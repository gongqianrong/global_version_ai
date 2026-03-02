package api

import (
	"context"
	"fmt"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/i18n"
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

	lang := i18n.Lang(query.UserLang)

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
			Status:        i18n.StatusLabel(product.Status, lang),
			Condition:     i18n.ConditionLabel(product.Condition, lang),
		}
		if t, ok := product.TitleTranslated[string(lang)]; ok && t != "" {
			summary.Title = t
			summary.IsTranslated = true
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

// GetProduct fetches a single product from a platform adapter and normalizes it.
// The productID should be in unified format: "platform_rawID" (e.g. "surugaya_663043159").
func (s *PlatformSearchService) GetProduct(ctx context.Context, productID string) (*domain.UnifiedProduct, error) {
	platform, rawID := domain.ParseProductID(productID)
	if platform == "" || rawID == "" {
		return nil, fmt.Errorf("platform service: invalid product ID %q", productID)
	}

	adapter, err := s.registry.GetAdapter(platform)
	if err != nil {
		return nil, fmt.Errorf("platform service: %w", err)
	}

	raw, err := adapter.GetProduct(ctx, rawID)
	if err != nil {
		return nil, fmt.Errorf("platform service: get product %s/%s: %w", platform, rawID, err)
	}

	product, err := s.normalizer.Normalize(platform, *raw)
	if err != nil {
		return nil, fmt.Errorf("platform service: normalize %s/%s: %w", platform, rawID, err)
	}

	return product, nil
}

// SearchPlatformFull is like SearchPlatform but also returns the full UnifiedProduct
// slice so callers can persist them to Elasticsearch.
func (s *PlatformSearchService) SearchPlatformFull(ctx context.Context, platformID string, query domain.SearchQuery) ([]domain.ProductSummary, []domain.UnifiedProduct, int64, error) {
	adapter, err := s.registry.GetAdapter(platformID)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("platform service: %w", err)
	}

	result, err := adapter.Search(ctx, query)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("platform service: search %s: %w", platformID, err)
	}

	lang := i18n.Lang(query.UserLang)

	summaries := make([]domain.ProductSummary, 0, len(result.Products))
	fullProducts := make([]domain.UnifiedProduct, 0, len(result.Products))
	for _, raw := range result.Products {
		product, err := s.normalizer.Normalize(platformID, raw)
		if err != nil {
			continue
		}

		fullProducts = append(fullProducts, *product)

		summary := domain.ProductSummary{
			ID:            product.ID,
			Title:         product.Title,
			TitleOriginal: product.Title,
			PriceJPY:      product.PriceJPY,
			Platform:      product.SourcePlatform,
			Status:        i18n.StatusLabel(product.Status, lang),
			Condition:     i18n.ConditionLabel(product.Condition, lang),
		}
		if t, ok := product.TitleTranslated[string(lang)]; ok && t != "" {
			summary.Title = t
			summary.IsTranslated = true
		}
		if len(product.Images) > 0 {
			summary.Image = product.Images[0]
		}
		if product.Brand != nil {
			summary.Brand = product.Brand.Name
		}
		summaries = append(summaries, summary)
	}

	return summaries, fullProducts, result.Total, nil
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
