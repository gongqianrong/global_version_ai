package domain

import (
	"testing"
)

func TestSearchQuery_IsGlobalSearch_NoPlatforms(t *testing.T) {
	q := SearchQuery{Keyword: "gucci bag"}
	if !q.IsGlobalSearch() {
		t.Error("IsGlobalSearch() should return true when no platforms are specified")
	}
}

func TestSearchQuery_IsGlobalSearch_WithPlatforms(t *testing.T) {
	q := SearchQuery{Platforms: []string{"mercari"}}
	if q.IsGlobalSearch() {
		t.Error("IsGlobalSearch() should return false when platforms are specified")
	}
}

func TestSearchQuery_IsGlobalSearch_EmptyPlatforms(t *testing.T) {
	q := SearchQuery{Keyword: "test", Platforms: []string{}}
	if !q.IsGlobalSearch() {
		t.Error("IsGlobalSearch() should return true when platforms slice is empty")
	}
}

func TestSearchQuery_HasFilters_NoFilters(t *testing.T) {
	q := SearchQuery{Keyword: "bag"}
	if q.HasFilters() {
		t.Error("HasFilters() should return false when only keyword is set")
	}
}

func TestSearchQuery_HasFilters_BrandID(t *testing.T) {
	q := SearchQuery{BrandID: "b_001"}
	if !q.HasFilters() {
		t.Error("HasFilters() should return true when BrandID is set")
	}
}

func TestSearchQuery_HasFilters_Categories(t *testing.T) {
	q := SearchQuery{Categories: []string{"bags"}}
	if !q.HasFilters() {
		t.Error("HasFilters() should return true when Categories is set")
	}
}

func TestSearchQuery_HasFilters_PriceMin(t *testing.T) {
	q := SearchQuery{PriceMin: 1000}
	if !q.HasFilters() {
		t.Error("HasFilters() should return true when PriceMin is set")
	}
}

func TestSearchQuery_HasFilters_PriceMax(t *testing.T) {
	q := SearchQuery{PriceMax: 50000}
	if !q.HasFilters() {
		t.Error("HasFilters() should return true when PriceMax is set")
	}
}

func TestSearchQuery_HasFilters_Condition(t *testing.T) {
	q := SearchQuery{Condition: []string{ConditionNew, ConditionGood}}
	if !q.HasFilters() {
		t.Error("HasFilters() should return true when Condition is set")
	}
}

func TestSearchQuery_AllFields(t *testing.T) {
	q := SearchQuery{
		Keyword:       "gucci",
		KeywordJA:     "グッチ",
		Platforms:     []string{"mercari", "yahoo"},
		BrandID:       "b_gucci",
		Categories:    []string{"bags"},
		PriceMin:      5000,
		PriceMax:      100000,
		Condition:     []string{ConditionNew},
		SortBy:        "price_asc",
		Page:          1,
		PageSize:      20,
		UserLang:      "en",
		ContentRating: ContentRatingGeneral,
	}

	if q.Keyword != "gucci" {
		t.Errorf("Keyword = %q, want %q", q.Keyword, "gucci")
	}
	if q.KeywordJA != "グッチ" {
		t.Errorf("KeywordJA = %q, want %q", q.KeywordJA, "グッチ")
	}
	if q.PageSize != 20 {
		t.Errorf("PageSize = %d, want %d", q.PageSize, 20)
	}
	if q.UserLang != "en" {
		t.Errorf("UserLang = %q, want %q", q.UserLang, "en")
	}
}

