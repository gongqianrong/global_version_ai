package api

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/i18n"
	"github.com/rakutao/collection-gateway/internal/search"
	"github.com/rakutao/collection-gateway/internal/translate"
)

// PlatformSearchHandler handles direct platform search requests (bypasses ES).
type PlatformSearchHandler struct {
	platformService *PlatformSearchService
	productWriter   ProductWriter
	esFetcher       *search.ESProductFetcher
	translator      translate.Translator
}

// NewPlatformSearchHandler creates a PlatformSearchHandler.
// pw (ProductWriter) is optional; if nil, search results are not persisted.
// esFetcher is optional; if set, existing translations are loaded from ES
// so that the user sees translated titles immediately.
// tr is optional; if set, untranslated titles are translated synchronously.
func NewPlatformSearchHandler(ps *PlatformSearchService, pw ProductWriter, esf *search.ESProductFetcher, tr translate.Translator) *PlatformSearchHandler {
	return &PlatformSearchHandler{platformService: ps, productWriter: pw, esFetcher: esf, translator: tr}
}

// HandleSearch handles GET /api/v1/platform/search.
// Required: keyword, platform
func (h *PlatformSearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := parseSearchParams(r)
	query.UserLang = string(i18n.FromContext(r.Context()))

	platform := r.URL.Query().Get("platform")
	if platform == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing required parameter: platform")
		return
	}
	if query.Keyword == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing required parameter: keyword")
		return
	}

	summaries, fullProducts, total, err := h.platformService.SearchPlatformFull(r.Context(), platform, query)
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, err.Error())
		return
	}

	// Load existing translations from ES so user sees them immediately.
	if h.esFetcher != nil && len(summaries) > 0 {
		ids := make([]string, len(summaries))
		for i, s := range summaries {
			ids[i] = s.ID
		}
		existing, err := h.esFetcher.BulkGetTranslations(r.Context(), ids)
		if err == nil && len(existing) > 0 {
			lang := query.UserLang
			for i := range summaries {
				if summaries[i].IsTranslated {
					continue
				}
				if tr, ok := existing[summaries[i].ID]; ok {
					if t, ok := tr.TitleTranslated[lang]; ok && t != "" {
						summaries[i].Title = t
						summaries[i].IsTranslated = true
					}
				}
			}
		}
	}

	// Synchronously translate remaining untranslated titles for the current language.
	if h.translator != nil && query.UserLang != string(i18n.LangJA) {
		h.translateSummaryTitles(r.Context(), summaries, fullProducts, query.UserLang)
	}

	// Async write full products to ES (with full translation for all languages).
	if h.productWriter != nil && len(fullProducts) > 0 {
		go h.productWriter.Dispatch(context.Background(), fullProducts)
	}

	Success(w, r, map[string]interface{}{
		"products": summaries,
		"total":    total,
		"platform": platform,
	})
}

// translateSummaryTitles concurrently translates untranslated titles in summaries.
// It also writes the translations into fullProducts so the async pipeline can reuse them.
func (h *PlatformSearchHandler) translateSummaryTitles(ctx context.Context, summaries []domain.ProductSummary, fullProducts []domain.UnifiedProduct, lang string) {
	// Build index of which summaries need translation.
	type job struct {
		summaryIdx int
		productIdx int // index in fullProducts, -1 if not found
		text       string
	}

	// Build product ID → fullProducts index map.
	productIdx := make(map[string]int, len(fullProducts))
	for i, p := range fullProducts {
		productIdx[p.ID] = i
	}

	var jobs []job
	for i := range summaries {
		if summaries[i].IsTranslated || summaries[i].TitleOriginal == "" {
			continue
		}
		pIdx := -1
		if idx, ok := productIdx[summaries[i].ID]; ok {
			pIdx = idx
		}
		jobs = append(jobs, job{summaryIdx: i, productIdx: pIdx, text: summaries[i].TitleOriginal})
	}

	if len(jobs) == 0 {
		return
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)
	var mu sync.Mutex

	for _, j := range jobs {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			translated, err := h.translator.Translate(ctx, j.text, "ja", lang)
			if err != nil {
				log.Printf("[translate] sync title error for %s→%s: %v", summaries[j.summaryIdx].ID, lang, err)
				return
			}

			mu.Lock()
			summaries[j.summaryIdx].Title = translated
			summaries[j.summaryIdx].IsTranslated = true
			// Also write to fullProducts so async pipeline can skip this.
			if j.productIdx >= 0 {
				if fullProducts[j.productIdx].TitleTranslated == nil {
					fullProducts[j.productIdx].TitleTranslated = make(map[string]string)
				}
				fullProducts[j.productIdx].TitleTranslated[lang] = translated
			}
			mu.Unlock()
		}(j)
	}

	wg.Wait()
}
