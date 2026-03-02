package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLibreTranslate_Translate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		var req libreRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Q == "" || req.Source == "" || req.Target == "" {
			t.Errorf("missing fields: %+v", req)
		}
		json.NewEncoder(w).Encode(libreResponse{TranslatedText: "translated: " + req.Q})
	}))
	defer server.Close()

	tr := NewLibreTranslate(server.URL, nil)

	result, err := tr.Translate(context.Background(), "テスト", "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "translated: テスト" {
		t.Errorf("got %q, want %q", result, "translated: テスト")
	}
}

func TestLibreTranslate_Translate_LangMapping(t *testing.T) {
	var gotTarget string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req libreRequest
		json.NewDecoder(r.Body).Decode(&req)
		gotTarget = req.Target
		json.NewEncoder(w).Encode(libreResponse{TranslatedText: "ok"})
	}))
	defer server.Close()

	tr := NewLibreTranslate(server.URL, nil)
	tr.Translate(context.Background(), "test", "ja", "zh-TW")
	if gotTarget != "zt" {
		t.Errorf("zh-TW should map to 'zt', got %q", gotTarget)
	}
}

func TestLibreTranslate_Translate_EmptyText(t *testing.T) {
	tr := NewLibreTranslate("http://unused", nil)
	result, err := tr.Translate(context.Background(), "", "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("got %q, want empty string", result)
	}
}

func TestLibreTranslate_Translate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	tr := NewLibreTranslate(server.URL, nil)
	_, err := tr.Translate(context.Background(), "テスト", "ja", "en")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}

func TestLibreTranslate_Translate_ErrorField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(libreResponse{Error: "language not supported"})
	}))
	defer server.Close()

	tr := NewLibreTranslate(server.URL, nil)
	_, err := tr.Translate(context.Background(), "テスト", "ja", "xx")
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
}

func TestLibreTranslate_BatchTranslate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req libreRequest
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(libreResponse{TranslatedText: "t:" + req.Q})
	}))
	defer server.Close()

	tr := NewLibreTranslate(server.URL, nil)
	results, err := tr.BatchTranslate(context.Background(), []string{"a", "b", "c"}, "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	if results[1] != "t:b" {
		t.Errorf("results[1] = %q, want %q", results[1], "t:b")
	}
}
