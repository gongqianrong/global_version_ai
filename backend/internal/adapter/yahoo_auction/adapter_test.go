package yahoo_auction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

func TestPlatformID(t *testing.T) {
	a := New("http://localhost:9999", nil)
	if got := a.PlatformID(); got != "yahoo_auction" {
		t.Errorf("PlatformID() = %q, want %q", got, "yahoo_auction")
	}
}

func TestCapabilities(t *testing.T) {
	a := New("http://localhost:9999", nil)
	caps := a.Capabilities()
	if !caps.SupportsSearch {
		t.Error("expected SupportsSearch = true")
	}
	if !caps.SupportsRealtime {
		t.Error("expected SupportsRealtime = true")
	}
	if !caps.HasCategoryField {
		t.Error("expected HasCategoryField = true")
	}
}

func TestSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		q := r.URL.Query()
		if got := q.Get("keyword"); got != "グッチ" {
			t.Errorf("keyword = %q, want %q", got, "グッチ")
		}
		if got := q.Get("page"); got != "1" {
			t.Errorf("page = %q, want %q", got, "1")
		}

		resp := searchResponse{
			Items: []item{
				{ID: "ya-001", Title: "グッチ バッグ", Price: 15000, Status: "on_sale", Condition: "good", Category: "バッグ"},
				{ID: "ya-002", Title: "グッチ 財布", Price: 8000, Images: []string{"img1.jpg"}},
			},
			Total: 25,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	result, err := a.Search(context.Background(), domain.SearchQuery{
		Keyword:   "gucci",
		KeywordJA: "グッチ",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if result.Total != 25 {
		t.Errorf("Total = %d, want 25", result.Total)
	}
	if len(result.Products) != 2 {
		t.Fatalf("len(Products) = %d, want 2", len(result.Products))
	}

	p0 := result.Products[0]
	if p0.Platform != "yahoo_auction" {
		t.Errorf("Platform = %q, want %q", p0.Platform, "yahoo_auction")
	}
	if p0.RawID != "ya-001" {
		t.Errorf("RawID = %q, want %q", p0.RawID, "ya-001")
	}
	if p0.RawData["title"] != "グッチ バッグ" {
		t.Errorf("RawData[title] = %v", p0.RawData["title"])
	}
	if p0.RawData["category"] != "バッグ" {
		t.Errorf("RawData[category] = %v", p0.RawData["category"])
	}
	if !result.HasMore {
		t.Error("HasMore = false, want true (25 > 1*20)")
	}
}

func TestSearch_UsesKeywordJA(t *testing.T) {
	var receivedKeyword string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKeyword = r.URL.Query().Get("keyword")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{})
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	a.Search(context.Background(), domain.SearchQuery{
		Keyword:   "gucci",
		KeywordJA: "グッチ",
		Page:      1,
		PageSize:  20,
	})

	if receivedKeyword != "グッチ" {
		t.Errorf("sent keyword = %q, want %q (should prefer KeywordJA)", receivedKeyword, "グッチ")
	}
}

func TestSearch_FallbackToKeyword(t *testing.T) {
	var receivedKeyword string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKeyword = r.URL.Query().Get("keyword")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{})
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	a.Search(context.Background(), domain.SearchQuery{
		Keyword:  "グッチ",
		Page:     1,
		PageSize: 20,
	})

	if receivedKeyword != "グッチ" {
		t.Errorf("sent keyword = %q, want %q (should fallback to Keyword)", receivedKeyword, "グッチ")
	}
}

func TestSearch_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	_, err := a.Search(context.Background(), domain.SearchQuery{Keyword: "test", Page: 1, PageSize: 20})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGetProduct_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/product/ya-001" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		resp := productResponse{
			Item: item{
				ID:          "ya-001",
				Title:       "グッチ バッグ",
				Price:       15000,
				Description: "美品",
				Images:      []string{"img1.jpg", "img2.jpg"},
				Status:      "on_sale",
				Condition:   "good",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	rp, err := a.GetProduct(context.Background(), "ya-001")
	if err != nil {
		t.Fatalf("GetProduct error: %v", err)
	}

	if rp.RawID != "ya-001" {
		t.Errorf("RawID = %q, want %q", rp.RawID, "ya-001")
	}
	if rp.RawData["title"] != "グッチ バッグ" {
		t.Errorf("RawData[title] = %v", rp.RawData["title"])
	}
	if rp.RawData["description"] != "美品" {
		t.Errorf("RawData[description] = %v", rp.RawData["description"])
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	_, err := a.GetProduct(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestHealthCheck_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	status := a.HealthCheck(context.Background())
	if status.Status != "healthy" {
		t.Errorf("Status = %q, want %q", status.Status, "healthy")
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	status := a.HealthCheck(context.Background())
	if status.Status != "unhealthy" {
		t.Errorf("Status = %q, want %q", status.Status, "unhealthy")
	}
}
