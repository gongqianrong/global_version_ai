package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/db"
)

// FollowedSeller represents a row in the followed_sellers table.
type FollowedSeller struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	SellerID   string    `json:"seller_id"`
	SellerName string    `json:"seller_name"`
	FollowedAt time.Time `json:"followed_at"`
}

// SellerFollowRepo handles followed-seller persistence in PostgreSQL.
type SellerFollowRepo struct {
	db *db.DB
}

// NewSellerFollowRepo creates a SellerFollowRepo.
func NewSellerFollowRepo(d *db.DB) *SellerFollowRepo {
	return &SellerFollowRepo{db: d}
}

// Follow inserts a follow record, ignoring duplicates (updates seller_name on conflict).
func (r *SellerFollowRepo) Follow(ctx context.Context, userID int64, sellerID, sellerName string) (*FollowedSeller, error) {
	var item FollowedSeller
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO followed_sellers (user_id, seller_id, seller_name)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, seller_id) DO UPDATE SET seller_name = EXCLUDED.seller_name
		 RETURNING id, user_id, seller_id, seller_name, followed_at`,
		userID, sellerID, sellerName,
	).Scan(&item.ID, &item.UserID, &item.SellerID, &item.SellerName, &item.FollowedAt)
	if err != nil {
		return nil, fmt.Errorf("seller_follow_repo: follow: %w", err)
	}
	return &item, nil
}

// Unfollow deletes a follow record.
func (r *SellerFollowRepo) Unfollow(ctx context.Context, userID int64, sellerID string) error {
	tag, err := r.db.Pool.Exec(ctx,
		`DELETE FROM followed_sellers WHERE user_id = $1 AND seller_id = $2`,
		userID, sellerID,
	)
	if err != nil {
		return fmt.Errorf("seller_follow_repo: unfollow: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

// ListByUser returns followed sellers for a user with offset-based pagination.
func (r *SellerFollowRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]FollowedSeller, int64, error) {
	var total int64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM followed_sellers WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("seller_follow_repo: count: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, seller_id, seller_name, followed_at
		 FROM followed_sellers WHERE user_id = $1
		 ORDER BY followed_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("seller_follow_repo: list: %w", err)
	}
	defer rows.Close()

	var items []FollowedSeller
	for rows.Next() {
		var item FollowedSeller
		if err := rows.Scan(&item.ID, &item.UserID, &item.SellerID, &item.SellerName, &item.FollowedAt); err != nil {
			return nil, 0, fmt.Errorf("seller_follow_repo: list scan: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}

// BatchIsFollowing checks which of the given seller IDs are followed by the user.
func (r *SellerFollowRepo) BatchIsFollowing(ctx context.Context, userID int64, sellerIDs []string) (map[string]bool, error) {
	if len(sellerIDs) == 0 {
		return map[string]bool{}, nil
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT seller_id FROM followed_sellers WHERE user_id = $1 AND seller_id = ANY($2)`,
		userID, sellerIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("seller_follow_repo: batch_check: %w", err)
	}
	defer rows.Close()

	result := make(map[string]bool, len(sellerIDs))
	for rows.Next() {
		var sid string
		if err := rows.Scan(&sid); err != nil {
			return nil, fmt.Errorf("seller_follow_repo: batch_check scan: %w", err)
		}
		result[sid] = true
	}
	return result, rows.Err()
}
