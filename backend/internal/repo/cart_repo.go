package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/db"
)

// CartItem represents a row in the cart_items table.
type CartItem struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	ProductID  string    `json:"product_id"`
	Quantity   int       `json:"quantity"`
	PriceAtAdd int64     `json:"price_at_add"`
	AddedAt    time.Time `json:"added_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CartRepo handles cart persistence in PostgreSQL.
type CartRepo struct {
	db *db.DB
}

// NewCartRepo creates a CartRepo.
func NewCartRepo(d *db.DB) *CartRepo {
	return &CartRepo{db: d}
}

// Add inserts a cart item or increments quantity on conflict.
func (r *CartRepo) Add(ctx context.Context, userID int64, productID string, quantity int, priceAtAdd int64) (*CartItem, error) {
	var item CartItem
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO cart_items (user_id, product_id, quantity, price_at_add)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id, product_id)
		 DO UPDATE SET quantity = cart_items.quantity + EXCLUDED.quantity, updated_at = NOW()
		 RETURNING id, user_id, product_id, quantity, price_at_add, added_at, updated_at`,
		userID, productID, quantity, priceAtAdd,
	).Scan(&item.ID, &item.UserID, &item.ProductID, &item.Quantity, &item.PriceAtAdd, &item.AddedAt, &item.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("cart_repo: add: %w", err)
	}
	return &item, nil
}

// UpdateQuantity sets the quantity for a specific cart item.
func (r *CartRepo) UpdateQuantity(ctx context.Context, userID int64, productID string, quantity int) error {
	tag, err := r.db.Pool.Exec(ctx,
		`UPDATE cart_items SET quantity = $1, updated_at = NOW()
		 WHERE user_id = $2 AND product_id = $3`,
		quantity, userID, productID,
	)
	if err != nil {
		return fmt.Errorf("cart_repo: update_quantity: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// Remove deletes a cart item.
func (r *CartRepo) Remove(ctx context.Context, userID int64, productID string) error {
	tag, err := r.db.Pool.Exec(ctx,
		`DELETE FROM cart_items WHERE user_id = $1 AND product_id = $2`,
		userID, productID,
	)
	if err != nil {
		return fmt.Errorf("cart_repo: remove: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListByUser returns all cart items for a user, ordered by added_at desc.
func (r *CartRepo) ListByUser(ctx context.Context, userID int64) ([]CartItem, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, product_id, quantity, price_at_add, added_at, updated_at
		 FROM cart_items WHERE user_id = $1 ORDER BY added_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("cart_repo: list: %w", err)
	}
	defer rows.Close()

	var items []CartItem
	for rows.Next() {
		var item CartItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.ProductID, &item.Quantity, &item.PriceAtAdd, &item.AddedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("cart_repo: list scan: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
