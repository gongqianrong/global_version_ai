package domain

import "time"

// UserPreference represents a user's selected category preference.
type UserPreference struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Category  string    `json:"category"`
	Weight    float32   `json:"weight"`
	CreatedAt time.Time `json:"createdAt"`
}

// BrowsingRecord represents a product view event.
type BrowsingRecord struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	ProductID string    `json:"productId"`
	Category  string    `json:"category"`
	Brand     string    `json:"brand"`
	SellerID  string    `json:"sellerId"`
	Platform  string    `json:"platform"`
	ViewedAt  time.Time `json:"viewedAt"`
}

// SearchRecord represents a search event.
type SearchRecord struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"userId"`
	Keyword   string    `json:"keyword"`
	KeywordJA string    `json:"keywordJa"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"createdAt"`
}

// RecWeight represents a computed recommendation weight for one signal+dimension+value.
type RecWeight struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"userId"`
	SignalType string    `json:"signalType"` // preference, browse, order, search, favorite, seller, cart
	Dimension  string    `json:"dimension"`  // category, brand, seller, keyword, platform
	Value      string    `json:"value"`
	Weight     float32   `json:"weight"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// RecommendedProduct is a product in a recommendation list.
type RecommendedProduct struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	Image         string `json:"image"`
	PriceJpy      int64  `json:"priceJpy"`
	Platform      string `json:"platform"`
	Brand         string `json:"brand"`
	Condition     string `json:"condition"`
	IsHighlighted bool   `json:"isHighlighted"`
}

// RecommendationList is a typed list of recommended products.
type RecommendationList struct {
	Type  string               `json:"type"`
	Title string               `json:"title"`
	Items []RecommendedProduct `json:"items"`
}
