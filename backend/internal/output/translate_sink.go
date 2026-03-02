package output

import (
	"context"
	"log"
	"sync"

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

// maxConcurrent limits the number of concurrent translation API calls.
const maxConcurrent = 10

// translateJob represents a single translation task.
type translateJob struct {
	productIdx int
	field      string // "title" or "desc"
	lang       string
	text       string
}

// Enrich populates TitleTranslated and DescTranslated for all supported languages.
// Products are modified in place. Source language is assumed to be Japanese (ja).
// Translations are executed concurrently with a bounded worker pool.
func (s *TranslateSink) Enrich(ctx context.Context, products []domain.UnifiedProduct) error {
	// Collect all translation jobs.
	var jobs []translateJob
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
				continue
			}
			langStr := string(lang)

			if _, ok := p.TitleTranslated[langStr]; !ok && p.Title != "" {
				jobs = append(jobs, translateJob{
					productIdx: i, field: "title", lang: langStr, text: p.Title,
				})
			}
			if _, ok := p.DescTranslated[langStr]; !ok && p.Description != "" {
				jobs = append(jobs, translateJob{
					productIdx: i, field: "desc", lang: langStr, text: p.Description,
				})
			}
		}
	}

	if len(jobs) == 0 {
		return nil
	}

	// Execute concurrently with a semaphore.
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, job := range jobs {
		wg.Add(1)
		go func(j translateJob) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			translated, err := s.translator.Translate(ctx, j.text, "ja", j.lang)
			if err != nil {
				log.Printf("[translate] %s error for %s→%s: %v", j.field, products[j.productIdx].ID, j.lang, err)
				return
			}

			mu.Lock()
			switch j.field {
			case "title":
				products[j.productIdx].TitleTranslated[j.lang] = translated
			case "desc":
				products[j.productIdx].DescTranslated[j.lang] = translated
			}
			mu.Unlock()
		}(job)
	}

	wg.Wait()
	return nil
}
