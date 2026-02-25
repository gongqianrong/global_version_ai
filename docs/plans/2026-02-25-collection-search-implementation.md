# Collection & Search System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the Rakutao International backend data layer: multi-platform collection gateway, unified search with translation, and brand recognition system.

**Architecture:** Go unified gateway with Adapter pattern for multi-platform collection, Elasticsearch for search, Python microservice for AI (brand extraction + translation + crawlers). Output Router with interface-based sinks for ES/Redis/future WMS.

**Tech Stack:** Go 1.22+, Python 3.12+ (FastAPI + Scrapy), Elasticsearch 8.x (kuromoji), Redis 7.x, PostgreSQL 16, gRPC (Go ↔ Python), Docker Compose for local dev.

**Design Doc:** `docs/plans/2026-02-25-collection-search-system-design.md`

---

## Phase 1: Go 项目脚手架 + 核心领域模型

### Task 1: Go 项目初始化

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/gateway/main.go`
- Create: `backend/Makefile`
- Create: `backend/.gitignore`

**Step 1: 初始化 Go module**

```bash
mkdir -p backend/cmd/gateway
cd backend
go mod init github.com/rakutao/collection-gateway
```

**Step 2: 创建 main.go 入口**

```go
// backend/cmd/gateway/main.go
package main

import (
    "fmt"
    "os"
)

func main() {
    fmt.Println("Rakutao Collection Gateway starting...")
    os.Exit(0)
}
```

**Step 3: 创建 Makefile**

```makefile
.PHONY: build test run lint

build:
	go build -o bin/gateway ./cmd/gateway

test:
	go test ./... -v -race

run:
	go run ./cmd/gateway

lint:
	golangci-lint run ./...
```

**Step 4: 创建 .gitignore**

```
bin/
*.exe
*.out
vendor/
.env
```

**Step 5: 验证编译通过**

Run: `cd backend && go build ./cmd/gateway`
Expected: 编译成功，无输出

**Step 6: Commit**

```bash
git add backend/
git commit -m "feat: init Go project scaffold for collection gateway"
```

---

### Task 2: 统一商品数据结构 (UnifiedProduct Schema)

**Files:**
- Create: `backend/internal/domain/product.go`
- Create: `backend/internal/domain/product_test.go`

**Step 1: 写测试 — 验证 Product ID 生成规则**

```go
// backend/internal/domain/product_test.go
package domain

import "testing"

func TestNewProductID(t *testing.T) {
    tests := []struct {
        platform string
        sourceID string
        want     string
    }{
        {"mercari", "m123456", "mercari_m123456"},
        {"rakuma", "r789", "rakuma_r789"},
    }
    for _, tt := range tests {
        got := NewProductID(tt.platform, tt.sourceID)
        if got != tt.want {
            t.Errorf("NewProductID(%q, %q) = %q, want %q", tt.platform, tt.sourceID, got, tt.want)
        }
    }
}

