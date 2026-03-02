package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/i18n"
)

func TestRecovery_NoPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRecovery_WithPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 50001 {
		t.Errorf("code = %d, want 50001", resp.Code)
	}
}

func TestRequestID_SetsHeader(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := getRequestID(r)
		if id == "" {
			t.Error("expected request ID in context, got empty")
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header, got empty")
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	var ids []string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids = append(ids, getRequestID(r))
	}))

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(w, r)
	}

	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate request ID: %s", id)
		}
		seen[id] = true
	}
}

func TestLogger_PassesThrough(t *testing.T) {
	called := false
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestStatusWriter_CapturesStatus(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	sw.WriteHeader(http.StatusNotFound)

	if sw.status != http.StatusNotFound {
		t.Errorf("status = %d, want %d", sw.status, http.StatusNotFound)
	}
}

func TestLanguageMiddleware(t *testing.T) {
	handler := Language(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := i18n.FromContext(r.Context())
		w.Write([]byte(string(lang)))
	}))

	// With Accept-Language: ja
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "ja")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Body.String() != "ja" {
		t.Errorf("got %q, want %q", w.Body.String(), "ja")
	}

	// Without header — default zh-TW
	req2 := httptest.NewRequest("GET", "/", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Body.String() != "zh-TW" {
		t.Errorf("got %q, want %q", w2.Body.String(), "zh-TW")
	}
}
