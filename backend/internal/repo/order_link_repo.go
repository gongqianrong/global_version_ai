package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// OrderLinkRepo handles order_link persistence in PostgreSQL.
type OrderLinkRepo struct {
	db *db.DB
}

// NewOrderLinkRepo creates an OrderLinkRepo.
func NewOrderLinkRepo(d *db.DB) *OrderLinkRepo {
	return &OrderLinkRepo{db: d}
}

// Create inserts an order_link and its items within a single transaction.
func (r *OrderLinkRepo) Create(ctx context.Context, ol *domain.OrderLink) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("order_link_repo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO order_links (link_no, user_id, state, total_amount, remark)
		 VALUES ($1,$2,$3,$4,$5)
		 RETURNING id, created_at, updated_at`,
		ol.LinkNo, ol.UserID, ol.State, ol.TotalAmount, ol.Remark,
	).Scan(&ol.ID, &ol.CreatedAt, &ol.UpdatedAt)
	if err != nil {
		return fmt.Errorf("order_link_repo: insert order_link: %w", err)
	}

	for i := range ol.Items {
		ol.Items[i].OrderLinkID = ol.ID
		err = tx.QueryRow(ctx,
			`INSERT INTO order_link_items (order_link_id, goods_url, goods_name, goods_img, quantity, unit_price, remark)
			 VALUES ($1,$2,$3,$4,$5,$6,$7)
			 RETURNING id, created_at`,
			ol.Items[i].OrderLinkID, ol.Items[i].GoodsURL, ol.Items[i].GoodsName,
			ol.Items[i].GoodsImg, ol.Items[i].Quantity, ol.Items[i].UnitPrice,
			ol.Items[i].Remark,
		).Scan(&ol.Items[i].ID, &ol.Items[i].CreatedAt)
		if err != nil {
			return fmt.Errorf("order_link_repo: insert item[%d]: %w", i, err)
		}
	}

	return tx.Commit(ctx)
}

// GetByLinkNo returns an order_link with its items by link_no.
func (r *OrderLinkRepo) GetByLinkNo(ctx context.Context, linkNo string) (*domain.OrderLink, error) {
	var ol domain.OrderLink
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, link_no, user_id, state, total_amount, order_number, remark, created_at, updated_at
		 FROM order_links WHERE link_no = $1`,
		linkNo,
	).Scan(&ol.ID, &ol.LinkNo, &ol.UserID, &ol.State, &ol.TotalAmount,
		&ol.OrderNumber, &ol.Remark, &ol.CreatedAt, &ol.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("order_link_repo: get by link_no: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, order_link_id, goods_url, goods_name, goods_img, quantity, unit_price, remark, created_at
		 FROM order_link_items WHERE order_link_id = $1 ORDER BY id`,
		ol.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("order_link_repo: get items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.OrderLinkItem
		if err := rows.Scan(&item.ID, &item.OrderLinkID, &item.GoodsURL, &item.GoodsName,
			&item.GoodsImg, &item.Quantity, &item.UnitPrice, &item.Remark, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("order_link_repo: scan item: %w", err)
		}
		ol.Items = append(ol.Items, item)
	}
	return &ol, rows.Err()
}

// ListByUser returns paginated order_links for a user, optionally filtered by state.
// Pass state < 0 to skip state filtering.
func (r *OrderLinkRepo) ListByUser(ctx context.Context, userID int64, state, limit, offset int) ([]domain.OrderLink, int64, error) {
	countQ := `SELECT COUNT(*) FROM order_links WHERE user_id = $1`
	listQ := `SELECT id, link_no, user_id, state, total_amount, order_number, remark, created_at, updated_at
	           FROM order_links WHERE user_id = $1`

	if state >= 0 {
		countQ += fmt.Sprintf(" AND state = %d", state)
		listQ += fmt.Sprintf(" AND state = %d", state)
	}
	listQ += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQ, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("order_link_repo: count: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx, listQ, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("order_link_repo: list: %w", err)
	}
	defer rows.Close()

	var links []domain.OrderLink
	for rows.Next() {
		var ol domain.OrderLink
		if err := rows.Scan(&ol.ID, &ol.LinkNo, &ol.UserID, &ol.State, &ol.TotalAmount,
			&ol.OrderNumber, &ol.Remark, &ol.CreatedAt, &ol.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("order_link_repo: scan: %w", err)
		}
		links = append(links, ol)
	}
	return links, total, rows.Err()
}

// UpdateState atomically updates the order_link state, requiring the expected current state (optimistic lock).
func (r *OrderLinkRepo) UpdateState(ctx context.Context, linkNo string, fromState, toState int) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE order_links SET state = $1, updated_at = NOW()
		 WHERE link_no = $2 AND state = $3`,
		toState, linkNo, fromState,
	)
	if err != nil {
		return fmt.Errorf("order_link_repo: update_state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// QuoteItem represents the pricing for a single item in a quote.
type QuoteItem struct {
	ItemID    int64 `json:"item_id"`
	UnitPrice int64 `json:"unit_price"`
}

// Quote sets the quote for an order_link: updates each item's unit_price,
// computes total_amount, and transitions state from Pending to Quoted.
func (r *OrderLinkRepo) Quote(ctx context.Context, linkNo string, items []QuoteItem, totalAmount int64) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("order_link_repo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Lock and verify state.
	var olID int64
	var state int
	err = tx.QueryRow(ctx,
		`SELECT id, state FROM order_links WHERE link_no = $1 FOR UPDATE`,
		linkNo,
	).Scan(&olID, &state)
	if err != nil {
		return fmt.Errorf("order_link_repo: quote lock: %w", err)
	}
	if state != domain.OrderLinkStatePending {
		return fmt.Errorf("order_link_repo: invalid state for quote: %d", state)
	}

	// Update each item's unit_price.
	for _, qi := range items {
		tag, err := tx.Exec(ctx,
			`UPDATE order_link_items SET unit_price = $1
			 WHERE id = $2 AND order_link_id = $3`,
			qi.UnitPrice, qi.ItemID, olID,
		)
		if err != nil {
			return fmt.Errorf("order_link_repo: update item price: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("order_link_repo: item %d not found", qi.ItemID)
		}
	}

	// Update total_amount and state.
	_, err = tx.Exec(ctx,
		`UPDATE order_links SET total_amount = $1, state = $2, updated_at = NOW()
		 WHERE id = $3`,
		totalAmount, domain.OrderLinkStateQuoted, olID,
	)
	if err != nil {
		return fmt.Errorf("order_link_repo: update quote: %w", err)
	}

	return tx.Commit(ctx)
}

// SetOrderNumber sets the real order_number on an order_link after payment.
func (r *OrderLinkRepo) SetOrderNumber(ctx context.Context, linkNo string, orderNumber string) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE order_links SET order_number = $1, state = $2, updated_at = NOW()
		 WHERE link_no = $3 AND state = $4`,
		orderNumber, domain.OrderLinkStatePaid, linkNo, domain.OrderLinkStateQuoted,
	)
	if err != nil {
		return fmt.Errorf("order_link_repo: set_order_number: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}
