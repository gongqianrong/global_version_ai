package repo

import (
	"context"
	"time"

	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

type SearchHistoryRepo struct {
	db *db.DB
}

func NewSearchHistoryRepo(d *db.DB) *SearchHistoryRepo {
	return &SearchHistoryRepo{db: d}
}

// Record inserts a search event.
func (r *SearchHistoryRepo) Record(ctx context.Context, rec *domain.SearchRecord) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO search_history (user_id, keyword, keyword_ja, platform)
		 VALUES ($1, $2, $3, $4)`,
		rec.UserID, rec.Keyword, rec.KeywordJA, rec.Platform,
	)
	return err
}

// ListRecent returns recent search records for a user within the given number of days.
func (r *SearchHistoryRepo) ListRecent(ctx context.Context, userID int64, days int) ([]domain.SearchRecord, error) {
	since := time.Now().AddDate(0, 0, -days)
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, keyword, keyword_ja, platform, created_at
		 FROM search_history
		 WHERE user_id = $1 AND created_at > $2
		 ORDER BY created_at DESC LIMIT 200`,
		userID, since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []domain.SearchRecord
	for rows.Next() {
		var s domain.SearchRecord
		if err := rows.Scan(&s.ID, &s.UserID, &s.Keyword, &s.KeywordJA, &s.Platform, &s.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, s)
	}
	return records, rows.Err()
}
