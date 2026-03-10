package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/repo"
)

// --- mock ---

type mockSellerFollowStore struct {
	followFn          func(ctx context.Context, userID int64, sellerID, sellerName string) (*repo.FollowedSeller, error)
	unfollowFn        func(ctx context.Context, userID int64, sellerID string) error
	listByUserFn      func(ctx context.Context, userID int64, limit, offset int) ([]repo.FollowedSeller, int64, error)
	batchIsFollowingFn func(ctx context.Context, userID int64, sellerIDs []string) (map[string]bool, error)
}

func (m *mockSellerFollowStore) Follow(ctx context.Context, userID int64, sellerID, sellerName string) (*repo.FollowedSeller, error) {
	return m.followFn(ctx, userID, sellerID, sellerName)
}
func (m *mockSellerFollowStore) Unfollow(ctx context.Context, userID int64, sellerID string) error {
	return m.unfollowFn(ctx, userID, sellerID)
}
func (m *mockSellerFollowStore) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]repo.FollowedSeller, int64, error) {
	return m.listByUserFn(ctx, userID, limit, offset)
}
func (m *mockSellerFollowStore) BatchIsFollowing(ctx context.Context, userID int64, sellerIDs []string) (map[string]bool, error) {
	return m.batchIsFollowingFn(ctx, userID, sellerIDs)
}

// helper to inject userID + chi URL params
func sellerReq(method, url string, body string, userID int64, params map[string]string) *http.Request {
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

// --- tests ---

func TestHandleFollow_Success(t *testing.T) {
	store := &mockSellerFollowStore{
		followFn: func(_ context.Context, uid int64, sid, sname string) (*repo.FollowedSeller, error) {
			return &repo.FollowedSeller{ID: 1, UserID: uid, SellerID: sid, SellerName: sname, FollowedAt: time.Now()}, nil
		},
	}
	h := NewSellerHandler(store)
	req := sellerReq(http.MethodPost, "/sellers/abc/follow", `{"seller_name":"Shop"}`, 10, map[string]string{"sellerID": "abc"})
	rec := httptest.NewRecorder()
	h.HandleFollow(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
}

func TestHandleFollow_EmptySellerID(t *testing.T) {
	h := NewSellerHandler(&mockSellerFollowStore{})
	req := sellerReq(http.MethodPost, "/sellers//follow", "", 10, map[string]string{"sellerID": ""})
	rec := httptest.NewRecorder()
	h.HandleFollow(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleUnfollow_Success(t *testing.T) {
	store := &mockSellerFollowStore{
		unfollowFn: func(_ context.Context, _ int64, _ string) error { return nil },
	}
	h := NewSellerHandler(store)
	req := sellerReq(http.MethodDelete, "/sellers/abc/follow", "", 10, map[string]string{"sellerID": "abc"})
	rec := httptest.NewRecorder()
	h.HandleUnfollow(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleUnfollow_NotFound(t *testing.T) {
	store := &mockSellerFollowStore{
		unfollowFn: func(_ context.Context, _ int64, _ string) error { return pgx.ErrNoRows },
	}
	h := NewSellerHandler(store)
	req := sellerReq(http.MethodDelete, "/sellers/abc/follow", "", 10, map[string]string{"sellerID": "abc"})
	rec := httptest.NewRecorder()
	h.HandleUnfollow(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleListFollowed_Empty(t *testing.T) {
	store := &mockSellerFollowStore{
		listByUserFn: func(_ context.Context, _ int64, _, _ int) ([]repo.FollowedSeller, int64, error) {
			return nil, 0, nil
		},
	}
	h := NewSellerHandler(store)
	req := sellerReq(http.MethodGet, "/sellers/followed", "", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleListFollowed(rec, req)

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

func TestHandleListFollowed_WithItems(t *testing.T) {
	store := &mockSellerFollowStore{
		listByUserFn: func(_ context.Context, _ int64, _, _ int) ([]repo.FollowedSeller, int64, error) {
			return []repo.FollowedSeller{
				{ID: 1, SellerID: "s1", SellerName: "Shop1"},
				{ID: 2, SellerID: "s2", SellerName: "Shop2"},
			}, 2, nil
		},
	}
	h := NewSellerHandler(store)
	req := sellerReq(http.MethodGet, "/sellers/followed?page=1&page_size=20", "", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleListFollowed(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	total := data["total"].(float64)
	if total != 2 {
		t.Errorf("total = %v, want 2", total)
	}
}

func TestHandleCheckFollowed_Success(t *testing.T) {
	store := &mockSellerFollowStore{
		batchIsFollowingFn: func(_ context.Context, _ int64, ids []string) (map[string]bool, error) {
			return map[string]bool{"s1": true}, nil
		},
	}
	h := NewSellerHandler(store)
	req := sellerReq(http.MethodGet, "/sellers/check?ids=s1,s2", "", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleCheckFollowed(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if data["s1"] != true {
		t.Errorf("s1 = %v, want true", data["s1"])
	}
	if data["s2"] != false {
		t.Errorf("s2 = %v, want false", data["s2"])
	}
}

func TestHandleCheckFollowed_MissingIDs(t *testing.T) {
	h := NewSellerHandler(&mockSellerFollowStore{})
	req := sellerReq(http.MethodGet, "/sellers/check", "", 10, nil)
	rec := httptest.NewRecorder()
	h.HandleCheckFollowed(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
