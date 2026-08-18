package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// OrderRepo handles order persistence in PostgreSQL.
type OrderRepo struct {
	db *db.DB
}

// NewOrderRepo creates an OrderRepo.
func NewOrderRepo(d *db.DB) *OrderRepo {
	return &OrderRepo{db: d}
}

// CreateOrder inserts an order and its details within a single transaction.
// It returns the created order with its generated ID.
func (r *OrderRepo) CreateOrder(ctx context.Context, order *domain.Order, details []domain.OrderDetail) (*domain.Order, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("order_repo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO orders (order_number, user_id, order_state, order_total_jp, commission_fee_jp,
		  shipping_fee_jp, order_inprice_jp, order_paytype, order_remark, order_purchase_type)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		 RETURNING id, created_at, updated_at`,
		order.OrderNumber, order.UserID, order.OrderState, order.OrderTotalJp,
		order.CommissionFeeJp, order.ShippingFeeJp, order.OrderInpriceJp,
		order.OrderPaytype, order.OrderRemark, order.OrderPurchaseType,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("order_repo: insert order: %w", err)
	}

	for i := range details {
		details[i].OrderID = order.ID
		err = tx.QueryRow(ctx,
			`INSERT INTO order_details (order_id, goods_mid, goods_name, goods_num, goods_img,
			  goods_url, goods_amount_jp, commission_fee_jp, shipping_fee_jp,
			  seller_id, seller_name, platform, condition, state)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			 RETURNING id, created_at`,
			details[i].OrderID, details[i].GoodsMid, details[i].GoodsName,
			details[i].GoodsNum, details[i].GoodsImg, details[i].GoodsUrl,
			details[i].GoodsAmountJp, details[i].CommissionFeeJp, details[i].ShippingFeeJp,
			details[i].SellerID, details[i].SellerName, details[i].Platform,
			details[i].Condition, details[i].State,
		).Scan(&details[i].ID, &details[i].CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("order_repo: insert detail[%d]: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("order_repo: commit: %w", err)
	}
	return order, nil
}

// GetByOrderNumber returns an order and its details by order number.
func (r *OrderRepo) GetByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, []domain.OrderDetail, error) {
	var o domain.Order
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, order_number, user_id, order_state, order_total_jp, commission_fee_jp,
		  shipping_fee_jp, order_inprice_jp, order_paytype, order_remark, order_purchase_type,
		  created_at, updated_at
		 FROM orders WHERE order_number = $1`,
		orderNumber,
	).Scan(&o.ID, &o.OrderNumber, &o.UserID, &o.OrderState, &o.OrderTotalJp,
		&o.CommissionFeeJp, &o.ShippingFeeJp, &o.OrderInpriceJp, &o.OrderPaytype,
		&o.OrderRemark, &o.OrderPurchaseType, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("order_repo: get order: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, order_id, goods_mid, goods_name, goods_num, goods_img, goods_url,
		  goods_amount_jp, commission_fee_jp, shipping_fee_jp,
		  seller_id, seller_name, platform, condition, state, created_at
		 FROM order_details WHERE order_id = $1 ORDER BY id`,
		o.ID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("order_repo: get details: %w", err)
	}
	defer rows.Close()

	var details []domain.OrderDetail
	for rows.Next() {
		var d domain.OrderDetail
		if err := rows.Scan(&d.ID, &d.OrderID, &d.GoodsMid, &d.GoodsName, &d.GoodsNum,
			&d.GoodsImg, &d.GoodsUrl, &d.GoodsAmountJp, &d.CommissionFeeJp,
			&d.ShippingFeeJp, &d.SellerID, &d.SellerName, &d.Platform,
			&d.Condition, &d.State, &d.CreatedAt); err != nil {
			return nil, nil, fmt.Errorf("order_repo: scan detail: %w", err)
		}
		details = append(details, d)
	}
	return &o, details, rows.Err()
}

// UpdateState atomically updates order state, requiring the expected current state.
func (r *OrderRepo) UpdateState(ctx context.Context, orderNumber string, fromState, toState int) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE orders SET order_state = $1, updated_at = NOW()
		 WHERE order_number = $2 AND order_state = $3`,
		toState, orderNumber, fromState,
	)
	if err != nil {
		return fmt.Errorf("order_repo: update_state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// UpdateStateWithTx updates order state within an existing transaction.
// CRITICAL: Caller must manage the transaction lifecycle (commit/rollback).
func (r *OrderRepo) UpdateStateWithTx(ctx context.Context, tx pgx.Tx, orderNumber string, fromState, toState int) error {
	tag, err := tx.Exec(ctx,
		`UPDATE orders SET order_state = $1, updated_at = NOW()
		 WHERE order_number = $2 AND order_state = $3`,
		toState, orderNumber, fromState,
	)
	if err != nil {
		return fmt.Errorf("order_repo: update_state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListByUser returns paginated orders for a user, optionally filtered by state.
// Pass state < 0 to skip state filtering.
func (r *OrderRepo) ListByUser(ctx context.Context, userID int64, state, limit, offset int) ([]domain.Order, int64, error) {
	var total int64
	countQ := `SELECT COUNT(*) FROM orders WHERE user_id = $1`
	listQ := `SELECT id, order_number, user_id, order_state, order_total_jp, commission_fee_jp,
	  shipping_fee_jp, order_inprice_jp, order_paytype, order_remark, order_purchase_type,
	  created_at, updated_at FROM orders WHERE user_id = $1`

	if state >= 0 {
		countQ += fmt.Sprintf(" AND order_state = %d", state)
		listQ += fmt.Sprintf(" AND order_state = %d", state)
	}
	listQ += ` ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	if err := r.db.Pool.QueryRow(ctx, countQ, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("order_repo: count: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx, listQ, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("order_repo: list: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.UserID, &o.OrderState, &o.OrderTotalJp,
			&o.CommissionFeeJp, &o.ShippingFeeJp, &o.OrderInpriceJp, &o.OrderPaytype,
			&o.OrderRemark, &o.OrderPurchaseType, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("order_repo: scan: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}

// GetDetailsByOrderIDs returns order details for multiple orders at once.
func (r *OrderRepo) GetDetailsByOrderIDs(ctx context.Context, orderIDs []int64) (map[int64][]domain.OrderDetail, error) {
	if len(orderIDs) == 0 {
		return nil, nil
	}
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, order_id, goods_mid, goods_name, goods_num, goods_img, goods_url,
		  goods_amount_jp, commission_fee_jp, shipping_fee_jp,
		  seller_id, seller_name, platform, condition, state, created_at
		 FROM order_details WHERE order_id = ANY($1) ORDER BY id`,
		orderIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("order_repo: get details by ids: %w", err)
	}
	defer rows.Close()

	result := make(map[int64][]domain.OrderDetail)
	for rows.Next() {
		var d domain.OrderDetail
		if err := rows.Scan(&d.ID, &d.OrderID, &d.GoodsMid, &d.GoodsName, &d.GoodsNum,
			&d.GoodsImg, &d.GoodsUrl, &d.GoodsAmountJp, &d.CommissionFeeJp,
			&d.ShippingFeeJp, &d.SellerID, &d.SellerName, &d.Platform,
			&d.Condition, &d.State, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("order_repo: scan detail: %w", err)
		}
		result[d.OrderID] = append(result[d.OrderID], d)
	}
	return result, rows.Err()
}

// CancelExpired cancels all pending orders older than the given timeout.
// Returns the number of cancelled orders.
func (r *OrderRepo) CancelExpired(ctx context.Context, timeout time.Duration) (int64, error) {
	cutoff := time.Now().Add(-timeout)
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE orders SET order_state = $1, updated_at = NOW()
		 WHERE order_state = $2 AND created_at < $3`,
		domain.OrderStateCancelled, domain.OrderStatePending, cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("order_repo: cancel_expired: %w", err)
	}
	return tag.RowsAffected(), nil
}

// PayOrderAtomic performs atomic payment: deduct wallet + update order state in ONE transaction.
// This is the CRITICAL FIX for P0 payment transaction consistency issue.
// Returns the wallet transaction and updated order state on success.
func (r *OrderRepo) PayOrderAtomic(
	ctx context.Context,
	orderNumber string,
	userID int64,
	amount int64,
	walletRepo *WalletRepo,
) (*domain.WalletTransaction, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("order_repo: begin payment tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Step 1: Deduct wallet balance (within this transaction)
	debitAmount := -amount
	desc := fmt.Sprintf("Order %s payment", orderNumber)
	wtx, err := walletRepo.AdjustWithTx(ctx, tx, userID, debitAmount, domain.TxTypePurchase, desc, &orderNumber)
	if err != nil {
		return nil, fmt.Errorf("order_repo: payment deduct wallet: %w", err)
	}

	// Step 2: Update order state to Paid (within same transaction)
	if err := r.UpdateStateWithTx(ctx, tx, orderNumber, domain.OrderStatePending, domain.OrderStatePaid); err != nil {
		return nil, fmt.Errorf("order_repo: payment update state: %w", err)
	}

	// Step 3: Commit transaction (both operations succeed or both fail)
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("order_repo: payment commit: %w", err)
	}

	return wtx, nil
}
