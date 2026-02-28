# Platform Adapters Design (Phase 1)

## Goal

Build platform adapters for Yahoo Auction Japan and Surugaya, proxying to existing domestic API backends. Implement PlatformSearchService to connect adapters to the API layer for real-time search.

## Architecture

Each platform gets its own Go package under `adapter/`. Each package contains a client (HTTP calls + JSON parsing) and an adapter (implements `domain.PlatformAdapter`). Both platforms proxy to separate domestic API services with different response formats.

A new `PlatformSearchService` bridges the adapter layer to the API layer by implementing `PlatformSearcher` and `PlatformLister` interfaces.

## Scope

- **In scope:** yahoo_auction adapter, surugaya adapter, PlatformSearchService, registry wiring, main.go update
- **Out of scope:** ProductFetcher (needs ES), BatchCollect, retry/circuit-breaker, other platforms (amazon_jp, animate, lashinbang)

## File Structure

```
backend/internal/adapter/
  yahoo_auction/
    client.go          HTTP client for Yahoo Auction domestic API
    adapter.go         PlatformAdapter implementation
    adapter_test.go    Tests with httptest mock server
  surugaya/
    client.go          HTTP client for Surugaya domestic API
    adapter.go         PlatformAdapter implementation
    adapter_test.go    Tests with httptest mock server

backend/internal/api/
  platform_service.go       PlatformSearchService (PlatformSearcher + PlatformLister)
  platform_service_test.go  Tests

backend/cmd/gateway/
  main.go                   Updated wiring
```

## Adapter Design

### Construction

```go
func NewAdapter(baseURL string, opts ...Option) *Adapter
```

- `baseURL`: domestic API base URL (e.g. `http://localhost:3001`)
- Options: custom HTTP client, custom timeout

### PlatformAdapter Methods

| Method | yahoo_auction | surugaya |
|--------|--------------|----------|
| PlatformID() | `"yahoo_auction"` | `"surugaya"` |
| Capabilities() | Search+Realtime, MaxQPS=10 | Search+Realtime, MaxQPS=10 |
| Search() | GET `{baseURL}/api/search` | GET `{baseURL}/api/search` |
| GetProduct() | GET `{baseURL}/api/product/{id}` | GET `{baseURL}/api/product/{id}` |
| BatchCollect() | not implemented | not implemented |
| HealthCheck() | GET `{baseURL}/health` | GET `{baseURL}/health` |

### Client Internal Structures

Each platform defines its own JSON response structs in client.go. Currently using stub format:

```go
type searchResponse struct {
    Items []item `json:"items"`
    Total int64  `json:"total"`
}
type item struct {
    ID, Title, Description, Status, Condition, Category string
    Price float64
    Images []string
}
```

These structs will be updated when actual API response formats are provided.

### RawProduct Conversion

Adapter converts client items to `domain.RawProduct` with RawData map containing fields expected by Normalizer:

- `title` (string) — required
- `price` (float64) — required
- `description` (string) — optional
- `images` ([]string) — optional
- `status` (string) — optional
- `condition` (string) — optional
- `category` (string) — optional

## PlatformSearchService

Implements `api.PlatformSearcher` and `api.PlatformLister`:

```
SearchPlatform(ctx, platformID, query):
  1. registry.GetAdapter(platformID)
  2. adapter.Search(ctx, query)
  3. normalizer.Normalize(rawProduct) for each result
  4. Map UnifiedProduct → ProductSummary
  5. Return []ProductSummary, total, nil

RealtimePlatformIDs():
  1. registry.RealtimeSearchable()
  2. Return []string of platform IDs
```

## Registration (main.go)

```go
reg := registry.New()
reg.Register(yahooMeta, yahoo_auction.NewAdapter(os.Getenv("YAHOO_AUCTION_API_URL")))
reg.Register(surugayaMeta, surugaya.NewAdapter(os.Getenv("SURUGAYA_API_URL")))

norm := normalizer.New(nil) // brand extractor optional
platformService := api.NewPlatformSearchService(reg, norm)

realtimeHandler := api.NewRealtimeHandler(sm, platformService, platformService)
```

## Error Handling

- HTTP errors from domestic API → adapter returns error → PlatformSearcher sends EventError to stream
- Timeout → context cancellation propagates through adapter
- Adapter not found in registry → PlatformSearchService returns error

## Testing Strategy

- Each adapter: httptest mock server simulating domestic API responses
- PlatformSearchService: mock registry + mock adapter + mock normalizer
- All tests run with `-race` flag

## YAGNI — Not Doing

- ProductFetcher implementation (needs ES client)
- BatchCollect implementation
- Retry/circuit-breaker middleware
- Rate limiting enforcement
- Other platforms (amazon_jp, animate, lashinbang)
