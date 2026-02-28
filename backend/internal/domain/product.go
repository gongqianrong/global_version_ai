// Package domain defines the core domain models for the Rakutao collection gateway.
package domain

import "time"

// Status constants represent the lifecycle state of a product listing.
const (
	StatusAvailable = "available"
	StatusSold      = "sold"
	StatusReserved  = "reserved"
	StatusDelisted  = "delisted"
)

// Condition constants describe the physical condition of a product.
const (
	ConditionNew     = "new"
	ConditionLikeNew = "like_new"
	ConditionGood    = "good"
	ConditionFair    = "fair"
	ConditionPoor    = "poor"
)

// Shipping constants describe who bears the shipping cost.
const (
	ShippingFree      = "free"
	ShippingBuyerPays = "buyer_pays"
	ShippingIncluded  = "included"
)

// Content rating constants for age-gating product visibility.
const (
	ContentRatingGeneral = "general"
	ContentRatingR18     = "r18"
)

// NewProductID builds a unified product ID by combining platform and source ID.
func NewProductID(platform, sourceID string) string {
	return platform + "_" + sourceID
}

// UnifiedProduct is the canonical representation of a product collected from
// any supported marketplace platform.
type UnifiedProduct struct {
	ID              string            `json:"id"`
	SourcePlatform  string            `json:"source_platform"`
	SourceID        string            `json:"source_id"`
	SourceURL       string            `json:"source_url"`
	Title           string            `json:"title"`
	TitleTranslated map[string]string `json:"title_translated,omitempty"`
	Description     string            `json:"description"`
	DescTranslated  map[string]string `json:"desc_translated,omitempty"`
	Images          []string          `json:"images"`
	PriceJPY        int64             `json:"price_jpy"`
	ServiceFeeJPY   int64             `json:"service_fee_jpy"`
	OriginalPrice   int64             `json:"original_price"`
	ShippingType    string            `json:"shipping_type"`
	ShippingFeeJPY  int64             `json:"shipping_fee_jpy"`
	Brand           *Brand            `json:"brand,omitempty"`
	Categories      []string          `json:"categories"`
	SourceCategory  string            `json:"source_category"`
	Tags            []string          `json:"tags"`
	Status          string            `json:"status"`
	Condition       string            `json:"condition"`
	Quantity        int               `json:"quantity"`
	Seller          SellerInfo        `json:"seller"`
	Variants        []Variant         `json:"variants,omitempty"`
	ListedAt        time.Time         `json:"listed_at"`
	CollectedAt     time.Time         `json:"collected_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	CacheTTL        int               `json:"cache_ttl"`
	ContentRating   string            `json:"content_rating"`
	Logistics       *LogisticsInfo    `json:"logistics,omitempty"`
}

// IsAvailable reports whether the product is currently available for purchase.
func (p *UnifiedProduct) IsAvailable() bool {
	return p.Status == StatusAvailable
}

// Brand holds brand information normalised across platforms.
type Brand struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	NameJA string `json:"name_ja"`
	Source string `json:"source"`
}

// SellerInfo contains summary information about the seller of a product.
type SellerInfo struct {
	SellerID   string  `json:"seller_id"`
	SellerName string  `json:"seller_name"`
	Rating     float64 `json:"rating"`
	ItemCount  int     `json:"item_count"`
}

// Variant represents a product variant axis (e.g. colour, size).
type Variant struct {
	Name    string   `json:"name"`
	Options []string `json:"options"`
}

// LogisticsInfo provides estimated logistics data for warehouse operations.
type LogisticsInfo struct {
	EstimatedWeight float64 `json:"estimated_weight"`
	EstimatedSize   string  `json:"estimated_size"`
	WarehouseRegion string  `json:"warehouse_region"`
}
