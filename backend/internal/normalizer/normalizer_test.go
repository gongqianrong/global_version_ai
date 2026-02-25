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
	if product.PriceJPY != 5000 {
		t.Errorf("PriceJPY = %d, want %d", product.PriceJPY, 5000)
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
