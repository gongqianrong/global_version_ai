package output

import (
	"context"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/i18n"
	"github.com/rakutao/collection-gateway/internal/translate"
)

func TestTranslateSink_Enrich(t *testing.T) {
	tr := translate.NewNoop()
	sink := NewTranslateSink(tr)

	products := []domain.UnifiedProduct{
		{
			ID:          "test_1",
			Title:       "テスト商品",
			Description: "テスト説明",
		},
	}

	err := sink.Enrich(context.Background(), products)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Noop translator returns original text
	for _, lang := range i18n.SupportedLangs {
		if lang == i18n.LangJA {
			continue
		}
		langStr := string(lang)
		if _, ok := products[0].TitleTranslated[langStr]; !ok {
			t.Errorf("missing TitleTranslated for %s", langStr)
		}
		if _, ok := products[0].DescTranslated[langStr]; !ok {
			t.Errorf("missing DescTranslated for %s", langStr)
		}
	}
}

func TestTranslateSink_SkipsExisting(t *testing.T) {
	tr := translate.NewNoop()
	sink := NewTranslateSink(tr)

	products := []domain.UnifiedProduct{
		{
			ID:              "test_2",
			Title:           "テスト",
			TitleTranslated: map[string]string{"en": "Existing Translation"},
		},
	}

	sink.Enrich(context.Background(), products)

	// Should NOT overwrite existing translation
	if products[0].TitleTranslated["en"] != "Existing Translation" {
		t.Errorf("existing translation was overwritten: %q", products[0].TitleTranslated["en"])
	}
}
