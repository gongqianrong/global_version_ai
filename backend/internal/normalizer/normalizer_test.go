package normalizer

import (
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// --- Test 1: Normalize basic fields ---

func TestNormalize_BasicFields(t *testing.T) {
	n := New(nil) // no brand extractor

	raw := domain.RawProduct{
		Platform: "mercari",
		RawID:    "m001",
		RawData: map[string]interface{}{
			"title":       "テスト商品",
			"price":       float64(5000),
			"description": "テスト商品の説明",
			"images":      []interface{}{"https://img.example.com/1.jpg", "https://img.example.com/2.jpg"},
			"category":    "ファッション",
			"status":      "on_sale",
			"condition":   "新品、未使用",
		},
	}

	product, err := n.Normalize("mercari", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if product == nil {
		t.Fatal("expected product, got nil")
	}
	if product.ID != "mercari_m001" {
		t.Errorf("ID = %q, want %q", product.ID, "mercari_m001")
	}
	if product.SourcePlatform != "mercari" {
		t.Errorf("SourcePlatform = %q, want %q", product.SourcePlatform, "mercari")
	}
	if product.SourceID != "m001" {
		t.Errorf("SourceID = %q, want %q", product.SourceID, "m001")
	}
	if product.Title != "テスト商品" {
		t.Errorf("Title = %q, want %q", product.Title, "テスト商品")
	}
	if product.PriceJPY != 5500 {
		t.Errorf("PriceJPY = %d, want %d", product.PriceJPY, 5500)
	}
	if product.ServiceFeeJPY != 500 {
		t.Errorf("ServiceFeeJPY = %d, want %d", product.ServiceFeeJPY, 500)
	}
	if product.OriginalPrice != 5000 {
		t.Errorf("OriginalPrice = %d, want %d", product.OriginalPrice, 5000)
	}
	if product.Description != "テスト商品の説明" {
		t.Errorf("Description = %q, want %q", product.Description, "テスト商品の説明")
	}
	if len(product.Images) != 2 {
		t.Errorf("len(Images) = %d, want 2", len(product.Images))
	}
	if product.SourceCategory != "ファッション" {
		t.Errorf("SourceCategory = %q, want %q", product.SourceCategory, "ファッション")
	}
	if product.Status != domain.StatusAvailable {
		t.Errorf("Status = %q, want %q", product.Status, domain.StatusAvailable)
	}
	if product.Condition != domain.ConditionNew {
		t.Errorf("Condition = %q, want %q", product.Condition, domain.ConditionNew)
	}
}

// --- Test 2: Missing required fields ---

func TestNormalize_MissingRequiredFields(t *testing.T) {
	n := New(nil)

	raw := domain.RawProduct{
		Platform: "mercari",
		RawID:    "m002",
		RawData:  map[string]interface{}{},
	}

	_, err := n.Normalize("mercari", raw)
	if err == nil {
		t.Fatal("expected error for missing required fields, got nil")
	}
}

// --- Test: Status mapping ---

func TestNormalize_StatusMapping(t *testing.T) {
	n := New(nil)

	tests := []struct {
		input string
		want  string
	}{
		{"on_sale", domain.StatusAvailable},
		{"available", domain.StatusAvailable},
		{"active", domain.StatusAvailable},
		{"sold", domain.StatusSold},
		{"sold_out", domain.StatusSold},
		{"品切れ", domain.StatusSold},
		{"reserved", domain.StatusReserved},
		{"trading", domain.StatusReserved},
		{"delisted", domain.StatusDelisted},
		{"cancelled", domain.StatusDelisted},
		{"unknown_status", domain.StatusAvailable}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			raw := domain.RawProduct{
				Platform: "mercari",
				RawID:    "s001",
				RawData: map[string]interface{}{
					"title":  "テスト",
					"price":  float64(1000),
					"status": tt.input,
				},
			}
			p, err := n.Normalize("mercari", raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Status != tt.want {
				t.Errorf("status %q → %q, want %q", tt.input, p.Status, tt.want)
			}
		})
	}
}

// --- Test: Condition mapping (Japanese) ---

func TestNormalize_ConditionMapping(t *testing.T) {
	n := New(nil)

	tests := []struct {
		input string
		want  string
	}{
		{"新品、未使用", domain.ConditionNew},
		{"未使用に近い", domain.ConditionLikeNew},
		{"目立った傷や汚れなし", domain.ConditionGood},
		{"やや傷や汚れあり", domain.ConditionFair},
		{"傷や汚れあり", domain.ConditionPoor},
		{"全体的に状態が悪い", domain.ConditionPoor},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			rawData := map[string]interface{}{
				"title": "テスト",
				"price": float64(1000),
			}
			if tt.input != "" {
				rawData["condition"] = tt.input
			}
			raw := domain.RawProduct{
				Platform: "mercari",
				RawID:    "c001",
				RawData:  rawData,
			}
			p, err := n.Normalize("mercari", raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Condition != tt.want {
				t.Errorf("condition %q → %q, want %q", tt.input, p.Condition, tt.want)
			}
		})
	}
}

