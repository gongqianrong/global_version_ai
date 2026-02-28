# AI Service Integration Design

## Overview

Wire the Python AI service (translation + brand extraction) to the Go backend via HTTP. This involves:
1. Adding FastAPI HTTP endpoints to the Python AI service
2. Creating a Go HTTP client package (`aiclient`) that implements `Translator` and `AIExtractor` interfaces
3. Wiring everything together in `main.go`

## Python Side — FastAPI Endpoints

Install `fastapi` + `uvicorn`. New file `app/main.py`.

### POST /translate

```json
Request:  {"keyword": "gucci bag", "source_lang": "en"}
Response: {"keyword_ja": "グッチ バッグ", "original": "gucci bag",
           "source_lang": "en", "was_translated": true}
```

### POST /extract-brand

```json
Request:  {"title": "GUCCI バッグ", "description": "新品", "category": "ファッション"}
Response: {"brand_name": "GUCCI", "confidence": 0.95}
No match: {"brand_name": null, "confidence": 0.0}
```

### GET /health

```json
Response: {"status": "healthy"}
```

App instantiates `TranslationService` and `BrandExtractor` singletons. Endpoints delegate to existing class methods.

Dockerfile updated to run: `uvicorn app.main:app --host 0.0.0.0 --port 8000`

## Go Side — aiclient Package

New package: `internal/aiclient/`

### Client

```go
type Client struct {
    baseURL    string
    httpClient *http.Client
}
func New(baseURL string, httpClient *http.Client) *Client
```

### Translator Interface (search.Translator)

```go
func (c *Client) Translate(ctx context.Context, keyword, sourceLang string) (string, error)
```
POST `{baseURL}/translate` → returns `keyword_ja` field.

### AIExtractor Interface (brand.AIExtractor)

```go
func (c *Client) Extract(title, description, category string) (*brand.AIExtractionResult, error)
```
POST `{baseURL}/extract-brand` → returns `{brand_name, confidence}`.

### Error Handling

- Network errors / non-200 → return error
- Gateway already falls back to original keyword on translation failure
- Brand Pipeline Level 3 already handles nil/error gracefully

## Wiring in main.go

```go
aiClient := aiclient.New(aiServiceURL, nil)
gateway := search.NewGateway(aiClient, keywordFilter)
brandPipeline := brand.NewPipeline(brandRegistry, aiClient)
norm := normalizer.New(brandPipeline)
```

Environment variable: `AI_SERVICE_URL` (default: `http://localhost:8000`)

## Testing

- Python: pytest with FastAPI TestClient
- Go: httptest mock server for aiclient tests
