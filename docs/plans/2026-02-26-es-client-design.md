# Elasticsearch Client Design

## Overview

Implement the Elasticsearch client layer for the Rakutao collection gateway, covering three interfaces: search (Searcher), single product fetch (ProductFetcher), and bulk write (ESSink). Uses the official `go-elasticsearch/v8` client library.

## Architecture

All ES read operations live in `internal/search/` alongside the existing `buildESQuery()`. The write operation stays in `internal/output/` where the ESSink stub already exists. The official ES Go client is injected via constructor, enabling mock-based testing without a running ES instance.

## File Structure

```
internal/search/
├── es_client.go          # existing — buildESQuery(), buildSort()
├── es_searcher.go        # NEW — ESSearcher + ESProductFetcher
├── es_searcher_test.go   # NEW — tests with mock HTTP transport
├── gateway.go            # unchanged
└── index.json            # unchanged

internal/output/
├── es_sink.go            # MODIFY — real Bulk API implementation
├── es_sink_test.go       # NEW — Bulk write tests

cmd/gateway/main.go       # MODIFY — wire ES client
```

## Component Design

### ESSearcher

Implements `api.Searcher` interface.

```go
type ESSearcher struct {
    client    *elasticsearch.Client
    indexName string
}
```

**Search flow:**
1. `buildESQuery(query)` → JSON bytes
2. `es.Search(index, body)` → ES HTTP response
3. Parse response JSON into internal `esResponse` struct
4. Map `hits.hits[]._source` → `[]ProductSummary`
5. Map `aggregations` → `SearchAggs`
6. Return `*SearchResponse`

**ProductSummary mapping from ES `_source`:**
- `ID` = `id`
- `Title` = `title_zh_tw` / `title_en` / `title` (based on UserLang)
- `TitleOriginal` = `title`
- `Image` = `images[0]`
- `PriceJPY` = `price_jpy`
- `Platform` = `source_platform`
- `Brand` = `brand_name`
- `Condition` = `condition`
- `Tags` = `tags`
- `IsTranslated` = title differs from original

### ESProductFetcher

Implements `api.ProductFetcher` interface.

```go
type ESProductFetcher struct {
    client    *elasticsearch.Client
    indexName string
}
```

- `GetProduct(ctx, id)` → ES GET `/{index}/_doc/{id}`
- Parse `_source` → `UnifiedProduct`
- Not found → `domain.ErrAdapterNotFound`

### ESSink (modified)

```go
type ESSink struct {
    client    *elasticsearch.Client
    indexName string
}
```

- `Write(ctx, products)` → ES Bulk API
- Each product serialized as JSON with `product.ID` as `_id`
- Returns error if any bulk item fails

## Configuration (Environment Variables)

| Variable | Default | Description |
|----------|---------|-------------|
| `ELASTICSEARCH_URL` | `http://localhost:9200` | ES cluster URL |
| `ES_INDEX_NAME` | `rakutao_products` | Target index name |

## Testing Strategy

Mock HTTP transport via `elasticsearch.Config{Transport: roundTripFunc(...)}` — no real ES instance needed.

Test cases:
- Search: normal results, empty results, ES error, aggregation parsing
- ProductFetcher: found, not found, ES error
- ESSink: bulk success, bulk partial error

## Dependency

```
github.com/elastic/go-elasticsearch/v8
```
