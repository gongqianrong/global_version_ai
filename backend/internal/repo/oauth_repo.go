package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/db"
)

// ErrOAuthNotFound is returned when no oauth_accounts row matches.
var ErrOAuthNotFound = errors.New("oauth account not found")

// OAuthRepo handles oauth_accounts persistence.
type OAuthRepo struct {
	db *db.DB
}

// NewOAuthRepo creates an OAuthRepo.
func NewOAuthRepo(d *db.DB) *OAuthRepo {
	return &OAuthRepo{db: d}
}

// FindByProvider returns the user_id linked to the given provider + provider_user_id.
func (r *OAuthRepo) FindByProvider(ctx context.Context, provider, providerUserID string) (int64, error) {
	var userID int64
	err := r.db.Pool.QueryRow(ctx,
		`SELECT user_id FROM oauth_accounts WHERE provider = $1 AND provider_user_id = $2`,
		provider, providerUserID,
	).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrOAuthNotFound
		}
		return 0, fmt.Errorf("oauth_repo: find_by_provider: %w", err)
	}
	return userID, nil
}

// Link inserts an oauth_accounts row. If the row already exists it does nothing.
func (r *OAuthRepo) Link(ctx context.Context, userID int64, provider, providerUserID string) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO oauth_accounts (user_id, provider, provider_user_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (provider, provider_user_id) DO NOTHING`,
		userID, provider, providerUserID,
	)
	if err != nil {
		return fmt.Errorf("oauth_repo: link: %w", err)
	}
	return nil
}
