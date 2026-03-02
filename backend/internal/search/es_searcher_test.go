package search

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// roundTripFunc is a helper to create mock HTTP transports for testing.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockClient(statusCode int, body string) *elasticsearch.Client {
	client, _ := elasticsearch.NewClient(elasticsearch.Config{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: statusCode,
				Header: http.Header{
					"Content-Type":       []string{"application/json"},
					"X-Elastic-Product":  []string{"Elasticsearch"},
				},
				Body: io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	})
	return client
}

func newMockClientError() *elasticsearch.Client {
	client, _ := elasticsearch.NewClient(elasticsearch.Config{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return nil, errors.New("connection refused")
		}),
	})
	return client
}

const searchResponseJSON = `{
	"hits": {
		"total": {"value": 2},
		"hits": [
			{
				"_source": {
					"id": "yahoo_auction_ya-001",
					"source_platform": "yahoo_auction",
					"status": "available",
					"title": "グッチ バッグ",
					"title_translated": {"en": "Gucci Bag", "zh-TW": "Gucci 包包"},
					"price_jpy": 16500,
					"brand_name": "Gucci",
					"condition": "like_new",
					"images": ["img1.jpg", "img2.jpg"],
					"tags": ["luxury", "bag"]
				}
			},
			{
				"_source": {
					"id": "surugaya_sg-001",
					"source_platform": "surugaya",
					"status": "available",
					"title": "ガンダム プラモデル",
					"price_jpy": 3300,
					"condition": "new",
					"images": ["gundam1.jpg"]
				}
			}
		]
	},
	"aggregations": {
		"platforms": {
			"buckets": [
				{"key": "yahoo_auction", "doc_count": 150},
				{"key": "surugaya", "doc_count": 80}
			]
		},
		"brands": {
			"buckets": [
				{"key": "Gucci", "doc_count": 30}
			]
		},
		"categories": {
			"buckets": [
				{"key": "バッグ", "doc_count": 45}
			]
		},
		"price_stats": {
			"count": 230,
			"min": 500,
			"max": 100000,
			"avg": 15000,
			"sum": 3450000
		}
	}
}`

