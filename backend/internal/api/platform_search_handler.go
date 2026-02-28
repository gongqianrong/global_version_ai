package api

import (
	"net/http"
)

// PlatformSearchHandler handles direct platform search requests (bypasses ES).
type PlatformSearchHandler struct {
	platformService *PlatformSearchService
}

// NewPlatformSearchHandler creates a PlatformSearchHandler.
func NewPlatformSearchHandler(ps *PlatformSearchService) *PlatformSearchHandler {
	return &PlatformSearchHandler{platformService: ps}
}

// HandleSearch handles GET /api/v1/platform/search.
// Required: keyword, platform
func (h *PlatformSearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := parseSearchParams(r)

	platform := r.URL.Query().Get("platform")
	if platform == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing required parameter: platform")
		return
	}
	if query.Keyword == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing required parameter: keyword")
		return
	}

	products, total, err := h.platformService.SearchPlatform(r.Context(), platform, query)
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, err.Error())
		return
	}

	Success(w, r, map[string]interface{}{
		"products": products,
		"total":    total,
		"platform": platform,
	})
}
