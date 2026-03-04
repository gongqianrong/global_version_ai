package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- mock ---

type mockAdminChecker struct {
	isAdminFn func(ctx context.Context, userID int64) (bool, error)
}

func (m *mockAdminChecker) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	return m.isAdminFn(ctx, userID)
}

// --- tests ---

func TestAdminRequired_Admin(t *testing.T) {
	checker := &mockAdminChecker{
		isAdminFn: func(_ context.Context, _ int64) (bool, error) { return true, nil },
	}
	called := false
	handler := AdminRequired(checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	ctx := context.WithValue(req.Context(), userIDCtxKey, int64(1))
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("next handler was not called for admin user")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestAdminRequired_NonAdmin(t *testing.T) {
	checker := &mockAdminChecker{
		isAdminFn: func(_ context.Context, _ int64) (bool, error) { return false, nil },
	}
	handler := AdminRequired(checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called for non-admin user")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	ctx := context.WithValue(req.Context(), userIDCtxKey, int64(1))
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestAdminRequired_NoUserID(t *testing.T) {
	checker := &mockAdminChecker{
		isAdminFn: func(_ context.Context, _ int64) (bool, error) { return false, nil },
	}
	handler := AdminRequired(checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called without user ID")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestAdminRequired_CheckError(t *testing.T) {
	checker := &mockAdminChecker{
		isAdminFn: func(_ context.Context, _ int64) (bool, error) { return false, errors.New("db error") },
	}
	handler := AdminRequired(checker)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called on error")
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin/test", nil)
	ctx := context.WithValue(req.Context(), userIDCtxKey, int64(1))
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}
