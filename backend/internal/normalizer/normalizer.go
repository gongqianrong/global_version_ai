// Package normalizer transforms RawProduct data from platform adapters into
// the canonical UnifiedProduct representation.
package normalizer

import (
	"errors"
	"strings"
	"time"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// BrandExtractor is an optional dependency that can identify brand information
// from product text fields.
type BrandExtractor interface {
	Extract(title, description, category string) (*domain.Brand, error)
}

// Normalizer converts RawProduct data into UnifiedProduct.
type Normalizer struct {
	brandExtractor BrandExtractor
}

// New creates a Normalizer with an optional BrandExtractor.
// Pass nil if brand extraction is not needed.
func New(brandExtractor BrandExtractor) *Normalizer {
	return &Normalizer{
		brandExtractor: brandExtractor,
	}
}

// Normalize transforms a RawProduct into a UnifiedProduct. The platform string
// and raw.RawID are used to generate the canonical product ID. Title and price
// are required fields; their absence causes an error.
func (n *Normalizer) Normalize(platform string, raw domain.RawProduct) (*domain.UnifiedProduct, error) {
	if raw.RawData == nil {
		return nil, errors.New("normalizer: RawData is nil")
	}

	title := extractString(raw.RawData, "title")
	if title == "" {
		return nil, errors.New("normalizer: missing required field 'title'")
	}

	price := extractFloat64(raw.RawData, "price")
	if price == 0 {
		return nil, errors.New("normalizer: missing required field 'price'")
	}

	description := extractString(raw.RawData, "description")
	images := extractStringSlice(raw.RawData, "images")
	category := extractString(raw.RawData, "category")
	statusStr := extractString(raw.RawData, "status")
	conditionStr := extractString(raw.RawData, "condition")

	// Optional fields provided by platform adapters.
	sourceURL := extractString(raw.RawData, "source_url")
	contentRating := extractString(raw.RawData, "content_rating")
	sellerID := extractString(raw.RawData, "seller_id")
	sellerName := extractString(raw.RawData, "seller_name")
	subcategory := extractString(raw.RawData, "subcategory")
	janCode := extractString(raw.RawData, "jan_code")
	manufacturer := extractString(raw.RawData, "manufacturer")

	basePriceJPY := int64(price)
	serviceFeeJPY := basePriceJPY * 10 / 100
	finalPriceJPY := basePriceJPY + serviceFeeJPY

	product := &domain.UnifiedProduct{
		ID:             domain.NewProductID(platform, raw.RawID),
		SourcePlatform: platform,
		SourceID:       raw.RawID,
		SourceURL:      sourceURL,
		Title:          title,
		Description:    description,
		Images:         images,
		PriceJPY:       finalPriceJPY,
		ServiceFeeJPY:  serviceFeeJPY,
		OriginalPrice:  basePriceJPY,
		SourceCategory: category,
		Status:         mapStatus(statusStr),
		Condition:      mapCondition(conditionStr),
		Quantity:        1,
		ContentRating:   domain.ContentRatingGeneral,
		CollectedAt:    time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Override content rating if adapter provides it.
	if contentRating == domain.ContentRatingR18 {
		product.ContentRating = domain.ContentRatingR18
	}

	// Set seller info if adapter provides it.
	if sellerID != "" {
		product.Seller = domain.SellerInfo{
			SellerID:   sellerID,
			SellerName: sellerName,
		}
	}

	// Build tags from optional metadata fields.
	var tags []string
	if subcategory != "" {
		tags = append(tags, subcategory)
	}
	if manufacturer != "" {
		tags = append(tags, manufacturer)
	}
	if janCode != "" {
		tags = append(tags, "JAN:"+janCode)
	}
	product.Tags = tags

	// If a brand_name is directly provided by the adapter, use it.
	brandName := extractString(raw.RawData, "brand_name")
	if brandName != "" {
		product.Brand = &domain.Brand{
			Name:   brandName,
			Source: "platform_field",
		}
	}

	// Call brand extractor if available and no brand set yet.
	if product.Brand == nil && n.brandExtractor != nil {
		brand, err := n.brandExtractor.Extract(title, description, category)
		if err == nil && brand != nil {
			product.Brand = brand
		}
	}

	return product, nil
}

// --- Helper functions ---

// extractString retrieves a string value from a map by key.
func extractString(data map[string]interface{}, key string) string {
	v, ok := data[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// extractFloat64 retrieves a numeric value from a map by key.
func extractFloat64(data map[string]interface{}, key string) float64 {
	v, ok := data[key]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

// extractStringSlice retrieves a []string from a map by key. It handles both
// []string and []interface{} values (the latter is common from JSON decoding).
func extractStringSlice(data map[string]interface{}, key string) []string {
	v, ok := data[key]
	if !ok {
		return nil
	}
	switch s := v.(type) {
	case []string:
		return s
	case []interface{}:
		result := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}
		return result
	default:
		return nil
	}
}

// --- Status mapping ---

// mapStatus converts platform-specific status strings to domain status constants.
func mapStatus(s string) string {
	switch strings.ToLower(s) {
	case "on_sale", "available", "active":
		return domain.StatusAvailable
	case "sold", "sold_out", "品切れ":
		return domain.StatusSold
	case "reserved", "trading":
		return domain.StatusReserved
	case "delisted", "cancelled":
		return domain.StatusDelisted
	default:
		return domain.StatusAvailable
	}
}

// --- Condition mapping ---

// mapCondition converts condition strings (Japanese and English) to domain
// condition constants. English values are matched case-insensitively.
func mapCondition(s string) string {
	// Try exact match first for Japanese strings (Mercari/Yahoo style).
	switch s {
	case "新品、未使用":
		return domain.ConditionNew
	case "未使用に近い":
		return domain.ConditionLikeNew
	case "目立った傷や汚れなし":
		return domain.ConditionGood
	case "やや傷や汚れあり":
		return domain.ConditionFair
	case "傷や汚れあり", "全体的に状態が悪い":
		return domain.ConditionPoor
	}

	// Surugaya condition grades.
	switch s {
	case "新品":
		return domain.ConditionNew
	case "A+":
		return domain.ConditionLikeNew
	case "A":
		return domain.ConditionGood
	case "B+":
		return domain.ConditionGood
	case "B":
		return domain.ConditionFair
	case "C":
		return domain.ConditionPoor
	case "ジャンク":
		return domain.ConditionPoor
	}

	// Case-insensitive match for English condition strings.
	switch strings.ToLower(s) {
	case "new":
		return domain.ConditionNew
	case "like_new":
		return domain.ConditionLikeNew
	case "good":
		return domain.ConditionGood
	case "fair":
		return domain.ConditionFair
	case "poor":
		return domain.ConditionPoor
	default:
		return ""
	}
}
