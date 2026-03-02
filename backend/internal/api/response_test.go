package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/i18n"
)

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r = r.WithContext(context.WithValue(r.Context(), requestIDKey, "req-123"))

	Success(w, r, map[string]string{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
	if resp.RequestID != "req-123" {
		t.Errorf("request_id = %q, want %q", resp.RequestID, "req-123")
	}
	if resp.Message != "" {
		t.Errorf("message = %q, want empty", resp.Message)
	}
}

func TestErrorWithCode(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r = r.WithContext(context.WithValue(r.Context(), requestIDKey, "req-456"))

	ErrorWithCode(w, r, http.StatusBadRequest, 40001, "keyword blocked")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40001 {
		t.Errorf("code = %d, want 40001", resp.Code)
	}
	// Message is now auto-translated; default lang (zh-TW) is used when no lang in context.
	if resp.Message != "關鍵字被內容政策封鎖" {
		t.Errorf("message = %q, want zh-TW translation", resp.Message)
	}
}

func TestSuccess_NoRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	Success(w, r, "ok")

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.RequestID != "" {
		t.Errorf("request_id = %q, want empty", resp.RequestID)
	}
}

func TestWriteJSON_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"a": "b"})

	ct := w.Header().Get("Content-Type")
	want := "application/json; charset=utf-8"
	if ct != want {
		t.Errorf("Content-Type = %q, want %q", ct, want)
	}
}

func TestErrorWithCode_I18N(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	ctx := i18n.WithLang(req.Context(), i18n.LangJA)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	ErrorWithCode(w, req, http.StatusBadRequest, 40002, "missing required parameter: keyword")

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Message != "必須パラメータが不足しています" {
		t.Errorf("message = %q, want Japanese translation", resp.Message)
	}
}

func TestErrorWithCode_DefaultLang(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	ErrorWithCode(w, req, http.StatusNotFound, 40401, "product not found")

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Message != "找不到商品" {
		t.Errorf("message = %q, want zh-TW translation", resp.Message)
	}
}
