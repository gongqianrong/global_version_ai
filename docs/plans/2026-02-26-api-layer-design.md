# API Layer Design — Rakutao International Collection Gateway

**Version:** 1.0
**Date:** 2026-02-26

## 1. Scope

First-phase API layer covering the core search-to-detail loop:

- **Search** (HTTP + WebSocket real-time)
- **Product detail**
- **Health check**

Out of scope: cart, orders, wallet, payments, auth, live stream.

## 2. Tech Stack

| Component | Choice | Rationale |
|-----------|--------|-----------|
| HTTP router | `chi/v5` | Lightweight, net/http compatible, middleware chains, route groups |
| CORS | `go-chi/cors` | chi ecosystem, simple config |
| WebSocket | `nhooyr.io/websocket` | Modern API, context-aware, production-grade |
| JSON | `encoding/json` (stdlib) | No external dependency needed |

## 3. Architecture

### 3.1 Request Flow

```
Client
  │
  ├── GET /api/v1/search ──────► SearchHandler
  │     │                            ├── KeywordFilter.IsBlocked()
  │     │                            ├── Gateway.prepareQuery() (translate keyword)
  │     │                            ├── ESClient.Search() (cached results)
  │     │                            ├── StreamManager.Create() (realtime stream)
  │     │                            └── JSON response (results + streamID)
  │     │
  │     └── WS /api/v1/search/stream/{streamID} ──► RealtimeHandler
  │                                                    ├── StreamManager.Get(streamID)
  │                                                    ├── Registry.RealtimeSearchable()
  │                                                    ├── Adapter.Search() (concurrent, per platform)
  │                                                    ├── Normalizer.Normalize() (per result)
  │                                                    └── WebSocket push (incremental)
  │
  ├── GET /api/v1/products/{id} ──► ProductHandler
  │                                    ├── ES cache lookup (by ID)
  │                                    ├── Cache miss → Adapter.GetProduct()
  │                                    ├── Normalizer.Normalize()
  │                                    └── JSON response
  │
  └── GET /health ─────────────► HealthHandler
                                    ├── Registry.All()
                                    ├── Adapter.HealthCheck() (per adapter)
                                    └── JSON response
```

### 3.2 Dependency Injection

`cmd/gateway/main.go` assembles all dependencies:

```
main.go
  ├── KeywordFilter (with political keyword list)
  ├── Gateway (translator, keywordFilter)
  ├── Registry (platform adapters)
  ├── Normalizer (brandExtractor)
  ├── API handlers (receive above dependencies)
  ├── Router (mount handlers + middleware)
  └── http.Server (listen :8080)
```

### 3.3 Middleware Chain

```
Recovery → Logger → CORS → RequestID → [route match] → Handler
```

- **Recovery**: catch panics, return 500
- **Logger**: log method, path, status, latency
- **CORS**: configurable allowed origins
- **RequestID**: UUID per request for tracing

## 4. Endpoints

### 4.1 GET /api/v1/search

**Query Parameters:**

| Param | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `keyword` | string | yes | - | Search keyword (any language) |
| `platforms` | string | no | all | Comma-separated platform IDs |
| `brand_id` | string | no | - | Brand filter |
| `categories` | string | no | - | Comma-separated categories |
| `price_min` | int | no | 0 | Min price (JPY) |
| `price_max` | int | no | 0 | Max price (JPY) |
| `condition` | string | no | - | Comma-separated conditions |
| `sort` | string | no | relevance | relevance/price_asc/price_desc/newest |
| `page` | int | no | 1 | Page number |
| `page_size` | int | no | 20 | Page size (max 100) |
| `lang` | string | no | zh-TW | User language |
| `content_rating` | string | no | general | general/all |

**Response:**

```json
{
  "code": 0,
  "data": {
    "cached_results": [
      {
        "id": "yahoo_auction_abc123",
        "title": "古馳手提包",
        "title_original": "グッチ ハンドバッグ",
        "image": "https://...",
        "price_jpy": 16500,
        "platform": "yahoo_auction",
        "status": "available",
        "brand": "Gucci",
        "condition": "good",
        "is_translated": true
      }
    ],
    "cached_total": 128,
    "realtime_stream_id": "stream_abc123",
    "aggregations": {
      "platforms": [{"key": "yahoo_auction", "count": 45}],
      "brands": [{"key": "Gucci", "count": 12}],
      "categories": [{"key": "bags", "count": 30}],
      "price_ranges": [{"min": 0, "max": 5000, "count": 20}]
    },
    "translated_keyword": "グッチ バッグ"
  },
  "request_id": "uuid"
}
```

### 4.2 WS /api/v1/search/stream/{streamID}