func TestUnifiedProduct_IsAvailable(t *testing.T) {
    p := UnifiedProduct{Status: StatusAvailable}
    if !p.IsAvailable() {
        t.Error("expected available product to return true")
    }
    p.Status = StatusSold
    if p.IsAvailable() {
        t.Error("expected sold product to return false")
    }
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/domain/ -v`
Expected: FAIL — 编译错误，类型未定义

**Step 3: 实现领域模型**

```go
// backend/internal/domain/product.go
package domain

import (
    "fmt"
    "time"
)

// Product status constants
const (
    StatusAvailable = "available"
    StatusSold      = "sold"
    StatusReserved  = "reserved"
    StatusDelisted  = "delisted"
)

// Product condition constants
const (
    ConditionNew     = "new"
    ConditionLikeNew = "like_new"
    ConditionGood    = "good"
    ConditionFair    = "fair"
    ConditionPoor    = "poor"
)

// Shipping type constants
const (
    ShippingFree      = "free"
    ShippingBuyerPays = "buyer_pays"
    ShippingIncluded  = "included"
)

// Content rating constants
const (
    ContentRatingGeneral = "general"
    ContentRatingR18     = "r18"
)

// NewProductID generates a globally unique ID from platform and source ID.
func NewProductID(platform, sourceID string) string {
    return fmt.Sprintf("%s_%s", platform, sourceID)
}

// UnifiedProduct is the canonical product representation across all platforms.
type UnifiedProduct struct {
    ID              string
    SourcePlatform  string
    SourceID        string
    SourceURL       string

    Title           string
    TitleTranslated map[string]string
    Description     string
    DescTranslated  map[string]string
    Images          []string

    PriceJPY       int64
    OriginalPrice  int64
    ShippingType   string
    ShippingFeeJPY int64

    Brand          *Brand
    Categories     []string
    SourceCategory string
    Tags           []string

    Status    string
    Condition string
    Quantity  int

    Seller   SellerInfo
    Variants []Variant

    ListedAt    time.Time
    CollectedAt time.Time
    UpdatedAt   time.Time
    CacheTTL    int

    ContentRating string
    Logistics     *LogisticsInfo
}

// IsAvailable returns true if the product can be purchased.
func (p *UnifiedProduct) IsAvailable() bool {
    return p.Status == StatusAvailable
}

type Brand struct {
    ID     string
    Name   string
    NameJA string
    Source string // "platform_field" | "rule_matched" | "ai_extracted"
}

type SellerInfo struct {
    SellerID   string
    SellerName string
    Rating     float64
    ItemCount  int
}

type Variant struct {
    Name    string
    Options []string
}

// LogisticsInfo is a placeholder for future WMS integration.
type LogisticsInfo struct {
    EstimatedWeight float64
    EstimatedSize   string
    WarehouseRegion string
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/domain/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/domain/
git commit -m "feat: add UnifiedProduct domain model with Brand, Seller, Variant types"
```

---

### Task 3: RawProduct + 平台能力声明类型

**Files:**
- Create: `backend/internal/domain/adapter.go`
- Create: `backend/internal/domain/adapter_test.go`

**Step 1: 写测试 — 验证能力声明逻辑**

```go
// backend/internal/domain/adapter_test.go
package domain

import "testing"

func TestAdapterCaps_CanSearch(t *testing.T) {
    caps := AdapterCaps{SupportsSearch: true, SupportsRealtime: true}
    if !caps.CanRealtimeSearch() {
        t.Error("expected CanRealtimeSearch to be true")
    }
    caps.SupportsRealtime = false
    if caps.CanRealtimeSearch() {
        t.Error("expected CanRealtimeSearch to be false when realtime not supported")
    }
}

func TestRawProduct_HasRawData(t *testing.T) {
    rp := RawProduct{
        Platform: "mercari",
        RawID:    "m123",
        RawData:  map[string]interface{}{"title": "テスト"},
    }
    if rp.Platform != "mercari" {
        t.Errorf("expected platform mercari, got %s", rp.Platform)
    }
    if rp.RawData["title"] != "テスト" {
        t.Error("expected raw data to contain title")
    }
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/domain/ -v -run TestAdapterCaps`
Expected: FAIL

**Step 3: 实现 Adapter 相关类型**

```go
// backend/internal/domain/adapter.go
package domain

import "context"

// AdapterCaps declares what a platform adapter can do.
type AdapterCaps struct {
    SupportsSearch       bool
    SupportsRealtime     bool
    SupportsBatchCollect bool
    HasBrandField        bool
    HasCategoryField     bool
    MaxQPS               int
}

// CanRealtimeSearch returns true if the adapter supports live proxy search.
func (c AdapterCaps) CanRealtimeSearch() bool {
    return c.SupportsSearch && c.SupportsRealtime
}

// RawProduct holds the original data from a platform adapter before normalization.
type RawProduct struct {
    Platform   string
    RawID      string
    RawData    map[string]interface{}
    Normalized *UnifiedProduct
}

// HealthStatus represents a platform's health state.
type HealthStatus struct {
    Status  string // "healthy" | "degraded" | "offline"
    Message string
}

// CollectParams controls batch collection behavior.
type CollectParams struct {
    Categories []string
    MaxItems   int
    SinceTime  int64 // unix timestamp
}

// PlatformAdapter is the interface every platform must implement.
type PlatformAdapter interface {
    PlatformID() string
    Capabilities() AdapterCaps
    Search(ctx context.Context, query SearchQuery) (*SearchResult, error)
    GetProduct(ctx context.Context, productID string) (*RawProduct, error)
    BatchCollect(ctx context.Context, params CollectParams) (<-chan RawProduct, error)
    HealthCheck(ctx context.Context) HealthStatus
}

// SearchResult holds results returned from an adapter search call.
type SearchResult struct {
    Products []RawProduct
    Total    int64
    HasMore  bool
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/domain/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add backend/internal/domain/
git commit -m "feat: add PlatformAdapter interface, AdapterCaps, RawProduct types"
```

---

### Task 4: 搜索请求/响应类型

**Files:**
- Create: `backend/internal/domain/search.go`
- Create: `backend/internal/domain/search_test.go`

**Step 1: 写测试**

```go
// backend/internal/domain/search_test.go
package domain

import "testing"

func TestSearchQuery_IsGlobalSearch(t *testing.T) {
    q := SearchQuery{Keyword: "gucci bag"}
    if !q.IsGlobalSearch() {
        t.Error("empty platforms should be global search")
    }
    q.Platforms = []string{"mercari"}
    if q.IsGlobalSearch() {
        t.Error("specific platform should not be global search")
    }
}

func TestSearchQuery_HasFilters(t *testing.T) {
    q := SearchQuery{Keyword: "bag"}
    if q.HasFilters() {
        t.Error("no filters set")
    }
    q.BrandID = "b_001"
    if !q.HasFilters() {
        t.Error("brand filter set")
    }
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/domain/ -v -run TestSearchQuery`
Expected: FAIL

**Step 3: 实现搜索类型**

```go
// backend/internal/domain/search.go
package domain

// SearchQuery represents a search request from the client.
type SearchQuery struct {
    Keyword       string
    KeywordJA     string
    Platforms     []string
    BrandID       string
    Categories    []string
    PriceMin      int64
    PriceMax      int64
    Condition     []string
    SortBy        string // "relevance" | "price_asc" | "price_desc" | "newest"
    Page          int
    PageSize      int
    UserLang      string
    ContentRating string
}

// IsGlobalSearch returns true when no specific platform is targeted.
func (q *SearchQuery) IsGlobalSearch() bool {
    return len(q.Platforms) == 0
}

// HasFilters returns true when any filter beyond keyword is set.
func (q *SearchQuery) HasFilters() bool {
    return q.BrandID != "" ||
        len(q.Categories) > 0 ||
        q.PriceMin > 0 ||
        q.PriceMax > 0 ||
        len(q.Condition) > 0
}

// SearchResponse is returned to the client.
type SearchResponse struct {
    CachedResults     []ProductSummary
    CachedTotal       int64
    RealtimeStreamID  string
    Aggregations      SearchAggs
    TranslatedKeyword string
}

// ProductSummary is a lightweight product representation for search results.
type ProductSummary struct {
    ID            string
    Title         string
    TitleOriginal string
    Image         string
    PriceJPY      int64
    Platform      string
    Status        string
    Brand         string
    Tags          []string
    IsTranslated  bool
}

// SearchAggs holds aggregation data for filter panels.
type SearchAggs struct {
    Platforms  []AggBucket
    Brands     []AggBucket
    PriceRange PriceRange
    Categories []AggBucket
}

// AggBucket is a single aggregation entry.
type AggBucket struct {
    Key   string
    Count int64
}

// PriceRange represents min/max price boundaries.
type PriceRange struct {
    Min int64
    Max int64
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/domain/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add backend/internal/domain/
git commit -m "feat: add SearchQuery, SearchResponse, ProductSummary types"
```

---

## Phase 2: Platform Registry + Adapter 框架

### Task 5: Platform Registry 实现

**Files:**
- Create: `backend/internal/registry/registry.go`
- Create: `backend/internal/registry/registry_test.go`

**Step 1: 写测试**

```go
// backend/internal/registry/registry_test.go
package registry

import (
    "testing"

    "github.com/rakutao/collection-gateway/internal/domain"
)

type mockAdapter struct {
    id   string
    caps domain.AdapterCaps
}

func (m *mockAdapter) PlatformID() string              { return m.id }
func (m *mockAdapter) Capabilities() domain.AdapterCaps { return m.caps }
func (m *mockAdapter) Search(_ context.Context, _ domain.SearchQuery) (*domain.SearchResult, error) { return nil, nil }
func (m *mockAdapter) GetProduct(_ context.Context, _ string) (*domain.RawProduct, error) { return nil, nil }
func (m *mockAdapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) { return nil, nil }
func (m *mockAdapter) HealthCheck(_ context.Context) domain.HealthStatus { return domain.HealthStatus{Status: "healthy"} }

func TestRegistry_RegisterAndGet(t *testing.T) {
    r := New()
    adapter := &mockAdapter{id: "mercari", caps: domain.AdapterCaps{SupportsSearch: true}}

    r.Register(PlatformMeta{
        ID: "mercari", Name: "メルカリ", NameEN: "Mercari",
        Type: TypeDomesticProxy, Status: StatusActive,
    }, adapter)

    got, err := r.GetAdapter("mercari")
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if got.PlatformID() != "mercari" {
        t.Errorf("expected mercari, got %s", got.PlatformID())
    }
}

func TestRegistry_GetAdapter_NotFound(t *testing.T) {
    r := New()
    _, err := r.GetAdapter("nonexistent")
    if err == nil {
        t.Error("expected error for nonexistent platform")
    }
}

func TestRegistry_ActivePlatforms(t *testing.T) {
    r := New()
    r.Register(PlatformMeta{ID: "mercari", Status: StatusActive}, &mockAdapter{id: "mercari"})
    r.Register(PlatformMeta{ID: "tobu", Status: StatusOffline}, &mockAdapter{id: "tobu"})

    active := r.ActivePlatforms()
    if len(active) != 1 {
        t.Errorf("expected 1 active platform, got %d", len(active))
    }
    if active[0].ID != "mercari" {
        t.Errorf("expected mercari, got %s", active[0].ID)
    }
}

func TestRegistry_RealtimeSearchable(t *testing.T) {
    r := New()
    r.Register(PlatformMeta{ID: "mercari", Status: StatusActive,
        Caps: domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: true}},
        &mockAdapter{id: "mercari", caps: domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: true}})
    r.Register(PlatformMeta{ID: "tobu", Status: StatusActive,
        Caps: domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: false}},
        &mockAdapter{id: "tobu", caps: domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: false}})

    rt := r.RealtimeSearchable()
    if len(rt) != 1 {
        t.Errorf("expected 1 realtime-searchable, got %d", len(rt))
    }
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/registry/ -v`
Expected: FAIL

**Step 3: 实现 Registry**

```go
// backend/internal/registry/registry.go
package registry

import (
    "fmt"
    "sync"

    "github.com/rakutao/collection-gateway/internal/domain"
)

const (
    TypeDomesticProxy = "domestic_proxy"
    TypeSelfCrawler   = "self_crawler"
    TypeSelfAPI       = "self_api"

    StatusActive   = "active"
    StatusDegraded = "degraded"
    StatusOffline  = "offline"
)

// PlatformMeta holds metadata about a registered platform.
type PlatformMeta struct {
    ID     string
    Name   string
    NameEN string
    Icon   string
    Type   string
    Status string
    Caps   domain.AdapterCaps
}

type entry struct {
    meta    PlatformMeta
    adapter domain.PlatformAdapter
}

// Registry manages all registered platform adapters.
type Registry struct {
    mu      sync.RWMutex
    entries map[string]entry
}

// New creates an empty registry.
func New() *Registry {
    return &Registry{entries: make(map[string]entry)}
}

// Register adds a platform adapter to the registry.
func (r *Registry) Register(meta PlatformMeta, adapter domain.PlatformAdapter) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.entries[meta.ID] = entry{meta: meta, adapter: adapter}
}

// GetAdapter returns the adapter for the given platform ID.
func (r *Registry) GetAdapter(platformID string) (domain.PlatformAdapter, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    e, ok := r.entries[platformID]
    if !ok {
        return nil, fmt.Errorf("platform %q not registered", platformID)
    }
    return e.adapter, nil
}

// ActivePlatforms returns metadata for all platforms with status "active".
func (r *Registry) ActivePlatforms() []PlatformMeta {
    r.mu.RLock()
    defer r.mu.RUnlock()
    var result []PlatformMeta
    for _, e := range r.entries {
        if e.meta.Status == StatusActive {
            result = append(result, e.meta)
        }
    }
    return result
}

// RealtimeSearchable returns entries that support realtime proxy search.
func (r *Registry) RealtimeSearchable() []entry {
    r.mu.RLock()
    defer r.mu.RUnlock()
    var result []entry
    for _, e := range r.entries {
        if e.meta.Status == StatusActive && e.meta.Caps.CanRealtimeSearch() {
            result = append(result, e)
        }
    }
    return result
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/registry/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add backend/internal/registry/
git commit -m "feat: add PlatformRegistry with register, lookup, active/realtime filtering"
```

---

### Task 6: Domestic Proxy Adapter (类型 A — 国内版代理)

**Files:**
- Create: `backend/internal/adapter/domestic/adapter.go`
- Create: `backend/internal/adapter/domestic/adapter_test.go`

**Step 1: 写测试 — 用 httptest 模拟国内版接口**

```go
// backend/internal/adapter/domestic/adapter_test.go
package domestic

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/rakutao/collection-gateway/internal/domain"
)

func TestAdapter_PlatformID(t *testing.T) {
    a := New("mercari", "http://localhost", http.DefaultClient)
    if a.PlatformID() != "mercari" {
        t.Errorf("expected mercari, got %s", a.PlatformID())
    }
}

func TestAdapter_Search(t *testing.T) {
    // Mock domestic API response
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        resp := domesticSearchResponse{
            Items: []domesticItem{
                {ID: "m001", Title: "テスト商品", Price: 5000},
            },
            Total: 1,
        }
        json.NewEncoder(w).Encode(resp)
    }))
    defer server.Close()

    a := New("mercari", server.URL, server.Client())
    result, err := a.Search(context.Background(), domain.SearchQuery{Keyword: "テスト"})
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if len(result.Products) != 1 {
        t.Fatalf("expected 1 product, got %d", len(result.Products))
    }
    if result.Products[0].RawID != "m001" {
        t.Errorf("expected m001, got %s", result.Products[0].RawID)
    }
}

