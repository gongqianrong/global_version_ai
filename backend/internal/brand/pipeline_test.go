package brand

import (
	"testing"
	"time"
)

// --- Mock AI Extractor ---

type mockAIExtractor struct {
	result *AIExtractionResult
	err    error
}

func (m *mockAIExtractor) Extract(title, description, category string) (*AIExtractionResult, error) {
	return m.result, m.err
}

// Helper to build a registry for pipeline tests.
func newPipelineRegistry() *Registry {
	r := NewRegistry()

	r.Add(Entry{
		ID:         "gucci",
		NameStd:    "Gucci",
		NameJA:     "グッチ",
		NameZhTW:   "古馳",
		Aliases:    []string{"GUCCI", "グッチ"},
		Category:   "luxury",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})

	r.Add(Entry{
		ID:         "louis_vuitton",
		NameStd:    "Louis Vuitton",
		NameJA:     "ルイ・ヴィトン",
		NameZhTW:   "路易威登",
		Aliases:    []string{"LV", "ルイヴィトン", "LOUIS VUITTON"},
		Category:   "luxury",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})

	r.Add(Entry{
		ID:         "prada",
		NameStd:    "Prada",
		NameJA:     "プラダ",
		NameZhTW:   "普拉達",
		Aliases:    []string{"PRADA", "プラダ"},
		Category:   "luxury",
		IsVerified: true,
		CreatedAt:  time.Now(),
	})

	return r
}

// --- Test 1: Level 1 hit — rawData has brand field ---

func TestPipeline_Level1_PlatformField(t *testing.T) {
	reg := newPipelineRegistry()
	pipeline := NewPipeline(reg, nil)

	rawData := map[string]interface{}{
		"brand": "GUCCI",
	}

	brand := pipeline.Identify(rawData, "Some title", "Some description", "fashion")
	if brand == nil {
		t.Fatal("expected brand, got nil")
	}
	if brand.Name != "Gucci" {
		t.Errorf("Name = %q, want %q", brand.Name, "Gucci")
	}
	if brand.Source != "platform_field" {
		t.Errorf("Source = %q, want %q", brand.Source, "platform_field")
	}
}

// --- Test: Level 1 hit with various raw field names ---

func TestPipeline_Level1_BrandNameField(t *testing.T) {
	reg := newPipelineRegistry()
	pipeline := NewPipeline(reg, nil)

	rawData := map[string]interface{}{
		"brand_name": "LV",
	}

	brand := pipeline.Identify(rawData, "Bag", "Nice bag", "")
	if brand == nil {
		t.Fatal("expected brand, got nil")
	}
	if brand.Name != "Louis Vuitton" {
		t.Errorf("Name = %q, want %q", brand.Name, "Louis Vuitton")
	}
	if brand.Source != "platform_field" {
		t.Errorf("Source = %q, want %q", brand.Source, "platform_field")
	}
}

// --- Test 2: Level 2 hit — brand found in title ---

func TestPipeline_Level2_TitleMatch(t *testing.T) {
	reg := newPipelineRegistry()
	pipeline := NewPipeline(reg, nil)

	rawData := map[string]interface{}{} // no brand field

	brand := pipeline.Identify(rawData, "ルイ・ヴィトン モノグラム バッグ", "綺麗なバッグです", "ファッション")
	if brand == nil {
		t.Fatal("expected brand, got nil")
	}
	if brand.Name != "Louis Vuitton" {
		t.Errorf("Name = %q, want %q", brand.Name, "Louis Vuitton")
	}
	if brand.Source != "rule_matched" {
		t.Errorf("Source = %q, want %q", brand.Source, "rule_matched")
	}
}

// --- Test: Level 2 hit from description ---

func TestPipeline_Level2_DescriptionMatch(t *testing.T) {
	reg := newPipelineRegistry()
	pipeline := NewPipeline(reg, nil)

	rawData := map[string]interface{}{}

	brand := pipeline.Identify(rawData, "レザーバッグ", "プラダの正規品です", "")
	if brand == nil {
		t.Fatal("expected brand, got nil")
	}
	if brand.Name != "Prada" {
		t.Errorf("Name = %q, want %q", brand.Name, "Prada")
	}
	if brand.Source != "rule_matched" {
		t.Errorf("Source = %q, want %q", brand.Source, "rule_matched")
	}
}

