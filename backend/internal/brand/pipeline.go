package brand

import (
	"strings"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// aiConfidenceThreshold is the minimum confidence level required for an AI
// extraction result to be accepted.
const aiConfidenceThreshold = 0.7

// AIExtractionResult holds the output of an AI-based brand identification.
type AIExtractionResult struct {
	BrandName  string  `json:"brand_name"`
	Confidence float64 `json:"confidence"`
}

// AIExtractor is an interface for AI-powered brand identification.
type AIExtractor interface {
	Extract(title, description, category string) (*AIExtractionResult, error)
}

// Pipeline implements 3-level brand identification:
//  1. Platform field — check the raw data for a brand/brand_name field.
//  2. Rule-based — scan title and description for known brand aliases.
//  3. AI extraction — call an external AI service and validate confidence.
type Pipeline struct {
	registry    *Registry
	aiExtractor AIExtractor
}

// NewPipeline creates a Pipeline with the given brand registry and an optional
// AI extractor. Pass nil for aiExtractor if AI extraction is not available.
func NewPipeline(registry *Registry, aiExtractor AIExtractor) *Pipeline {
	return &Pipeline{
		registry:    registry,
		aiExtractor: aiExtractor,
	}
}

// Identify runs the 3-level pipeline and returns a *domain.Brand if a brand is
// identified, or nil otherwise.
func (p *Pipeline) Identify(rawData map[string]interface{}, title, description, category string) *domain.Brand {
	// Level 1: Check platform field.
	if brand := p.level1(rawData); brand != nil {
		return brand
	}

	// Level 2: Rule-based text matching.
	if brand := p.level2(title, description); brand != nil {
		return brand
	}

	// Level 3: AI extraction.
	if brand := p.level3(title, description, category); brand != nil {
		return brand
	}

	return nil
}

// level1 checks for brand information in the raw platform data.
// It looks for "brand" and "brand_name" fields.
func (p *Pipeline) level1(rawData map[string]interface{}) *domain.Brand {
	if rawData == nil {
		return nil
	}

	// Try common field names for brand.
	brandFields := []string{"brand", "brand_name", "brandName"}
	for _, field := range brandFields {
		if v, ok := rawData[field]; ok {
			if brandStr, ok := v.(string); ok && brandStr != "" {
				brandStr = strings.TrimSpace(brandStr)
				// Look up in registry for canonical data.
				if entry := p.registry.FindByAlias(brandStr); entry != nil {
					return &domain.Brand{
						ID:     entry.ID,
						Name:   entry.NameStd,
						NameJA: entry.NameJA,
						Source: "platform_field",
					}
				}
				// Not in registry, return raw value.
				return &domain.Brand{
					Name:   brandStr,
					Source: "platform_field",
				}
			}
		}
	}

	return nil
}

// level2 performs rule-based matching by scanning the title and description
// for known brand aliases from the registry.
func (p *Pipeline) level2(title, description string) *domain.Brand {
	// Try title first.
	if title != "" {
		if entry := p.registry.MatchInText(title); entry != nil {
			return &domain.Brand{
				ID:     entry.ID,
				Name:   entry.NameStd,
				NameJA: entry.NameJA,
				Source: "rule_matched",
			}
		}
	}

	// Then try description.
	if description != "" {
		if entry := p.registry.MatchInText(description); entry != nil {
			return &domain.Brand{
				ID:     entry.ID,
				Name:   entry.NameStd,
				NameJA: entry.NameJA,
				Source: "rule_matched",
			}
		}
	}

	return nil
}

// level3 calls the AI extractor if available, validates the confidence threshold,
// and optionally matches the result against the registry.
func (p *Pipeline) level3(title, description, category string) *domain.Brand {
	if p.aiExtractor == nil {
		return nil
	}

	result, err := p.aiExtractor.Extract(title, description, category)
	if err != nil || result == nil {
		return nil
	}

	if result.Confidence < aiConfidenceThreshold {
		return nil
	}

	// Try to match against registry for canonical data.
	if entry := p.registry.FindByAlias(result.BrandName); entry != nil {
		return &domain.Brand{
			ID:     entry.ID,
			Name:   entry.NameStd,
			NameJA: entry.NameJA,
			Source: "ai_extracted",
		}
	}

	// Brand not in registry — still return AI result.
	return &domain.Brand{
		Name:   result.BrandName,
		Source: "ai_extracted",
	}
}
