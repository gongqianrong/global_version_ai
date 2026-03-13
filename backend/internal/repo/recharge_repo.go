package repo

import (
	"context"
	"fmt"

	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// RechargeRepo handles recharge order persistence in PostgreSQL.
// It implements api.RechargeStore.
type RechargeRepo struct {
	db *db.DB
}

// NewRechargeRepo creates a RechargeRepo.
func NewRechargeRepo(d *db.DB) *RechargeRepo {
	return &RechargeRepo{db: d}
}

// Create inserts a new recharge order and returns it with the generated ID.
func (r *RechargeRepo) Create(ctx context.Context, order *domain.RechargeOrder) (*domain.RechargeOrder, error) {
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO recharge_orders (recharge_no, user_id, amount_jpy, pay_method, state)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		order.RechargeNo, order.UserID, order.AmountJPY, order.PayMethod, order.State,
	).Scan(&order.ID, &order.CreatedAt, &order.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("recharge_repo: create: %w", err)
	}
	return order, nil
}

// GetByRechargeNo returns a recharge order by its unique number.
func (r *RechargeRepo) GetByRechargeNo(ctx context.Context, rechargeNo string) (*domain.RechargeOrder, error) {
	var o domain.RechargeOrder
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, recharge_no, user_id, amount_jpy, pay_method, state, created_at, updated_at
		 FROM recharge_orders WHERE recharge_no = $1`,
		rechargeNo,
	).Scan(&o.ID, &o.RechargeNo, &o.UserID, &o.AmountJPY, &o.PayMethod, &o.State,
		&o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("recharge_repo: get: %w", err)
	}
	return &o, nil
}

// UpdateState updates the state of a recharge order.
func (r *RechargeRepo) UpdateState(ctx context.Context, rechargeNo string, state int) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE recharge_orders SET state = $1, updated_at = NOW() WHERE recharge_no = $2`,
		state, rechargeNo,
	)
	if err != nil {
		return fmt.Errorf("recharge_repo: update_state: %w", err)
	}
	return nil
}
