package repo

import (
	"context"
	"time"

	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

type RecWeightRepo struct {
	db *db.DB
}

func NewRecWeightRepo(d *db.DB) *RecWeightRepo {
	return &RecWeightRepo{db: d}
}

// UpsertWeights replaces all weights for a user atomically.
func (r *RecWeightRepo) UpsertWeights(ctx context.Context, userID int64, weights []domain.RecWeight) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM user_rec_weights WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}

	for _, w := range weights {
		_, err = tx.Exec(ctx,
			`INSERT INTO user_rec_weights (user_id, signal_type, dimension, value, weight, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			userID, w.SignalType, w.Dimension, w.Value, w.Weight, time.Now(),
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetWeights returns all recommendation weights for a user.
func (r *RecWeightRepo) GetWeights(ctx context.Context, userID int64) ([]domain.RecWeight, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, signal_type, dimension, value, weight, updated_at
		 FROM user_rec_weights WHERE user_id = $1 ORDER BY weight DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var weights []domain.RecWeight
	for rows.Next() {
		var w domain.RecWeight
		if err := rows.Scan(&w.ID, &w.UserID, &w.SignalType, &w.Dimension, &w.Value, &w.Weight, &w.UpdatedAt); err != nil {
			return nil, err
		}
		weights = append(weights, w)
	}
	return weights, rows.Err()
}

// GetActiveUserIDs returns user IDs that have had any activity in the given number of days.
func (r *RecWeightRepo) GetActiveUserIDs(ctx context.Context, days int) ([]int64, error) {
	since := time.Now().AddDate(0, 0, -days)
	rows, err := r.db.Pool.Query(ctx,
		`SELECT DISTINCT user_id FROM (
			SELECT user_id FROM browsing_history WHERE viewed_at > $1
			UNION
			SELECT user_id FROM search_history WHERE created_at > $1
			UNION
			SELECT user_id FROM cart_items WHERE updated_at > $1
			UNION
			SELECT user_id FROM favorites WHERE added_at > $1
			UNION
			SELECT user_id FROM orders WHERE created_at > $1
			UNION
			SELECT user_id FROM followed_sellers WHERE followed_at > $1
			UNION
			SELECT user_id FROM user_preferences WHERE created_at > $1
		 ) active_users`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
