package search

import (
	"context"
	"errors"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// mockTranslator implements the Translator interface for testing.
type mockTranslator struct {
	result string
	err    error
}

func (m *mockTranslator) Translate(_ context.Context, _, _ string) (string, error) {
	return m.result, m.err
}

func TestPrepareQuery_EnglishKeyword_Translated(t *testing.T) {
	translator := &mockTranslator{result: "グッチ バッグ"}
	gw := NewGateway(translator)

	q := domain.SearchQuery{
		Keyword:  "gucci bag",
		UserLang: "en",
	}

	got, err := gw.prepareQuery(context.Background(), q)
	if err != nil {
		t.Fatalf("prepareQuery() unexpected error: %v", err)
	}

	if got.KeywordJA != "グッチ バッグ" {
		t.Errorf("KeywordJA = %q, want %q", got.KeywordJA, "グッチ バッグ")
	}
}

func TestPrepareQuery_JapaneseKeyword_PassThrough(t *testing.T) {
	translator := &mockTranslator{result: "should not be called"}
	gw := NewGateway(translator)

	q := domain.SearchQuery{
		Keyword:  "グッチ バッグ",
		UserLang: "ja",
	}

	got, err := gw.prepareQuery(context.Background(), q)
	if err != nil {
		t.Fatalf("prepareQuery() unexpected error: %v", err)
	}

	if got.KeywordJA != "グッチ バッグ" {
		t.Errorf("KeywordJA = %q, want %q", got.KeywordJA, "グッチ バッグ")
	}
}

func TestPrepareQuery_JapaneseKeyword_AutoDetect(t *testing.T) {
	// Even without UserLang="ja", Japanese text should be auto-detected.
	translator := &mockTranslator{result: "should not be called"}
	gw := NewGateway(translator)

	q := domain.SearchQuery{
		Keyword:  "グッチ バッグ",
		UserLang: "en",
	}

	got, err := gw.prepareQuery(context.Background(), q)
	if err != nil {
		t.Fatalf("prepareQuery() unexpected error: %v", err)
	}

	if got.KeywordJA != "グッチ バッグ" {
		t.Errorf("KeywordJA = %q, want %q", got.KeywordJA, "グッチ バッグ")
	}
}

func TestPrepareQuery_TranslationError_FallbackToOriginal(t *testing.T) {
	translator := &mockTranslator{err: errors.New("translation service unavailable")}
	gw := NewGateway(translator)

	q := domain.SearchQuery{
		Keyword:  "gucci bag",
		UserLang: "en",
	}

	got, err := gw.prepareQuery(context.Background(), q)
	if err != nil {
		t.Fatalf("prepareQuery() unexpected error: %v", err)
	}

	// On translation error, KeywordJA should fall back to the original keyword.
	if got.KeywordJA != "gucci bag" {
		t.Errorf("KeywordJA = %q, want %q (fallback to original)", got.KeywordJA, "gucci bag")
	}
}

func TestPrepareQuery_EmptyKeyword(t *testing.T) {
	translator := &mockTranslator{result: "should not be called"}
	gw := NewGateway(translator)

	q := domain.SearchQuery{
		Keyword:  "",
		UserLang: "en",
	}

	got, err := gw.prepareQuery(context.Background(), q)
	if err != nil {
		t.Fatalf("prepareQuery() unexpected error: %v", err)
	}

	if got.KeywordJA != "" {
		t.Errorf("KeywordJA = %q, want empty string", got.KeywordJA)
	}
}

func TestPrepareQuery_HiraganaKeyword(t *testing.T) {
	translator := &mockTranslator{result: "should not be called"}
	gw := NewGateway(translator)

	q := domain.SearchQuery{
		Keyword:  "ぐっち",
		UserLang: "ja",
	}

	got, err := gw.prepareQuery(context.Background(), q)
	if err != nil {
		t.Fatalf("prepareQuery() unexpected error: %v", err)
	}

	if got.KeywordJA != "ぐっち" {
		t.Errorf("KeywordJA = %q, want %q", got.KeywordJA, "ぐっち")
	}
}

func TestIsJapanese(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{"katakana", "グッチ", true},
		{"hiragana", "ぐっち", true},
		{"mixed katakana and space", "グッチ バッグ", true},
		{"english", "gucci bag", false},
		{"chinese", "古驰包", false},
		{"mixed english and katakana", "gucci グッチ", true},
		{"empty", "", false},
		{"numbers", "12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isJapanese(tt.s)
			if got != tt.want {
				t.Errorf("isJapanese(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}
