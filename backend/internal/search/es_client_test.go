package search

import (
	"strings"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

func TestBuildESQuery_GlobalSearch_MultiMatch(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "グッチ バッグ",
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "グッチ バッグ") {
		t.Errorf("expected query JSON to contain keyword 'グッチ バッグ', got:\n%s", got)
	}
	if !strings.Contains(got, "multi_match") {
		t.Errorf("expected query JSON to contain 'multi_match', got:\n%s", got)
	}
	if !strings.Contains(got, "title^3") {
		t.Errorf("expected query JSON to contain 'title^3' boost, got:\n%s", got)
	}
	if !strings.Contains(got, "title_en^2") {
		t.Errorf("expected query JSON to contain 'title_en^2' boost, got:\n%s", got)
	}
	if !strings.Contains(got, "title_zh_tw^2") {
		t.Errorf("expected query JSON to contain 'title_zh_tw^2' boost, got:\n%s", got)
	}
	if !strings.Contains(got, "description") {
		t.Errorf("expected query JSON to contain 'description' field, got:\n%s", got)
	}
	if !strings.Contains(got, "brand_name_ja") {
		t.Errorf("expected query JSON to contain 'brand_name_ja' field, got:\n%s", got)
	}
}

func TestBuildESQuery_PlatformFilter(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		Platforms: []string{"mercari"},
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "mercari") {
		t.Errorf("expected query JSON to contain 'mercari' in terms filter, got:\n%s", got)
	}
	if !strings.Contains(got, "source_platform") {
		t.Errorf("expected query JSON to contain 'source_platform' field, got:\n%s", got)
	}
}

func TestBuildESQuery_PriceRange(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		PriceMin:  1000,
		PriceMax:  50000,
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "price_jpy") {
		t.Errorf("expected query JSON to contain 'price_jpy', got:\n%s", got)
	}
	if !strings.Contains(got, "1000") {
		t.Errorf("expected query JSON to contain '1000' for gte, got:\n%s", got)
	}
	if !strings.Contains(got, "50000") {
		t.Errorf("expected query JSON to contain '50000' for lte, got:\n%s", got)
	}
	if !strings.Contains(got, "gte") {
		t.Errorf("expected query JSON to contain 'gte' for range, got:\n%s", got)
	}
	if !strings.Contains(got, "lte") {
		t.Errorf("expected query JSON to contain 'lte' for range, got:\n%s", got)
	}
}

func TestBuildESQuery_BrandFilter(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		BrandID:   "b_gucci",
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "brand_id") {
		t.Errorf("expected query JSON to contain 'brand_id', got:\n%s", got)
	}
	if !strings.Contains(got, "b_gucci") {
		t.Errorf("expected query JSON to contain 'b_gucci', got:\n%s", got)
	}
}

func TestBuildESQuery_ConditionFilter(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		Condition: []string{"new", "like_new"},
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "condition") {
		t.Errorf("expected query JSON to contain 'condition', got:\n%s", got)
	}
	if !strings.Contains(got, "new") {
		t.Errorf("expected query JSON to contain 'new', got:\n%s", got)
	}
	if !strings.Contains(got, "like_new") {
		t.Errorf("expected query JSON to contain 'like_new', got:\n%s", got)
	}
}

func TestBuildESQuery_CategoriesFilter(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA:  "バッグ",
		Categories: []string{"bags", "accessories"},
		Page:       1,
		PageSize:   20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "categories") {
		t.Errorf("expected query JSON to contain 'categories', got:\n%s", got)
	}
	if !strings.Contains(got, "bags") {
		t.Errorf("expected query JSON to contain 'bags', got:\n%s", got)
	}
	if !strings.Contains(got, "accessories") {
		t.Errorf("expected query JSON to contain 'accessories', got:\n%s", got)
	}
}

func TestBuildESQuery_StatusAvailableFilter(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "available") {
		t.Errorf("expected query JSON to contain status 'available' filter, got:\n%s", got)
	}
}

func TestBuildESQuery_SortPriceAsc(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		SortBy:    "price_asc",
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "price_jpy") {
		t.Errorf("expected query JSON to contain 'price_jpy' for sort, got:\n%s", got)
	}
	if !strings.Contains(got, "asc") {
		t.Errorf("expected query JSON to contain 'asc' sort order, got:\n%s", got)
	}
}

func TestBuildESQuery_SortPriceDesc(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		SortBy:    "price_desc",
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "price_jpy") {
		t.Errorf("expected query JSON to contain 'price_jpy' for sort, got:\n%s", got)
	}
	if !strings.Contains(got, "desc") {
		t.Errorf("expected query JSON to contain 'desc' sort order, got:\n%s", got)
	}
}

func TestBuildESQuery_SortNewest(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		SortBy:    "newest",
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "listed_at") {
		t.Errorf("expected query JSON to contain 'listed_at' for newest sort, got:\n%s", got)
	}
	if !strings.Contains(got, "desc") {
		t.Errorf("expected query JSON to contain 'desc' for newest sort, got:\n%s", got)
	}
}

func TestBuildESQuery_SortDefaultScore(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "_score") {
		t.Errorf("expected query JSON to contain '_score' for default sort, got:\n%s", got)
	}
}

func TestBuildESQuery_Aggregations(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "platforms") {
		t.Errorf("expected query JSON to contain 'platforms' aggregation, got:\n%s", got)
	}
	if !strings.Contains(got, "brands") {
		t.Errorf("expected query JSON to contain 'brands' aggregation, got:\n%s", got)
	}
	if !strings.Contains(got, "price_stats") {
		t.Errorf("expected query JSON to contain 'price_stats' aggregation, got:\n%s", got)
	}
}

func TestBuildESQuery_Pagination(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		Page:      3,
		PageSize:  10,
	}

	got := string(buildESQuery(q))

	// from = (3-1)*10 = 20
	if !strings.Contains(got, `"from":20`) {
		t.Errorf("expected query JSON to contain '\"from\":20', got:\n%s", got)
	}
	if !strings.Contains(got, `"size":10`) {
		t.Errorf("expected query JSON to contain '\"size\":10', got:\n%s", got)
	}
}

func TestBuildESQuery_DefaultPageSize(t *testing.T) {
	q := domain.SearchQuery{
		KeywordJA: "バッグ",
		Page:      1,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, `"size":20`) {
		t.Errorf("expected default page size of 20, got:\n%s", got)
	}
}

func TestBuildESQuery_NoKeyword(t *testing.T) {
	q := domain.SearchQuery{
		Platforms: []string{"mercari"},
		Page:      1,
		PageSize:  20,
	}

	got := string(buildESQuery(q))

	if !strings.Contains(got, "match_all") {
		t.Errorf("expected query JSON to contain 'match_all' when no keyword is provided, got:\n%s", got)
	}
}
