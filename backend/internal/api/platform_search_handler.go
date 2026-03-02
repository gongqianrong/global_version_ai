package api

import (
	"context"
	"net/http"

	"github.com/rakutao/collection-gateway/internal/i18n"
)

// PlatformSearchHandler handles direct platform search requests (bypasses ES).
type PlatformSearchHandler struct {
	platformService *PlatformSearchService
	productWriter   ProductWriter
}

// NewPlatformSearchHandler creates a PlatformSearchHandler.
// pw (ProductWriter) is optional; if nil, search results are not persisted.
func NewPlatformSearchHandler(ps *PlatformSearchService, pw ProductWriter) *PlatformSearchHandler {
	return &PlatformSearchHandler{platformService: ps, productWriter: pw}
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
