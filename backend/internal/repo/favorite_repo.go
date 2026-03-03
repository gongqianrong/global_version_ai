package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/db"
)

// FavoriteItem represents a row in the favorites table.
type FavoriteItem struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	ProductID string    `json:"product_id"`
	AddedAt   time.Time `json:"added_at"`
}

// FavoriteRepo handles favorite persistence in PostgreSQL.
type FavoriteRepo struct {
	db *db.DB
}

// NewFavoriteRepo creates a FavoriteRepo.
func NewFavoriteRepo(d *db.DB) *FavoriteRepo {
	return &FavoriteRepo{db: d}
}

// Add inserts a favorite, ignoring duplicates.
func (r *FavoriteRepo) Add(ctx context.Context, userID int64, productID string) (*FavoriteItem, error) {
	var item FavoriteItem
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO favorites (user_id, product_id)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id, product_id) DO UPDATE SET user_id = favorites.user_id
		 RETURNING id, user_id, product_id, added_at`,
		userID, productID,
	).Scan(&item.ID, &item.UserID, &item.ProductID, &item.AddedAt)
	if err != nil {
		return nil, fmt.Errorf("favorite_repo: add: %w", err)
	}
	return &item, nil
}

// Remove deletes a favorite.
func (r *FavoriteRepo) Remove(ctx context.Context, userID int64, productID string) error {
	tag, err := r.db.Pool.Exec(ctx,
		`DELETE FROM favorites WHERE user_id = $1 AND product_id = $2`,
		userID, productID,
	)
	if err != nil {
		return fmt.Errorf("favorite_repo: remove: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListByUser returns favorites for a user with offset-based pagination.
func (r *FavoriteRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]FavoriteItem, int64, error) {
	// Count total.
	var total int64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM favorites WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("favorite_repo: count: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, product_id, added_at
		 FROM favorites WHERE user_id = $1
		 ORDER BY added_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("favorite_repo: list: %w", err)
	}
	defer rows.Close()

	var items []FavoriteItem
	for rows.Next() {
		var item FavoriteItem
		if err := rows.Scan(&item.ID, &item.UserID, &item.ProductID, &item.AddedAt); err != nil {
			return nil, 0, fmt.Errorf("favorite_repo: list scan: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

// BatchIsFavorited checks which of the given product IDs are favorited by the user.
// Returns a map of productID → true for favorited items.
func (r *FavoriteRepo) BatchIsFavorited(ctx context.Context, userID int64, productIDs []string) (map[string]bool, error) {
	if len(productIDs) == 0 {
		return map[string]bool{}, nil
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT product_id FROM favorites WHERE user_id = $1 AND product_id = ANY($2)`,
		userID, productIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("favorite_repo: batch_check: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool, len(productIDs))
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			return nil, fmt.Errorf("favorite_repo: batch_check scan: %w", err)
		}
		result[pid] = true
	}
	return result, rows.Err()
}
