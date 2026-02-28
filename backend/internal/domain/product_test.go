package domain

import (
	"testing"
	"time"
)

func TestNewProductID(t *testing.T) {
	got := NewProductID("mercari", "m123456")
	want := "mercari_m123456"
	if got != want {
		t.Errorf("NewProductID(\"mercari\", \"m123456\") = %q, want %q", got, want)
	}
}

func TestUnifiedProduct_IsAvailable_True(t *testing.T) {
	p := UnifiedProduct{Status: StatusAvailable}
	if !p.IsAvailable() {
		t.Error("expected IsAvailable() to return true for status \"available\"")
	}
}

func TestUnifiedProduct_IsAvailable_False(t *testing.T) {
	p := UnifiedProduct{Status: StatusSold}
	if p.IsAvailable() {
		t.Error("expected IsAvailable() to return false for status \"sold\"")
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"StatusAvailable", StatusAvailable, "available"},
		{"StatusSold", StatusSold, "sold"},
		{"StatusReserved", StatusReserved, "reserved"},
		{"StatusDelisted", StatusDelisted, "delisted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.want)
			}
		})
	}
}

func TestConditionConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"ConditionNew", ConditionNew, "new"},
		{"ConditionLikeNew", ConditionLikeNew, "like_new"},
		{"ConditionGood", ConditionGood, "good"},
		{"ConditionFair", ConditionFair, "fair"},
		{"ConditionPoor", ConditionPoor, "poor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.want)
			}
		})
	}
}

func TestShippingConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"ShippingFree", ShippingFree, "free"},
		{"ShippingBuyerPays", ShippingBuyerPays, "buyer_pays"},
		{"ShippingIncluded", ShippingIncluded, "included"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.want)
			}
		})
	}
}

func TestContentRatingConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{"ContentRatingGeneral", ContentRatingGeneral, "general"},
		{"ContentRatingR18", ContentRatingR18, "r18"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.want)
			}
		})
	}
}

func TestUnifiedProductStructFields(t *testing.T) {
	now := time.Now()
	p := UnifiedProduct{
		ID:              "mercari_m123456",
		SourcePlatform:  "mercari",
		SourceID:        "m123456",
		SourceURL:       "https://mercari.com/item/m123456",
		Title:           "Gucci Bag",
		TitleTranslated: map[string]string{"en": "Gucci Bag"},
		Description:     "A nice bag",
		DescTranslated:  map[string]string{"en": "A nice bag"},
		Images:          []string{"img1.jpg", "img2.jpg"},
		PriceJPY:        15000,
		ServiceFeeJPY:   1500,
		OriginalPrice:   18000,
		ShippingType:    ShippingFree,
		ShippingFeeJPY:  0,
		Brand:           &Brand{ID: "b_gucci", Name: "Gucci", NameJA: "グッチ", Source: "mercari"},
		Categories:      []string{"bags", "luxury"},
		SourceCategory:  "バッグ",
		Tags:            []string{"gucci", "bag"},
		Status:          StatusAvailable,
		Condition:       ConditionGood,
		Quantity:        1,
		Seller:          SellerInfo{SellerID: "s001", SellerName: "TopSeller", Rating: 4.8, ItemCount: 120},
		Variants:        []Variant{{Name: "Color", Options: []string{"Red", "Blue"}}},
		ListedAt:        now,
		CollectedAt:     now,
		UpdatedAt:       now,
		CacheTTL:        300,
		ContentRating:   ContentRatingGeneral,
		Logistics:       &LogisticsInfo{EstimatedWeight: 0.5, EstimatedSize: "30x20x10", WarehouseRegion: "JP-Tokyo"},
	}

	if p.ID != "mercari_m123456" {
		t.Errorf("ID = %q, want %q", p.ID, "mercari_m123456")
	}
	if p.SourcePlatform != "mercari" {
		t.Errorf("SourcePlatform = %q, want %q", p.SourcePlatform, "mercari")
	}
	if p.PriceJPY != 15000 {
		t.Errorf("PriceJPY = %d, want %d", p.PriceJPY, 15000)
	}
	if p.ServiceFeeJPY != 1500 {
		t.Errorf("ServiceFeeJPY = %d, want %d", p.ServiceFeeJPY, 1500)
	}
	if p.Brand.Name != "Gucci" {
		t.Errorf("Brand.Name = %q, want %q", p.Brand.Name, "Gucci")
	}
	if len(p.Images) != 2 {
		t.Errorf("len(Images) = %d, want %d", len(p.Images), 2)
	}
	if p.Seller.Rating != 4.8 {
		t.Errorf("Seller.Rating = %f, want %f", p.Seller.Rating, 4.8)
	}
	if len(p.Variants) != 1 || p.Variants[0].Name != "Color" {
		t.Errorf("Variants not set correctly")
	}
	if p.Logistics.WarehouseRegion != "JP-Tokyo" {
		t.Errorf("Logistics.WarehouseRegion = %q, want %q", p.Logistics.WarehouseRegion, "JP-Tokyo")
	}
}
