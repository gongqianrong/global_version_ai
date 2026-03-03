package api

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/rakutao/collection-gateway/internal/auth"
	"github.com/rakutao/collection-gateway/internal/cache"
	"github.com/rakutao/collection-gateway/internal/email"
	"github.com/rakutao/collection-gateway/internal/repo"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles user registration and login.
type AuthHandler struct {
	users       *repo.UserRepo
	jwt         *auth.JWTManager
	redis       *cache.Client
	emailSender *email.Sender
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(users *repo.UserRepo, jwt *auth.JWTManager, redis *cache.Client, emailSender *email.Sender) *AuthHandler {
	return &AuthHandler{users: users, jwt: jwt, redis: redis, emailSender: emailSender}
}

type sendCodeRequest struct {
	Email string `json:"email"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Nickname string `json:"nickname"`
	Code     string `json:"code"`
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

// HandleSendCode handles POST /api/v1/auth/send-code.
func (h *AuthHandler) HandleSendCode(w http.ResponseWriter, r *http.Request) {
	var req sendCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)

	// Basic email format validation.
	if !strings.Contains(req.Email, "@") || !strings.Contains(req.Email, ".") {
		ErrorWithCode(w, r, http.StatusBadRequest, 40004, "invalid email format")
		return
	}

	ctx := r.Context()

	// Check 60s cooldown.
	cdKey := "verify_cd:" + req.Email
	existing, _ := h.redis.Get(ctx, cdKey)
	if existing != nil {
		ErrorWithCode(w, r, http.StatusTooManyRequests, 40005, "please wait before requesting another code")
		return
	}

	// Generate 6-digit code.
	code := generateCode()

	// Store code in Redis with 10min TTL.
	verifyKey := "verify:" + req.Email
	if err := h.redis.Set(ctx, verifyKey, []byte(code), 10*time.Minute); err != nil {
		log.Printf("[AUTH] redis set verify code error: %v", err)
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "internal error")
		return
	}

	// Store cooldown in Redis with 60s TTL.
	if err := h.redis.Set(ctx, cdKey, []byte("1"), 60*time.Second); err != nil {
		log.Printf("[AUTH] redis set cooldown error: %v", err)
	}

	// Send email.
	if err := h.emailSender.SendVerifyCode(req.Email, code); err != nil {
		log.Printf("[AUTH] send verify code email error: %v", err)
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to send email")
		return
	}

	Success(w, r, map[string]string{"message": "verification code sent"})
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
	req.Code = strings.TrimSpace(req.Code)

	if req.Email == "" || req.Password == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40003, "email and password are required")
		return
	}
	if len(req.Password) < 6 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40003, "password too short")
		return
	}
	if req.Code == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40006, "verification code is required")
		return
	}

	// Verify code from Redis.
	ctx := r.Context()
	verifyKey := "verify:" + req.Email
	stored, err := h.redis.Get(ctx, verifyKey)
	if err != nil {
		log.Printf("[AUTH] redis get verify code error: %v", err)
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "internal error")
		return
	}
	if stored == nil || string(stored) != req.Code {
		ErrorWithCode(w, r, http.StatusBadRequest, 40006, "invalid or expired verification code")
		return
	}

	// Delete code to prevent reuse.
	_ = h.redis.Del(ctx, verifyKey)

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

// generateCode returns a cryptographically random 6-digit string.
func generateCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "000000"
	}
	return fmt.Sprintf("%06d", n.Int64())
}
