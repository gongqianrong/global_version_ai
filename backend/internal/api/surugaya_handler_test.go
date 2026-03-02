package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/adapter/surugaya"
)

// newTestSurugayaHandler creates a SurugayaHandler backed by a mock HTTP server.
// The caller provides a handler func to simulate the upstream Surugaya API.
func newTestSurugayaHandler(upstream http.HandlerFunc) (*SurugayaHandler, *httptest.Server) {
	srv := httptest.NewServer(upstream)
	client := surugaya.NewClient(srv.URL, nil)
	return NewSurugayaHandler(client, nil), srv
}

// mockSurugayaAPI returns a handler that routes surugaya API paths to JSON responses.
func mockSurugayaAPI(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t.Helper()
		w.Header().Set("Content-Type", "application/json")

		var data interface{}

		switch {
		case r.URL.Path == "/suruga/product/extend/663043159":
			data = map[string]interface{}{
				"comments": []map[string]string{
					{"star": "5.0", "title": "素晴らしい", "userId": "u1", "userName": "テスト", "comment": "良い商品", "other": ""},
				},
				"otherList": []map[string]interface{}{
					{"img": "https://img.jpg", "mid": "663049229", "name": "類似商品", "price": 3000, "url": "https://suruga-ya.jp/product/detail/663049229"},
				},
			}

		case r.URL.Path == "/suruga/product/extend/store/663043159":
			data = map[string]interface{}{
				"stores": []map[string]interface{}{
					{"storeName": "駿河屋本店", "price": 4950, "stock": 2, "buyState": true, "state": "中古"},
				},
			}

		case r.URL.Path == "/suruga/product/discount":
			data = map[string]interface{}{
				"timeSale":    []map[string]string{{"title": "タイムセール", "content": "期間限定", "picLink": "https://pic.jpg"}},
				"unifiedSale": []map[string]interface{}{{"title": "まとめ買い", "content": "活動内容", "activeTime": "2026-03-01 ~ 2026-03-31", "conditions": []map[string]string{{"count": "3", "discount": "10%"}}}},
			}

		case r.URL.Path == "/suruga/product/user/comments":
			q := r.URL.Query()
			if q.Get("user_id") == "" {
				http.Error(w, "missing user_id", http.StatusBadRequest)
				return
			}
			data = map[string]interface{}{
				"comments": []map[string]string{
					{"comment": "良い商品です", "star": "4.0", "userId": q.Get("user_id"), "userName": "ユーザー"},
				},
				"platform": 0,
			}

		case r.URL.Path == "/suruga/product/campaign":
			data = map[string]interface{}{
				"campaigns": []map[string]interface{}{
					{"title": "春のキャンペーン", "detail": "詳細", "image": "https://img.jpg",
						"conditions": []map[string]string{{"num": "3", "discount": "10%OFF"}},
						"urlList":    []map[string]string{{"alt": "詳細", "url": "https://suruga-ya.jp/campaign/xxx"}}},
				},
			}

		case r.URL.Path == "/suruga/product/campaign/detail":
			if r.URL.Query().Get("url") == "" {
				http.Error(w, "missing url", http.StatusBadRequest)
				return
			}
			data = map[string]interface{}{
				"campaigns": []map[string]interface{}{
					{"title": "キャンペーン名", "desc": "説明", "image": "https://img.jpg", "condition": "条件"},
				},
				"groups": []map[string]interface{}{
					{"title": "グループ名", "items": []interface{}{}, "otherSearchParams": map[string]interface{}{}},
				},
			}

		default:
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		dataBytes, _ := json.Marshal(data)
		resp := map[string]interface{}{"code": 200, "data": json.RawMessage(dataBytes), "msg": "成功"}
		json.NewEncoder(w).Encode(resp)
	}
}

// --- Product Reviews ---

func TestHandleProductReviews_Success(t *testing.T) {
	handler, srv := newTestSurugayaHandler(mockSurugayaAPI(t))
	defer srv.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/surugaya/products/{id}/reviews", handler.HandleProductReviews)

	req := httptest.NewRequest("GET", "/api/v1/surugaya/products/663043159/reviews", nil)
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
	var result map[string]interface{}
	json.Unmarshal(data, &result)

	if _, ok := result["comments"]; !ok {
		t.Error("expected 'comments' in response data")
	}
	if _, ok := result["otherList"]; !ok {
		t.Error("expected 'otherList' in response data")
	}
}

func TestHandleProductReviews_MissingID(t *testing.T) {
	handler, srv := newTestSurugayaHandler(mockSurugayaAPI(t))
	defer srv.Close()

	// Use a route without {id} to simulate missing ID
	r := chi.NewRouter()
	r.Get("/api/v1/surugaya/products//reviews", handler.HandleProductReviews)

	req := httptest.NewRequest("GET", "/api/v1/surugaya/products//reviews", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// chi will 405/404 for empty path segments, so test handler directly
	req2 := httptest.NewRequest("GET", "/reviews", nil)
	w2 := httptest.NewRecorder()
	handler.HandleProductReviews(w2, req2)

	if w2.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w2.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w2.Body).Decode(&resp)
	if resp.Code != 40002 {
		t.Errorf("code = %d, want 40002", resp.Code)
	}
}

