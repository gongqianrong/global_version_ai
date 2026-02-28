# Elasticsearch Client Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement the Elasticsearch client layer (search, product fetch, bulk write) using the official go-elasticsearch/v8 library.

**Architecture:** ESSearcher and ESProductFetcher in `internal/search/`, ESSink modification in `internal/output/`. All use mock HTTP transport for testing. Wired in `cmd/gateway/main.go`.

**Tech Stack:** Go 1.22, github.com/elastic/go-elasticsearch/v8

---

### Task 1: Add go-elasticsearch dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Steps:**

1. Run `go get github.com/elastic/go-elasticsearch/v8@latest` in the backend directory.
2. Verify `go.mod` contains the new dependency.

**Verify:**
```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go build ./...
```

---

### Task 2: ESSearcher — implement Search

**Files:**
- Create: `internal/search/es_searcher.go`
- Create: `internal/search/es_searcher_test.go`

**Implementation (`es_searcher.go`):**

```go
package search

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"

    "github.com/elastic/go-elasticsearch/v8"
    "github.com/rakutao/collection-gateway/internal/domain"
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
        Platforms:  mapBuckets(resp.Aggregations.Platforms),
        Brands:    mapBuckets(resp.Aggregations.Brands),
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

    return domain.ProductSummary{
        ID:            doc.ID,
        Title:         title,
        TitleOriginal: doc.Title,
        Image:         image,
        PriceJPY:      doc.PriceJPY,
        Platform:      doc.SourcePlatform,
        Status:        doc.Status,
        Brand:         doc.BrandName,
        Condition:     doc.Condition,
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
```

**Tests (`es_searcher_test.go`):**

Test using mock transport:
1. `TestESSearcher_Search_Success` — mock ES returns 2 hits + aggregations, verify ProductSummary mapping + aggregations
2. `TestESSearcher_Search_Empty` — mock ES returns 0 hits, verify empty results
3. `TestESSearcher_Search_ESError` — mock ES returns 500, verify error returned
4. `TestESSearcher_Search_LanguageSelection` — verify zh-TW/en/ja title selection
5. `TestMapDocToSummary` — unit test title selection and isTranslated flag

**Verify:**
```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/search/... -v -run ESSearcher
```

---

### Task 3: ESProductFetcher — implement GetProduct

**Files:**
- Modify: `internal/search/es_searcher.go` (add ESProductFetcher to same file)
- Modify: `internal/search/es_searcher_test.go` (add tests)

**Implementation (append to `es_searcher.go`):**

```go
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
```

**Tests:**
1. `TestESProductFetcher_GetProduct_Success` — mock returns a valid document
2. `TestESProductFetcher_GetProduct_NotFound` — mock returns 404, verify ErrAdapterNotFound
3. `TestESProductFetcher_GetProduct_ESError` — mock returns 500

**Verify:**
```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/search/... -v -run ESProductFetcher
```

---

### Task 4: ESSink — implement Bulk write

**Files:**
- Modify: `internal/output/es_sink.go`
- Create: `internal/output/es_sink_test.go`

**Implementation (replace `es_sink.go`):**

```go
package output

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"

    "github.com/elastic/go-elasticsearch/v8"
    "github.com/rakutao/collection-gateway/internal/domain"
)

// ESSink writes UnifiedProduct documents to Elasticsearch using the Bulk API.
type ESSink struct {
    client    *elasticsearch.Client
    indexName string
}

// NewESSink creates an ESSink targeting the given index.
func NewESSink(client *elasticsearch.Client, indexName string) *ESSink {
    return &ESSink{client: client, indexName: indexName}
}

// Name returns the sink identifier.
func (s *ESSink) Name() string { return "elasticsearch" }

// Write bulk-indexes the given products into Elasticsearch.
func (s *ESSink) Write(ctx context.Context, products []domain.UnifiedProduct) error {
    if len(products) == 0 {
        return nil
    }

    var buf bytes.Buffer
    for _, p := range products {
        meta := map[string]interface{}{
            "index": map[string]interface{}{
                "_index": s.indexName,
                "_id":    p.ID,
            },
        }
        metaLine, _ := json.Marshal(meta)
        buf.Write(metaLine)
        buf.WriteByte('\n')
        docLine, err := json.Marshal(p)
        if err != nil {
            return fmt.Errorf("es sink: marshal product %s: %w", p.ID, err)
        }
        buf.Write(docLine)
        buf.WriteByte('\n')
    }

    res, err := s.client.Bulk(
        bytes.NewReader(buf.Bytes()),
        s.client.Bulk.WithContext(ctx),
        s.client.Bulk.WithIndex(s.indexName),
    )
    if err != nil {
        return fmt.Errorf("es sink: bulk request: %w", err)
    }
    defer res.Body.Close()

    if res.IsError() {
        return fmt.Errorf("es sink: bulk response: %s", res.Status())
    }

    // Check for per-item errors.
    body, err := io.ReadAll(res.Body)
    if err != nil {
        return fmt.Errorf("es sink: read bulk response: %w", err)
    }

    var bulkResp bulkResponse
    if err := json.Unmarshal(body, &bulkResp); err != nil {
        return fmt.Errorf("es sink: decode bulk response: %w", err)
    }

    if bulkResp.Errors {
        return fmt.Errorf("es sink: bulk indexing had errors")
    }

    return nil
}

type bulkResponse struct {
    Errors bool `json:"errors"`
}
```

**Tests (`es_sink_test.go`):**
1. `TestESSink_Write_Success` — mock returns 200 with `{"errors": false}`, verify no error
2. `TestESSink_Write_Empty` — empty products slice, verify no error and no HTTP call
3. `TestESSink_Write_BulkError` — mock returns 200 with `{"errors": true}`, verify error
4. `TestESSink_Write_ESError` — mock returns 500

**Verify:**
```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/output/... -v
```

---

### Task 5: Wire ES client in main.go + full verification

**Files:**
- Modify: `cmd/gateway/main.go`

**Implementation:**

Add imports for `elasticsearch` and `search` packages. Add ES client initialization:

```go
import (
    "github.com/elastic/go-elasticsearch/v8"
    "github.com/rakutao/collection-gateway/internal/search"
)

// In main():
esURL := os.Getenv("ELASTICSEARCH_URL")
if esURL == "" {
    esURL = "http://localhost:9200"
}
esIndexName := os.Getenv("ES_INDEX_NAME")
if esIndexName == "" {
    esIndexName = "rakutao_products"
}
esClient, err := elasticsearch.NewClient(elasticsearch.Config{
    Addresses: []string{esURL},
})
if err != nil {
    log.Fatalf("elasticsearch client: %v", err)
}

esSearcher := search.NewESSearcher(esClient, esIndexName)
esFetcher := search.NewESProductFetcher(esClient, esIndexName)

// Update handler wiring:
searchHandler := api.NewSearchHandler(gateway, esSearcher, streamManager)
productHandler := api.NewProductHandler(esFetcher)
```

Also update ESSink in output package (if output router is wired).

**Verify:**
```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go build ./... && go test ./... -v -race
```

Expected: all packages compile, all tests pass.
