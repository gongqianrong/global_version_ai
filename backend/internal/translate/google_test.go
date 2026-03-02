package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGoogle_Translate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req googleRequest
		json.NewDecoder(r.Body).Decode(&req)
		if len(req.Q) != 1 || req.Q[0] != "テスト" {
			t.Errorf("unexpected q: %v", req.Q)
		}
		json.NewEncoder(w).Encode(googleResponse{
			Data: struct {
				Translations []struct {
					TranslatedText string `json:"translatedText"`
				} `json:"translations"`
			}{
				Translations: []struct {
					TranslatedText string `json:"translatedText"`
				}{{TranslatedText: "test"}},
			},
		})
	}))
	defer server.Close()

	// Override URL for testing.
	g := NewGoogle("fake-key", nil)
	g.apiKey = ""
	origURL := googleTranslateURL

	// Use a custom client that redirects to test server.
	tr := &testTransport{server: server}
	g.client = &http.Client{Transport: tr}
	_ = origURL

	result, err := g.Translate(context.Background(), "テスト", "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "test" {
		t.Errorf("got %q, want %q", result, "test")
	}
}

func TestGoogle_Translate_EmptyText(t *testing.T) {
	g := NewGoogle("fake-key", nil)
	result, err := g.Translate(context.Background(), "", "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}

func TestGoogle_BatchTranslate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req googleRequest
		json.NewDecoder(r.Body).Decode(&req)
		resp := googleResponse{}
		resp.Data.Translations = make([]struct {
			TranslatedText string `json:"translatedText"`
		}, len(req.Q))
		for i, q := range req.Q {
			resp.Data.Translations[i].TranslatedText = "t:" + q
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	g := NewGoogle("", nil)
	g.client = &http.Client{Transport: &testTransport{server: server}}

	results, err := g.BatchTranslate(context.Background(), []string{"a", "", "b"}, "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	if results[0] != "t:a" {
		t.Errorf("results[0] = %q, want %q", results[0], "t:a")
	}
	if results[1] != "" {
		t.Errorf("results[1] = %q, want empty", results[1])
	}
	if results[2] != "t:b" {
		t.Errorf("results[2] = %q, want %q", results[2], "t:b")
	}
}

func TestGoogle_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(googleResponse{
			Error: &struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			}{Message: "API key not valid", Code: 400},
		})
	}))
	defer server.Close()

	g := NewGoogle("bad-key", nil)
	g.client = &http.Client{Transport: &testTransport{server: server}}

	_, err := g.Translate(context.Background(), "test", "ja", "en")
	if err == nil {
		t.Fatal("expected error for bad API key")
	}
}

// testTransport redirects all requests to the test server.
type testTransport struct {
	server *httptest.Server
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = t.server.Listener.Addr().String()
	return http.DefaultTransport.RoundTrip(req)
}
