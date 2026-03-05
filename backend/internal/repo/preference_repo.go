package repo

import (
	"context"

	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

type PreferenceRepo struct {
	db *db.DB
}

func NewPreferenceRepo(d *db.DB) *PreferenceRepo {
	return &PreferenceRepo{db: d}
}

// SetPreferences replaces all preferences for a user.
func (r *PreferenceRepo) SetPreferences(ctx context.Context, userID int64, categories []string) ([]domain.UserPreference, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Delete existing preferences
	_, err = tx.Exec(ctx, `DELETE FROM user_preferences WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}

	// Insert new ones
	result := make([]domain.UserPreference, 0, len(categories))
	for _, cat := range categories {
		var p domain.UserPreference
		err = tx.QueryRow(ctx,
			`INSERT INTO user_preferences (user_id, category, weight)
			 VALUES ($1, $2, 1.0)
			 RETURNING id, user_id, category, weight, created_at`,
			userID, cat,
		).Scan(&p.ID, &p.UserID, &p.Category, &p.Weight, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPreferences returns all preferences for a user.
func (r *PreferenceRepo) GetPreferences(ctx context.Context, userID int64) ([]domain.UserPreference, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, category, weight, created_at
		 FROM user_preferences WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []domain.UserPreference
	for rows.Next() {
		var p domain.UserPreference
		if err := rows.Scan(&p.ID, &p.UserID, &p.Category, &p.Weight, &p.CreatedAt); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, rows.Err()
}