func TestSearchResponse_Fields(t *testing.T) {
	resp := SearchResponse{
		CachedResults: []ProductSummary{
			{
				ID:            "mercari_m1",
				Title:         "Gucci Bag",
				TitleOriginal: "グッチ バッグ",
				Image:         "https://img.example.com/1.jpg",
				PriceJPY:      15000,
				Platform:      "mercari",
				Status:        StatusAvailable,
				Brand:         "Gucci",
				Tags:          []string{"gucci", "bag"},
				IsTranslated:  true,
			},
		},
		CachedTotal:       100,
		RealtimeStreamID:  "stream_abc123",
		TranslatedKeyword: "グッチ バッグ",
		Aggregations: SearchAggs{
			Platforms: []AggBucket{
				{Key: "mercari", Count: 50},
				{Key: "yahoo", Count: 50},
			},
			Brands: []AggBucket{
				{Key: "Gucci", Count: 30},
			},
			Categories: []AggBucket{
				{Key: "bags", Count: 80},
			},
			PriceRanges: []PriceRange{
				{Min: 0, Max: 10000, Count: 40},
				{Min: 10000, Max: 50000, Count: 60},
			},
		},
	}

	if len(resp.CachedResults) != 1 {
		t.Fatalf("len(CachedResults) = %d, want %d", len(resp.CachedResults), 1)
	}
	ps := resp.CachedResults[0]
	if ps.ID != "mercari_m1" {
		t.Errorf("ProductSummary.ID = %q, want %q", ps.ID, "mercari_m1")
	}
	if ps.PriceJPY != 15000 {
		t.Errorf("ProductSummary.PriceJPY = %d, want %d", ps.PriceJPY, 15000)
	}
	if !ps.IsTranslated {
		t.Error("ProductSummary.IsTranslated should be true")
	}
	if resp.CachedTotal != 100 {
		t.Errorf("CachedTotal = %d, want %d", resp.CachedTotal, 100)
	}
	if resp.RealtimeStreamID != "stream_abc123" {
		t.Errorf("RealtimeStreamID = %q, want %q", resp.RealtimeStreamID, "stream_abc123")
	}
	if len(resp.Aggregations.Platforms) != 2 {
		t.Errorf("len(Aggregations.Platforms) = %d, want %d", len(resp.Aggregations.Platforms), 2)
	}
	if resp.Aggregations.PriceRanges[0].Max != 10000 {
		t.Errorf("PriceRanges[0].Max = %d, want %d", resp.Aggregations.PriceRanges[0].Max, 10000)
	}
}

func TestProductSummary_Fields(t *testing.T) {
	ps := ProductSummary{
		ID:            "yahoo_y001",
		Title:         "Nike Shoes",
		TitleOriginal: "ナイキ シューズ",
		Image:         "https://img.example.com/2.jpg",
		PriceJPY:      8000,
		Platform:      "yahoo",
		Status:        StatusAvailable,
		Brand:         "Nike",
		Tags:          []string{"nike", "shoes"},
		IsTranslated:  false,
	}

	if ps.Platform != "yahoo" {
		t.Errorf("Platform = %q, want %q", ps.Platform, "yahoo")
	}
	if ps.Brand != "Nike" {
		t.Errorf("Brand = %q, want %q", ps.Brand, "Nike")
	}
	if len(ps.Tags) != 2 {
		t.Errorf("len(Tags) = %d, want %d", len(ps.Tags), 2)
	}
}

func TestAggBucket_Fields(t *testing.T) {
	b := AggBucket{Key: "mercari", Count: 42}
	if b.Key != "mercari" {
		t.Errorf("Key = %q, want %q", b.Key, "mercari")
	}
	if b.Count != 42 {
		t.Errorf("Count = %d, want %d", b.Count, 42)
	}
}

func TestPriceRange_Fields(t *testing.T) {
	pr := PriceRange{Min: 5000, Max: 20000, Count: 15}
	if pr.Min != 5000 {
		t.Errorf("Min = %d, want %d", pr.Min, 5000)
	}
	if pr.Max != 20000 {
		t.Errorf("Max = %d, want %d", pr.Max, 20000)
	}
	if pr.Count != 15 {
		t.Errorf("Count = %d, want %d", pr.Count, 15)
	}
}