func TestAdapter_HealthCheck(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    }))
    defer server.Close()

    a := New("mercari", server.URL, server.Client())
    status := a.HealthCheck(context.Background())
    if status.Status != "healthy" {
        t.Errorf("expected healthy, got %s", status.Status)
    }
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/adapter/domestic/ -v`
Expected: FAIL

**Step 3: 实现 Domestic Proxy Adapter**

```go
// backend/internal/adapter/domestic/adapter.go
package domestic

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"

    "github.com/rakutao/collection-gateway/internal/domain"
)

// domesticSearchResponse mirrors the domestic API response format.
type domesticSearchResponse struct {
    Items []domesticItem `json:"items"`
    Total int64          `json:"total"`
}

type domesticItem struct {
    ID    string `json:"id"`
    Title string `json:"title"`
    Price int64  `json:"price"`
    // Domestic API returns platform-specific fields as raw JSON.
    // These are preserved in RawData for the Normalizer.
}

// Adapter proxies requests to the domestic version's collection API.
type Adapter struct {
    platformID  string
    domesticURL string
    client      *http.Client
}

// New creates a domestic proxy adapter for the given platform.
func New(platformID, domesticURL string, client *http.Client) *Adapter {
    return &Adapter{
        platformID:  platformID,
        domesticURL: domesticURL,
        client:      client,
    }
}

func (a *Adapter) PlatformID() string { return a.platformID }

func (a *Adapter) Capabilities() domain.AdapterCaps {
    return domain.AdapterCaps{
        SupportsSearch:       true,
        SupportsRealtime:     true,
        SupportsBatchCollect: true,
        HasBrandField:        false, // varies by platform, conservative default
        HasCategoryField:     true,
        MaxQPS:               10,
    }
}

func (a *Adapter) Search(ctx context.Context, query domain.SearchQuery) (*domain.SearchResult, error) {
    u, err := url.Parse(fmt.Sprintf("%s/api/search", a.domesticURL))
    if err != nil {
        return nil, fmt.Errorf("parse url: %w", err)
    }
    q := u.Query()
    q.Set("platform", a.platformID)
    q.Set("keyword", query.Keyword)
    q.Set("page", fmt.Sprintf("%d", query.Page))
    q.Set("page_size", fmt.Sprintf("%d", query.PageSize))
    u.RawQuery = q.Encode()

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    resp, err := a.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("do request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("domestic API returned status %d", resp.StatusCode)
    }

    var domesticResp domesticSearchResponse
    if err := json.NewDecoder(resp.Body).Decode(&domesticResp); err != nil {
        return nil, fmt.Errorf("decode response: %w", err)
    }

    products := make([]domain.RawProduct, len(domesticResp.Items))
    for i, item := range domesticResp.Items {
        products[i] = domain.RawProduct{
            Platform: a.platformID,
            RawID:    item.ID,
            RawData: map[string]interface{}{
                "id":    item.ID,
                "title": item.Title,
                "price": item.Price,
            },
        }
    }

    return &domain.SearchResult{
        Products: products,
        Total:    domesticResp.Total,
        HasMore:  int64(len(products)) < domesticResp.Total,
    }, nil
}

func (a *Adapter) GetProduct(ctx context.Context, productID string) (*domain.RawProduct, error) {
    // TODO: implement when domestic API endpoint is confirmed
    return nil, fmt.Errorf("GetProduct not yet implemented for domestic proxy")
}

func (a *Adapter) BatchCollect(ctx context.Context, params domain.CollectParams) (<-chan domain.RawProduct, error) {
    // TODO: implement batch collection via domestic API
    return nil, fmt.Errorf("BatchCollect not yet implemented for domestic proxy")
}

