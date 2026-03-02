package api

import (
	"context"
	"net/http"

	"github.com/rakutao/collection-gateway/internal/i18n"
	"github.com/rakutao/collection-gateway/internal/search"
)

// PlatformSearchHandler handles direct platform search requests (bypasses ES).
type PlatformSearchHandler struct {
	platformService *PlatformSearchService
	productWriter   ProductWriter
	esFetcher       *search.ESProductFetcher
}

// NewPlatformSearchHandler creates a PlatformSearchHandler.
// pw (ProductWriter) is optional; if nil, search results are not persisted.
// esFetcher is optional; if set, existing translations are loaded from ES
// so that the user sees translated titles immediately.
func NewPlatformSearchHandler(ps *PlatformSearchService, pw ProductWriter, esf *search.ESProductFetcher) *PlatformSearchHandler {
	return &PlatformSearchHandler{platformService: ps, productWriter: pw, esFetcher: esf}
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
					continue // already has translation
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

	// Async write full products to ES.
	if h.productWriter != nil && len(fullProducts) > 0 {
		go h.productWriter.Dispatch(context.Background(), fullProducts)
	}

	Success(w, r, map[string]interface{}{
		"products": summaries,
		"total":    total,
		"platform": platform,
	})
}
