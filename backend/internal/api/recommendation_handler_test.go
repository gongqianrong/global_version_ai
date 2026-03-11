package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// --- mocks ---

type mockPreferenceStore struct {
	setFn func(ctx context.Context, userID int64, categories []string) ([]domain.UserPreference, error)
	getFn func(ctx context.Context, userID int64) ([]domain.UserPreference, error)
}

func (m *mockPreferenceStore) SetPreferences(ctx context.Context, userID int64, categories []string) ([]domain.UserPreference, error) {
	return m.setFn(ctx, userID, categories)
}
func (m *mockPreferenceStore) GetPreferences(ctx context.Context, userID int64) ([]domain.UserPreference, error) {
	return m.getFn(ctx, userID)
}

type mockBrowsingStore struct {
	recordFn func(ctx context.Context, rec *domain.BrowsingRecord) error
}

func (m *mockBrowsingStore) Record(ctx context.Context, rec *domain.BrowsingRecord) error {
	return m.recordFn(ctx, rec)
}

type mockSearchHistoryStore struct {
	recordFn func(ctx context.Context, rec *domain.SearchRecord) error
}

func (m *mockSearchHistoryStore) Record(ctx context.Context, rec *domain.SearchRecord) error {
	return m.recordFn(ctx, rec)
}

type mockRecService struct {
	getFn         func(ctx context.Context, userID int64, listType string, refresh bool) ([]domain.RecommendationList, error)
	invalidateFn  func(ctx context.Context, userID int64)
}

func (m *mockRecService) GetRecommendations(ctx context.Context, userID int64, listType string, refresh bool) ([]domain.RecommendationList, error) {
	return m.getFn(ctx, userID, listType, refresh)
}

func (m *mockRecService) InvalidateCache(ctx context.Context, userID int64) {
	if m.invalidateFn != nil {
		m.invalidateFn(ctx, userID)
	}
}

func recReq(method, url string, body string, userID int64) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	ctx := context.WithValue(r.Context(), userIDCtxKey, userID)
	return r.WithContext(ctx)
}

