package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/i18n"
)

type mockProductFetcher struct {
	product *domain.UnifiedProduct
	err     error
}

func (m *mockProductFetcher) GetProduct(_ context.Context, _ string) (*domain.UnifiedProduct, error) {
	return m.product, m.err
}

func TestHandleGetProduct_Success(t *testing.T) {
	fetcher := &mockProductFetcher{
		product: &domain.UnifiedProduct{
			ID:              "yahoo_auction_abc123",
			SourcePlatform:  "yahoo_auction",
			Title:           "グッチ バッグ",
			TitleTranslated: map[string]string{"zh-TW": "古馳手提包"},
			Description:     "良い状態",
			PriceJPY:        16500,
			ServiceFeeJPY:   1500,
			OriginalPrice:   15000,
			Status:          "available",
			Condition:       "good",
			Quantity:        1,
			Images:          []string{"img1.jpg"},
			ListedAt:        time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewProductHandler(fetcher, nil)

	r := chi.NewRouter()
	r.Get("/products/{id}", handler.HandleGetProduct)

	req := httptest.NewRequest("GET", "/products/yahoo_auction_abc123", nil)
	req = req.WithContext(i18n.WithLang(req.Context(), i18n.LangZhTW))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}

	data, _ := json.Marshal(resp.Data)
	var product ProductResponse
	json.Unmarshal(data, &product)

	if product.Title != "古馳手提包" {
		t.Errorf("title = %q, want translated title", product.Title)
	}
	if product.TitleOriginal != "古馳手提包" {
		t.Errorf("title_original = %q, want translated", product.TitleOriginal)
	}
	if !product.IsTranslated {
		t.Error("expected is_translated = true")
	}
	if product.PriceJPY != 16500 {
		t.Errorf("price_jpy = %d, want 16500", product.PriceJPY)
	}
}

func TestHandleGetProduct_NotFound(t *testing.T) {
	fetcher := &mockProductFetcher{err: errors.New("not found")}
	handler := NewProductHandler(fetcher, nil)

	r := chi.NewRouter()
	r.Get("/products/{id}", handler.HandleGetProduct)

	req := httptest.NewRequest("GET", "/products/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40401 {
		t.Errorf("code = %d, want 40401", resp.Code)
	}
}

func TestHandleGetProduct_DefaultLang(t *testing.T) {
	fetcher := &mockProductFetcher{
		product: &domain.UnifiedProduct{
			ID:              "p1",
			SourcePlatform:  "test",
			Title:           "テスト",
			TitleTranslated: map[string]string{"zh-TW": "測試"},
			ListedAt:        time.Now(),
		},
	}
	handler := NewProductHandler(fetcher, nil)

	r := chi.NewRouter()
	r.Get("/products/{id}", handler.HandleGetProduct)

	// No lang param — should default to zh-TW
	req := httptest.NewRequest("GET", "/products/p1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	data, _ := json.Marshal(resp.Data)
	var product ProductResponse
	json.Unmarshal(data, &product)

	if product.Title != "測試" {
		t.Errorf("title = %q, want %q (default zh-TW)", product.Title, "測試")
	}
}

func TestHandleGetProduct_Fallback(t *testing.T) {
	esFetcher := &mockProductFetcher{err: errors.New("es: not found")}
	fallback := &mockProductFetcher{
		product: &domain.UnifiedProduct{
			ID:             "surugaya_663043159",
			SourcePlatform: "surugaya",
			Title:          "FW GUNDAM CONVERGE SB ニカーヤ",
			PriceJPY:       5445,
			ListedAt:       time.Now(),
		},
	}
	handler := NewProductHandler(esFetcher, fallback)

	r := chi.NewRouter()
	r.Get("/products/{id}", handler.HandleGetProduct)

	req := httptest.NewRequest("GET", "/products/surugaya_663043159", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (fallback should succeed)", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}

	data, _ := json.Marshal(resp.Data)
	var product ProductResponse
	json.Unmarshal(data, &product)
	if product.ID != "surugaya_663043159" {
		t.Errorf("ID = %q, want %q", product.ID, "surugaya_663043159")
	}
	if product.PriceJPY != 5445 {
		t.Errorf("PriceJPY = %d, want 5445", product.PriceJPY)
	}
}

func TestHandleGetProduct_BothFail(t *testing.T) {
	esFetcher := &mockProductFetcher{err: errors.New("es: not found")}
	fallback := &mockProductFetcher{err: errors.New("adapter: not found")}
	handler := NewProductHandler(esFetcher, fallback)

	r := chi.NewRouter()
	r.Get("/products/{id}", handler.HandleGetProduct)

	req := httptest.NewRequest("GET", "/products/surugaya_999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40401 {
		t.Errorf("code = %d, want 40401", resp.Code)
	}
}

func TestBuildProductResponse_NoTranslation(t *testing.T) {
	p := &domain.UnifiedProduct{
		ID:       "p1",
		Title:    "テスト",
		ListedAt: time.Now(),
	}
	resp := buildProductResponse(p, i18n.LangZhTW)

	if resp.Title != "テスト" {
		t.Errorf("title = %q, want original when no translation", resp.Title)
	}
	if resp.IsTranslated {
		t.Error("expected is_translated = false when no translation available")
	}
}

func TestBuildProductResponse_I18N(t *testing.T) {
	p := &domain.UnifiedProduct{
		Status:          "available",
		Condition:       "new",
		ShippingType:    "free",
		ContentRating:   "general",
		Title:           "テスト商品",
		TitleTranslated: map[string]string{"zh-TW": "測試商品", "en": "Test Product"},
		Description:     "テスト説明",
		DescTranslated:  map[string]string{"zh-TW": "測試說明", "en": "Test Desc"},
		Images:          []string{"img.jpg"},
		ListedAt:        time.Now(),
	}

	resp := buildProductResponse(p, i18n.LangEN)
	if resp.Status != "available" {
		t.Errorf("status = %q, want available", resp.Status)
	}
	if resp.Condition != "new" {
		t.Errorf("condition = %q, want new", resp.Condition)
	}
	if resp.Title != "Test Product" {
		t.Errorf("title = %q, want Test Product", resp.Title)
	}
	if resp.TitleOriginal != "Test Product" {
		t.Errorf("title_original = %q, want Test Product", resp.TitleOriginal)
	}
	if resp.Description != "Test Desc" {
		t.Errorf("description = %q, want Test Desc", resp.Description)
	}
	if resp.DescriptionOriginal != "Test Desc" {
		t.Errorf("description_original = %q, want Test Desc", resp.DescriptionOriginal)
	}

	// JA: no translation, all fields show original Japanese
	respJA := buildProductResponse(p, i18n.LangJA)
	if respJA.Title != "テスト商品" {
		t.Errorf("title(ja) = %q, want テスト商品", respJA.Title)
	}
	if respJA.TitleOriginal != "テスト商品" {
		t.Errorf("title_original(ja) = %q, want テスト商品", respJA.TitleOriginal)
	}
	if respJA.Description != "テスト説明" {
		t.Errorf("description(ja) = %q, want テスト説明", respJA.Description)
	}
}
