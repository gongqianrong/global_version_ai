package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

type mockHealthAdapter struct {
	status domain.HealthStatus
}

func (m *mockHealthAdapter) PlatformID() string               { return "test" }
func (m *mockHealthAdapter) Capabilities() domain.AdapterCaps { return domain.AdapterCaps{} }
func (m *mockHealthAdapter) Search(_ context.Context, _ domain.SearchQuery) (*domain.SearchResult, error) {
	return nil, nil
}
func (m *mockHealthAdapter) GetProduct(_ context.Context, _ string) (*domain.RawProduct, error) {
	return nil, nil
}
func (m *mockHealthAdapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) {
	return nil, nil
}
func (m *mockHealthAdapter) HealthCheck(_ context.Context) domain.HealthStatus {
	return m.status
}

type mockHealthChecker struct {
	adapters map[string]domain.PlatformAdapter
}

func (m *mockHealthChecker) GetAdapter(id string) (domain.PlatformAdapter, error) {
	a, ok := m.adapters[id]
	if !ok {
		return nil, domain.ErrAdapterNotFound
	}
	return a, nil
}

func (m *mockHealthChecker) AllPlatformIDs() []string {
	ids := make([]string, 0, len(m.adapters))
	for id := range m.adapters {
		ids = append(ids, id)
	}
	return ids
}

func TestHandleHealth_AllHealthy(t *testing.T) {
	checker := &mockHealthChecker{
		adapters: map[string]domain.PlatformAdapter{
			"yahoo_auction": &mockHealthAdapter{status: domain.HealthStatus{Status: "healthy"}},
			"amazon_jp":     &mockHealthAdapter{status: domain.HealthStatus{Status: "healthy"}},
		},
	}
	handler := NewHealthHandler(checker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)
	handler.HandleHealth(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

func TestHandleHealth_Degraded(t *testing.T) {
	checker := &mockHealthChecker{
		adapters: map[string]domain.PlatformAdapter{
			"yahoo_auction": &mockHealthAdapter{status: domain.HealthStatus{Status: "healthy"}},
			"surugaya":      &mockHealthAdapter{status: domain.HealthStatus{Status: "degraded", Message: "high latency"}},
		},
	}
	handler := NewHealthHandler(checker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)
	handler.HandleHealth(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "degraded" {
		t.Errorf("status = %q, want %q", resp["status"], "degraded")
	}
}

func TestHandleHealth_NoPlatforms(t *testing.T) {
	checker := &mockHealthChecker{adapters: map[string]domain.PlatformAdapter{}}
	handler := NewHealthHandler(checker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)
	handler.HandleHealth(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}
