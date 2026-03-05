package repo

import (
	"context"
	"time"

	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

type BrowsingRepo struct {
	db *db.DB
}

func NewBrowsingRepo(d *db.DB) *BrowsingRepo {
	return &BrowsingRepo{db: d}
}

// Record inserts a browsing event. Deduplicates: same user+product within 5 minutes is skipped.
func (r *BrowsingRepo) Record(ctx context.Context, rec *domain.BrowsingRecord) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO browsing_history (user_id, product_id, category, brand, seller_id, platform)
		 SELECT $1, $2, $3, $4, $5, $6
		 WHERE NOT EXISTS (
		     SELECT 1 FROM browsing_history
		     WHERE user_id = $1 AND product_id = $2 AND viewed_at > NOW() - INTERVAL '5 minutes'
		 )`,
		rec.UserID, rec.ProductID, rec.Category, rec.Brand, rec.SellerID, rec.Platform,
	)
	return err
}

// ListRecent returns recent browsing records for a user within the given number of days.
func (r *BrowsingRepo) ListRecent(ctx context.Context, userID int64, days int) ([]domain.BrowsingRecord, error) {
	since := time.Now().AddDate(0, 0, -days)
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, product_id, category, brand, seller_id, platform, viewed_at
		 FROM browsing_history
		 WHERE user_id = $1 AND viewed_at > $2
		 ORDER BY viewed_at DESC LIMIT 500`,
		userID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []domain.BrowsingRecord
	for rows.Next() {
		var b domain.BrowsingRecord
		if err := rows.Scan(&b.ID, &b.UserID, &b.ProductID, &b.Category, &b.Brand, &b.SellerID, &b.Platform, &b.ViewedAt); err != nil {
			return nil, err
		}
		records = append(records, b)
	}
	return records, rows.Err()
}

// RecentProductIDs returns product IDs the user has recently viewed (for exclusion).
func (r *BrowsingRepo) RecentProductIDs(ctx context.Context, userID int64, limit int) ([]string, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT DISTINCT product_id FROM browsing_history
		 WHERE user_id = $1 ORDER BY product_id LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