func TestHandleProductReviews_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := surugaya.NewClient(srv.URL, nil)
	handler := NewSurugayaHandler(client, nil)

	r := chi.NewRouter()
	r.Get("/api/v1/surugaya/products/{id}/reviews", handler.HandleProductReviews)

	req := httptest.NewRequest("GET", "/api/v1/surugaya/products/663043159/reviews", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 50003 {
		t.Errorf("code = %d, want 50003", resp.Code)
	}
}

// --- Product Stores ---

func TestHandleProductStores_Success(t *testing.T) {
	handler, srv := newTestSurugayaHandler(mockSurugayaAPI(t))
	defer srv.Close()

	r := chi.NewRouter()
	r.Get("/api/v1/surugaya/products/{id}/stores", handler.HandleProductStores)

	req := httptest.NewRequest("GET", "/api/v1/surugaya/products/663043159/stores", nil)
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
}

// --- Discounts ---

func TestHandleDiscounts_Success(t *testing.T) {
	handler, srv := newTestSurugayaHandler(mockSurugayaAPI(t))
	defer srv.Close()

	req := httptest.NewRequest("GET", "/api/v1/surugaya/discounts", nil)
	w := httptest.NewRecorder()
	handler.HandleDiscounts(w, req)

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

	if _, ok := result["timeSale"]; !ok {
		t.Error("expected 'timeSale' in response data")
	}
	if _, ok := result["unifiedSale"]; !ok {
		t.Error("expected 'unifiedSale' in response data")
	}
}

// --- User Comments ---

func TestHandleUserComments_Success(t *testing.T) {
	handler, srv := newTestSurugayaHandler(mockSurugayaAPI(t))
	defer srv.Close()

	req := httptest.NewRequest("GET", "/api/v1/surugaya/comments?user_id=user123&page=2&page_size=10&sort=-rate", nil)
	w := httptest.NewRecorder()
	handler.HandleUserComments(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
}

func TestHandleUserComments_MissingUserID(t *testing.T) {
	handler, srv := newTestSurugayaHandler(mockSurugayaAPI(t))
	defer srv.Close()

	req := httptest.NewRequest("GET", "/api/v1/surugaya/comments", nil)
	w := httptest.NewRecorder()
	handler.HandleUserComments(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40002 {
		t.Errorf("code = %d, want 40002", resp.Code)
	}
}

func TestHandleUserComments_Defaults(t *testing.T) {
	var capturedPage, capturedPageSize, capturedSort string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/suruga/product/user/comments" {
			q := r.URL.Query()
			capturedPage = q.Get("page_num")
			capturedPageSize = q.Get("page_size")
			capturedSort = q.Get("sort_type")

			data, _ := json.Marshal(map[string]interface{}{"comments": []interface{}{}, "platform": 0})
			resp := map[string]interface{}{"code": 200, "data": json.RawMessage(data), "msg": "成功"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	client := surugaya.NewClient(srv.URL, nil)
	handler := NewSurugayaHandler(client, nil)

	req := httptest.NewRequest("GET", "/comments?user_id=u1", nil)
	w := httptest.NewRecorder()
	handler.HandleUserComments(w, req)

	if capturedPage != "1" {
		t.Errorf("default page = %q, want %q", capturedPage, "1")
	}
	if capturedPageSize != "20" {
		t.Errorf("default page_size = %q, want %q", capturedPageSize, "20")
	}
	if capturedSort != "-created_at" {
		t.Errorf("default sort = %q, want %q", capturedSort, "-created_at")
	}
}

// --- Campaigns ---

func TestHandleCampaigns_Success(t *testing.T) {
	handler, srv := newTestSurugayaHandler(mockSurugayaAPI(t))
	defer srv.Close()

	req := httptest.NewRequest("GET", "/api/v1/surugaya/campaigns", nil)
	w := httptest.NewRecorder()
	handler.HandleCampaigns(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
}

// --- Campaign Detail ---

func TestHandleCampaignDetail_Success(t *testing.T) {
	handler, srv := newTestSurugayaHandler(mockSurugayaAPI(t))
	defer srv.Close()

	req := httptest.NewRequest("GET", "/api/v1/surugaya/campaigns/detail?url=https://www.suruga-ya.jp/campaign/xxx", nil)
	w := httptest.NewRecorder()
	handler.HandleCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
}

func TestHandleCampaignDetail_MissingURL(t *testing.T) {
	handler, srv := newTestSurugayaHandler(mockSurugayaAPI(t))
	defer srv.Close()

	req := httptest.NewRequest("GET", "/api/v1/surugaya/campaigns/detail", nil)
	w := httptest.NewRecorder()
	handler.HandleCampaignDetail(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40002 {
		t.Errorf("code = %d, want 40002", resp.Code)
	}
}
