package translate

import (
	"context"
	"testing"
)

func TestNoop_Translate(t *testing.T) {
	tr := NewNoop()
	result, err := tr.Translate(context.Background(), "テスト", "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "テスト" {
		t.Errorf("got %q, want %q", result, "テスト")
	}
}

func TestNoop_BatchTranslate(t *testing.T) {
	tr := NewNoop()
	results, err := tr.BatchTranslate(context.Background(), []string{"a", "b"}, "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 || results[0] != "a" || results[1] != "b" {
		t.Errorf("got %v, want [a b]", results)
	}
}
