package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/normalizer"
	"github.com/rakutao/collection-gateway/internal/registry"
)

func TestPlatformSearchHandler_Success(t *testing.T) {
	reg := registry.New()
	adapter := &testAdapter{
		id: "surugaya",
		result: &domain.SearchResult{
			Products: []domain.RawProduct{
				{
					Platform: "surugaya",
					RawID:    "663043159",
					RawData: map[string]interface{}{
						"title": "FW GUNDAM CONVERGE SB ニカーヤ",
						"price": float64(4950),
					},
				},
			},
			Total: 1,
		},
	}
	reg.Register(registry.PlatformMeta{
		ID:     "surugaya",
		Status: registry.StatusActive,
		Caps:   adapter.Capabilities(),
	}, adapter)

	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)
	handler := NewPlatformSearchHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/platform/search?keyword=gundam&platform=surugaya", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}

	data, _ := json.Marshal(resp.Data)
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if result["platform"] != "surugaya" {
		t.Errorf("platform = %v, want surugaya", result["platform"])
	}
	products, ok := result["products"].([]interface{})
	if !ok || len(products) != 1 {
		t.Errorf("products count = %v, want 1", result["products"])
	}
	total, _ := result["total"].(float64)
	if int64(total) != 1 {
		t.Errorf("total = %v, want 1", result["total"])
	}
}

func TestPlatformSearchHandler_MissingKeyword(t *testing.T) {
	svc := NewPlatformSearchService(registry.New(), normalizer.New(nil))
	handler := NewPlatformSearchHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/platform/search?platform=surugaya", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40002 {
		t.Errorf("code = %d, want 40002", resp.Code)
	}
}

func TestPlatformSearchHandler_MissingPlatform(t *testing.T) {
	svc := NewPlatformSearchService(registry.New(), normalizer.New(nil))
	handler := NewPlatformSearchHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/platform/search?keyword=gundam", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40002 {
		t.Errorf("code = %d, want 40002", resp.Code)
	}
}

func TestPlatformSearchHandler_PlatformNotFound(t *testing.T) {
	svc := NewPlatformSearchService(registry.New(), normalizer.New(nil))
	handler := NewPlatformSearchHandler(svc)

	req := httptest.NewRequest("GET", "/api/v1/platform/search?keyword=gundam&platform=nonexistent", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 50003 {
		t.Errorf("code = %d, want 50003", resp.Code)
	}
}
