package aiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranslate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/translate" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req translateRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Keyword != "gucci bag" {
			t.Errorf("keyword = %q, want %q", req.Keyword, "gucci bag")
		}
		if req.SourceLang != "en" {
			t.Errorf("source_lang = %q, want %q", req.SourceLang, "en")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(translateResponse{
			KeywordJA:     "グッチ bag",
			Original:      "gucci bag",
			SourceLang:    "en",
			WasTranslated: true,
		})
	}))
	defer srv.Close()

	client := New(srv.URL, nil)
	result, err := client.Translate(context.Background(), "gucci bag", "en")
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if result != "グッチ bag" {
		t.Errorf("result = %q, want %q", result, "グッチ bag")
	}
}

func TestTranslate_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, nil)
	_, err := client.Translate(context.Background(), "test", "en")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestTranslate_ConnectionError(t *testing.T) {
	client := New("http://localhost:1", nil)
	_, err := client.Translate(context.Background(), "test", "en")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestExtract_BrandFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/extract-brand" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var req extractBrandRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Title != "GUCCI バッグ" {
			t.Errorf("title = %q", req.Title)
		}

		brandName := "GUCCI"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(extractBrandResponse{
			BrandName:  &brandName,
			Confidence: 0.95,
		})
	}))
	defer srv.Close()

	client := New(srv.URL, nil)
	result, err := client.Extract("GUCCI バッグ", "新品", "ファッション")
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.BrandName != "GUCCI" {
		t.Errorf("BrandName = %q", result.BrandName)
	}
	if result.Confidence != 0.95 {
		t.Errorf("Confidence = %f", result.Confidence)
	}
}

func TestExtract_NoBrand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(extractBrandResponse{
			BrandName:  nil,
			Confidence: 0.0,
		})
	}))
	defer srv.Close()

	client := New(srv.URL, nil)
	result, err := client.Extract("手作りバッグ", "", "")
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for no brand, got %+v", result)
	}
}

func TestExtract_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, nil)
	_, err := client.Extract("test", "", "")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
