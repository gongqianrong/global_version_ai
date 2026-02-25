package domain

// SearchQuery represents an inbound search request from a client. It supports
// cross-platform (global) search as well as platform-specific queries with
// optional filters.
type SearchQuery struct {
	Keyword       string   `json:"keyword"`
	KeywordJA     string   `json:"keyword_ja"`
	Platforms     []string `json:"platforms,omitempty"`
	BrandID       string   `json:"brand_id,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	PriceMin      int64    `json:"price_min,omitempty"`
	PriceMax      int64    `json:"price_max,omitempty"`
	Condition     []string `json:"condition,omitempty"`
	SortBy        string   `json:"sort_by,omitempty"`
	Page          int      `json:"page"`
	PageSize      int      `json:"page_size"`
	UserLang      string   `json:"user_lang,omitempty"`
	ContentRating string   `json:"content_rating,omitempty"`
}

// IsGlobalSearch reports whether the query targets all platforms (i.e. no
// specific platforms were requested).
func (q *SearchQuery) IsGlobalSearch() bool {
	return len(q.Platforms) == 0
}

// HasFilters reports whether any structured filters (BrandID, Categories,
// PriceMin, PriceMax, or Condition) are active on the query.
func (q *SearchQuery) HasFilters() bool {
	return q.BrandID != "" ||
		len(q.Categories) > 0 ||
		q.PriceMin > 0 ||
		q.PriceMax > 0 ||
		len(q.Condition) > 0
}

// SearchResponse is the response returned to the client for a search request.
// It contains cached results (from Elasticsearch) alongside a realtime stream
// ID that the client can use to receive live results via SSE/WebSocket.
type SearchResponse struct {
	CachedResults     []ProductSummary `json:"cached_results"`
	CachedTotal       int64            `json:"cached_total"`
	RealtimeStreamID  string           `json:"realtime_stream_id,omitempty"`
	TranslatedKeyword string           `json:"translated_keyword,omitempty"`
	Aggregations      SearchAggs       `json:"aggregations"`
}

// ProductSummary is a lightweight projection of a product used in search
// result listings.
type ProductSummary struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	TitleOriginal string   `json:"title_original"`
	Image         string   `json:"image"`
	PriceJPY      int64    `json:"price_jpy"`
	Platform      string   `json:"platform"`
	Status        string   `json:"status"`
	Brand         string   `json:"brand,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	IsTranslated  bool     `json:"is_translated"`
}

// SearchAggs holds the facet aggregations returned alongside search results,
// enabling the client to build filter UIs.
type SearchAggs struct {
	Platforms   []AggBucket  `json:"platforms"`
	Brands      []AggBucket  `json:"brands"`
	Categories  []AggBucket  `json:"categories"`
	PriceRanges []PriceRange `json:"price_ranges"`
}

// AggBucket is a single aggregation bucket with a key and a document count.
type AggBucket struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

// PriceRange represents a price histogram bucket.
type PriceRange struct {
	Min   int64 `json:"min"`
	Max   int64 `json:"max"`
	Count int64 `json:"count"`
}