Client connects after receiving `realtime_stream_id` from search response. Server concurrently queries platform adapters and pushes results incrementally.

**Server-sent message types:**

```json
// Platform results arrived
{
  "type": "results",
  "platform": "amazon_jp",
  "products": [{ ... }],
  "total": 35
}

// All platforms finished
{
  "type": "done",
  "platforms_searched": ["yahoo_auction", "amazon_jp", "surugaya"]
}

// Platform error/timeout
{
  "type": "error",
  "platform": "animate",
  "message": "timeout"
}
```

**Lifecycle:**
- streamID valid for 30 seconds after creation
- Total search timeout: 5 seconds
- After all platforms respond or timeout, sends `done` and closes connection
- Unclaimed streams auto-expire and get garbage-collected

### 4.3 GET /api/v1/products/{id}

**Path Parameter:** `id` — unified product ID (`{platform}_{sourceID}`)

**Query Parameter:** `lang` (default: `zh-TW`)

**Data Source Strategy:**
1. Look up ES cache by product ID
2. Cache miss → parse platform from ID, call adapter `GetProduct()`
3. Normalize result, write to ES, return to client

**Response:**

```json
{
  "code": 0,
  "data": {
    "id": "yahoo_auction_abc123",
    "platform": "yahoo_auction",
    "title": "古馳手提包",
    "title_original": "グッチ ハンドバッグ",
    "description": "狀態良好的古馳手提包...",
    "description_original": "状態の良いグッチハンドバッグ...",
    "images": ["https://..."],
    "price_jpy": 16500,
    "service_fee_jpy": 1500,
    "original_price": 15000,
    "shipping_type": "free",
    "shipping_fee_jpy": 0,
    "brand": {"id": "b_gucci", "name": "Gucci", "name_ja": "グッチ"},
    "categories": ["bags", "luxury"],
    "condition": "good",
    "status": "available",
    "quantity": 1,
    "seller": {"seller_id": "s001", "seller_name": "TopSeller", "rating": 4.8},
    "variants": [{"name": "Color", "options": ["Red", "Blue"]}],
    "content_rating": "general",
    "listed_at": "2026-02-20T10:00:00Z",
    "is_translated": true
  },
  "request_id": "uuid"
}
```

### 4.4 GET /health

**Response:**

```json
{
  "status": "ok",
  "platforms": {
    "yahoo_auction": {"status": "healthy"},
    "amazon_jp": {"status": "healthy"},
    "surugaya": {"status": "degraded", "message": "high latency"}
  }
}
```

## 5. StreamManager

In-memory manager for active real-time search sessions.

```
StreamManager
  ├── Create(query SearchQuery) → streamID string
  ├── Get(streamID) → *Stream, bool
  └── background cleanup goroutine (30s TTL)
```

Each `Stream` holds:
- `SearchQuery` — the original query
- `chan StreamEvent` — channel for platform results
- `createdAt` — for TTL expiration
- `claimed` — whether a WebSocket client has connected

WebSocket handler reads from channel, writes to socket. Platform adapter goroutines write to channel.

## 6. Unified Response Format

```go
// Success
{"code": 0, "data": {...}, "request_id": "uuid"}

// Error
{"code": 40001, "message": "keyword is blocked by content policy", "request_id": "uuid"}
```

## 7. Error Codes

| Code | HTTP Status | Meaning |
|------|-------------|---------|
| `0` | 200 | Success |
| `40001` | 400 | Keyword blocked (political) |
| `40002` | 400 | Missing required parameter |
| `40003` | 400 | Invalid parameter format |
| `40401` | 404 | Product not found |
| `50001` | 500 | Internal server error |
| `50002` | 502 | Platform adapter error |
| `50003` | 503 | Search service unavailable |

## 8. File Structure

```
backend/internal/api/
  ├── router.go       — chi route definitions, middleware mounting
  ├── search.go       — SearchHandler (HTTP search + param parsing)
  ├── product.go      — ProductHandler (product detail)
  ├── realtime.go     — RealtimeHandler (WebSocket) + StreamManager
  ├── health.go       — HealthHandler
  ├── middleware.go    — Recovery, Logger, CORS, RequestID
  └── response.go     — JSON response helpers (Success/Error)

backend/cmd/gateway/main.go  — dependency assembly, HTTP server startup
```

## 9. External Dependencies (go.mod additions)

```
github.com/go-chi/chi/v5
github.com/go-chi/cors
nhooyr.io/websocket
```

## 10. YAGNI — Explicitly Not Doing

- No auth/authorization (no user system in phase 1)
- No rate limiting (handled by Nginx/API gateway later)
- No API version negotiation (fixed v1)
- No gRPC (Python AI service called via HTTP)
- No cart/order/wallet/payment endpoints (future phases)
