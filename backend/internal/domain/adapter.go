package domain

import (
	"context"
	"errors"
)

// ErrAdapterNotFound is returned when a platform adapter is not registered.
var ErrAdapterNotFound = errors.New("domain: adapter not found")

// AdapterCaps declares the capabilities of a platform adapter, allowing the
// gateway to decide how to route searches and collection jobs.
type AdapterCaps struct {
	SupportsSearch       bool `json:"supports_search"`
	SupportsRealtime     bool `json:"supports_realtime"`
	SupportsBatchCollect bool `json:"supports_batch_collect"`
	HasBrandField        bool `json:"has_brand_field"`
	HasCategoryField     bool `json:"has_category_field"`
	MaxQPS               int  `json:"max_qps"`
}

// CanRealtimeSearch reports whether the adapter supports real-time search,
// which requires both search capability and realtime capability.
func (c AdapterCaps) CanRealtimeSearch() bool {
	return c.SupportsSearch && c.SupportsRealtime
}

// RawProduct holds the unprocessed product data fetched from a platform,
// together with an optional normalised form.
type RawProduct struct {
	Platform   string                 `json:"platform"`
	RawID      string                 `json:"raw_id"`
	RawData    map[string]interface{} `json:"raw_data"`
	Normalized *UnifiedProduct        `json:"normalized,omitempty"`
}

// HealthStatus describes the current health of a platform adapter.
type HealthStatus struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// CollectParams configures a batch collection job against a platform.
type CollectParams struct {
	Categories []string `json:"categories"`
	MaxItems   int      `json:"max_items"`
	SinceTime  int64    `json:"since_time"`
}

// SearchResult is the response returned by a platform adapter's Search method.
type SearchResult struct {
	Products []RawProduct `json:"products"`
	Total    int64        `json:"total"`
	HasMore  bool         `json:"has_more"`
}

// PlatformAdapter is the interface every marketplace platform must implement
// in order to be plugged into the collection gateway.
type PlatformAdapter interface {
	// PlatformID returns the unique identifier for this platform
	// (e.g. "yahoo_auction", "amazon_jp", "surugaya", "animate", "lashinbang").
	PlatformID() string

	// Capabilities returns the declared capabilities of this adapter.
	Capabilities() AdapterCaps

	// Search performs a keyword search on the platform and returns raw results.
	Search(ctx context.Context, query SearchQuery) (*SearchResult, error)

	// GetProduct fetches a single product by its platform-specific ID.
	GetProduct(ctx context.Context, productID string) (*RawProduct, error)

	// BatchCollect runs a batch collection job with the given parameters.
	BatchCollect(ctx context.Context, params CollectParams) (<-chan RawProduct, error)

	// HealthCheck returns the current health status of the adapter.
	HealthCheck(ctx context.Context) HealthStatus
}
