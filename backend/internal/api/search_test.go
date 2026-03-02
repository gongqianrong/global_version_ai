package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/i18n"
	"github.com/rakutao/collection-gateway/internal/search"
)

// --- Mocks ---

type mockPreparer struct {
	query domain.SearchQuery
	err   error
}

func (m *mockPreparer) PrepareQuery(_ context.Context, q domain.SearchQuery) (domain.SearchQuery, error) {
	if m.err != nil {
		return q, m.err
	}
	q.KeywordJA = m.query.KeywordJA
	return q, nil
}

type mockSearcher struct {
	result *domain.SearchResponse
	err    error
}

func (m *mockSearcher) Search(_ context.Context, _ domain.SearchQuery) (*domain.SearchResponse, error) {
	return m.result, m.err
}

// --- Tests ---

func TestHandleSearch_Success(t *testing.T) {
	preparer := &mockPreparer{query: domain.SearchQuery{KeywordJA: "グッチ"}}
	searcher := &mockSearcher{result: &domain.SearchResponse{
		CachedResults:     []domain.ProductSummary{{ID: "p1", Title: "Gucci Bag"}},
		CachedTotal:       1,
		TranslatedKeyword: "グッチ",
	}}
	sm := NewStreamManager()
	defer sm.Stop()
	handler := NewSearchHandler(preparer, searcher, sm)

	r := httptest.NewRequest("GET", "/api/v1/search?keyword=gucci", nil)
	r = r.WithContext(i18n.WithLang(r.Context(), i18n.LangEN))
	w := httptest.NewRecorder()
	handler.HandleSearch(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}

	// Check that data contains realtime_stream_id
	data, _ := json.Marshal(resp.Data)
	var searchResp domain.SearchResponse
	json.Unmarshal(data, &searchResp)
	if searchResp.RealtimeStreamID == "" {
		t.Error("expected non-empty realtime_stream_id")
	}
	if searchResp.CachedTotal != 1 {
		t.Errorf("cached_total = %d, want 1", searchResp.CachedTotal)
	}
}

func TestHandleSearch_MissingKeyword(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()
	handler := NewSearchHandler(&mockPreparer{}, &mockSearcher{}, sm)

	r := httptest.NewRequest("GET", "/api/v1/search", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40002 {
		t.Errorf("code = %d, want 40002", resp.Code)
	}
}

func TestHandleSearch_BlockedKeyword(t *testing.T) {
	preparer := &mockPreparer{err: search.ErrBlockedKeyword}
	sm := NewStreamManager()
	defer sm.Stop()
	handler := NewSearchHandler(preparer, &mockSearcher{}, sm)

	r := httptest.NewRequest("GET", "/api/v1/search?keyword=blocked", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40001 {
		t.Errorf("code = %d, want 40001", resp.Code)
	}
}

func TestHandleSearch_SearchError(t *testing.T) {
	preparer := &mockPreparer{query: domain.SearchQuery{KeywordJA: "test"}}
	searcher := &mockSearcher{err: errors.New("es unavailable")}
	sm := NewStreamManager()
	defer sm.Stop()
	handler := NewSearchHandler(preparer, searcher, sm)

	r := httptest.NewRequest("GET", "/api/v1/search?keyword=test", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 50003 {
		t.Errorf("code = %d, want 50003", resp.Code)
	}
}

func TestParseSearchParams_Defaults(t *testing.T) {
	r := httptest.NewRequest("GET", "/search?keyword=test", nil)
	q := parseSearchParams(r)

	if q.Keyword != "test" {
		t.Errorf("keyword = %q, want %q", q.Keyword, "test")
	}
	if q.Page != 1 {
		t.Errorf("page = %d, want 1", q.Page)
	}
	if q.PageSize != 20 {
		t.Errorf("page_size = %d, want 20", q.PageSize)
	}
	if q.UserLang != "zh-TW" {
		t.Errorf("lang = %q, want %q", q.UserLang, "zh-TW")
	}
	if q.ContentRating != "general" {
		t.Errorf("content_rating = %q, want %q", q.ContentRating, "general")
	}
}

func TestParseSearchParams_AllParams(t *testing.T) {
	r := httptest.NewRequest("GET", "/search?keyword=gucci&platforms=yahoo_auction,amazon_jp&brand_id=b1&categories=bags,shoes&price_min=1000&price_max=50000&condition=new,good&sort=price_asc&page=3&page_size=50&content_rating=all", nil)
	r = r.WithContext(i18n.WithLang(r.Context(), i18n.LangEN))
	q := parseSearchParams(r)

	if q.Keyword != "gucci" {
		t.Errorf("keyword = %q", q.Keyword)
	}
	if len(q.Platforms) != 2 || q.Platforms[0] != "yahoo_auction" {
		t.Errorf("platforms = %v", q.Platforms)
	}
	if q.BrandID != "b1" {
		t.Errorf("brand_id = %q", q.BrandID)
	}
	if len(q.Categories) != 2 {
		t.Errorf("categories = %v", q.Categories)
	}
	if q.PriceMin != 1000 {
		t.Errorf("price_min = %d", q.PriceMin)
	}
	if q.PriceMax != 50000 {
		t.Errorf("price_max = %d", q.PriceMax)
	}
	if len(q.Condition) != 2 {
		t.Errorf("condition = %v", q.Condition)
	}
	if q.SortBy != "price_asc" {
		t.Errorf("sort = %q", q.SortBy)
	}
	if q.Page != 3 {
		t.Errorf("page = %d", q.Page)
	}
	if q.PageSize != 50 {
		t.Errorf("page_size = %d", q.PageSize)
	}
}

func TestParseSearchParams_PageSizeCap(t *testing.T) {
	r := httptest.NewRequest("GET", "/search?keyword=test&page_size=999", nil)
	q := parseSearchParams(r)

	if q.PageSize != 100 {
		t.Errorf("page_size = %d, want 100 (capped)", q.PageSize)
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"a", 1},
		{"a,b,c", 3},
		{"a, b , c", 3},
		{",,,", 0},
	}
	for _, tt := range tests {
		got := splitCSV(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitCSV(%q) = %d items, want %d", tt.input, len(got), tt.want)
		}
	}
}
