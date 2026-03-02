package output

import (
	"context"
	"log"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/i18n"
	"github.com/rakutao/collection-gateway/internal/translate"
)

// TranslateSink enriches products with translations before they are persisted.
// It is NOT a Sink — it's a pre-processing step called before dispatch.
type TranslateSink struct {
	translator translate.Translator
}

func NewTranslateSink(tr translate.Translator) *TranslateSink {
	return &TranslateSink{translator: tr}
}

// Enrich populates TitleTranslated and DescTranslated for all supported languages.
// Products are modified in place. Source language is assumed to be Japanese (ja).
func (s *TranslateSink) Enrich(ctx context.Context, products []domain.UnifiedProduct) error {
	for i := range products {
		p := &products[i]
		if p.TitleTranslated == nil {
			p.TitleTranslated = make(map[string]string)
		}
		if p.DescTranslated == nil {
			p.DescTranslated = make(map[string]string)
		}

		for _, lang := range i18n.SupportedLangs {
			if lang == i18n.LangJA {
				continue // source language, no translation needed
			}
			langStr := string(lang)

			// Skip if already translated
			if _, ok := p.TitleTranslated[langStr]; ok {
				continue
			}

			if p.Title != "" {
				translated, err := s.translator.Translate(ctx, p.Title, "ja", langStr)
				if err != nil {
					log.Printf("[translate] title error for %s→%s: %v", p.ID, langStr, err)
					continue
				}
				p.TitleTranslated[langStr] = translated
			}

			if p.Description != "" {
				translated, err := s.translator.Translate(ctx, p.Description, "ja", langStr)
				if err != nil {
					log.Printf("[translate] desc error for %s→%s: %v", p.ID, langStr, err)
					continue
				}
				p.DescTranslated[langStr] = translated
			}
		}
	}
	return nil
}