// --- Test 3: Level 3 hit — AI extractor returns brand with confidence >= 0.7 ---

func TestPipeline_Level3_AIExtractor(t *testing.T) {
	reg := newPipelineRegistry()
	ai := &mockAIExtractor{
		result: &AIExtractionResult{
			BrandName:  "Gucci",
			Confidence: 0.85,
		},
	}
	pipeline := NewPipeline(reg, ai)

	rawData := map[string]interface{}{} // no brand field

	brand := pipeline.Identify(rawData, "高級バッグ", "イタリア製のバッグ", "ファッション")
	if brand == nil {
		t.Fatal("expected brand, got nil")
	}
	if brand.Name != "Gucci" {
		t.Errorf("Name = %q, want %q", brand.Name, "Gucci")
	}
	if brand.Source != "ai_extracted" {
		t.Errorf("Source = %q, want %q", brand.Source, "ai_extracted")
	}
}

// --- Test: Level 3 — AI confidence below threshold ---

func TestPipeline_Level3_LowConfidence(t *testing.T) {
	reg := newPipelineRegistry()
	ai := &mockAIExtractor{
		result: &AIExtractionResult{
			BrandName:  "Gucci",
			Confidence: 0.5,
		},
	}
	pipeline := NewPipeline(reg, ai)

	rawData := map[string]interface{}{}

	brand := pipeline.Identify(rawData, "高級バッグ", "イタリア製", "")
	if brand != nil {
		t.Errorf("expected nil for low confidence, got %+v", brand)
	}
}

// --- Test 4: No brand found at any level ---

func TestPipeline_NoBrandFound(t *testing.T) {
	reg := newPipelineRegistry()
	pipeline := NewPipeline(reg, nil)

	rawData := map[string]interface{}{}

	brand := pipeline.Identify(rawData, "普通のバッグ", "特にブランドなし", "雑貨")
	if brand != nil {
		t.Errorf("expected nil, got %+v", brand)
	}
}

// --- Test: Level 3 — AI returns brand not in registry ---

func TestPipeline_Level3_AIBrandNotInRegistry(t *testing.T) {
	reg := newPipelineRegistry()
	ai := &mockAIExtractor{
		result: &AIExtractionResult{
			BrandName:  "UnknownBrand",
			Confidence: 0.90,
		},
	}
	pipeline := NewPipeline(reg, ai)

	rawData := map[string]interface{}{}

	brand := pipeline.Identify(rawData, "バッグ", "高品質", "")
	// AI brand not found in registry — we still return the AI result
	if brand == nil {
		t.Fatal("expected brand from AI (even if not in registry), got nil")
	}
	if brand.Name != "UnknownBrand" {
		t.Errorf("Name = %q, want %q", brand.Name, "UnknownBrand")
	}
	if brand.Source != "ai_extracted" {
		t.Errorf("Source = %q, want %q", brand.Source, "ai_extracted")
	}
}

// --- Test: Nil rawData ---

func TestPipeline_NilRawData(t *testing.T) {
	reg := newPipelineRegistry()
	pipeline := NewPipeline(reg, nil)

	brand := pipeline.Identify(nil, "", "", "")
	if brand != nil {
		t.Errorf("expected nil for nil rawData, got %+v", brand)
	}
}

// --- Test: Level 1 with exact confidence threshold (0.7) ---

func TestPipeline_Level3_ExactThreshold(t *testing.T) {
	reg := newPipelineRegistry()
	ai := &mockAIExtractor{
		result: &AIExtractionResult{
			BrandName:  "Prada",
			Confidence: 0.7,
		},
	}
	pipeline := NewPipeline(reg, ai)

	rawData := map[string]interface{}{}

	brand := pipeline.Identify(rawData, "バッグ", "イタリア製", "")
	if brand == nil {
		t.Fatal("expected brand at exact threshold 0.7, got nil")
	}
	if brand.Source != "ai_extracted" {
		t.Errorf("Source = %q, want %q", brand.Source, "ai_extracted")
	}
}
