package domestic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

func TestPlatformID(t *testing.T) {
	a := New("mercari", "http://localhost:9999", http.DefaultClient)
	if got := a.PlatformID(); got != "mercari" {
		t.Errorf("PlatformID() = %q, want %q", got, "mercari")
	}
}

func TestSearch(t *testing.T) {
	// Set up a mock domestic API server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request path and query parameters.
		if r.URL.Path != "/api/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		q := r.URL.Query()
		if got := q.Get("platform"); got != "mercari" {
			t.Errorf("platform param = %q, want %q", got, "mercari")
		}
		if got := q.Get("keyword"); got != "nike" {
			t.Errorf("keyword param = %q, want %q", got, "nike")
		}
		if got := q.Get("page"); got != "1" {
			t.Errorf("page param = %q, want %q", got, "1")
		}
		if got := q.Get("page_size"); got != "20" {
			t.Errorf("page_size param = %q, want %q", got, "20")
		}

		resp := domesticSearchResponse{
			Items: []domesticItem{
				{ID: "item-1", Title: "Nike Air Max", Price: 12000},
				{ID: "item-2", Title: "Nike Dunk Low", Price: 15000},
			},
			Total: 42,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New("mercari", srv.URL, http.DefaultClient)
	result, err := a.Search(context.Background(), domain.SearchQuery{
		Keyword:  "nike",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("Search returned unexpected error: %v", err)
	}

	if result.Total != 42 {
		t.Errorf("Total = %d, want %d", result.Total, 42)
	}
	if len(result.Products) != 2 {
		t.Fatalf("len(Products) = %d, want %d", len(result.Products), 2)
	}

	p0 := result.Products[0]
	if p0.Platform != "mercari" {
		t.Errorf("Products[0].Platform = %q, want %q", p0.Platform, "mercari")
	}
	if p0.RawID != "item-1" {
		t.Errorf("Products[0].RawID = %q, want %q", p0.RawID, "item-1")
	}
	if p0.RawData["title"] != "Nike Air Max" {
		t.Errorf("Products[0].RawData[title] = %v, want %q", p0.RawData["title"], "Nike Air Max")
	}

	// HasMore should be true since total (42) > page*pageSize (1*20 = 20)
	if !result.HasMore {
		t.Error("HasMore = false, want true")
	}
}

func TestHealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	a := New("mercari", srv.URL, http.DefaultClient)
	status := a.HealthCheck(context.Background())
	if status.Status != "healthy" {
		t.Errorf("HealthCheck().Status = %q, want %q", status.Status, "healthy")
	}
}
