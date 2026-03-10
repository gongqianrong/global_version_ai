package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/repo"
)

// --- mock ---

type mockFavoriteStore struct {
	addFn              func(ctx context.Context, userID int64, productID string) (*repo.FavoriteItem, error)
	removeFn           func(ctx context.Context, userID int64, productID string) error
	listByUserFn       func(ctx context.Context, userID int64, limit, offset int) ([]repo.FavoriteItem, int64, error)
	batchIsFavoritedFn func(ctx context.Context, userID int64, productIDs []string) (map[string]bool, error)
}

func (m *mockFavoriteStore) Add(ctx context.Context, userID int64, productID string) (*repo.FavoriteItem, error) {
	return m.addFn(ctx, userID, productID)
}
func (m *mockFavoriteStore) Remove(ctx context.Context, userID int64, productID string) error {
	return m.removeFn(ctx, userID, productID)
}
func (m *mockFavoriteStore) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]repo.FavoriteItem, int64, error) {
	return m.listByUserFn(ctx, userID, limit, offset)
}
func (m *mockFavoriteStore) BatchIsFavorited(ctx context.Context, userID int64, productIDs []string) (map[string]bool, error) {
	return m.batchIsFavoritedFn(ctx, userID, productIDs)
}

// helper
func favReq(method, url string, userID int64, params map[string]string) *http.Request {
	r := httptest.NewRequest(method, url, nil)
	ctx := context.WithValue(r.Context(), userIDCtxKey, userID)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

// --- tests ---

func TestHandleAddFavorite_Success(t *testing.T) {
	fetcher := &mockProductFetcher{
		product: &domain.UnifiedProduct{ID: "prod1", Status: domain.StatusAvailable},
	}
	store := &mockFavoriteStore{
		addFn: func(_ context.Context, uid int64, pid string) (*repo.FavoriteItem, error) {
			return &repo.FavoriteItem{ID: 1, UserID: uid, ProductID: pid, AddedAt: time.Now()}, nil
		},
	}
	h := NewFavoriteHandler(store, fetcher, nil)
	req := favReq(http.MethodPost, "/favorites/prod1", 10, map[string]string{"productID": "prod1"})
	rec := httptest.NewRecorder()
	h.HandleAddFavorite(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleAddFavorite_ProductNotFound(t *testing.T) {
	fetcher := &mockProductFetcher{
		err: fmt.Errorf("not found"),
	}
	h := NewFavoriteHandler(&mockFavoriteStore{}, fetcher, nil)
	req := favReq(http.MethodPost, "/favorites/prod1", 10, map[string]string{"productID": "prod1"})
	rec := httptest.NewRecorder()
	h.HandleAddFavorite(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleRemoveFavorite_Success(t *testing.T) {
	store := &mockFavoriteStore{
		removeFn: func(_ context.Context, _ int64, _ string) error { return nil },
	}
	h := NewFavoriteHandler(store, nil, nil)
	req := favReq(http.MethodDelete, "/favorites/prod1", 10, map[string]string{"productID": "prod1"})
	rec := httptest.NewRecorder()
	h.HandleRemoveFavorite(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleRemoveFavorite_NotFound(t *testing.T) {
	store := &mockFavoriteStore{
		removeFn: func(_ context.Context, _ int64, _ string) error { return pgx.ErrNoRows },
	}
	h := NewFavoriteHandler(store, nil, nil)
	req := favReq(http.MethodDelete, "/favorites/prod1", 10, map[string]string{"productID": "prod1"})
	rec := httptest.NewRecorder()
	h.HandleRemoveFavorite(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleListFavorites_Empty(t *testing.T) {
	store := &mockFavoriteStore{
		listByUserFn: func(_ context.Context, _ int64, _, _ int) ([]repo.FavoriteItem, int64, error) {
			return nil, 0, nil
		},
	}
	h := NewFavoriteHandler(store, nil, nil)
	req := favReq(http.MethodGet, "/favorites", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleListFavorites(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 0 {
		t.Errorf("items len = %d, want 0", len(items))
	}
}

func TestHandleListFavorites_WithItems(t *testing.T) {
	store := &mockFavoriteStore{
		listByUserFn: func(_ context.Context, _ int64, _, _ int) ([]repo.FavoriteItem, int64, error) {
			return []repo.FavoriteItem{
				{ID: 1, UserID: 10, ProductID: "prod1", AddedAt: time.Now()},
				{ID: 2, UserID: 10, ProductID: "prod2", AddedAt: time.Now()},
			}, 2, nil
		},
	}
	fetcher := &mockProductFetcher{
		product: &domain.UnifiedProduct{
			ID:             "prod1",
			Title:          "テスト商品",
			Images:         []string{"https://example.com/img.jpg"},
			PriceJPY:       1500,
			SourcePlatform: "surugaya",
			Status:         domain.StatusAvailable,
		},
	}
	h := NewFavoriteHandler(store, fetcher, nil)
	req := favReq(http.MethodGet, "/favorites?page=1&page_size=20", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleListFavorites(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 2 {
		t.Fatalf("items len = %d, want 2", len(items))
	}
	first := items[0].(map[string]interface{})
	if first["title"] != "テスト商品" {
		t.Errorf("title = %v, want テスト商品", first["title"])
	}
	if first["image"] != "https://example.com/img.jpg" {
		t.Errorf("image = %v, want https://example.com/img.jpg", first["image"])
	}
	if first["platform"] != "surugaya" {
		t.Errorf("platform = %v, want surugaya", first["platform"])
	}
	if int64(first["price_jpy"].(float64)) != 1500 {
		t.Errorf("price_jpy = %v, want 1500", first["price_jpy"])
	}
	total := data["total"].(float64)
	if total != 2 {
		t.Errorf("total = %v, want 2", total)
	}
}

func TestHandleListFavorites_ESNotFound(t *testing.T) {
	store := &mockFavoriteStore{
		listByUserFn: func(_ context.Context, _ int64, _, _ int) ([]repo.FavoriteItem, int64, error) {
			return []repo.FavoriteItem{
				{ID: 1, UserID: 10, ProductID: "gone1", AddedAt: time.Now()},
			}, 1, nil
		},
	}
	fetcher := &mockProductFetcher{err: fmt.Errorf("not found in ES")}
	h := NewFavoriteHandler(store, fetcher, nil)
	req := favReq(http.MethodGet, "/favorites", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleListFavorites(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	items := data["items"].([]interface{})
	first := items[0].(map[string]interface{})
	if first["status"] != "unknown" {
		t.Errorf("status = %v, want unknown (product not in ES)", first["status"])
	}
}

func TestHandleCheckFavorites_Success(t *testing.T) {
	store := &mockFavoriteStore{
		batchIsFavoritedFn: func(_ context.Context, _ int64, ids []string) (map[string]bool, error) {
			return map[string]bool{"p1": true}, nil
		},
	}
	h := NewFavoriteHandler(store, nil, nil)
	req := favReq(http.MethodGet, "/favorites/check?product_ids=p1,p2", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleCheckFavorites(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if data["p1"] != true {
		t.Errorf("p1 = %v, want true", data["p1"])
	}
	if data["p2"] != false {
		t.Errorf("p2 = %v, want false", data["p2"])
	}
}
