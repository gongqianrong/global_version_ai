package yahoo_auction

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// Adapter implements domain.PlatformAdapter for Yahoo Auction Japan,
// proxying requests to a domestic API backend.
type Adapter struct {
	client *Client
}

// New creates a Yahoo Auction adapter that proxies to the given domestic API URL.
func New(baseURL string, httpClient *http.Client) *Adapter {
	return &Adapter{
		client: NewClient(baseURL, httpClient),
	}
}

// PlatformID returns "yahoo_auction".
func (a *Adapter) PlatformID() string {
	return "yahoo_auction"
}

// Capabilities returns the adapter's declared capabilities.
func (a *Adapter) Capabilities() domain.AdapterCaps {
	return domain.AdapterCaps{
		SupportsSearch:       true,
		SupportsRealtime:     true,
		SupportsBatchCollect: false,
		HasBrandField:        false,
		HasCategoryField:     true,
		MaxQPS:               10,
	}
}

// Search performs a keyword search via the domestic API and returns raw products.
func (a *Adapter) Search(ctx context.Context, query domain.SearchQuery) (*domain.SearchResult, error) {
	keyword := query.KeywordJA
	if keyword == "" {
		keyword = query.Keyword
	}

	resp, err := a.client.Search(ctx, keyword, query.Page, query.PageSize)
	if err != nil {
		return nil, err
	}

	products := make([]domain.RawProduct, 0, len(resp.Items))
	for _, it := range resp.Items {
		products = append(products, itemToRawProduct(it))
	}

	hasMore := int64(query.Page*query.PageSize) < resp.Total

	return &domain.SearchResult{
		Products: products,
		Total:    resp.Total,
		HasMore:  hasMore,
	}, nil
}

// GetProduct fetches a single product by ID via the domestic API.
func (a *Adapter) GetProduct(ctx context.Context, productID string) (*domain.RawProduct, error) {
	it, err := a.client.GetProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	rp := itemToRawProduct(*it)
	return &rp, nil
}

// BatchCollect is not implemented.
func (a *Adapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) {
	return nil, fmt.Errorf("yahoo_auction: BatchCollect not implemented")
}

// HealthCheck checks the domestic API health endpoint.
func (a *Adapter) HealthCheck(ctx context.Context) domain.HealthStatus {
	if err := a.client.Health(ctx); err != nil {
		return domain.HealthStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}
	return domain.HealthStatus{
		Status:  "healthy",
		Message: "yahoo_auction domestic API is reachable",
	}
}

// itemToRawProduct converts a domestic API item to a domain.RawProduct.
func itemToRawProduct(it item) domain.RawProduct {
	rawData := map[string]interface{}{
		"title": it.Title,
		"price": it.Price,
	}
	if it.Description != "" {
		rawData["description"] = it.Description
	}
	if len(it.Images) > 0 {
		rawData["images"] = it.Images
	}
	if it.Status != "" {
		rawData["status"] = it.Status
	}
	if it.Condition != "" {
		rawData["condition"] = it.Condition
	}
	if it.Category != "" {
		rawData["category"] = it.Category
	}
	return domain.RawProduct{
		Platform: "yahoo_auction",
		RawID:    it.ID,
		RawData:  rawData,
	}
}
