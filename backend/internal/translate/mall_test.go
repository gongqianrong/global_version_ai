package translate

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMall_Translate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		src := r.URL.Query().Get("sourceText")
		tgt := r.URL.Query().Get("target")
		if src == "" || tgt == "" {
			t.Errorf("missing params: sourceText=%q, target=%q", src, tgt)
		}
		json.NewEncoder(w).Encode(mallResponse{Code: "000000", Data: "translated:" + src, Msg: "成功"})
	}))
	defer server.Close()

	m := NewMall(server.URL, nil)
	result, err := m.Translate(context.Background(), "テスト", "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "translated:テスト" {
		t.Errorf("got %q, want %q", result, "translated:テスト")
	}
}

func TestMall_Translate_ZhTW(t *testing.T) {
	var gotTarget string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotTarget = r.URL.Query().Get("target")
		// Return simplified Chinese, should be converted to traditional.
		json.NewEncoder(w).Encode(mallResponse{Code: "000000", Data: "测试商品", Msg: "成功"})
	}))
	defer server.Close()

	m := NewMall(server.URL, nil)
	result, err := m.Translate(context.Background(), "テスト商品", "ja", "zh-TW")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTarget != "zh-CN" {
		t.Errorf("zh-TW should map to zh-CN, got %q", gotTarget)
	}
	// "测试商品" → "測試商品" (traditional)
	if result != "測試商品" {
		t.Errorf("got %q, want traditional Chinese %q", result, "測試商品")
	}
}

func TestMall_Translate_EmptyText(t *testing.T) {
	m := NewMall("http://unused", nil)
	result, err := m.Translate(context.Background(), "", "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}

func TestMall_Translate_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(mallResponse{Code: "-999996", Msg: "请求失败"})
	}))
	defer server.Close()

	m := NewMall(server.URL, nil)
	_, err := m.Translate(context.Background(), "test", "ja", "en")
	if err == nil {
		t.Fatal("expected error for failed API call")
	}
}

func TestMall_BatchTranslate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		src := r.URL.Query().Get("sourceText")
		json.NewEncoder(w).Encode(mallResponse{Code: "000000", Data: "t:" + src, Msg: "成功"})
	}))
	defer server.Close()

	m := NewMall(server.URL, nil)
	results, err := m.BatchTranslate(context.Background(), []string{"a", "b"}, "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	if results[0] != "t:a" || results[1] != "t:b" {
		t.Errorf("got %v", results)
	}
}
