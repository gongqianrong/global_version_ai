// Package domestic implements a PlatformAdapter that proxies requests to a
// domestic version collection API. This is the primary integration path for
// platforms that already have a domestic service layer.
package domestic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// domesticSearchResponse is the JSON shape returned by the domestic API's
// /api/search endpoint.
type domesticSearchResponse struct {
	Items []domesticItem `json:"items"`
	Total int64          `json:"total"`
}

// domesticItem is a single product returned by the domestic API.
type domesticItem struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Price int64  `json:"price"`
}

// Adapter proxies requests to a domestic version collection API.
type Adapter struct {
	platformID  string
	domesticURL string
	client      *http.Client
}

// New creates a new domestic proxy adapter.
func New(platformID, domesticURL string, client *http.Client) *Adapter {
	return &Adapter{
		platformID:  platformID,
		domesticURL: domesticURL,
		client:      client,
	}
}

// PlatformID returns the unique identifier for this platform.
func (a *Adapter) PlatformID() string {
	return a.platformID
}

// Capabilities returns reasonable default capabilities for a domestic proxy.
func (a *Adapter) Capabilities() domain.AdapterCaps {
	return domain.AdapterCaps{
		SupportsSearch:       true,
		SupportsRealtime:     true,
		SupportsBatchCollect: false,
		HasBrandField:        false,
		HasCategoryField:     false,
		MaxQPS:               10,
	}
}

// Search calls the domestic API's /api/search endpoint with the given
// query and converts the response into a domain.SearchResult.
func (a *Adapter) Search(ctx context.Context, query domain.SearchQuery) (*domain.SearchResult, error) {
	u, err := url.Parse(a.domesticURL + "/api/search")
	if err != nil {
		return nil, fmt.Errorf("domestic: parse URL: %w", err)
	}

	q := u.Query()
	q.Set("platform", a.platformID)
	q.Set("keyword", query.Keyword)
	q.Set("page", strconv.Itoa(query.Page))
	q.Set("page_size", strconv.Itoa(query.PageSize))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("domestic: create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("domestic: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("domestic: unexpected status %d", resp.StatusCode)
	}

	var body domesticSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("domestic: decode response: %w", err)
	}

	products := make([]domain.RawProduct, 0, len(body.Items))
	for _, item := range body.Items {
		products = append(products, domain.RawProduct{
			Platform: a.platformID,
			RawID:    item.ID,
			RawData: map[string]interface{}{
				"id":    item.ID,
				"title": item.Title,
				"price": item.Price,
			},
		})
	}

	hasMore := int64(query.Page*query.PageSize) < body.Total

	return &domain.SearchResult{
		Products: products,
		Total:    body.Total,
		HasMore:  hasMore,
	}, nil
}

// GetProduct is not yet implemented for the domestic proxy adapter.
func (a *Adapter) GetProduct(_ context.Context, _ string) (*domain.RawProduct, error) {
	return nil, fmt.Errorf("domestic: GetProduct not yet implemented")
}

// BatchCollect is not yet implemented for the domestic proxy adapter.
func (a *Adapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) {
	return nil, fmt.Errorf("domestic: BatchCollect not yet implemented")
}

// HealthCheck calls the domestic API's /health endpoint and reports whether
// the service is healthy.
func (a *Adapter) HealthCheck(ctx context.Context) domain.HealthStatus {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.domesticURL+"/health", nil)
	if err != nil {
		return domain.HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("domestic: create health request: %v", err),
		}
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return domain.HealthStatus{
			Status:  "unhealthy",
			Message: fmt.Sprintf("domestic: health request: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return domain.HealthStatus{
			Status:  "healthy",
			Message: "domestic API is reachable",
		}
	}

	return domain.HealthStatus{
		Status:  "unhealthy",
		Message: fmt.Sprintf("domestic API returned status %d", resp.StatusCode),
	}
}