func recReqWithChi(method, url string, body string, userID int64, params map[string]string) *http.Request {
	r := recReq(method, url, body, userID)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

// --- tests ---

func TestHandleSetPreferences_Success(t *testing.T) {
	now := time.Now()
	store := &mockPreferenceStore{
		setFn: func(_ context.Context, uid int64, cats []string) ([]domain.UserPreference, error) {
			result := make([]domain.UserPreference, len(cats))
			for i, c := range cats {
				result[i] = domain.UserPreference{ID: int64(i + 1), UserID: uid, Category: c, Weight: 1.0, CreatedAt: now}
			}
			return result, nil
		},
	}
	h := NewRecommendationHandler(store, nil, nil, nil)
	req := recReq(http.MethodPost, "/preferences", `{"categories":["フィギュア","ゲーム"]}`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleSetPreferences(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["code"].(float64) != 0 {
		t.Fatalf("code = %v, want 0", resp["code"])
	}
	data := resp["data"].(map[string]interface{})
	prefs := data["preferences"].([]interface{})
	if len(prefs) != 2 {
		t.Fatalf("preferences len = %d, want 2", len(prefs))
	}
}

func TestHandleSetPreferences_EmptyCategories(t *testing.T) {
	h := NewRecommendationHandler(&mockPreferenceStore{}, nil, nil, nil)
	req := recReq(http.MethodPost, "/preferences", `{"categories":[]}`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleSetPreferences(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleSetPreferences_InvalidBody(t *testing.T) {
	h := NewRecommendationHandler(&mockPreferenceStore{}, nil, nil, nil)
	req := recReq(http.MethodPost, "/preferences", `{invalid`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleSetPreferences(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleSetPreferences_StoreError(t *testing.T) {
	store := &mockPreferenceStore{
		setFn: func(_ context.Context, _ int64, _ []string) ([]domain.UserPreference, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewRecommendationHandler(store, nil, nil, nil)
	req := recReq(http.MethodPost, "/preferences", `{"categories":["test"]}`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleSetPreferences(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestHandleGetPreferences_Success(t *testing.T) {
	store := &mockPreferenceStore{
		getFn: func(_ context.Context, uid int64) ([]domain.UserPreference, error) {
			return []domain.UserPreference{
				{ID: 1, UserID: uid, Category: "フィギュア", Weight: 1.0, CreatedAt: time.Now()},
			}, nil
		},
	}
	h := NewRecommendationHandler(store, nil, nil, nil)
	req := recReq(http.MethodGet, "/preferences", "", 1)
	rec := httptest.NewRecorder()
	h.HandleGetPreferences(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	prefs := data["preferences"].([]interface{})
	if len(prefs) != 1 {
		t.Fatalf("preferences len = %d, want 1", len(prefs))
	}
}

func TestHandleGetPreferences_Empty(t *testing.T) {
	store := &mockPreferenceStore{
		getFn: func(_ context.Context, _ int64) ([]domain.UserPreference, error) {
			return nil, nil
		},
	}
	h := NewRecommendationHandler(store, nil, nil, nil)
	req := recReq(http.MethodGet, "/preferences", "", 1)
	rec := httptest.NewRecorder()
	h.HandleGetPreferences(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	prefs := data["preferences"].([]interface{})
	if len(prefs) != 0 {
		t.Fatalf("preferences should be empty, got %d", len(prefs))
	}
}

func TestHandleGetPreferences_Error(t *testing.T) {
	store := &mockPreferenceStore{
		getFn: func(_ context.Context, _ int64) ([]domain.UserPreference, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewRecommendationHandler(store, nil, nil, nil)
	req := recReq(http.MethodGet, "/preferences", "", 1)
	rec := httptest.NewRecorder()
	h.HandleGetPreferences(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestHandleTrackView_Success(t *testing.T) {
	store := &mockBrowsingStore{
		recordFn: func(_ context.Context, _ *domain.BrowsingRecord) error {
			return nil
		},
	}
	h := NewRecommendationHandler(nil, store, nil, nil)
	req := recReq(http.MethodPost, "/track/view", `{"productId":"test123","category":"フィギュア","brand":"Bandai","sellerId":"s1","platform":"surugaya"}`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleTrackView(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleTrackView_MissingProductID(t *testing.T) {
	h := NewRecommendationHandler(nil, &mockBrowsingStore{}, nil, nil)
	req := recReq(http.MethodPost, "/track/view", `{"category":"test"}`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleTrackView(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleTrackView_InvalidBody(t *testing.T) {
	h := NewRecommendationHandler(nil, &mockBrowsingStore{}, nil, nil)
	req := recReq(http.MethodPost, "/track/view", `{bad`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleTrackView(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleTrackSearch_Success(t *testing.T) {
	store := &mockSearchHistoryStore{
		recordFn: func(_ context.Context, _ *domain.SearchRecord) error {
			return nil
		},
	}
	h := NewRecommendationHandler(nil, nil, store, nil)
	req := recReq(http.MethodPost, "/track/search", `{"keyword":"ガンダム","keywordJa":"ガンダム","platform":"surugaya"}`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleTrackSearch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleTrackSearch_MissingKeyword(t *testing.T) {
	h := NewRecommendationHandler(nil, nil, &mockSearchHistoryStore{}, nil)
	req := recReq(http.MethodPost, "/track/search", `{"platform":"surugaya"}`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleTrackSearch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleTrackSearch_InvalidBody(t *testing.T) {
	h := NewRecommendationHandler(nil, nil, &mockSearchHistoryStore{}, nil)
	req := recReq(http.MethodPost, "/track/search", `{bad`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleTrackSearch(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleGetRecommendationList_InvalidType(t *testing.T) {
	h := NewRecommendationHandler(nil, nil, nil, nil)
	req := recReqWithChi(http.MethodGet, "/recommendations/invalid", "", 1, map[string]string{"listType": "invalid"})
	rec := httptest.NewRecorder()
	h.HandleGetRecommendationList(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleGetRecommendations_Success(t *testing.T) {
	svc := &mockRecService{
		getFn: func(_ context.Context, userID int64, listType string, refresh bool) ([]domain.RecommendationList, error) {
			return []domain.RecommendationList{
				{Type: "for_you", Title: "おすすめ", Items: []domain.RecommendedProduct{{ID: "p1"}}},
			}, nil
		},
	}
	h := NewRecommendationHandler(nil, nil, nil, svc)
	req := recReq(http.MethodGet, "/recommendations", "", 1)
	rec := httptest.NewRecorder()
	h.HandleGetRecommendations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	lists := data["lists"].([]interface{})
	if len(lists) != 1 {
		t.Fatalf("lists len = %d, want 1", len(lists))
	}
}

func TestHandleGetRecommendations_Error(t *testing.T) {
	svc := &mockRecService{
		getFn: func(_ context.Context, _ int64, _ string, _ bool) ([]domain.RecommendationList, error) {
			return nil, errors.New("es error")
		},
	}
	h := NewRecommendationHandler(nil, nil, nil, svc)
	req := recReq(http.MethodGet, "/recommendations", "", 1)
	rec := httptest.NewRecorder()
	h.HandleGetRecommendations(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestHandleGetRecommendationList_Success(t *testing.T) {
	svc := &mockRecService{
		getFn: func(_ context.Context, _ int64, listType string, _ bool) ([]domain.RecommendationList, error) {
			return []domain.RecommendationList{
				{Type: listType, Title: "新着アイテム", Items: []domain.RecommendedProduct{{ID: "p2"}}},
			}, nil
		},
	}
	h := NewRecommendationHandler(nil, nil, nil, svc)
	req := recReqWithChi(http.MethodGet, "/recommendations/new_arrivals", "", 1, map[string]string{"listType": "new_arrivals"})
	rec := httptest.NewRecorder()
	h.HandleGetRecommendationList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleSetPreferences_InvalidatesCacheOnSuccess(t *testing.T) {
	invalidated := false
	store := &mockPreferenceStore{
		setFn: func(_ context.Context, _ int64, _ []string) ([]domain.UserPreference, error) {
			return []domain.UserPreference{{ID: 1, Category: "test"}}, nil
		},
	}
	svc := &mockRecService{
		getFn: func(_ context.Context, _ int64, _ string, _ bool) ([]domain.RecommendationList, error) {
			return nil, nil
		},
		invalidateFn: func(_ context.Context, _ int64) {
			invalidated = true
		},
	}
	h := NewRecommendationHandler(store, nil, nil, svc)
	req := recReq(http.MethodPost, "/preferences", `{"categories":["test"]}`, 1)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleSetPreferences(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !invalidated {
		t.Error("expected cache to be invalidated after setting preferences")
	}
}