// --- Test: Brand extractor integration ---

type mockBrandExtractor struct {
	brand *domain.Brand
	err   error
}

func (m *mockBrandExtractor) Extract(title, description, category string) (*domain.Brand, error) {
	return m.brand, m.err
}

func TestNormalize_WithBrandExtractor(t *testing.T) {
	extractor := &mockBrandExtractor{
		brand: &domain.Brand{
			ID:   "gucci",
			Name: "GUCCI",
			Source: "rule_matched",
		},
	}
	n := New(extractor)

	raw := domain.RawProduct{
		Platform: "mercari",
		RawID:    "b001",
		RawData: map[string]interface{}{
			"title": "グッチ バッグ",
			"price": float64(30000),
		},
	}

	p, err := n.Normalize("mercari", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Brand == nil {
		t.Fatal("expected brand, got nil")
	}
	if p.Brand.Name != "GUCCI" {
		t.Errorf("Brand.Name = %q, want %q", p.Brand.Name, "GUCCI")
	}
}

// --- Test: Nil RawData ---

func TestNormalize_NilRawData(t *testing.T) {
	n := New(nil)

	raw := domain.RawProduct{
		Platform: "mercari",
		RawID:    "n001",
		RawData:  nil,
	}

	_, err := n.Normalize("mercari", raw)
	if err == nil {
		t.Fatal("expected error for nil RawData, got nil")
	}
}

// --- Test: Brand from brand_name field ---

func TestNormalize_BrandNameField(t *testing.T) {
	n := New(nil) // no brand extractor

	raw := domain.RawProduct{
		Platform: "surugaya",
		RawID:    "sg-brand",
		RawData: map[string]interface{}{
			"title":      "テスト商品",
			"price":      float64(3000),
			"brand_name": "[バンダイ]",
		},
	}

	p, err := n.Normalize("surugaya", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Brand == nil {
		t.Fatal("expected brand, got nil")
	}
	if p.Brand.Name != "[バンダイ]" {
		t.Errorf("Brand.Name = %q, want %q", p.Brand.Name, "[バンダイ]")
	}
	if p.Brand.Source != "platform_field" {
		t.Errorf("Brand.Source = %q, want %q", p.Brand.Source, "platform_field")
	}
}

func TestNormalize_BrandExtractorOverriddenByBrandName(t *testing.T) {
	// When brand_name is directly provided, brand extractor should NOT override it.
	extractor := &mockBrandExtractor{
		brand: &domain.Brand{ID: "gucci", Name: "GUCCI", Source: "rule_matched"},
	}
	n := New(extractor)

	raw := domain.RawProduct{
		Platform: "surugaya",
		RawID:    "sg-brand2",
		RawData: map[string]interface{}{
			"title":      "テスト",
			"price":      float64(1000),
			"brand_name": "[バンダイ]",
		},
	}

	p, err := n.Normalize("surugaya", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// brand_name should win over brand extractor.
	if p.Brand.Name != "[バンダイ]" {
		t.Errorf("Brand.Name = %q, want %q (brand_name should take priority)", p.Brand.Name, "[バンダイ]")
	}
}

// --- Test: Surugaya condition mapping ---

func TestNormalize_SurugayaConditionMapping(t *testing.T) {
	n := New(nil)

	tests := []struct {
		input string
		want  string
	}{
		{"新品", domain.ConditionNew},
		{"A+", domain.ConditionLikeNew},
		{"A", domain.ConditionGood},
		{"B+", domain.ConditionGood},
		{"B", domain.ConditionFair},
		{"C", domain.ConditionPoor},
		{"ジャンク", domain.ConditionPoor},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			raw := domain.RawProduct{
				Platform: "surugaya",
				RawID:    "sg001",
				RawData: map[string]interface{}{
					"title":     "テスト",
					"price":     float64(1000),
					"condition": tt.input,
				},
			}
			p, err := n.Normalize("surugaya", raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Condition != tt.want {
				t.Errorf("condition %q → %q, want %q", tt.input, p.Condition, tt.want)
			}
		})
	}
}

// --- Test: SourceURL from RawData ---

func TestNormalize_SourceURL(t *testing.T) {
	n := New(nil)

	raw := domain.RawProduct{
		Platform: "surugaya",
		RawID:    "sg001",
		RawData: map[string]interface{}{
			"title":      "テスト",
			"price":      float64(1000),
			"source_url": "https://www.suruga-ya.jp/product/detail/sg001",
		},
	}
	p, err := n.Normalize("surugaya", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SourceURL != "https://www.suruga-ya.jp/product/detail/sg001" {
		t.Errorf("SourceURL = %q", p.SourceURL)
	}
}

// --- Test: ContentRating from RawData ---

func TestNormalize_ContentRating(t *testing.T) {
	n := New(nil)

	t.Run("r18", func(t *testing.T) {
		raw := domain.RawProduct{
			Platform: "surugaya",
			RawID:    "sg002",
			RawData: map[string]interface{}{
				"title":          "R18テスト",
				"price":          float64(2000),
				"content_rating": "r18",
			},
		}
		p, err := n.Normalize("surugaya", raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ContentRating != domain.ContentRatingR18 {
			t.Errorf("ContentRating = %q, want %q", p.ContentRating, domain.ContentRatingR18)
		}
	})

	t.Run("default general", func(t *testing.T) {
		raw := domain.RawProduct{
			Platform: "surugaya",
			RawID:    "sg003",
			RawData: map[string]interface{}{
				"title": "テスト",
				"price": float64(1000),
			},
		}
		p, err := n.Normalize("surugaya", raw)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.ContentRating != domain.ContentRatingGeneral {
			t.Errorf("ContentRating = %q, want %q", p.ContentRating, domain.ContentRatingGeneral)
		}
	})
}

// --- Test: Seller info from RawData ---

func TestNormalize_SellerInfo(t *testing.T) {
	n := New(nil)

	raw := domain.RawProduct{
		Platform: "surugaya",
		RawID:    "sg004",
		RawData: map[string]interface{}{
			"title":       "テスト",
			"price":       float64(1000),
			"seller_id":   "surugaya",
			"seller_name": "駿河屋",
		},
	}
	p, err := n.Normalize("surugaya", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Seller.SellerID != "surugaya" {
		t.Errorf("Seller.SellerID = %q, want %q", p.Seller.SellerID, "surugaya")
	}
	if p.Seller.SellerName != "駿河屋" {
		t.Errorf("Seller.SellerName = %q, want %q", p.Seller.SellerName, "駿河屋")
	}
}

// --- Test: Tags from subcategory, manufacturer, JAN ---

func TestNormalize_Tags(t *testing.T) {
	n := New(nil)

	raw := domain.RawProduct{
		Platform: "surugaya",
		RawID:    "sg005",
		RawData: map[string]interface{}{
			"title":        "テスト",
			"price":        float64(1000),
			"subcategory":  "PS5ソフト",
			"manufacturer": "バンダイナムコ",
			"jan_code":     "4573102630001",
		},
	}
	p, err := n.Normalize("surugaya", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Tags) != 3 {
		t.Fatalf("len(Tags) = %d, want 3; got %v", len(p.Tags), p.Tags)
	}
	if p.Tags[0] != "PS5ソフト" {
		t.Errorf("Tags[0] = %q, want %q", p.Tags[0], "PS5ソフト")
	}
	if p.Tags[1] != "バンダイナムコ" {
		t.Errorf("Tags[1] = %q, want %q", p.Tags[1], "バンダイナムコ")
	}
	if p.Tags[2] != "JAN:4573102630001" {
		t.Errorf("Tags[2] = %q, want %q", p.Tags[2], "JAN:4573102630001")
	}
}

// --- Test: Backward compatibility (no new fields) ---

func TestNormalize_BackwardCompatibility(t *testing.T) {
	n := New(nil)

	raw := domain.RawProduct{
		Platform: "yahoo_auction",
		RawID:    "ya001",
		RawData: map[string]interface{}{
			"title":       "テスト商品",
			"price":       float64(3000),
			"description": "説明文",
			"images":      []interface{}{"img1.jpg"},
			"status":      "on_sale",
			"condition":   "未使用に近い",
			"category":    "ファッション",
		},
	}
	p, err := n.Normalize("yahoo_auction", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Existing fields should still work.
	if p.Title != "テスト商品" {
		t.Errorf("Title = %q", p.Title)
	}
	if p.Condition != domain.ConditionLikeNew {
		t.Errorf("Condition = %q, want %q", p.Condition, domain.ConditionLikeNew)
	}

	// New fields should default gracefully.
	if p.SourceURL != "" {
		t.Errorf("SourceURL = %q, want empty", p.SourceURL)
	}
	if p.ContentRating != domain.ContentRatingGeneral {
		t.Errorf("ContentRating = %q, want %q", p.ContentRating, domain.ContentRatingGeneral)
	}
	if p.Seller.SellerID != "" {
		t.Errorf("Seller.SellerID = %q, want empty", p.Seller.SellerID)
	}
	if p.Tags != nil {
		t.Errorf("Tags = %v, want nil", p.Tags)
	}
}

// --- Test: Service fee calculation ---

func TestNormalize_ServiceFeeCalculation(t *testing.T) {
	n := New(nil)

	tests := []struct {
		name           string
		rawPrice       float64
		wantOriginal   int64
		wantFee        int64
		wantFinalPrice int64
	}{
		{"1000 yen item", 1000, 1000, 100, 1100},
		{"5000 yen item", 5000, 5000, 500, 5500},
		{"10000 yen item", 10000, 10000, 1000, 11000},
		{"99 yen item", 99, 99, 9, 108},
		{"1 yen item", 1, 1, 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := domain.RawProduct{
				Platform: "yahoo_auction",
				RawID:    "fee001",
				RawData: map[string]interface{}{
					"title": "テスト商品",
					"price": tt.rawPrice,
				},
			}
			p, err := n.Normalize("yahoo_auction", raw)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.OriginalPrice != tt.wantOriginal {
				t.Errorf("OriginalPrice = %d, want %d", p.OriginalPrice, tt.wantOriginal)
			}
			if p.ServiceFeeJPY != tt.wantFee {
				t.Errorf("ServiceFeeJPY = %d, want %d", p.ServiceFeeJPY, tt.wantFee)
			}
			if p.PriceJPY != tt.wantFinalPrice {
				t.Errorf("PriceJPY = %d, want %d", p.PriceJPY, tt.wantFinalPrice)
			}
		})
	}
}