func (a *Adapter) HealthCheck(ctx context.Context) domain.HealthStatus {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/health", a.domesticURL), nil)
    if err != nil {
        return domain.HealthStatus{Status: "offline", Message: err.Error()}
    }
    resp, err := a.client.Do(req)
    if err != nil {
        return domain.HealthStatus{Status: "offline", Message: err.Error()}
    }
    defer resp.Body.Close()
    if resp.StatusCode == http.StatusOK {
        return domain.HealthStatus{Status: "healthy"}
    }
    return domain.HealthStatus{Status: "degraded", Message: fmt.Sprintf("status %d", resp.StatusCode)}
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/adapter/domestic/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add backend/internal/adapter/
git commit -m "feat: add DomesticProxyAdapter — proxies to domestic version collection API"
```

---

## Phase 3: Normalizer + Output Router

### Task 7: Normalizer — 数据标准化引擎

**Files:**
- Create: `backend/internal/normalizer/normalizer.go`
- Create: `backend/internal/normalizer/normalizer_test.go`

**Step 1: 写测试**

```go
// backend/internal/normalizer/normalizer_test.go
package normalizer

import (
    "testing"
    "time"

    "github.com/rakutao/collection-gateway/internal/domain"
)

func TestNormalize_BasicFields(t *testing.T) {
    n := New(nil) // no brand extractor for basic test
    raw := domain.RawProduct{
        Platform: "mercari",
        RawID:    "m001",
        RawData: map[string]interface{}{
            "id":          "m001",
            "title":       "テスト商品",
            "description": "これはテストです",
            "price":       float64(5000),
            "images":      []interface{}{"https://img.example.com/1.jpg"},
            "status":      "on_sale",
            "condition":   "目立った傷や汚れなし",
        },
    }

    product, err := n.Normalize("mercari", raw)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if product.ID != "mercari_m001" {
        t.Errorf("expected mercari_m001, got %s", product.ID)
    }
    if product.PriceJPY != 5000 {
        t.Errorf("expected price 5000, got %d", product.PriceJPY)
    }
    if product.Title != "テスト商品" {
        t.Errorf("expected テスト商品, got %s", product.Title)
    }
    if product.SourcePlatform != "mercari" {
        t.Errorf("expected mercari, got %s", product.SourcePlatform)
    }
}

func TestNormalize_MissingRequiredFields(t *testing.T) {
    n := New(nil)
    raw := domain.RawProduct{
        Platform: "mercari",
        RawID:    "m002",
        RawData:  map[string]interface{}{}, // missing everything
    }
    _, err := n.Normalize("mercari", raw)
    if err == nil {
        t.Error("expected error for missing required fields")
    }
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/normalizer/ -v`
Expected: FAIL

**Step 3: 实现 Normalizer**

```go
// backend/internal/normalizer/normalizer.go
package normalizer

import (
    "fmt"
    "time"

    "github.com/rakutao/collection-gateway/internal/domain"
)

// BrandExtractor is called by the Normalizer when brand detection is needed.
type BrandExtractor interface {
    Extract(title, description, category string) (*domain.Brand, error)
}

// Normalizer converts RawProduct data into UnifiedProduct.
type Normalizer struct {
    brandExtractor BrandExtractor
}

// New creates a Normalizer. brandExtractor may be nil if brand detection is not needed.
func New(brandExtractor BrandExtractor) *Normalizer {
    return &Normalizer{brandExtractor: brandExtractor}
}

// Normalize converts a RawProduct into a UnifiedProduct.
func (n *Normalizer) Normalize(platform string, raw domain.RawProduct) (*domain.UnifiedProduct, error) {
    title, _ := extractString(raw.RawData, "title")
    if title == "" {
        return nil, fmt.Errorf("missing required field: title")
    }

    price, err := extractFloat64(raw.RawData, "price")
    if err != nil {
        return nil, fmt.Errorf("missing or invalid required field: price: %w", err)
    }

    description, _ := extractString(raw.RawData, "description")
    images := extractStringSlice(raw.RawData, "images")
    sourceCategory, _ := extractString(raw.RawData, "category")

    product := &domain.UnifiedProduct{
        ID:             domain.NewProductID(platform, raw.RawID),
        SourcePlatform: platform,
        SourceID:       raw.RawID,
        Title:          title,
        Description:    description,
        Images:         images,
        PriceJPY:       int64(price),
        Status:         normalizeStatus(raw.RawData),
        Condition:      normalizeCondition(raw.RawData),
        Quantity:       1,
        SourceCategory: sourceCategory,
        CollectedAt:    time.Now(),
        UpdatedAt:      time.Now(),
        ContentRating:  domain.ContentRatingGeneral,
    }

    // Brand extraction (if extractor is available)
    if n.brandExtractor != nil {
        brand, _ := n.brandExtractor.Extract(title, description, sourceCategory)
        product.Brand = brand
    }

    return product, nil
}

func extractString(data map[string]interface{}, key string) (string, bool) {
    v, ok := data[key]
    if !ok {
        return "", false
    }
    s, ok := v.(string)
    return s, ok
}

func extractFloat64(data map[string]interface{}, key string) (float64, error) {
    v, ok := data[key]
    if !ok {
        return 0, fmt.Errorf("key %q not found", key)
    }
    f, ok := v.(float64)
    if !ok {
        return 0, fmt.Errorf("key %q is not a number", key)
    }
    return f, nil
}

func extractStringSlice(data map[string]interface{}, key string) []string {
    v, ok := data[key]
    if !ok {
        return nil
    }
    arr, ok := v.([]interface{})
    if !ok {
        return nil
    }
    result := make([]string, 0, len(arr))
    for _, item := range arr {
        if s, ok := item.(string); ok {
            result = append(result, s)
        }
    }
    return result
}

func normalizeStatus(data map[string]interface{}) string {
    s, _ := extractString(data, "status")
    switch s {
    case "on_sale", "available", "active":
        return domain.StatusAvailable
    case "sold", "sold_out":
        return domain.StatusSold
    case "reserved", "trading":
        return domain.StatusReserved
    default:
        return domain.StatusAvailable
    }
}

func normalizeCondition(data map[string]interface{}) string {
    s, _ := extractString(data, "condition")
    switch s {
    case "new", "新品、未使用":
        return domain.ConditionNew
    case "like_new", "未使用に近い":
        return domain.ConditionLikeNew
    case "good", "目立った傷や汚れなし":
        return domain.ConditionGood
    case "fair", "やや傷や汚れあり":
        return domain.ConditionFair
    case "poor", "傷や汚れあり", "全体的に状態が悪い":
        return domain.ConditionPoor
    default:
        return domain.ConditionGood
    }
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/normalizer/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add backend/internal/normalizer/
git commit -m "feat: add Normalizer — converts RawProduct to UnifiedProduct with status/condition mapping"
```

---

### Task 8: Output Router + ES Sink + Redis Sink

**Files:**
- Create: `backend/internal/output/router.go`
- Create: `backend/internal/output/router_test.go`
- Create: `backend/internal/output/es_sink.go`
- Create: `backend/internal/output/redis_sink.go`

**Step 1: 写测试 — 用 mock sink 验证路由逻辑**

```go
// backend/internal/output/router_test.go
package output

import (
    "context"
    "testing"

    "github.com/rakutao/collection-gateway/internal/domain"
)

type mockSink struct {
    name     string
    received []domain.UnifiedProduct
    err      error
}

func (m *mockSink) Name() string { return m.name }
func (m *mockSink) Write(ctx context.Context, products []domain.UnifiedProduct) error {
    m.received = append(m.received, products...)
    return m.err
}

func TestRouter_FanOut(t *testing.T) {
    s1 := &mockSink{name: "sink1"}
    s2 := &mockSink{name: "sink2"}
    router := NewRouter(s1, s2)

    products := []domain.UnifiedProduct{
        {ID: "mercari_m001", Title: "テスト"},
    }

    errs := router.Dispatch(context.Background(), products)
    if len(errs) != 0 {
        t.Fatalf("unexpected errors: %v", errs)
    }
    if len(s1.received) != 1 {
        t.Errorf("sink1 expected 1 product, got %d", len(s1.received))
    }
    if len(s2.received) != 1 {
        t.Errorf("sink2 expected 1 product, got %d", len(s2.received))
    }
}

func TestRouter_PartialFailure(t *testing.T) {
    s1 := &mockSink{name: "ok_sink"}
    s2 := &mockSink{name: "fail_sink", err: fmt.Errorf("write failed")}
    router := NewRouter(s1, s2)

    products := []domain.UnifiedProduct{{ID: "test_001"}}
    errs := router.Dispatch(context.Background(), products)

    if len(errs) != 1 {
        t.Fatalf("expected 1 error, got %d", len(errs))
    }
    // s1 should still have received the data
    if len(s1.received) != 1 {
        t.Errorf("ok_sink should still receive data despite other sink failing")
    }
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/output/ -v`
Expected: FAIL

**Step 3: 实现 Router**

```go
// backend/internal/output/router.go
package output

import (
    "context"
    "fmt"
    "sync"

    "github.com/rakutao/collection-gateway/internal/domain"
)

// Sink is the interface for downstream data consumers.
type Sink interface {
    Name() string
    Write(ctx context.Context, products []domain.UnifiedProduct) error
}

// Router fans out normalized products to all registered sinks.
type Router struct {
    sinks []Sink
}

// NewRouter creates a router with the given sinks.
func NewRouter(sinks ...Sink) *Router {
    return &Router{sinks: sinks}
}

// AddSink dynamically adds a sink (e.g., future WMS sink).
func (r *Router) AddSink(sink Sink) {
    r.sinks = append(r.sinks, sink)
}

// SinkError pairs a sink name with its error.
type SinkError struct {
    Sink string
    Err  error
}

func (e SinkError) Error() string {
    return fmt.Sprintf("sink %q: %v", e.Sink, e.Err)
}

// Dispatch sends products to all sinks concurrently.
// Returns errors from any sinks that failed; successful sinks are not affected.
func (r *Router) Dispatch(ctx context.Context, products []domain.UnifiedProduct) []SinkError {
    var mu sync.Mutex
    var errs []SinkError
    var wg sync.WaitGroup

    for _, s := range r.sinks {
        wg.Add(1)
        go func(sink Sink) {
            defer wg.Done()
            if err := sink.Write(ctx, products); err != nil {
                mu.Lock()
                errs = append(errs, SinkError{Sink: sink.Name(), Err: err})
                mu.Unlock()
            }
        }(s)
    }

    wg.Wait()
    return errs
}
```

```go
// backend/internal/output/es_sink.go
package output

import (
    "context"

    "github.com/rakutao/collection-gateway/internal/domain"
)

// ESSink writes products to Elasticsearch.
type ESSink struct {
    // TODO: add ES client when elasticsearch-go dependency is added
    indexName string
}

// NewESSink creates an Elasticsearch sink.
func NewESSink(indexName string) *ESSink {
    return &ESSink{indexName: indexName}
}

func (s *ESSink) Name() string { return "elasticsearch" }

func (s *ESSink) Write(ctx context.Context, products []domain.UnifiedProduct) error {
    // TODO: implement bulk index to ES
    // Will use elasticsearch-go official client
    // Bulk API for batch writes
    return nil
}
```

```go
// backend/internal/output/redis_sink.go
package output

import (
    "context"

    "github.com/rakutao/collection-gateway/internal/domain"
)

// RedisSink caches hot product data in Redis.
type RedisSink struct {
    // TODO: add Redis client when go-redis dependency is added
    prefix string
}

// NewRedisSink creates a Redis cache sink.
func NewRedisSink(prefix string) *RedisSink {
    return &RedisSink{prefix: prefix}
}

func (s *RedisSink) Name() string { return "redis" }

func (s *RedisSink) Write(ctx context.Context, products []domain.UnifiedProduct) error {
    // TODO: implement SET with TTL per product
    // Key format: {prefix}:{product.ID}
    // Value: JSON serialized UnifiedProduct
    // TTL: product.CacheTTL seconds
    return nil
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/output/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add backend/internal/output/
git commit -m "feat: add OutputRouter with fan-out dispatch, ESSink and RedisSink stubs"
```

---

## Phase 4: 品牌识别系统

### Task 9: Brand Registry (品牌库)

**Files:**
- Create: `backend/internal/brand/registry.go`
- Create: `backend/internal/brand/registry_test.go`

**Step 1: 写测试**

```go
// backend/internal/brand/registry_test.go
package brand

import "testing"

func TestRegistry_FindByAlias(t *testing.T) {
    r := NewRegistry()
    r.Add(Entry{
        ID:      "b_001",
        NameStd: "Louis Vuitton",
        NameJA:  "ルイ・ヴィトン",
        Aliases: []string{"LV", "ルイヴィトン", "LOUIS VUITTON", "louis vuitton"},
    })

    tests := []struct {
        input string
        want  string
    }{
        {"LV", "b_001"},
        {"ルイヴィトン", "b_001"},
        {"LOUIS VUITTON", "b_001"},
        {"louis vuitton", "b_001"},
        {"unknown brand", ""},
    }

    for _, tt := range tests {
        got := r.FindByAlias(tt.input)
        if tt.want == "" && got != nil {
            t.Errorf("FindByAlias(%q) expected nil, got %+v", tt.input, got)
        } else if tt.want != "" && (got == nil || got.ID != tt.want) {
            t.Errorf("FindByAlias(%q) expected %s, got %v", tt.input, tt.want, got)
        }
    }
}

func TestRegistry_MatchInText(t *testing.T) {
    r := NewRegistry()
    r.Add(Entry{
        ID: "b_001", NameStd: "Gucci", NameJA: "グッチ",
        Aliases: []string{"GUCCI", "gucci", "グッチ"},
    })
    r.Add(Entry{
        ID: "b_002", NameStd: "Nike", NameJA: "ナイキ",
        Aliases: []string{"NIKE", "ナイキ"},
    })

    text := "グッチ レザー ショルダーバッグ 美品"
    got := r.MatchInText(text)
    if got == nil || got.ID != "b_001" {
        t.Errorf("expected Gucci match, got %v", got)
    }

    text2 := "ノーブランド バッグ"
    got2 := r.MatchInText(text2)
    if got2 != nil {
        t.Errorf("expected no match, got %v", got2)
    }
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/brand/ -v`
Expected: FAIL

**Step 3: 实现品牌库**

```go
// backend/internal/brand/registry.go
package brand

import "strings"

// Entry represents a brand in the brand registry.
type Entry struct {
    ID         string
    NameStd    string
    NameJA     string
    NameZhTW   string
    Aliases    []string
    Category   string
    LogoURL    string
    IsVerified bool
}

// Registry is an in-memory brand lookup store.
type Registry struct {
    entries  []Entry
    aliasMap map[string]*Entry // lowercase alias → entry
}

// NewRegistry creates an empty brand registry.
func NewRegistry() *Registry {
    return &Registry{
        aliasMap: make(map[string]*Entry),
    }
}

// Add registers a brand entry and indexes all its aliases.
func (r *Registry) Add(e Entry) {
    r.entries = append(r.entries, e)
    ptr := &r.entries[len(r.entries)-1]

    // Index standard names
    r.aliasMap[strings.ToLower(e.NameStd)] = ptr
    if e.NameJA != "" {
        r.aliasMap[e.NameJA] = ptr
    }
    // Index all aliases
    for _, alias := range e.Aliases {
        r.aliasMap[strings.ToLower(alias)] = ptr
    }
}

// FindByAlias looks up a brand by exact alias match (case-insensitive for latin).
func (r *Registry) FindByAlias(alias string) *Entry {
    // Try exact (for Japanese)
    if e, ok := r.aliasMap[alias]; ok {
        return e
    }
    // Try lowercase (for Latin)
    if e, ok := r.aliasMap[strings.ToLower(alias)]; ok {
        return e
    }
    return nil
}

// MatchInText scans text for any known brand alias. Returns the first match found.
func (r *Registry) MatchInText(text string) *Entry {
    lower := strings.ToLower(text)
    for alias, entry := range r.aliasMap {
        if strings.Contains(lower, strings.ToLower(alias)) || strings.Contains(text, alias) {
            return entry
        }
    }
    return nil
}

// All returns all registered brand entries.
func (r *Registry) All() []Entry {
    return r.entries
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/brand/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add backend/internal/brand/
git commit -m "feat: add BrandRegistry with alias lookup and text matching"
```

---

### Task 10: 三级品牌识别流水线

**Files:**
- Create: `backend/internal/brand/pipeline.go`
- Create: `backend/internal/brand/pipeline_test.go`

**Step 1: 写测试**

```go
// backend/internal/brand/pipeline_test.go
package brand

import (
    "testing"

    "github.com/rakutao/collection-gateway/internal/domain"
)

type mockAIExtractor struct {
    result *AIExtractionResult
    err    error
}

func (m *mockAIExtractor) Extract(title, description, category string) (*AIExtractionResult, error) {
    return m.result, m.err
}

func TestPipeline_Level1_PlatformField(t *testing.T) {
    reg := NewRegistry()
    reg.Add(Entry{ID: "b_001", NameStd: "Gucci", NameJA: "グッチ", Aliases: []string{"GUCCI"}})

    p := NewPipeline(reg, nil)
    rawData := map[string]interface{}{"brand": "GUCCI"}
    result := p.Identify(rawData, "タイトル", "説明", "")

    if result == nil {
        t.Fatal("expected brand match")
    }
    if result.ID != "b_001" {
        t.Errorf("expected b_001, got %s", result.ID)
    }
    if result.Source != "platform_field" {
        t.Errorf("expected platform_field, got %s", result.Source)
    }
}

func TestPipeline_Level2_RuleMatch(t *testing.T) {
    reg := NewRegistry()
    reg.Add(Entry{ID: "b_002", NameStd: "Louis Vuitton", NameJA: "ルイ・ヴィトン",
        Aliases: []string{"LV", "ルイ・ヴィトン"}})

    p := NewPipeline(reg, nil)
    rawData := map[string]interface{}{} // no brand field
    result := p.Identify(rawData, "ルイ・ヴィトン モノグラム バッグ", "美品", "")

    if result == nil {
        t.Fatal("expected brand match from title")
    }
    if result.ID != "b_002" {
        t.Errorf("expected b_002, got %s", result.ID)
    }
    if result.Source != "rule_matched" {
        t.Errorf("expected rule_matched, got %s", result.Source)
    }
}

func TestPipeline_Level3_AIExtract(t *testing.T) {
    reg := NewRegistry()
    reg.Add(Entry{ID: "b_003", NameStd: "Prada", NameJA: "プラダ", Aliases: []string{"PRADA"}})

    ai := &mockAIExtractor{
        result: &AIExtractionResult{BrandName: "PRADA", Confidence: 0.85},
    }
    p := NewPipeline(reg, ai)
    rawData := map[string]interface{}{} // no brand field
    // title doesn't contain brand alias directly (uses obscure reference)
    result := p.Identify(rawData, "イタリア製 サフィアーノ バッグ", "プラダの最新コレクション", "")

    if result == nil {
        t.Fatal("expected brand from AI or rule match")
    }
}

func TestPipeline_NoBrand(t *testing.T) {
    reg := NewRegistry()
    p := NewPipeline(reg, nil)
    rawData := map[string]interface{}{}
    result := p.Identify(rawData, "ノーブランド バッグ", "", "")

    if result != nil {
        t.Errorf("expected nil for no-brand product, got %+v", result)
    }
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/brand/ -v -run TestPipeline`
Expected: FAIL

**Step 3: 实现流水线**

```go
// backend/internal/brand/pipeline.go
package brand

import "github.com/rakutao/collection-gateway/internal/domain"

// AIExtractionResult is the response from the Python AI brand extractor.
type AIExtractionResult struct {
    BrandName  string
    Confidence float64
}

// AIExtractor calls the Python AI service for brand extraction.
type AIExtractor interface {
    Extract(title, description, category string) (*AIExtractionResult, error)
}

// Pipeline runs the 3-level brand identification process.
type Pipeline struct {
    registry    *Registry
    aiExtractor AIExtractor // may be nil
}

// NewPipeline creates a brand identification pipeline.
func NewPipeline(registry *Registry, aiExtractor AIExtractor) *Pipeline {
    return &Pipeline{registry: registry, aiExtractor: aiExtractor}
}

const aiConfidenceThreshold = 0.7

// Identify runs through Level 1 → 2 → 3, returns on first hit.
func (p *Pipeline) Identify(rawData map[string]interface{}, title, description, category string) *domain.Brand {
    // Level 1: Platform field direct extraction
    if brand := p.level1PlatformField(rawData); brand != nil {
        return brand
    }

    // Level 2: Rule-based alias matching in title + description
    if brand := p.level2RuleMatch(title, description); brand != nil {
        return brand
    }

    // Level 3: AI extraction (if extractor is available)
    if brand := p.level3AIExtract(title, description, category); brand != nil {
        return brand
    }

    return nil
}

func (p *Pipeline) level1PlatformField(rawData map[string]interface{}) *domain.Brand {
    brandVal, ok := rawData["brand"]
    if !ok {
        return nil
    }
    brandStr, ok := brandVal.(string)
    if !ok || brandStr == "" {
        return nil
    }
    entry := p.registry.FindByAlias(brandStr)
    if entry == nil {
        return nil
    }
    return &domain.Brand{
        ID:     entry.ID,
        Name:   entry.NameStd,
        NameJA: entry.NameJA,
        Source: "platform_field",
    }
}

func (p *Pipeline) level2RuleMatch(title, description string) *domain.Brand {
    // Search in title first (higher priority)
    if entry := p.registry.MatchInText(title); entry != nil {
        return &domain.Brand{
            ID:     entry.ID,
            Name:   entry.NameStd,
            NameJA: entry.NameJA,
            Source: "rule_matched",
        }
    }
    // Then search in description
    if description != "" {
        if entry := p.registry.MatchInText(description); entry != nil {
            return &domain.Brand{
                ID:     entry.ID,
                Name:   entry.NameStd,
                NameJA: entry.NameJA,
                Source: "rule_matched",
            }
        }
    }
    return nil
}

func (p *Pipeline) level3AIExtract(title, description, category string) *domain.Brand {
    if p.aiExtractor == nil {
        return nil
    }
    result, err := p.aiExtractor.Extract(title, description, category)
    if err != nil || result == nil {
        return nil
    }
    if result.Confidence < aiConfidenceThreshold {
        return nil
    }
    // Try to match AI result against brand registry
    entry := p.registry.FindByAlias(result.BrandName)
    if entry != nil {
        return &domain.Brand{
            ID:     entry.ID,
            Name:   entry.NameStd,
            NameJA: entry.NameJA,
            Source: "ai_extracted",
        }
    }
    // AI found a brand not in registry — return with empty ID (pending review)
    return &domain.Brand{
        Name:   result.BrandName,
        Source: "ai_extracted",
    }
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/brand/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add backend/internal/brand/
git commit -m "feat: add 3-level brand identification pipeline (platform field → rule match → AI)"
```

---

## Phase 5: Python AI 微服务

### Task 11: Python 项目脚手架 + 翻译服务

**Files:**
- Create: `ai-service/pyproject.toml`
- Create: `ai-service/app/__init__.py`
- Create: `ai-service/app/main.py`
- Create: `ai-service/app/translation.py`
- Create: `ai-service/tests/test_translation.py`

**Step 1: 写测试**

```python
# ai-service/tests/test_translation.py
import pytest
from app.translation import TranslationService

@pytest.fixture
def svc():
    return TranslationService(brand_mappings={"gucci": "グッチ", "nike": "ナイキ"})

def test_detect_language_ja(svc):
    assert svc.detect_language("グッチ バッグ") == "ja"

def test_detect_language_en(svc):
    assert svc.detect_language("gucci bag") == "en"

def test_detect_language_zh(svc):
    assert svc.detect_language("包包 红色") == "zh"

def test_translate_brand_uses_mapping(svc):
    result = svc.translate_keyword("gucci bag", source_lang="en")
    # "gucci" should be mapped, not translated generically
    assert "グッチ" in result.keyword_ja

def test_translate_ja_passthrough(svc):
    result = svc.translate_keyword("グッチ バッグ", source_lang="ja")
    assert result.keyword_ja == "グッチ バッグ"
    assert result.was_translated is False
```

**Step 2: 创建 pyproject.toml**

```toml
# ai-service/pyproject.toml
[project]
name = "rakutao-ai-service"
version = "0.1.0"
requires-python = ">=3.12"
dependencies = [
    "fastapi>=0.115",
    "uvicorn>=0.34",
    "pydantic>=2.0",
]

[project.optional-dependencies]
dev = ["pytest>=8.0", "httpx>=0.28"]

[tool.pytest.ini_options]
testpaths = ["tests"]
```

**Step 3: 实现翻译服务**

```python
# ai-service/app/translation.py
from dataclasses import dataclass
import re

@dataclass
class TranslationResult:
    keyword_ja: str
    original: str
    source_lang: str
    was_translated: bool

class TranslationService:
    """Translates search keywords to Japanese with brand-aware mapping."""

    def __init__(self, brand_mappings: dict[str, str] | None = None):
        self.brand_mappings = {k.lower(): v for k, v in (brand_mappings or {}).items()}

    def detect_language(self, text: str) -> str:
        """Simple language detection based on character ranges."""
        # Check for Japanese characters (hiragana, katakana, kanji)
        if re.search(r'[\u3040-\u309F\u30A0-\u30FF\u4E00-\u9FFF]', text):
            # Distinguish Japanese from Chinese by checking for kana
            if re.search(r'[\u3040-\u309F\u30A0-\u30FF]', text):
                return "ja"
            return "zh"
        # Check for Chinese characters only
        if re.search(r'[\u4E00-\u9FFF]', text):
            return "zh"
        return "en"

    def translate_keyword(self, keyword: str, source_lang: str | None = None) -> TranslationResult:
        """Translate a search keyword to Japanese."""
        if source_lang is None:
            source_lang = self.detect_language(keyword)

        if source_lang == "ja":
            return TranslationResult(
                keyword_ja=keyword,
                original=keyword,
                source_lang="ja",
                was_translated=False,
            )

        # Split into tokens and process each
        tokens = keyword.split()
        ja_tokens = []
        for token in tokens:
            lower = token.lower()
            if lower in self.brand_mappings:
                # Brand name: use mapping, not generic translation
                ja_tokens.append(self.brand_mappings[lower])
            else:
                # TODO: call LLM for generic translation
                # For now, pass through (will be replaced with LLM call)
                ja_tokens.append(token)

        keyword_ja = " ".join(ja_tokens)
        return TranslationResult(
            keyword_ja=keyword_ja,
            original=keyword,
            source_lang=source_lang,
            was_translated=True,
        )
```

```python
# ai-service/app/__init__.py
```

```python
# ai-service/app/main.py
from fastapi import FastAPI
from app.translation import TranslationService

app = FastAPI(title="Rakutao AI Service")

translation_svc = TranslationService()

@app.get("/health")
def health():
    return {"status": "ok"}

@app.post("/translate")
def translate(keyword: str, source_lang: str | None = None):
    result = translation_svc.translate_keyword(keyword, source_lang)
    return {
        "keyword_ja": result.keyword_ja,
        "original": result.original,
        "source_lang": result.source_lang,
        "was_translated": result.was_translated,
    }
```

**Step 4: 运行测试确认通过**

Run: `cd ai-service && pip install -e ".[dev]" && pytest tests/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add ai-service/
git commit -m "feat: add Python AI service with TranslationService (brand-aware keyword translation)"
```

---

### Task 12: Python 品牌 AI 提取服务

**Files:**
- Create: `ai-service/app/brand_extractor.py`
- Create: `ai-service/tests/test_brand_extractor.py`

**Step 1: 写测试**

```python
# ai-service/tests/test_brand_extractor.py
import pytest
from app.brand_extractor import BrandExtractor

@pytest.fixture
def extractor():
    return BrandExtractor()

def test_extract_brand_from_title(extractor):
    result = extractor.extract(
        title="CHANEL シャネル マトラッセ チェーンショルダーバッグ",
        description="正規品です",
        category="バッグ",
    )
    assert result is not None
    assert result.brand_name in ("CHANEL", "シャネル")
    assert result.confidence > 0.0

def test_extract_no_brand(extractor):
    result = extractor.extract(
        title="ハンドメイド トートバッグ",
        description="手作りです",
        category="バッグ",
    )
    # Should return None or low confidence
    assert result is None or result.confidence < 0.5
```

**Step 2: 运行测试确认失败**

Run: `cd ai-service && pytest tests/test_brand_extractor.py -v`
Expected: FAIL

**Step 3: 实现品牌提取器**

```python
# ai-service/app/brand_extractor.py
from dataclasses import dataclass
import re

@dataclass
class ExtractionResult:
    brand_name: str
    confidence: float

# Known brand patterns (expandable, loaded from DB in production)
KNOWN_BRANDS = [
    "CHANEL", "シャネル", "GUCCI", "グッチ", "LOUIS VUITTON", "ルイ・ヴィトン",
    "PRADA", "プラダ", "HERMES", "エルメス", "BURBERRY", "バーバリー",
    "COACH", "コーチ", "NIKE", "ナイキ", "ADIDAS", "アディダス",
    "SUPREME", "シュプリーム", "BALENCIAGA", "バレンシアガ",
    "DIOR", "ディオール", "FENDI", "フェンディ", "CELINE", "セリーヌ",
]

class BrandExtractor:
    """Extracts brand names from product title and description.

    In production, this will call an LLM for complex cases.
    This implementation uses pattern matching as a baseline.
    """

    def __init__(self):
        # Build regex patterns (case-insensitive for latin)
        self.patterns = []
        for brand in KNOWN_BRANDS:
            self.patterns.append((brand, re.compile(re.escape(brand), re.IGNORECASE)))

    def extract(self, title: str, description: str, category: str) -> ExtractionResult | None:
        combined = f"{title} {description}"

        for brand_name, pattern in self.patterns:
            if pattern.search(combined):
                # Higher confidence if found in title vs description
                confidence = 0.95 if pattern.search(title) else 0.75
                return ExtractionResult(brand_name=brand_name, confidence=confidence)

        # TODO: Fall back to LLM extraction for unknown brands
        return None
```

**Step 4: 运行测试确认通过**

Run: `cd ai-service && pytest tests/ -v`
Expected: ALL PASS

**Step 5: 添加 API 端点并 commit**

在 `app/main.py` 中添加品牌提取端点：

```python
from app.brand_extractor import BrandExtractor

brand_extractor = BrandExtractor()

@app.post("/brand/extract")
def extract_brand(title: str, description: str = "", category: str = ""):
    result = brand_extractor.extract(title, description, category)
    if result is None:
        return {"brand_name": None, "confidence": 0.0}
    return {"brand_name": result.brand_name, "confidence": result.confidence}
```

```bash
git add ai-service/
git commit -m "feat: add BrandExtractor — pattern-based brand identification with LLM fallback stub"
```

---

## Phase 6: 搜索网关

### Task 13: ES 搜索客户端

**Files:**
- Create: `backend/internal/search/es_client.go`
- Create: `backend/internal/search/es_client_test.go`
- Create: `backend/internal/search/index.json` (ES mapping)

**Step 1: 保存 ES 索引 mapping 文件**

从设计文档复制 ES mapping JSON 到 `backend/internal/search/index.json`。

**Step 2: 写测试 — 测试查询构建逻辑（不需要真实 ES）**

```go
// backend/internal/search/es_client_test.go
package search

import (
    "testing"

    "github.com/rakutao/collection-gateway/internal/domain"
)

func TestBuildESQuery_GlobalSearch(t *testing.T) {
    q := domain.SearchQuery{
        Keyword:   "gucci bag",
        KeywordJA: "グッチ バッグ",
        UserLang:  "en",
        Page:      1,
        PageSize:  20,
    }
    body := buildESQuery(q)
    if body == nil {
        t.Fatal("expected non-nil query body")
    }
    // Verify it contains multi_match for ja + translated fields
    queryStr := string(body)
    if !contains(queryStr, "グッチ バッグ") {
        t.Error("expected Japanese keyword in query")
    }
}

func TestBuildESQuery_PlatformFilter(t *testing.T) {
    q := domain.SearchQuery{
        KeywordJA: "バッグ",
        Platforms: []string{"mercari"},
        Page:      1,
        PageSize:  20,
    }
    body := buildESQuery(q)
    queryStr := string(body)
    if !contains(queryStr, "mercari") {
        t.Error("expected platform filter in query")
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
    body := buildESQuery(q)
    queryStr := string(body)
    if !contains(queryStr, "price_jpy") {
        t.Error("expected price range filter in query")
    }
}

func contains(s, substr string) bool {
    return len(s) > 0 && len(substr) > 0 && // basic safety
        len(s) >= len(substr) &&
        stringContains(s, substr)
}

func stringContains(s, sub string) bool {
    for i := 0; i <= len(s)-len(sub); i++ {
        if s[i:i+len(sub)] == sub {
            return true
        }
    }
    return false
}
```

**Step 3: 实现 ES 查询构建**

```go
// backend/internal/search/es_client.go
package search

import (
    "encoding/json"

    "github.com/rakutao/collection-gateway/internal/domain"
)

// buildESQuery constructs an Elasticsearch query from a SearchQuery.
func buildESQuery(q domain.SearchQuery) []byte {
    must := []map[string]interface{}{}
    filter := []map[string]interface{}{}

    // Multi-match across title fields
    keyword := q.KeywordJA
    if keyword == "" {
        keyword = q.Keyword
    }
    if keyword != "" {
        must = append(must, map[string]interface{}{
            "multi_match": map[string]interface{}{
                "query":  keyword,
                "fields": []string{"title^3", "title_en^2", "title_zh_tw^2", "description", "brand_name_ja"},
                "type":   "best_fields",
            },
        })
    }

    // Platform filter
    if len(q.Platforms) > 0 {
        filter = append(filter, map[string]interface{}{
            "terms": map[string]interface{}{
                "source_platform": q.Platforms,
            },
        })
    }

    // Brand filter
    if q.BrandID != "" {
        filter = append(filter, map[string]interface{}{
            "term": map[string]interface{}{
                "brand_id": q.BrandID,
            },
        })
    }

    // Price range
    priceRange := map[string]interface{}{}
    if q.PriceMin > 0 {
        priceRange["gte"] = q.PriceMin
    }
    if q.PriceMax > 0 {
        priceRange["lte"] = q.PriceMax
    }
    if len(priceRange) > 0 {
        filter = append(filter, map[string]interface{}{
            "range": map[string]interface{}{
                "price_jpy": priceRange,
            },
        })
    }

    // Condition filter
    if len(q.Condition) > 0 {
        filter = append(filter, map[string]interface{}{
            "terms": map[string]interface{}{
                "condition": q.Condition,
            },
        })
    }

    // Only available products
    filter = append(filter, map[string]interface{}{
        "term": map[string]interface{}{
            "status": domain.StatusAvailable,
        },
    })

    // Categories
    if len(q.Categories) > 0 {
        filter = append(filter, map[string]interface{}{
            "terms": map[string]interface{}{
                "categories": q.Categories,
            },
        })
    }

    // Sort
    sortClause := []map[string]interface{}{}
    switch q.SortBy {
    case "price_asc":
        sortClause = append(sortClause, map[string]interface{}{"price_jpy": "asc"})
    case "price_desc":
        sortClause = append(sortClause, map[string]interface{}{"price_jpy": "desc"})
    case "newest":
        sortClause = append(sortClause, map[string]interface{}{"listed_at": "desc"})
    default:
        sortClause = append(sortClause, map[string]interface{}{"_score": "desc"})
    }

    // Aggregations
    aggs := map[string]interface{}{
        "platforms":  map[string]interface{}{"terms": map[string]interface{}{"field": "source_platform", "size": 20}},
        "brands":     map[string]interface{}{"terms": map[string]interface{}{"field": "brand_name", "size": 50}},
        "categories": map[string]interface{}{"terms": map[string]interface{}{"field": "categories", "size": 30}},
        "price_stats": map[string]interface{}{"stats": map[string]interface{}{"field": "price_jpy"}},
    }

    // Pagination
    from := 0
    if q.Page > 1 {
        from = (q.Page - 1) * q.PageSize
    }
    size := q.PageSize
    if size <= 0 {
        size = 20
    }

    query := map[string]interface{}{
        "query": map[string]interface{}{
            "bool": map[string]interface{}{
                "must":   must,
                "filter": filter,
            },
        },
        "sort": sortClause,
        "aggs": aggs,
        "from": from,
        "size": size,
    }

    body, _ := json.Marshal(query)
    return body
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/search/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add backend/internal/search/
git commit -m "feat: add ES query builder with multi-match, platform/brand/price filters, aggregations"
```

---

### Task 14: Search Gateway — 混合搜索协调器

**Files:**
- Create: `backend/internal/search/gateway.go`
- Create: `backend/internal/search/gateway_test.go`

**Step 1: 写测试**

```go
// backend/internal/search/gateway_test.go
package search

import (
    "context"
    "testing"

    "github.com/rakutao/collection-gateway/internal/domain"
)

type mockTranslator struct {
    keywordJA string
}

func (m *mockTranslator) Translate(ctx context.Context, keyword, lang string) (string, error) {
    return m.keywordJA, nil
}

func TestGateway_TranslatesKeyword(t *testing.T) {
    g := &Gateway{
        translator: &mockTranslator{keywordJA: "グッチ バッグ"},
    }
    q := domain.SearchQuery{Keyword: "gucci bag", UserLang: "en"}
    translated, err := g.prepareQuery(context.Background(), q)
    if err != nil {
        t.Fatal(err)
    }
    if translated.KeywordJA != "グッチ バッグ" {
        t.Errorf("expected グッチ バッグ, got %s", translated.KeywordJA)
    }
}

func TestGateway_JapaneseKeyword_NoTranslation(t *testing.T) {
    g := &Gateway{
        translator: &mockTranslator{},
    }
    q := domain.SearchQuery{Keyword: "グッチ バッグ", UserLang: "ja"}
    translated, err := g.prepareQuery(context.Background(), q)
    if err != nil {
        t.Fatal(err)
    }
    if translated.KeywordJA != "グッチ バッグ" {
        t.Errorf("expected keyword to pass through, got %s", translated.KeywordJA)
    }
}
```

**Step 2: 运行测试确认失败**

Run: `cd backend && go test ./internal/search/ -v -run TestGateway`
Expected: FAIL

**Step 3: 实现 Gateway**

```go
// backend/internal/search/gateway.go
package search

import (
    "context"
    "regexp"

    "github.com/rakutao/collection-gateway/internal/domain"
)

var jaPattern = regexp.MustCompile(`[\x{3040}-\x{309F}\x{30A0}-\x{30FF}]`)

// Translator translates keywords to Japanese.
type Translator interface {
    Translate(ctx context.Context, keyword, sourceLang string) (string, error)
}

// Gateway coordinates ES cache search and realtime proxy search.
type Gateway struct {
    translator Translator
    // TODO: add ES client, registry for realtime search
}

// NewGateway creates a search gateway.
func NewGateway(translator Translator) *Gateway {
    return &Gateway{translator: translator}
}

// prepareQuery detects language and translates keyword to Japanese if needed.
func (g *Gateway) prepareQuery(ctx context.Context, q domain.SearchQuery) (domain.SearchQuery, error) {
    // If keyword is already Japanese, use as-is
    if isJapanese(q.Keyword) {
        q.KeywordJA = q.Keyword
        return q, nil
    }

    // Translate to Japanese
    if g.translator != nil {
        ja, err := g.translator.Translate(ctx, q.Keyword, q.UserLang)
        if err != nil {
            // Fallback: use original keyword
            q.KeywordJA = q.Keyword
            return q, nil
        }
        q.KeywordJA = ja
    } else {
        q.KeywordJA = q.Keyword
    }

    return q, nil
}

func isJapanese(s string) bool {
    return jaPattern.MatchString(s)
}
```

**Step 4: 运行测试确认通过**

Run: `cd backend && go test ./internal/search/ -v`
Expected: ALL PASS

**Step 5: Commit**

```bash
git add backend/internal/search/
git commit -m "feat: add SearchGateway with language detection and keyword translation"
```

---

## Phase 7: Docker Compose + 集成

### Task 15: Docker Compose 开发环境

**Files:**
- Create: `docker-compose.yml`
- Create: `backend/Dockerfile`
- Create: `ai-service/Dockerfile`

**Step 1: 创建 docker-compose.yml**

```yaml
# docker-compose.yml
services:
  gateway:
    build: ./backend
    ports:
      - "8080:8080"
    depends_on:
      - elasticsearch
      - redis
      - postgres
      - ai-service
    environment:
      - ES_URL=http://elasticsearch:9200
      - REDIS_URL=redis://redis:6379
      - POSTGRES_URL=postgres://rakutao:rakutao@postgres:5432/rakutao?sslmode=disable
      - AI_SERVICE_URL=http://ai-service:8000

  ai-service:
    build: ./ai-service
    ports:
      - "8000:8000"

  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.17.0
    ports:
      - "9200:9200"
    environment:
      - discovery.type=single-node
      - xpack.security.enabled=false
      - "ES_JAVA_OPTS=-Xms512m -Xmx512m"
    volumes:
      - es-data:/usr/share/elasticsearch/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_USER=rakutao
      - POSTGRES_PASSWORD=rakutao
      - POSTGRES_DB=rakutao
    volumes:
      - pg-data:/var/lib/postgresql/data

volumes:
  es-data:
  pg-data:
```

**Step 2: 创建 backend Dockerfile**

```dockerfile
# backend/Dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /gateway ./cmd/gateway

FROM alpine:3.19
COPY --from=builder /gateway /gateway
EXPOSE 8080
CMD ["/gateway"]
```

**Step 3: 创建 ai-service Dockerfile**

```dockerfile
# ai-service/Dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY pyproject.toml .
RUN pip install --no-cache-dir .
COPY . .
EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

**Step 4: 验证 docker-compose config 合法**

Run: `docker-compose config`
Expected: 输出合法的 YAML 配置，无错误

**Step 5: Commit**

```bash
git add docker-compose.yml backend/Dockerfile ai-service/Dockerfile
git commit -m "feat: add Docker Compose dev environment (ES, Redis, Postgres, Go gateway, Python AI)"
```

---

### Task 16: ES 索引初始化脚本

**Files:**
- Create: `backend/scripts/init-es-index.sh`
- Modify: `backend/internal/search/index.json` (已在 Task 13 引用)

**Step 1: 将设计文档的 ES mapping 保存为文件**

将 `docs/plans/2026-02-25-collection-search-system-design.md` 中 §4.3 的 JSON mapping 保存到 `backend/internal/search/index.json`。

**Step 2: 创建索引初始化脚本**

```bash
#!/bin/bash
# backend/scripts/init-es-index.sh
# Creates the products index with kuromoji analyzer

ES_URL="${ES_URL:-http://localhost:9200}"
INDEX_NAME="${INDEX_NAME:-rakutao_products}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MAPPING_FILE="$SCRIPT_DIR/../internal/search/index.json"

echo "Creating index $INDEX_NAME on $ES_URL..."

# Install analysis-icu and analysis-kuromoji plugins if not present
# (These should be pre-installed in the ES Docker image for production)

curl -s -X PUT "$ES_URL/$INDEX_NAME" \
  -H "Content-Type: application/json" \
  -d @"$MAPPING_FILE"

echo ""
echo "Index $INDEX_NAME created."
```

**Step 3: 验证脚本语法**

Run: `bash -n backend/scripts/init-es-index.sh`
Expected: 无输出（语法正确）

**Step 4: Commit**

```bash
chmod +x backend/scripts/init-es-index.sh
git add backend/scripts/ backend/internal/search/index.json
git commit -m "feat: add ES index initialization script with kuromoji + ICU analyzers"
```

---

## Phase 8: 全量测试 + 项目结构验证

### Task 17: 全量测试通过

**Step 1: Go 全量测试**

Run: `cd backend && go test ./... -v -race`
Expected: ALL PASS

**Step 2: Python 全量测试**

Run: `cd ai-service && pytest tests/ -v`
Expected: ALL PASS

**Step 3: 验证项目结构**

Run: `find backend ai-service -type f -name "*.go" -o -name "*.py" | sort`
Expected: 输出完整的文件树，与设计文档对应

**Step 4: Final commit**

```bash
git add -A
git commit -m "chore: verify all tests pass, project structure complete"
```

---

## 项目结构总览

```
backend/
├── cmd/gateway/main.go              # Go 入口
├── internal/
│   ├── domain/                       # 领域模型
│   │   ├── product.go               # UnifiedProduct, Brand, SellerInfo, Variant
│   │   ├── adapter.go               # PlatformAdapter, AdapterCaps, RawProduct
│   │   ├── search.go                # SearchQuery, SearchResponse, ProductSummary
│   │   └── *_test.go
│   ├── registry/                     # 平台注册中心
│   │   ├── registry.go
│   │   └── registry_test.go
│   ├── adapter/                      # Adapter 实现
│   │   └── domestic/                 # 类型 A: 国内版代理
│   │       ├── adapter.go
│   │       └── adapter_test.go
│   ├── normalizer/                   # 数据标准化
│   │   ├── normalizer.go
│   │   └── normalizer_test.go
│   ├── output/                       # 输出路由
│   │   ├── router.go                # Router + Sink 接口
│   │   ├── es_sink.go               # ES sink (stub)
│   │   ├── redis_sink.go            # Redis sink (stub)
│   │   └── router_test.go
│   ├── brand/                        # 品牌识别
│   │   ├── registry.go              # 品牌库
│   │   ├── pipeline.go              # 三级识别流水线
│   │   └── *_test.go
│   └── search/                       # 搜索网关
│       ├── es_client.go             # ES 查询构建
│       ├── gateway.go               # 混合搜索协调
│       ├── index.json               # ES mapping
│       └── *_test.go
├── scripts/init-es-index.sh
├── Dockerfile
├── Makefile
└── go.mod

ai-service/
├── app/
│   ├── __init__.py
│   ├── main.py                      # FastAPI 入口
│   ├── translation.py               # 翻译服务 (品牌感知)
│   └── brand_extractor.py           # AI 品牌提取
├── tests/
│   ├── test_translation.py
│   └── test_brand_extractor.py
├── Dockerfile
└── pyproject.toml

docker-compose.yml                    # 本地开发环境
```

---

## 依赖关系图

```
Task 1 (Go 脚手架)
  └→ Task 2 (Product Schema)
      └→ Task 3 (Adapter 类型)
          ├→ Task 4 (Search 类型)
          └→ Task 5 (Platform Registry)
              └→ Task 6 (Domestic Adapter)
                  └→ Task 7 (Normalizer)
                      └→ Task 8 (Output Router)

Task 9 (Brand Registry)
  └→ Task 10 (Brand Pipeline) — 依赖 Task 2

Task 11 (Python 翻译) — 独立
Task 12 (Python 品牌提取) — 依赖 Task 11

Task 13 (ES 查询) — 依赖 Task 4
  └→ Task 14 (Search Gateway) — 依赖 Task 5, 13

Task 15 (Docker Compose) — 依赖 Task 1, 11
Task 16 (ES 索引脚本) — 依赖 Task 13
Task 17 (全量验证) — 依赖所有
```
