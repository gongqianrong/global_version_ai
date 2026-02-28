package api

import (
	"context"
	"fmt"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/normalizer"
	"github.com/rakutao/collection-gateway/internal/registry"
)

// PlatformSearchService implements PlatformSearcher and PlatformLister
// by delegating to the adapter registry and normalizer.
type PlatformSearchService struct {
	registry   *registry.Registry
	normalizer *normalizer.Normalizer
}

// NewPlatformSearchService creates a PlatformSearchService.
func NewPlatformSearchService(reg *registry.Registry, norm *normalizer.Normalizer) *PlatformSearchService {
	return &PlatformSearchService{
		registry:   reg,
		normalizer: norm,
	}
}

// SearchPlatform implements PlatformSearcher.
// It fetches the adapter from the registry, calls Search, normalizes results,
// and maps them to ProductSummary.
func (s *PlatformSearchService) SearchPlatform(ctx context.Context, platformID string, query domain.SearchQuery) ([]domain.ProductSummary, int64, error) {
	adapter, err := s.registry.GetAdapter(platformID)
	if err != nil {
		return nil, 0, fmt.Errorf("platform service: %w", err)
	}

	result, err := adapter.Search(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("platform service: search %s: %w", platformID, err)
	}

	summaries := make([]domain.ProductSummary, 0, len(result.Products))
	for _, raw := range result.Products {
		product, err := s.normalizer.Normalize(platformID, raw)
		if err != nil {
			continue // skip products that fail normalization
		}

		summary := domain.ProductSummary{
			ID:            product.ID,
			Title:         product.Title,
			TitleOriginal: product.Title,
			PriceJPY:      product.PriceJPY,
			Platform:      product.SourcePlatform,
			Status:        product.Status,
			Condition:     product.Condition,
		}
		if len(product.Images) > 0 {
			summary.Image = product.Images[0]
		}
		if product.Brand != nil {
			summary.Brand = product.Brand.Name
		}
		summaries = append(summaries, summary)
	}

	return summaries, result.Total, nil
}

// RealtimePlatformIDs implements PlatformLister.
// Returns the IDs of all registered platforms that support real-time search.
func (s *PlatformSearchService) RealtimePlatformIDs() []string {
	metas := s.registry.RealtimeSearchable()
	ids := make([]string, len(metas))
	for i, m := range metas {
		ids[i] = m.ID
	}
	return ids
}