func TestESSearcher_Search_Success(t *testing.T) {
	client := newMockClient(200, searchResponseJSON)
	searcher := NewESSearcher(client, "test_index")

	resp, err := searcher.Search(context.Background(), domain.SearchQuery{
		Keyword:  "gucci",
		Page:     1,
		PageSize: 20,
		UserLang: "zh-TW",
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if resp.CachedTotal != 2 {
		t.Errorf("CachedTotal = %d, want 2", resp.CachedTotal)
	}
	if len(resp.CachedResults) != 2 {
		t.Fatalf("len(CachedResults) = %d, want 2", len(resp.CachedResults))
	}

	// First hit: zh-TW title should be used
	p0 := resp.CachedResults[0]
	if p0.ID != "yahoo_auction_ya-001" {
		t.Errorf("ID = %q, want %q", p0.ID, "yahoo_auction_ya-001")
	}
	if p0.Title != "Gucci 包包" {
		t.Errorf("Title = %q, want %q", p0.Title, "Gucci 包包")
	}
	if p0.TitleOriginal != "グッチ バッグ" {
		t.Errorf("TitleOriginal = %q, want %q", p0.TitleOriginal, "グッチ バッグ")
	}
	if !p0.IsTranslated {
		t.Error("IsTranslated = false, want true")
	}
	if p0.Image != "img1.jpg" {
		t.Errorf("Image = %q, want %q", p0.Image, "img1.jpg")
	}
	if p0.PriceJPY != 16500 {
		t.Errorf("PriceJPY = %d, want 16500", p0.PriceJPY)
	}
	if p0.Platform != "yahoo_auction" {
		t.Errorf("Platform = %q, want %q", p0.Platform, "yahoo_auction")
	}
	if p0.Brand != "Gucci" {
		t.Errorf("Brand = %q, want %q", p0.Brand, "Gucci")
	}

	// Second hit: no zh-TW title, should fall back to original
	p1 := resp.CachedResults[1]
	if p1.Title != "ガンダム プラモデル" {
		t.Errorf("p1.Title = %q, want %q", p1.Title, "ガンダム プラモデル")
	}
	if p1.IsTranslated {
		t.Error("p1.IsTranslated = true, want false (no zh-TW title)")
	}

	// Aggregations
	if len(resp.Aggregations.Platforms) != 2 {
		t.Errorf("len(Platforms) = %d, want 2", len(resp.Aggregations.Platforms))
	}
	if resp.Aggregations.Platforms[0].Key != "yahoo_auction" {
		t.Errorf("Platforms[0].Key = %q", resp.Aggregations.Platforms[0].Key)
	}
	if resp.Aggregations.Platforms[0].Count != 150 {
		t.Errorf("Platforms[0].Count = %d, want 150", resp.Aggregations.Platforms[0].Count)
	}
	if len(resp.Aggregations.Brands) != 1 {
		t.Errorf("len(Brands) = %d, want 1", len(resp.Aggregations.Brands))
	}
	if len(resp.Aggregations.Categories) != 1 {
		t.Errorf("len(Categories) = %d, want 1", len(resp.Aggregations.Categories))
	}
	if len(resp.Aggregations.PriceRanges) != 1 {
		t.Fatalf("len(PriceRanges) = %d, want 1", len(resp.Aggregations.PriceRanges))
	}
	if resp.Aggregations.PriceRanges[0].Min != 500 {
		t.Errorf("PriceRanges[0].Min = %d, want 500", resp.Aggregations.PriceRanges[0].Min)
	}
	if resp.Aggregations.PriceRanges[0].Max != 100000 {
		t.Errorf("PriceRanges[0].Max = %d, want 100000", resp.Aggregations.PriceRanges[0].Max)
	}
}

func TestESSearcher_Search_Empty(t *testing.T) {
	body := `{
		"hits": {"total": {"value": 0}, "hits": []},
		"aggregations": {
			"platforms": {"buckets": []},
			"brands": {"buckets": []},
			"categories": {"buckets": []},
			"price_stats": {"count": 0, "min": 0, "max": 0, "avg": 0, "sum": 0}
		}
	}`
	client := newMockClient(200, body)
	searcher := NewESSearcher(client, "test_index")

	resp, err := searcher.Search(context.Background(), domain.SearchQuery{
		Keyword:  "nonexistent",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if resp.CachedTotal != 0 {
		t.Errorf("CachedTotal = %d, want 0", resp.CachedTotal)
	}
	if len(resp.CachedResults) != 0 {
		t.Errorf("len(CachedResults) = %d, want 0", len(resp.CachedResults))
	}
	if resp.Aggregations.PriceRanges != nil {
		t.Errorf("PriceRanges should be nil for empty stats")
	}
}

func TestESSearcher_Search_ESError(t *testing.T) {
	client := newMockClient(500, `{"error": "internal server error"}`)
	searcher := NewESSearcher(client, "test_index")

	_, err := searcher.Search(context.Background(), domain.SearchQuery{
		Keyword:  "test",
		Page:     1,
		PageSize: 20,
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestESSearcher_Search_ConnectionError(t *testing.T) {
	client := newMockClientError()
	searcher := NewESSearcher(client, "test_index")

	_, err := searcher.Search(context.Background(), domain.SearchQuery{
		Keyword:  "test",
		Page:     1,
		PageSize: 20,
	})
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestESSearcher_Search_LanguageEN(t *testing.T) {
	client := newMockClient(200, searchResponseJSON)
	searcher := NewESSearcher(client, "test_index")

	resp, err := searcher.Search(context.Background(), domain.SearchQuery{
		Keyword:  "gucci",
		Page:     1,
		PageSize: 20,
		UserLang: "en",
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	p0 := resp.CachedResults[0]
	if p0.Title != "Gucci Bag" {
		t.Errorf("Title = %q, want %q (en title)", p0.Title, "Gucci Bag")
	}
	if !p0.IsTranslated {
		t.Error("IsTranslated = false, want true")
	}
}

func TestESSearcher_Search_LanguageJA(t *testing.T) {
	client := newMockClient(200, searchResponseJSON)
	searcher := NewESSearcher(client, "test_index")

	resp, err := searcher.Search(context.Background(), domain.SearchQuery{
		Keyword:  "gucci",
		Page:     1,
		PageSize: 20,
		UserLang: "ja",
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	p0 := resp.CachedResults[0]
	if p0.Title != "グッチ バッグ" {
		t.Errorf("Title = %q, want %q (ja falls back to original)", p0.Title, "グッチ バッグ")
	}
	if p0.IsTranslated {
		t.Error("IsTranslated = true, want false (ja uses original)")
	}
}

func TestMapDocToSummary(t *testing.T) {
	doc := esDocument{
		ID:             "test-001",
		SourcePlatform: "yahoo_auction",
		Status:         "available",
		Title:          "テスト商品",
		TitleTranslated: map[string]string{"en": "Test Product", "zh-TW": "測試商品"},
		PriceJPY:       5000,
		BrandName:      "TestBrand",
		Condition:      "new",
		Images:         []string{"a.jpg", "b.jpg"},
		Tags:           []string{"test"},
	}

	// zh-TW
	s := mapDocToSummary(doc, "zh-TW")
	if s.Title != "測試商品" {
		t.Errorf("zh-TW: Title = %q", s.Title)
	}
	if !s.IsTranslated {
		t.Error("zh-TW: IsTranslated should be true")
	}
	if s.Image != "a.jpg" {
		t.Errorf("Image = %q, want %q", s.Image, "a.jpg")
	}

	// en
	s = mapDocToSummary(doc, "en")
	if s.Title != "Test Product" {
		t.Errorf("en: Title = %q", s.Title)
	}

	// no images
	doc.Images = nil
	s = mapDocToSummary(doc, "zh-TW")
	if s.Image != "" {
		t.Errorf("Image should be empty when no images")
	}
}

// --- ESProductFetcher tests ---

const getProductResponseJSON = `{
	"found": true,
	"_source": {
		"id": "yahoo_auction_ya-001",
		"source_platform": "yahoo_auction",
		"source_id": "ya-001",
		"title": "グッチ バッグ",
		"description": "美品",
		"price_jpy": 16500,
		"service_fee_jpy": 1500,
		"original_price": 15000,
		"status": "available",
		"condition": "like_new",
		"images": ["img1.jpg"]
	}
}`

func TestESProductFetcher_GetProduct_Success(t *testing.T) {
	client := newMockClient(200, getProductResponseJSON)
	fetcher := NewESProductFetcher(client, "test_index")

	product, err := fetcher.GetProduct(context.Background(), "yahoo_auction_ya-001")
	if err != nil {
		t.Fatalf("GetProduct error: %v", err)
	}

	if product.ID != "yahoo_auction_ya-001" {
		t.Errorf("ID = %q", product.ID)
	}
	if product.SourcePlatform != "yahoo_auction" {
		t.Errorf("SourcePlatform = %q", product.SourcePlatform)
	}
	if product.Title != "グッチ バッグ" {
		t.Errorf("Title = %q", product.Title)
	}
	if product.PriceJPY != 16500 {
		t.Errorf("PriceJPY = %d", product.PriceJPY)
	}
	if product.ServiceFeeJPY != 1500 {
		t.Errorf("ServiceFeeJPY = %d", product.ServiceFeeJPY)
	}
	if product.OriginalPrice != 15000 {
		t.Errorf("OriginalPrice = %d", product.OriginalPrice)
	}
}

func TestESProductFetcher_GetProduct_NotFound(t *testing.T) {
	client := newMockClient(404, `{"found": false}`)
	fetcher := NewESProductFetcher(client, "test_index")

	_, err := fetcher.GetProduct(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !errors.Is(err, domain.ErrAdapterNotFound) {
		t.Errorf("error = %v, want ErrAdapterNotFound", err)
	}
}

func TestESProductFetcher_GetProduct_ESError(t *testing.T) {
	client := newMockClient(500, `{"error": "internal"}`)
	fetcher := NewESProductFetcher(client, "test_index")

	_, err := fetcher.GetProduct(context.Background(), "test-001")
	if err == nil {
		t.Fatal("expected error for 500")
	}
}

func TestESProductFetcher_GetProduct_ConnectionError(t *testing.T) {
	client := newMockClientError()
	fetcher := NewESProductFetcher(client, "test_index")

	_, err := fetcher.GetProduct(context.Background(), "test-001")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}
