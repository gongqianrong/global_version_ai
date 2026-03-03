package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/rakutao/collection-gateway/internal/auth"
	"github.com/rakutao/collection-gateway/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles user registration and login.
type AuthHandler struct {
	users *repo.UserRepo
	jwt   *auth.JWTManager
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(users *repo.UserRepo, jwt *auth.JWTManager) *AuthHandler {
	return &AuthHandler{users: users, jwt: jwt}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token    string `json:"token"`
	UserID   int64  `json:"user_id"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
}

// HandleRegister handles POST /api/v1/auth/register.
func (h *AuthHandler) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Nickname = strings.TrimSpace(req.Nickname)

	if req.Email == "" || req.Password == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40003, "email and password are required")
		return
	}
	if len(req.Password) < 6 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40003, "password too short")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "internal error")
		return
	}

	user, err := h.users.Create(r.Context(), req.Email, req.Nickname, string(hash))
	if err != nil {
		if errors.Is(err, repo.ErrEmailExists) {
			ErrorWithCode(w, r, http.StatusConflict, 40901, "email already registered")
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "internal error")
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

// HandleLogin handles POST /api/v1/auth/login.
func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)

	if req.Email == "" || req.Password == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40003, "email and password are required")
		return
	}

	user, err := h.users.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, repo.ErrUserNotFound) {
			ErrorWithCode(w, r, http.StatusUnauthorized, 40100, "invalid credentials")
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "internal error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		ErrorWithCode(w, r, http.StatusUnauthorized, 40100, "invalid credentials")
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
