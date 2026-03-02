package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/i18n"
)

// ESSearcher implements the Searcher interface using Elasticsearch.
type ESSearcher struct {
	client    *elasticsearch.Client
	indexName string
}

// NewESSearcher creates an ESSearcher.
func NewESSearcher(client *elasticsearch.Client, indexName string) *ESSearcher {
	return &ESSearcher{client: client, indexName: indexName}
}

// Search executes a search query against Elasticsearch and returns results.
func (s *ESSearcher) Search(ctx context.Context, query domain.SearchQuery) (*domain.SearchResponse, error) {
	body := buildESQuery(query)

	res, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex(s.indexName),
		s.client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, fmt.Errorf("es search: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("es search: status %s", res.Status())
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("es search: read body: %w", err)
	}

	var esResp esResponse
	if err := json.Unmarshal(data, &esResp); err != nil {
		return nil, fmt.Errorf("es search: decode: %w", err)
	}

	return mapSearchResponse(esResp, query.UserLang), nil
}

// --- internal ES response types ---

type esResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
		Hits []esHit `json:"hits"`
	} `json:"hits"`
	Aggregations esAggregations `json:"aggregations"`
}

type esHit struct {
	Source json.RawMessage `json:"_source"`
}

type esAggregations struct {
	Platforms  esBucketAgg `json:"platforms"`
	Brands    esBucketAgg `json:"brands"`
	Categories esBucketAgg `json:"categories"`
	PriceStats esStatsAgg  `json:"price_stats"`
}

type esBucketAgg struct {
	Buckets []esBucket `json:"buckets"`
}

type esBucket struct {
	Key      string `json:"key"`
	DocCount int64  `json:"doc_count"`
}

type esStatsAgg struct {
	Count int64   `json:"count"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Avg   float64 `json:"avg"`
	Sum   float64 `json:"sum"`
}

// esDocument represents a document stored in the ES index.
type esDocument struct {
	ID             string   `json:"id"`
	SourcePlatform string   `json:"source_platform"`
	Status         string   `json:"status"`
	Title          string   `json:"title"`
	TitleEN        string   `json:"title_en"`
	TitleZHTW      string   `json:"title_zh_tw"`
	Description    string   `json:"description"`
	BrandID        string   `json:"brand_id"`
	BrandName      string   `json:"brand_name"`
	BrandNameJA    string   `json:"brand_name_ja"`
	Categories     []string `json:"categories"`
	Tags           []string `json:"tags"`
	Condition      string   `json:"condition"`
	ContentRating  string   `json:"content_rating"`
	PriceJPY       int64    `json:"price_jpy"`
	ServiceFeeJPY  int64    `json:"service_fee_jpy"`
	OriginalPrice  int64    `json:"original_price"`
	ShippingType   string   `json:"shipping_type"`
	SellerID       string   `json:"seller_id"`
	SellerRating   float64  `json:"seller_rating"`
	Images         []string `json:"images"`
	ListedAt       string   `json:"listed_at"`
	CollectedAt    string   `json:"collected_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// mapSearchResponse converts an ES response to a domain SearchResponse.
func mapSearchResponse(resp esResponse, userLang string) *domain.SearchResponse {
	summaries := make([]domain.ProductSummary, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		var doc esDocument
		if err := json.Unmarshal(hit.Source, &doc); err != nil {
			continue
		}
		summaries = append(summaries, mapDocToSummary(doc, userLang))
	}

	aggs := domain.SearchAggs{
		Platforms:   mapBuckets(resp.Aggregations.Platforms),
		Brands:     mapBuckets(resp.Aggregations.Brands),
		Categories: mapBuckets(resp.Aggregations.Categories),
		PriceRanges: mapPriceRanges(resp.Aggregations.PriceStats),
	}

	return &domain.SearchResponse{
		CachedResults: summaries,
		CachedTotal:   resp.Hits.Total.Value,
		Aggregations:  aggs,
	}
}

func mapDocToSummary(doc esDocument, userLang string) domain.ProductSummary {
	title := doc.Title
	isTranslated := false

	switch userLang {
	case "zh-TW":
		if doc.TitleZHTW != "" {
			title = doc.TitleZHTW
			isTranslated = true
		}
	case "en":
		if doc.TitleEN != "" {
			title = doc.TitleEN
			isTranslated = true
		}
	}

	var image string
	if len(doc.Images) > 0 {
		image = doc.Images[0]
	}

	lang := i18n.Lang(userLang)

	return domain.ProductSummary{
		ID:            doc.ID,
		Title:         title,
		TitleOriginal: doc.Title,
		Image:         image,
		PriceJPY:      doc.PriceJPY,
		Platform:      doc.SourcePlatform,
		Status:        i18n.StatusLabel(doc.Status, lang),
		Brand:         doc.BrandName,
		Condition:     i18n.ConditionLabel(doc.Condition, lang),
		Tags:          doc.Tags,
		IsTranslated:  isTranslated,
	}
}

func mapBuckets(agg esBucketAgg) []domain.AggBucket {
	buckets := make([]domain.AggBucket, len(agg.Buckets))
	for i, b := range agg.Buckets {
		buckets[i] = domain.AggBucket{Key: b.Key, Count: b.DocCount}
	}
	return buckets
}

func mapPriceRanges(stats esStatsAgg) []domain.PriceRange {
	if stats.Count == 0 {
		return nil
	}
	return []domain.PriceRange{
		{Min: int64(stats.Min), Max: int64(stats.Max), Count: stats.Count},
	}
}

// --- ESProductFetcher ---

// ESProductFetcher implements the ProductFetcher interface using Elasticsearch.
type ESProductFetcher struct {
	client    *elasticsearch.Client
	indexName string
}

// NewESProductFetcher creates an ESProductFetcher.
func NewESProductFetcher(client *elasticsearch.Client, indexName string) *ESProductFetcher {
	return &ESProductFetcher{client: client, indexName: indexName}
}

// GetProduct fetches a single product from Elasticsearch by its ID.
func (f *ESProductFetcher) GetProduct(ctx context.Context, id string) (*domain.UnifiedProduct, error) {
	res, err := f.client.Get(f.indexName, id,
		f.client.Get.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("es get product: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		return nil, domain.ErrAdapterNotFound
	}
	if res.IsError() {
		return nil, fmt.Errorf("es get product: status %s", res.Status())
	}

	var getResp esGetResponse
	if err := json.NewDecoder(res.Body).Decode(&getResp); err != nil {
		return nil, fmt.Errorf("es get product: decode: %w", err)
	}

	var product domain.UnifiedProduct
	if err := json.Unmarshal(getResp.Source, &product); err != nil {
		return nil, fmt.Errorf("es get product: unmarshal: %w", err)
	}

	return &product, nil
}

type esGetResponse struct {
	Found  bool            `json:"found"`
	Source json.RawMessage `json:"_source"`
}
