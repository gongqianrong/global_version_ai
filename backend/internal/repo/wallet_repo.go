package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// ErrInsufficientBalance is returned when a debit would make balance negative.
var ErrInsufficientBalance = errors.New("insufficient balance")

// WalletRepo handles wallet persistence in PostgreSQL.
type WalletRepo struct {
	db *db.DB
}

// NewWalletRepo creates a WalletRepo.
func NewWalletRepo(d *db.DB) *WalletRepo {
	return &WalletRepo{db: d}
}

// GetOrCreateWallet returns the wallet for the given user, creating one with
// zero balance if it doesn't exist yet.
func (r *WalletRepo) GetOrCreateWallet(ctx context.Context, userID int64) (*domain.Wallet, error) {
	var w domain.Wallet
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO wallets (user_id) VALUES ($1)
		 ON CONFLICT (user_id) DO UPDATE SET user_id = wallets.user_id
		 RETURNING id, user_id, balance, created_at, updated_at`,
		userID,
	).Scan(&w.ID, &w.UserID, &w.Balance, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("wallet_repo: get_or_create: %w", err)
	}
	return &w, nil
}

// Adjust atomically updates the wallet balance and inserts a transaction record.
// For debits (negative amount), it checks that the resulting balance >= 0.
func (r *WalletRepo) Adjust(ctx context.Context, userID int64, amount int64, txType, description string, relatedOrder *string) (*domain.WalletTransaction, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("wallet_repo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Ensure wallet exists and lock the row.
	var newBalance int64
	err = tx.QueryRow(ctx,
		`INSERT INTO wallets (user_id) VALUES ($1)
		 ON CONFLICT (user_id) DO UPDATE SET
		   balance = wallets.balance + $2,
		   updated_at = NOW()
		 RETURNING balance`,
		userID, amount,
	).Scan(&newBalance)
	if err != nil {
		return nil, fmt.Errorf("wallet_repo: update balance: %w", err)
	}

	if newBalance < 0 {
		return nil, ErrInsufficientBalance
	}

	// Insert transaction record.
	var wt domain.WalletTransaction
	err = tx.QueryRow(ctx,
		`INSERT INTO wallet_transactions (user_id, type, amount, balance_after, description, related_order)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, type, amount, balance_after, description, related_order, created_at`,
		userID, txType, amount, newBalance, description, relatedOrder,
	).Scan(&wt.ID, &wt.UserID, &wt.Type, &wt.Amount, &wt.BalanceAfter, &wt.Description, &wt.RelatedOrder, &wt.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("wallet_repo: insert tx: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("wallet_repo: commit: %w", err)
	}
	return &wt, nil
}

// ListTransactions returns paginated transaction history for a user.
func (r *WalletRepo) ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int64, error) {
	var total int64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM wallet_transactions WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("wallet_repo: count tx: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, user_id, type, amount, balance_after, description, related_order, created_at
		 FROM wallet_transactions WHERE user_id = $1
		 ORDER BY created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("wallet_repo: list tx: %w", err)
	}
	defer rows.Close()

	var items []domain.WalletTransaction
	for rows.Next() {
		var wt domain.WalletTransaction
		if err := rows.Scan(&wt.ID, &wt.UserID, &wt.Type, &wt.Amount, &wt.BalanceAfter, &wt.Description, &wt.RelatedOrder, &wt.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("wallet_repo: list tx scan: %w", err)
		}
		items = append(items, wt)
	}
	return items, total, rows.Err()
}

// IsAdmin checks whether a user has admin privileges.
func (r *WalletRepo) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	var isAdmin bool
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COALESCE(is_admin, FALSE) FROM users WHERE id = $1`, userID,
	).Scan(&isAdmin)
	if err != nil {
		return false, fmt.Errorf("wallet_repo: is_admin: %w", err)
	}
	return isAdmin, nil
}
