package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/search"
)

// Searcher executes a search query and returns results.
// This interface abstracts the Elasticsearch client for testability.
type Searcher interface {
	Search(ctx context.Context, query domain.SearchQuery) (*domain.SearchResponse, error)
}

// QueryPreparer translates keywords and checks the blacklist.
type QueryPreparer interface {
	PrepareQuery(ctx context.Context, q domain.SearchQuery) (domain.SearchQuery, error)
}

// SearchHandler handles HTTP search requests.
type SearchHandler struct {
	preparer      QueryPreparer
	searcher      Searcher
	streamManager *StreamManager
}

// NewSearchHandler creates a SearchHandler with the given dependencies.
func NewSearchHandler(preparer QueryPreparer, searcher Searcher, sm *StreamManager) *SearchHandler {
	return &SearchHandler{
		preparer:      preparer,
		searcher:      searcher,
		streamManager: sm,
	}
}

// HandleSearch handles GET /api/v1/search.
func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := parseSearchParams(r)

	if query.Keyword == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing required parameter: keyword")
		return
	}

	// Translate keyword and check blacklist.
	prepared, err := h.preparer.PrepareQuery(r.Context(), query)
	if err == search.ErrBlockedKeyword {
		ErrorWithCode(w, r, http.StatusBadRequest, 40001, "keyword is blocked by content policy")
		return
	}
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "internal server error")
		return
	}

	// Execute ES search.
	result, err := h.searcher.Search(r.Context(), prepared)
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, "search service unavailable")
		return
	}

	// Create real-time search stream.
	streamID := h.streamManager.Create(prepared)
	result.RealtimeStreamID = streamID

	Success(w, r, result)
}

// parseSearchParams extracts SearchQuery fields from URL query parameters.
func parseSearchParams(r *http.Request) domain.SearchQuery {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	priceMin, _ := strconv.ParseInt(q.Get("price_min"), 10, 64)
	priceMax, _ := strconv.ParseInt(q.Get("price_max"), 10, 64)

	lang := q.Get("lang")
	if lang == "" {
		lang = "zh-TW"
	}

	contentRating := q.Get("content_rating")
	if contentRating == "" {
		contentRating = "general"
	}

	return domain.SearchQuery{
		Keyword:       q.Get("keyword"),
		Platforms:     splitCSV(q.Get("platforms")),
		BrandID:       q.Get("brand_id"),
		Categories:    splitCSV(q.Get("categories")),
		PriceMin:      priceMin,
		PriceMax:      priceMax,
		Condition:     splitCSV(q.Get("condition")),
		SortBy:        q.Get("sort"),
		Page:          page,
		PageSize:      pageSize,
		UserLang:      lang,
		ContentRating: contentRating,
	}
}

// splitCSV splits a comma-separated string into a slice, trimming whitespace.
// Returns nil for empty input.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
