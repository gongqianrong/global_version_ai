package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rakutao/collection-gateway/internal/auth"
)

func newTestJWTManager() *auth.JWTManager {
	return auth.NewJWTManager("test-secret-key", 72*time.Hour)
}

func TestAuthRequired_NoHeader(t *testing.T) {
	handler := AuthRequired(newTestJWTManager())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != 40100 {
		t.Errorf("code = %d, want 40100", resp.Code)
	}
}

func TestAuthRequired_InvalidFormat(t *testing.T) {
	handler := AuthRequired(newTestJWTManager())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestAuthRequired_InvalidToken(t *testing.T) {
	handler := AuthRequired(newTestJWTManager())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-string")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != 40101 {
		t.Errorf("code = %d, want 40101", resp.Code)
	}
}

func TestAuthRequired_ValidToken(t *testing.T) {
	jm := newTestJWTManager()
	token, _ := jm.Generate(99, "test@example.com")

	var gotUserID int64
	handler := AuthRequired(jm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if gotUserID != 99 {
		t.Errorf("userID = %d, want 99", gotUserID)
	}
}

func TestUserIDFromContext_Missing(t *testing.T) {
	id := UserIDFromContext(context.Background())
	if id != 0 {
		t.Errorf("UserIDFromContext on empty ctx = %d, want 0", id)
	}
}
