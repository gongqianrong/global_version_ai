package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/repo"
)

// --- mock ---

type mockCartStore struct {
	addFn            func(ctx context.Context, userID int64, productID string, quantity int, priceAtAdd int64) (*repo.CartItem, error)
	updateQuantityFn func(ctx context.Context, userID int64, productID string, quantity int) error
	removeFn         func(ctx context.Context, userID int64, productID string) error
	listByUserFn     func(ctx context.Context, userID int64) ([]repo.CartItem, error)
}

func (m *mockCartStore) Add(ctx context.Context, userID int64, productID string, quantity int, priceAtAdd int64) (*repo.CartItem, error) {
	return m.addFn(ctx, userID, productID, quantity, priceAtAdd)
}
func (m *mockCartStore) UpdateQuantity(ctx context.Context, userID int64, productID string, quantity int) error {
	return m.updateQuantityFn(ctx, userID, productID, quantity)
}
func (m *mockCartStore) Remove(ctx context.Context, userID int64, productID string) error {
	return m.removeFn(ctx, userID, productID)
}
func (m *mockCartStore) ListByUser(ctx context.Context, userID int64) ([]repo.CartItem, error) {
	return m.listByUserFn(ctx, userID)
}

func cartReq(method, url, body string, userID int64, params map[string]string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	ctx := context.WithValue(r.Context(), userIDCtxKey, userID)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

// mapProductFetcher returns different products by ID.
type mapProductFetcher struct {
	products map[string]*domain.UnifiedProduct
}

func (m *mapProductFetcher) GetProduct(_ context.Context, id string) (*domain.UnifiedProduct, error) {
	p, ok := m.products[id]
	if !ok {
		return nil, fmt.Errorf("product %s not found", id)
	}
	return p, nil
}

// --- tests ---

func TestHandleAddToCart_Success(t *testing.T) {
	fetcher := &mockProductFetcher{
		product: &domain.UnifiedProduct{ID: "prod1", Status: domain.StatusAvailable, PriceJPY: 1000},
	}
	store := &mockCartStore{
		addFn: func(_ context.Context, uid int64, pid string, qty int, price int64) (*repo.CartItem, error) {
			return &repo.CartItem{ID: 1, UserID: uid, ProductID: pid, Quantity: qty, PriceAtAdd: price, AddedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}
	h := NewCartHandler(store, fetcher, nil)
	req := cartReq(http.MethodPost, "/cart", `{"product_id":"prod1","quantity":2}`, 10, nil)
	rec := httptest.NewRecorder()
	h.HandleAddToCart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleAddToCart_ProductNotFound(t *testing.T) {
	fetcher := &mockProductFetcher{
		err: fmt.Errorf("not found"),
	}
	h := NewCartHandler(&mockCartStore{}, fetcher, nil)
	req := cartReq(http.MethodPost, "/cart", `{"product_id":"prod1","quantity":1}`, 10, nil)
	rec := httptest.NewRecorder()
	h.HandleAddToCart(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleRemoveFromCart_Success(t *testing.T) {
	store := &mockCartStore{
		removeFn: func(_ context.Context, _ int64, _ string) error { return nil },
	}
	h := NewCartHandler(store, nil, nil)
	req := cartReq(http.MethodDelete, "/cart/prod1", "", 10, map[string]string{"productID": "prod1"})
	rec := httptest.NewRecorder()
	h.HandleRemoveFromCart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleRemoveFromCart_NotFound(t *testing.T) {
	store := &mockCartStore{
		removeFn: func(_ context.Context, _ int64, _ string) error { return pgx.ErrNoRows },
	}
	h := NewCartHandler(store, nil, nil)
	req := cartReq(http.MethodDelete, "/cart/prod1", "", 10, map[string]string{"productID": "prod1"})
	rec := httptest.NewRecorder()
	h.HandleRemoveFromCart(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleListCart_Empty(t *testing.T) {
	store := &mockCartStore{
		listByUserFn: func(_ context.Context, _ int64) ([]repo.CartItem, error) {
			return nil, nil
		},
	}
	h := NewCartHandler(store, nil, nil)
	req := cartReq(http.MethodGet, "/cart", "", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleListCart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	total := data["total_items"].(float64)
	if total != 0 {
		t.Errorf("total_items = %v, want 0", total)
	}
}

func TestHandleListCart_WithItems(t *testing.T) {
	store := &mockCartStore{
		listByUserFn: func(_ context.Context, _ int64) ([]repo.CartItem, error) {
			return []repo.CartItem{
				{ID: 1, UserID: 10, ProductID: "p1", Quantity: 2, PriceAtAdd: 1000, AddedAt: time.Now(), UpdatedAt: time.Now()},
				{ID: 2, UserID: 10, ProductID: "p2", Quantity: 1, PriceAtAdd: 2000, AddedAt: time.Now(), UpdatedAt: time.Now()},
			}, nil
		},
	}
	fetcher := &mapProductFetcher{
		products: map[string]*domain.UnifiedProduct{
			"p1": {
				ID: "p1", Title: "商品A", PriceJPY: 1000, Status: domain.StatusAvailable,
				SourcePlatform: "surugaya", Images: []string{"https://example.com/a.jpg"},
				Seller: domain.SellerInfo{SellerID: "seller1", SellerName: "Shop A"},
			},
			"p2": {
				ID: "p2", Title: "商品B", PriceJPY: 2500, Status: domain.StatusAvailable,
				SourcePlatform: "surugaya", Images: []string{"https://example.com/b.jpg"},
				Seller: domain.SellerInfo{SellerID: "seller2", SellerName: "Shop B"},
			},
		},
	}
	h := NewCartHandler(store, fetcher, nil)
	req := cartReq(http.MethodGet, "/cart", "", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleListCart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})

	groups := data["groups"].([]interface{})
	if len(groups) != 2 {
		t.Fatalf("groups len = %d, want 2 (two different sellers)", len(groups))
	}

	totalItems := data["total_items"].(float64)
	if totalItems != 3 {
		t.Errorf("total_items = %v, want 3 (2+1)", totalItems)
	}

	// Check price change detection
	g1 := groups[0].(map[string]interface{})
	items := g1["items"].([]interface{})
	item1 := items[0].(map[string]interface{})
	if item1["price_changed"] != false {
		t.Errorf("p1 price_changed = %v, want false (same price)", item1["price_changed"])
	}
}

func TestHandleListCart_PriceChanged(t *testing.T) {
	store := &mockCartStore{
		listByUserFn: func(_ context.Context, _ int64) ([]repo.CartItem, error) {
			return []repo.CartItem{
				{ID: 1, UserID: 10, ProductID: "p1", Quantity: 1, PriceAtAdd: 1000, AddedAt: time.Now(), UpdatedAt: time.Now()},
			}, nil
		},
	}
	fetcher := &mapProductFetcher{
		products: map[string]*domain.UnifiedProduct{
			"p1": {
				ID: "p1", Title: "商品A", PriceJPY: 1500, Status: domain.StatusAvailable,
				Seller: domain.SellerInfo{SellerID: "s1", SellerName: "Shop"},
			},
		},
	}
	h := NewCartHandler(store, fetcher, nil)
	req := cartReq(http.MethodGet, "/cart", "", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleListCart(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	groups := data["groups"].([]interface{})
	g := groups[0].(map[string]interface{})
	item := g["items"].([]interface{})[0].(map[string]interface{})
	if item["price_changed"] != true {
		t.Errorf("price_changed = %v, want true (1000→1500)", item["price_changed"])
	}
	if int64(item["current_price"].(float64)) != 1500 {
		t.Errorf("current_price = %v, want 1500", item["current_price"])
	}
}

func TestHandleAddToCart_ProductUnavailable(t *testing.T) {
	fetcher := &mockProductFetcher{
		product: &domain.UnifiedProduct{ID: "prod1", Status: domain.StatusSold, PriceJPY: 1000},
	}
	h := NewCartHandler(&mockCartStore{}, fetcher, nil)
	req := cartReq(http.MethodPost, "/cart", `{"product_id":"prod1","quantity":1}`, 10, nil)
	rec := httptest.NewRecorder()
	h.HandleAddToCart(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (product sold)", rec.Code)
	}
}

func TestHandleUpdateQuantity_Success(t *testing.T) {
	store := &mockCartStore{
		updateQuantityFn: func(_ context.Context, _ int64, _ string, _ int) error { return nil },
	}
	h := NewCartHandler(store, nil, nil)
	req := cartReq(http.MethodPut, "/cart/prod1", `{"quantity":3}`, 10, map[string]string{"productID": "prod1"})
	rec := httptest.NewRecorder()
	h.HandleUpdateQuantity(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if data["quantity"].(float64) != 3 {
		t.Errorf("quantity = %v, want 3", data["quantity"])
	}
}

func TestHandleUpdateQuantity_NotFound(t *testing.T) {
	store := &mockCartStore{
		updateQuantityFn: func(_ context.Context, _ int64, _ string, _ int) error { return pgx.ErrNoRows },
	}
	h := NewCartHandler(store, nil, nil)
	req := cartReq(http.MethodPut, "/cart/prod1", `{"quantity":3}`, 10, map[string]string{"productID": "prod1"})
	rec := httptest.NewRecorder()
	h.HandleUpdateQuantity(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
