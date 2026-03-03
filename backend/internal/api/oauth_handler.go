package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/rakutao/collection-gateway/internal/auth"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/oauth"
	"github.com/rakutao/collection-gateway/internal/repo"
)

// OAuthHandler handles OAuth login via Google and Apple.
type OAuthHandler struct {
	users          *repo.UserRepo
	oauthAccounts  *repo.OAuthRepo
	jwt            *auth.JWTManager
	googleClientID string
	appleBundleID  string
}

// NewOAuthHandler creates an OAuthHandler.
func NewOAuthHandler(
	users *repo.UserRepo,
	oauthAccounts *repo.OAuthRepo,
	jwt *auth.JWTManager,
	googleClientID, appleBundleID string,
) *OAuthHandler {
	return &OAuthHandler{
		users:          users,
		oauthAccounts:  oauthAccounts,
		jwt:            jwt,
		googleClientID: googleClientID,
		appleBundleID:  appleBundleID,
	}
}

type oauthRequest struct {
	IDToken string `json:"id_token"`
}

// HandleGoogle handles POST /api/v1/auth/google.
func (h *OAuthHandler) HandleGoogle(w http.ResponseWriter, r *http.Request) {
	var req oauthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.IDToken == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "id_token is required")
		return
	}

	oauthUser, err := oauth.VerifyGoogleToken(r.Context(), req.IDToken, h.googleClientID)
	if err != nil {
		log.Printf("[OAUTH] google verify error: %v", err)
		ErrorWithCode(w, r, http.StatusUnauthorized, 40007, "invalid OAuth token")
		return
	}

	h.handleOAuthLogin(w, r, "google", oauthUser)
}

// HandleApple handles POST /api/v1/auth/apple.
func (h *OAuthHandler) HandleApple(w http.ResponseWriter, r *http.Request) {
	var req oauthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.IDToken == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "id_token is required")
		return
	}

	oauthUser, err := oauth.VerifyAppleToken(r.Context(), req.IDToken, h.appleBundleID)
	if err != nil {
		log.Printf("[OAUTH] apple verify error: %v", err)
		ErrorWithCode(w, r, http.StatusUnauthorized, 40007, "invalid OAuth token")
		return
	}

	h.handleOAuthLogin(w, r, "apple", oauthUser)
}

// handleOAuthLogin is the shared logic: find or create user, then return JWT.
func (h *OAuthHandler) handleOAuthLogin(w http.ResponseWriter, r *http.Request, provider string, oauthUser *oauth.OAuthUser) {
	ctx := r.Context()

	user, err := h.findOrCreateOAuthUser(ctx, provider, oauthUser)
	if err != nil {
		log.Printf("[OAUTH] findOrCreate error: %v", err)
		ErrorWithCode(w, r, http.StatusInternalServerError, 40008, "OAuth provider error")
		return
	}

	token, err := h.jwt.Generate(user.ID, user.Email)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "internal error")
		return
	}

	Success(w, r, authResponse{
		Token:    token,
		UserID:   user.ID,
		Email:    user.Email,
		Nickname: user.Nickname,
	})
}

// findOrCreateOAuthUser looks up or creates a user for the given OAuth identity.
//  1. Check oauth_accounts for existing link → return that user.
//  2. Check users by email → link OAuth and return that user.
//  3. Create new user (empty password) → link OAuth → return.
func (h *OAuthHandler) findOrCreateOAuthUser(ctx context.Context, provider string, ou *oauth.OAuthUser) (*domain.User, error) {
	// 1. Already linked?
	userID, err := h.oauthAccounts.FindByProvider(ctx, provider, ou.ProviderID)
	if err == nil {
		return h.users.GetByID(ctx, userID)
	}
	if !errors.Is(err, repo.ErrOAuthNotFound) {
		return nil, err
	}

	// 2. Email match?
	if ou.Email != "" {
		user, err := h.users.GetByEmail(ctx, ou.Email)
		if err == nil {
			_ = h.oauthAccounts.Link(ctx, user.ID, provider, ou.ProviderID)
			return user, nil
		}
		if !errors.Is(err, repo.ErrUserNotFound) {
			return nil, err
		}
	}

	// 3. Create new user (password_hash empty → password login disabled).
	nickname := ou.Name
	if nickname == "" {
		nickname = provider + " user"
	}
	email := ou.Email
	if email == "" {
		// Apple may hide email; use a placeholder that won't conflict.
		email = provider + "_" + ou.ProviderID + "@oauth.local"
	}

	user, err := h.users.Create(ctx, email, nickname, "")
	if err != nil {
		// Race condition: another request created the user between our check and insert.
		if errors.Is(err, repo.ErrEmailExists) {
			user, err = h.users.GetByEmail(ctx, email)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	_ = h.oauthAccounts.Link(ctx, user.ID, provider, ou.ProviderID)
	return user, nil
}
