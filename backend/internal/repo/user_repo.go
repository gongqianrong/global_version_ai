package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// ErrUserNotFound is returned when a user lookup yields no results.
var ErrUserNotFound = errors.New("user not found")

// ErrEmailExists is returned when a duplicate email is detected on insert.
var ErrEmailExists = errors.New("email already exists")

// UserRepo handles user persistence in PostgreSQL.
type UserRepo struct {
	db *db.DB
}

// NewUserRepo creates a UserRepo.
func NewUserRepo(d *db.DB) *UserRepo {
	return &UserRepo{db: d}
}

// Create inserts a new user and returns the populated User (with ID and timestamps).
func (r *UserRepo) Create(ctx context.Context, email, nickname, passwordHash string) (*domain.User, error) {
	// 生成globalAccountID（使用email的哈希或UUID）
	globalAccountID := fmt.Sprintf("INTL_%d", hashString(email))
	
	var u domain.User
	err := r.db.Pool.QueryRow(ctx,
		`INSERT INTO users (email, nickname, password_hash, global_account_id)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, email, nickname, password_hash, global_account_id, created_at, updated_at`,
		email, nickname, passwordHash, globalAccountID,
	).Scan(&u.ID, &u.Email, &u.Nickname, &u.PasswordHash, &u.GlobalAccountID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if isDuplicateKey(err) {
			return nil, ErrEmailExists
		}
		return nil, fmt.Errorf("user_repo: create: %w", err)
	}
	return &u, nil
}

// hashString generates a simple hash from string
func hashString(s string) uint64 {
	var hash uint64
	for i := 0; i < len(s); i++ {
		hash = hash*31 + uint64(s[i])
	}
	return hash % 1000000000
}

// GetByEmail fetches a user by email address.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, email, nickname, password_hash, global_account_id, created_at, updated_at
		 FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.Nickname, &u.PasswordHash, &u.GlobalAccountID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("user_repo: get_by_email: %w", err)
	}
	return &u, nil
}

// GetByID fetches a user by primary key.
func (r *UserRepo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	var u domain.User
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, email, nickname, password_hash, global_account_id, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.Nickname, &u.PasswordHash, &u.GlobalAccountID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("user_repo: get_by_id: %w", err)
	}
	return &u, nil
}

// isDuplicateKey checks for PostgreSQL unique constraint violation (SQLSTATE 23505).
func isDuplicateKey(err error) bool {
	return err != nil && contains(err.Error(), "23505")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
