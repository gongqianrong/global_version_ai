# Platform Adapters Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build Yahoo Auction and Surugaya platform adapters (proxying to domestic APIs) and connect them to the API layer via PlatformSearchService for real-time search.

**Architecture:** Each platform gets its own package under `adapter/` with a client (HTTP calls + JSON parsing) and an adapter (implements `domain.PlatformAdapter`). A new `PlatformSearchService` in the `api` package bridges adapters to the WebSocket real-time handler by implementing `PlatformSearcher` and `PlatformLister` interfaces. The existing `domestic` adapter serves as the reference pattern.

**Tech Stack:** Go 1.22, net/http, httptest (testing), existing normalizer + registry packages

**Go binary:** `$HOME/go-sdk/go/bin/go`
**Project root:** `/Users/gongqianrong/Desktop/ai/backend`
**Run tests:** `export PATH="$HOME/go-sdk/go/bin:$PATH" && cd /Users/gongqianrong/Desktop/ai/backend && go test ./... -v -race`

---

## Task 1: Yahoo Auction Adapter

**Files:**
- Create: `backend/internal/adapter/yahoo_auction/client.go`
- Create: `backend/internal/adapter/yahoo_auction/adapter.go`
- Create: `backend/internal/adapter/yahoo_auction/adapter_test.go`

### client.go

```go
package yahoo_auction

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// searchResponse is the JSON shape returned by the Yahoo Auction domestic API search endpoint.
type searchResponse struct {
	Items []item `json:"items"`
	Total int64  `json:"total"`
}

// item is a single product returned by the Yahoo Auction domestic API.
type item struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Price       float64  `json:"price"`
	Description string   `json:"description,omitempty"`
	Images      []string `json:"images,omitempty"`
	Status      string   `json:"status,omitempty"`
	Condition   string   `json:"condition,omitempty"`
	Category    string   `json:"category,omitempty"`
}

// productResponse is the JSON shape returned by the Yahoo Auction domestic API product endpoint.
type productResponse struct {
	Item item `json:"item"`
}

// Client handles HTTP communication with the Yahoo Auction domestic API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a Client for the Yahoo Auction domestic API.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{baseURL: baseURL, http: httpClient}
}

// Search calls the domestic API search endpoint.
func (c *Client) Search(ctx context.Context, keyword string, page, pageSize int) (*searchResponse, error) {
	u, err := url.Parse(c.baseURL + "/api/search")
	if err != nil {
		return nil, fmt.Errorf("yahoo_auction: parse URL: %w", err)
	}

	q := u.Query()
	q.Set("keyword", keyword)
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo_auction: create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo_auction: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo_auction: unexpected status %d", resp.StatusCode)
	}

	var body searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("yahoo_auction: decode search response: %w", err)
	}
	return &body, nil
}

// GetProduct calls the domestic API product detail endpoint.
func (c *Client) GetProduct(ctx context.Context, productID string) (*item, error) {
	reqURL := c.baseURL + "/api/product/" + productID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo_auction: create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo_auction: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo_auction: unexpected status %d", resp.StatusCode)
	}

	var body productResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("yahoo_auction: decode product response: %w", err)
	}
	return &body.Item, nil
}

// Health calls the domestic API health endpoint.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("yahoo_auction: create health request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("yahoo_auction: health request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("yahoo_auction: health returned status %d", resp.StatusCode)
	}
	return nil
}
```

### adapter.go

```go
package yahoo_auction

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// Adapter implements domain.PlatformAdapter for Yahoo Auction Japan,
// proxying requests to a domestic API backend.
type Adapter struct {
	client *Client
}

// New creates a Yahoo Auction adapter that proxies to the given domestic API URL.
func New(baseURL string, httpClient *http.Client) *Adapter {
	return &Adapter{
		client: NewClient(baseURL, httpClient),
	}
}

// PlatformID returns "yahoo_auction".
func (a *Adapter) PlatformID() string {
	return "yahoo_auction"
}

// Capabilities returns the adapter's declared capabilities.
func (a *Adapter) Capabilities() domain.AdapterCaps {
	return domain.AdapterCaps{
		SupportsSearch:       true,
		SupportsRealtime:     true,
		SupportsBatchCollect: false,
		HasBrandField:        false,
		HasCategoryField:     true,
		MaxQPS:               10,
	}
}

// Search performs a keyword search via the domestic API and returns raw products.
func (a *Adapter) Search(ctx context.Context, query domain.SearchQuery) (*domain.SearchResult, error) {
	keyword := query.KeywordJA
	if keyword == "" {
		keyword = query.Keyword
	}

	resp, err := a.client.Search(ctx, keyword, query.Page, query.PageSize)
	if err != nil {
		return nil, err
	}

	products := make([]domain.RawProduct, 0, len(resp.Items))
	for _, it := range resp.Items {
		products = append(products, itemToRawProduct(it))
	}

	hasMore := int64(query.Page*query.PageSize) < resp.Total

	return &domain.SearchResult{
		Products: products,
		Total:    resp.Total,
		HasMore:  hasMore,
	}, nil
}

// GetProduct fetches a single product by ID via the domestic API.
func (a *Adapter) GetProduct(ctx context.Context, productID string) (*domain.RawProduct, error) {
	it, err := a.client.GetProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	rp := itemToRawProduct(*it)
	return &rp, nil
}

// BatchCollect is not implemented.
func (a *Adapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) {
	return nil, fmt.Errorf("yahoo_auction: BatchCollect not implemented")
}

// HealthCheck checks the domestic API health endpoint.
func (a *Adapter) HealthCheck(ctx context.Context) domain.HealthStatus {
	if err := a.client.Health(ctx); err != nil {
		return domain.HealthStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}
	return domain.HealthStatus{
		Status:  "healthy",
		Message: "yahoo_auction domestic API is reachable",
	}
}

// itemToRawProduct converts a domestic API item to a domain.RawProduct.
func itemToRawProduct(it item) domain.RawProduct {
	rawData := map[string]interface{}{
		"title": it.Title,
		"price": it.Price,
	}
	if it.Description != "" {
		rawData["description"] = it.Description
	}
	if len(it.Images) > 0 {
		rawData["images"] = it.Images
	}
	if it.Status != "" {
		rawData["status"] = it.Status
	}
	if it.Condition != "" {
		rawData["condition"] = it.Condition
	}
	if it.Category != "" {
		rawData["category"] = it.Category
	}
	return domain.RawProduct{
		Platform: "yahoo_auction",
		RawID:    it.ID,
		RawData:  rawData,
	}
}
```

### adapter_test.go

```go
package yahoo_auction

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

func TestPlatformID(t *testing.T) {
	a := New("http://localhost:9999", nil)
	if got := a.PlatformID(); got != "yahoo_auction" {
		t.Errorf("PlatformID() = %q, want %q", got, "yahoo_auction")
	}
}

func TestCapabilities(t *testing.T) {
	a := New("http://localhost:9999", nil)
	caps := a.Capabilities()
	if !caps.SupportsSearch {
		t.Error("expected SupportsSearch = true")
	}
	if !caps.SupportsRealtime {
		t.Error("expected SupportsRealtime = true")
	}
	if !caps.HasCategoryField {
		t.Error("expected HasCategoryField = true")
	}
}

func TestSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		q := r.URL.Query()
		if got := q.Get("keyword"); got != "グッチ" {
			t.Errorf("keyword = %q, want %q", got, "グッチ")
		}
		if got := q.Get("page"); got != "1" {
			t.Errorf("page = %q, want %q", got, "1")
		}

		resp := searchResponse{
			Items: []item{
				{ID: "ya-001", Title: "グッチ バッグ", Price: 15000, Status: "on_sale", Condition: "good", Category: "バッグ"},
				{ID: "ya-002", Title: "グッチ 財布", Price: 8000, Images: []string{"img1.jpg"}},
			},
			Total: 25,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	result, err := a.Search(context.Background(), domain.SearchQuery{
		Keyword:   "gucci",
		KeywordJA: "グッチ",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if result.Total != 25 {
		t.Errorf("Total = %d, want 25", result.Total)
	}
	if len(result.Products) != 2 {
		t.Fatalf("len(Products) = %d, want 2", len(result.Products))
	}

	p0 := result.Products[0]
	if p0.Platform != "yahoo_auction" {
		t.Errorf("Platform = %q, want %q", p0.Platform, "yahoo_auction")
	}
	if p0.RawID != "ya-001" {
		t.Errorf("RawID = %q, want %q", p0.RawID, "ya-001")
	}
	if p0.RawData["title"] != "グッチ バッグ" {
		t.Errorf("RawData[title] = %v", p0.RawData["title"])
	}
	if p0.RawData["category"] != "バッグ" {
		t.Errorf("RawData[category] = %v", p0.RawData["category"])
	}
	if !result.HasMore {
		t.Error("HasMore = false, want true (25 > 1*20)")
	}
}

func TestSearch_UsesKeywordJA(t *testing.T) {
	var receivedKeyword string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKeyword = r.URL.Query().Get("keyword")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{})
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	a.Search(context.Background(), domain.SearchQuery{
		Keyword:   "gucci",
		KeywordJA: "グッチ",
		Page:      1,
		PageSize:  20,
	})

	if receivedKeyword != "グッチ" {
		t.Errorf("sent keyword = %q, want %q (should prefer KeywordJA)", receivedKeyword, "グッチ")
	}
}

func TestSearch_FallbackToKeyword(t *testing.T) {
	var receivedKeyword string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedKeyword = r.URL.Query().Get("keyword")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(searchResponse{})
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	a.Search(context.Background(), domain.SearchQuery{
		Keyword:  "グッチ",
		Page:     1,
		PageSize: 20,
	})

	if receivedKeyword != "グッチ" {
		t.Errorf("sent keyword = %q, want %q (should fallback to Keyword)", receivedKeyword, "グッチ")
	}
}

func TestSearch_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	_, err := a.Search(context.Background(), domain.SearchQuery{Keyword: "test", Page: 1, PageSize: 20})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGetProduct_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/product/ya-001" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		resp := productResponse{
			Item: item{
				ID:          "ya-001",
				Title:       "グッチ バッグ",
				Price:       15000,
				Description: "美品",
				Images:      []string{"img1.jpg", "img2.jpg"},
				Status:      "on_sale",
				Condition:   "good",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	rp, err := a.GetProduct(context.Background(), "ya-001")
	if err != nil {
		t.Fatalf("GetProduct error: %v", err)
	}

	if rp.RawID != "ya-001" {
		t.Errorf("RawID = %q, want %q", rp.RawID, "ya-001")
	}
	if rp.RawData["title"] != "グッチ バッグ" {
		t.Errorf("RawData[title] = %v", rp.RawData["title"])
	}
	if rp.RawData["description"] != "美品" {
		t.Errorf("RawData[description] = %v", rp.RawData["description"])
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	_, err := a.GetProduct(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestHealthCheck_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	status := a.HealthCheck(context.Background())
	if status.Status != "healthy" {
		t.Errorf("Status = %q, want %q", status.Status, "healthy")
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	status := a.HealthCheck(context.Background())
	if status.Status != "unhealthy" {
		t.Errorf("Status = %q, want %q", status.Status, "unhealthy")
	}
}
```

### Run tests

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/adapter/yahoo_auction/ -v -race
```

Expected: 10 tests PASS.

---

## Task 2: Surugaya Adapter

**Files:**
- Create: `backend/internal/adapter/surugaya/client.go`
- Create: `backend/internal/adapter/surugaya/adapter.go`
- Create: `backend/internal/adapter/surugaya/adapter_test.go`

### client.go

```go
package surugaya

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// searchResponse is the JSON shape returned by the Surugaya domestic API search endpoint.
// Note: Surugaya may use different field names/structure than Yahoo Auction.
// Update these structs when the actual API response format is provided.
type searchResponse struct {
	Items []item `json:"items"`
	Total int64  `json:"total"`
}

// item is a single product returned by the Surugaya domestic API.
type item struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Price       float64  `json:"price"`
	Description string   `json:"description,omitempty"`
	Images      []string `json:"images,omitempty"`
	Status      string   `json:"status,omitempty"`
	Condition   string   `json:"condition,omitempty"`
	Category    string   `json:"category,omitempty"`
}

// productResponse is the JSON shape returned by the Surugaya domestic API product endpoint.
type productResponse struct {
	Item item `json:"item"`
}

// Client handles HTTP communication with the Surugaya domestic API.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient creates a Client for the Surugaya domestic API.
func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	return &Client{baseURL: baseURL, http: httpClient}
}

// Search calls the domestic API search endpoint.
func (c *Client) Search(ctx context.Context, keyword string, page, pageSize int) (*searchResponse, error) {
	u, err := url.Parse(c.baseURL + "/api/search")
	if err != nil {
		return nil, fmt.Errorf("surugaya: parse URL: %w", err)
	}

	q := u.Query()
	q.Set("keyword", keyword)
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("surugaya: create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("surugaya: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("surugaya: unexpected status %d", resp.StatusCode)
	}

	var body searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("surugaya: decode search response: %w", err)
	}
	return &body, nil
}

// GetProduct calls the domestic API product detail endpoint.
func (c *Client) GetProduct(ctx context.Context, productID string) (*item, error) {
	reqURL := c.baseURL + "/api/product/" + productID

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("surugaya: create request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("surugaya: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("surugaya: unexpected status %d", resp.StatusCode)
	}

	var body productResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("surugaya: decode product response: %w", err)
	}
	return &body.Item, nil
}

// Health calls the domestic API health endpoint.
func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("surugaya: create health request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("surugaya: health request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("surugaya: health returned status %d", resp.StatusCode)
	}
	return nil
}
```

### adapter.go

```go
package surugaya

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// Adapter implements domain.PlatformAdapter for Surugaya,
// proxying requests to a domestic API backend.
type Adapter struct {
	client *Client
}

// New creates a Surugaya adapter that proxies to the given domestic API URL.
func New(baseURL string, httpClient *http.Client) *Adapter {
	return &Adapter{
		client: NewClient(baseURL, httpClient),
	}
}

// PlatformID returns "surugaya".
func (a *Adapter) PlatformID() string {
	return "surugaya"
}

// Capabilities returns the adapter's declared capabilities.
func (a *Adapter) Capabilities() domain.AdapterCaps {
	return domain.AdapterCaps{
		SupportsSearch:       true,
		SupportsRealtime:     true,
		SupportsBatchCollect: false,
		HasBrandField:        false,
		HasCategoryField:     true,
		MaxQPS:               10,
	}
}

// Search performs a keyword search via the domestic API and returns raw products.
func (a *Adapter) Search(ctx context.Context, query domain.SearchQuery) (*domain.SearchResult, error) {
	keyword := query.KeywordJA
	if keyword == "" {
		keyword = query.Keyword
	}

	resp, err := a.client.Search(ctx, keyword, query.Page, query.PageSize)
	if err != nil {
		return nil, err
	}

	products := make([]domain.RawProduct, 0, len(resp.Items))
	for _, it := range resp.Items {
		products = append(products, itemToRawProduct(it))
	}

	hasMore := int64(query.Page*query.PageSize) < resp.Total

	return &domain.SearchResult{
		Products: products,
		Total:    resp.Total,
		HasMore:  hasMore,
	}, nil
}

// GetProduct fetches a single product by ID via the domestic API.
func (a *Adapter) GetProduct(ctx context.Context, productID string) (*domain.RawProduct, error) {
	it, err := a.client.GetProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	rp := itemToRawProduct(*it)
	return &rp, nil
}

// BatchCollect is not implemented.
func (a *Adapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) {
	return nil, fmt.Errorf("surugaya: BatchCollect not implemented")
}

// HealthCheck checks the domestic API health endpoint.
func (a *Adapter) HealthCheck(ctx context.Context) domain.HealthStatus {
	if err := a.client.Health(ctx); err != nil {
		return domain.HealthStatus{
			Status:  "unhealthy",
			Message: err.Error(),
		}
	}
	return domain.HealthStatus{
		Status:  "healthy",
		Message: "surugaya domestic API is reachable",
	}
}

// itemToRawProduct converts a domestic API item to a domain.RawProduct.
func itemToRawProduct(it item) domain.RawProduct {
	rawData := map[string]interface{}{
		"title": it.Title,
		"price": it.Price,
	}
	if it.Description != "" {
		rawData["description"] = it.Description
	}
	if len(it.Images) > 0 {
		rawData["images"] = it.Images
	}
	if it.Status != "" {
		rawData["status"] = it.Status
	}
	if it.Condition != "" {
		rawData["condition"] = it.Condition
	}
	if it.Category != "" {
		rawData["category"] = it.Category
	}
	return domain.RawProduct{
		Platform: "surugaya",
		RawID:    it.ID,
		RawData:  rawData,
	}
}
```

### adapter_test.go

```go
package surugaya

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

func TestPlatformID(t *testing.T) {
	a := New("http://localhost:9999", nil)
	if got := a.PlatformID(); got != "surugaya" {
		t.Errorf("PlatformID() = %q, want %q", got, "surugaya")
	}
}

func TestCapabilities(t *testing.T) {
	a := New("http://localhost:9999", nil)
	caps := a.Capabilities()
	if !caps.SupportsSearch {
		t.Error("expected SupportsSearch = true")
	}
	if !caps.SupportsRealtime {
		t.Error("expected SupportsRealtime = true")
	}
}

func TestSearch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		q := r.URL.Query()
		if got := q.Get("keyword"); got != "ガンダム" {
			t.Errorf("keyword = %q, want %q", got, "ガンダム")
		}

		resp := searchResponse{
			Items: []item{
				{ID: "sg-001", Title: "ガンダム プラモデル", Price: 3000, Category: "プラモデル", Condition: "new"},
				{ID: "sg-002", Title: "ガンダム フィギュア", Price: 5500, Images: []string{"fig1.jpg"}, Status: "on_sale"},
			},
			Total: 18,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	result, err := a.Search(context.Background(), domain.SearchQuery{
		Keyword:   "gundam",
		KeywordJA: "ガンダム",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}

	if result.Total != 18 {
		t.Errorf("Total = %d, want 18", result.Total)
	}
	if len(result.Products) != 2 {
		t.Fatalf("len(Products) = %d, want 2", len(result.Products))
	}

	p0 := result.Products[0]
	if p0.Platform != "surugaya" {
		t.Errorf("Platform = %q, want %q", p0.Platform, "surugaya")
	}
	if p0.RawID != "sg-001" {
		t.Errorf("RawID = %q, want %q", p0.RawID, "sg-001")
	}
	if p0.RawData["category"] != "プラモデル" {
		t.Errorf("RawData[category] = %v", p0.RawData["category"])
	}
	if !result.HasMore {
		t.Error("HasMore should be false (18 < 1*20)")
	}
}

func TestSearch_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	_, err := a.Search(context.Background(), domain.SearchQuery{Keyword: "test", Page: 1, PageSize: 20})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestGetProduct_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/product/sg-001" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		resp := productResponse{
			Item: item{
				ID:          "sg-001",
				Title:       "ガンダム プラモデル",
				Price:       3000,
				Description: "新品未開封",
				Condition:   "new",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	rp, err := a.GetProduct(context.Background(), "sg-001")
	if err != nil {
		t.Fatalf("GetProduct error: %v", err)
	}
	if rp.RawID != "sg-001" {
		t.Errorf("RawID = %q, want %q", rp.RawID, "sg-001")
	}
	if rp.RawData["condition"] != "new" {
		t.Errorf("RawData[condition] = %v", rp.RawData["condition"])
	}
}

func TestGetProduct_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	_, err := a.GetProduct(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestHealthCheck_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	status := a.HealthCheck(context.Background())
	if status.Status != "healthy" {
		t.Errorf("Status = %q, want %q", status.Status, "healthy")
	}
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	a := New(srv.URL, nil)
	status := a.HealthCheck(context.Background())
	if status.Status != "unhealthy" {
		t.Errorf("Status = %q, want %q", status.Status, "unhealthy")
	}
}
```

### Run tests

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/adapter/surugaya/ -v -race
```

Expected: 8 tests PASS.

---

## Task 3: PlatformSearchService

Bridges adapters + normalizer to the API layer's `PlatformSearcher` and `PlatformLister` interfaces.

**Files:**
- Create: `backend/internal/api/platform_service.go`
- Create: `backend/internal/api/platform_service_test.go`

### platform_service.go

```go
package api

import (
	"context"
	"fmt"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/normalizer"
	"github.com/rakutao/collection-gateway/internal/registry"
)

// PlatformSearchService implements PlatformSearcher and PlatformLister
// by delegating to the adapter registry and normalizer.
type PlatformSearchService struct {
	registry   *registry.Registry
	normalizer *normalizer.Normalizer
}

// NewPlatformSearchService creates a PlatformSearchService.
func NewPlatformSearchService(reg *registry.Registry, norm *normalizer.Normalizer) *PlatformSearchService {
	return &PlatformSearchService{
		registry:   reg,
		normalizer: norm,
	}
}

// SearchPlatform implements PlatformSearcher.
// It fetches the adapter from the registry, calls Search, normalizes results,
// and maps them to ProductSummary.
func (s *PlatformSearchService) SearchPlatform(ctx context.Context, platformID string, query domain.SearchQuery) ([]domain.ProductSummary, int64, error) {
	adapter, err := s.registry.GetAdapter(platformID)
	if err != nil {
		return nil, 0, fmt.Errorf("platform service: %w", err)
	}

	result, err := adapter.Search(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("platform service: search %s: %w", platformID, err)
	}

	summaries := make([]domain.ProductSummary, 0, len(result.Products))
	for _, raw := range result.Products {
		product, err := s.normalizer.Normalize(platformID, raw)
		if err != nil {
			continue // skip products that fail normalization
		}

		summary := domain.ProductSummary{
			ID:            product.ID,
			Title:         product.Title,
			TitleOriginal: product.Title,
			PriceJPY:      product.PriceJPY,
			Platform:      product.SourcePlatform,
			Status:        product.Status,
			Condition:     product.Condition,
		}
		if len(product.Images) > 0 {
			summary.Image = product.Images[0]
		}
		if product.Brand != nil {
			summary.Brand = product.Brand.Name
		}
		summaries = append(summaries, summary)
	}

	return summaries, result.Total, nil
}

// RealtimePlatformIDs implements PlatformLister.
// Returns the IDs of all registered platforms that support real-time search.
func (s *PlatformSearchService) RealtimePlatformIDs() []string {
	metas := s.registry.RealtimeSearchable()
	ids := make([]string, len(metas))
	for i, m := range metas {
		ids[i] = m.ID
	}
	return ids
}
```

### platform_service_test.go

```go
package api

import (
	"context"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/normalizer"
	"github.com/rakutao/collection-gateway/internal/registry"
)

// testAdapter is a minimal PlatformAdapter for testing PlatformSearchService.
type testAdapter struct {
	id     string
	result *domain.SearchResult
	err    error
}

func (a *testAdapter) PlatformID() string               { return a.id }
func (a *testAdapter) Capabilities() domain.AdapterCaps {
	return domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: true}
}
func (a *testAdapter) Search(_ context.Context, _ domain.SearchQuery) (*domain.SearchResult, error) {
	return a.result, a.err
}
func (a *testAdapter) GetProduct(_ context.Context, _ string) (*domain.RawProduct, error) {
	return nil, nil
}
func (a *testAdapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) {
	return nil, nil
}
func (a *testAdapter) HealthCheck(_ context.Context) domain.HealthStatus {
	return domain.HealthStatus{Status: "healthy"}
}

func TestPlatformSearchService_SearchPlatform(t *testing.T) {
	reg := registry.New()
	adapter := &testAdapter{
		id: "yahoo_auction",
		result: &domain.SearchResult{
			Products: []domain.RawProduct{
				{
					Platform: "yahoo_auction",
					RawID:    "ya-001",
					RawData: map[string]interface{}{
						"title": "グッチ バッグ",
						"price": float64(15000),
					},
				},
			},
			Total: 1,
		},
	}
	reg.Register(registry.PlatformMeta{
		ID:     "yahoo_auction",
		Name:   "ヤフオク",
		Status: registry.StatusActive,
		Caps:   adapter.Capabilities(),
	}, adapter)

	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	summaries, total, err := svc.SearchPlatform(context.Background(), "yahoo_auction", domain.SearchQuery{
		Keyword:  "gucci",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("SearchPlatform error: %v", err)
	}

	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %d, want 1", len(summaries))
	}
	if summaries[0].ID != "yahoo_auction_ya-001" {
		t.Errorf("ID = %q, want %q", summaries[0].ID, "yahoo_auction_ya-001")
	}
	if summaries[0].Title != "グッチ バッグ" {
		t.Errorf("Title = %q", summaries[0].Title)
	}
	// Price should include 10% service fee: 15000 + 1500 = 16500
	if summaries[0].PriceJPY != 16500 {
		t.Errorf("PriceJPY = %d, want 16500 (15000 + 10%% fee)", summaries[0].PriceJPY)
	}
	if summaries[0].Platform != "yahoo_auction" {
		t.Errorf("Platform = %q", summaries[0].Platform)
	}
}

func TestPlatformSearchService_SearchPlatform_NotFound(t *testing.T) {
	reg := registry.New()
	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	_, _, err := svc.SearchPlatform(context.Background(), "nonexistent", domain.SearchQuery{})
	if err == nil {
		t.Fatal("expected error for unregistered platform")
	}
}

func TestPlatformSearchService_SearchPlatform_SkipsBadProducts(t *testing.T) {
	reg := registry.New()
	adapter := &testAdapter{
		id: "yahoo_auction",
		result: &domain.SearchResult{
			Products: []domain.RawProduct{
				{Platform: "yahoo_auction", RawID: "good", RawData: map[string]interface{}{"title": "Valid", "price": float64(1000)}},
				{Platform: "yahoo_auction", RawID: "bad", RawData: map[string]interface{}{"title": ""}},  // missing title
				{Platform: "yahoo_auction", RawID: "good2", RawData: map[string]interface{}{"title": "Also Valid", "price": float64(2000)}},
			},
			Total: 3,
		},
	}
	reg.Register(registry.PlatformMeta{
		ID:     "yahoo_auction",
		Status: registry.StatusActive,
		Caps:   adapter.Capabilities(),
	}, adapter)

	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	summaries, _, err := svc.SearchPlatform(context.Background(), "yahoo_auction", domain.SearchQuery{Keyword: "test", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("len(summaries) = %d, want 2 (bad product should be skipped)", len(summaries))
	}
}

func TestPlatformSearchService_RealtimePlatformIDs(t *testing.T) {
	reg := registry.New()

	// Register two adapters: one realtime-capable, one not
	realtimeAdapter := &testAdapter{id: "yahoo_auction"}
	nonRealtimeAdapter := &testAdapter{id: "static_catalog"}

	reg.Register(registry.PlatformMeta{
		ID:     "yahoo_auction",
		Status: registry.StatusActive,
		Caps:   domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: true},
	}, realtimeAdapter)

	reg.Register(registry.PlatformMeta{
		ID:     "static_catalog",
		Status: registry.StatusActive,
		Caps:   domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: false},
	}, nonRealtimeAdapter)

	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	ids := svc.RealtimePlatformIDs()
	if len(ids) != 1 {
		t.Fatalf("len(ids) = %d, want 1", len(ids))
	}
	if ids[0] != "yahoo_auction" {
		t.Errorf("ids[0] = %q, want %q", ids[0], "yahoo_auction")
	}
}

func TestPlatformSearchService_RealtimePlatformIDs_Empty(t *testing.T) {
	reg := registry.New()
	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	ids := svc.RealtimePlatformIDs()
	if len(ids) != 0 {
		t.Errorf("len(ids) = %d, want 0", len(ids))
	}
}
```

### Run tests

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/api/ -v -race
```

Expected: 41 tests PASS (36 existing + 5 new).

---

## Task 4: Wire Adapters into main.go

Update `cmd/gateway/main.go` to register Yahoo Auction and Surugaya adapters, create PlatformSearchService, and inject into handlers.

**Files:**
- Modify: `backend/cmd/gateway/main.go`

### Updated main.go

```go
// backend/cmd/gateway/main.go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/rakutao/collection-gateway/internal/adapter/surugaya"
	yahoo "github.com/rakutao/collection-gateway/internal/adapter/yahoo_auction"
	"github.com/rakutao/collection-gateway/internal/api"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/filter"
	"github.com/rakutao/collection-gateway/internal/normalizer"
	"github.com/rakutao/collection-gateway/internal/registry"
	"github.com/rakutao/collection-gateway/internal/search"
)

func main() {
	// --- Build dependencies ---

	// Keyword filter (political keywords blacklist).
	keywordFilter := filter.NewKeywordFilter(
		[]string{"天安门", "六四", "法轮功"},
		[]string{"政治", "独立运动"},
	)

	// Search gateway (translator is nil — will be wired when AI service is ready).
	gateway := search.NewGateway(nil, keywordFilter)

	// Normalizer (brand extractor is nil — will be wired when brand pipeline is integrated).
	norm := normalizer.New(nil)

	// Platform registry.
	reg := registry.New()

	// Register Yahoo Auction adapter.
	yahooURL := os.Getenv("YAHOO_AUCTION_API_URL")
	if yahooURL == "" {
		yahooURL = "http://localhost:3001"
	}
	reg.Register(registry.PlatformMeta{
		ID:     "yahoo_auction",
		Name:   "ヤフオク",
		NameEN: "Yahoo Auctions",
		Type:   registry.TypeDomesticProxy,
		Status: registry.StatusActive,
		Caps: domain.AdapterCaps{
			SupportsSearch:   true,
			SupportsRealtime: true,
			HasCategoryField: true,
			MaxQPS:           10,
		},
	}, yahoo.New(yahooURL, nil))

	// Register Surugaya adapter.
	surugayaURL := os.Getenv("SURUGAYA_API_URL")
	if surugayaURL == "" {
		surugayaURL = "http://localhost:3002"
	}
	reg.Register(registry.PlatformMeta{
		ID:     "surugaya",
		Name:   "駿河屋",
		NameEN: "Surugaya",
		Type:   registry.TypeDomesticProxy,
		Status: registry.StatusActive,
		Caps: domain.AdapterCaps{
			SupportsSearch:   true,
			SupportsRealtime: true,
			HasCategoryField: true,
			MaxQPS:           10,
		},
	}, surugaya.New(surugayaURL, nil))

	// Platform search service (bridges adapters to API layer).
	platformService := api.NewPlatformSearchService(reg, norm)

	// Stream manager for real-time search.
	streamManager := api.NewStreamManager()
	defer streamManager.Stop()

	// --- Build handlers ---
	searchHandler := api.NewSearchHandler(gateway, nil, streamManager)
	realtimeHandler := api.NewRealtimeHandler(streamManager, platformService, platformService)
	productHandler := api.NewProductHandler(nil)
	healthHandler := api.NewHealthHandler(nil)

	// --- Build router ---
	router := api.NewRouter(api.RouterConfig{
		SearchHandler:   searchHandler,
		RealtimeHandler: realtimeHandler,
		ProductHandler:  productHandler,
		HealthHandler:   healthHandler,
	})

	// --- Start server ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("Rakutao Collection Gateway starting on %s", addr)
	log.Printf("  Yahoo Auction API: %s", yahooURL)
	log.Printf("  Surugaya API: %s", surugayaURL)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

### Verify build

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go build ./...
```

Expected: compiles without errors.

---

## Task 5: Full Verification

Run the complete test suite across all packages.

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go build ./... && go test ./... -v -race
```

Expected: 13 packages compile, all tests pass with `-race`:
- `adapter/domestic`: 3 tests
- `adapter/yahoo_auction`: 10 tests (new)
- `adapter/surugaya`: 8 tests (new)
- `api`: 41 tests (36 existing + 5 new)
- Plus all other existing packages

Also verify Python tests still pass:

```bash
cd /Users/gongqianrong/Desktop/ai/ai-service && python3 -m pytest tests/ -v
```

---

## Dependencies

```
Task 1 (Yahoo Auction) ─┐
                         ├─► Task 3 (PlatformSearchService)
Task 2 (Surugaya) ───────┘         │
                                   ▼
                         Task 4 (main.go wiring)
                                   │
                                   ▼
                         Task 5 (Full Verification)
```

Tasks 1 and 2 are independent and can be done in parallel.
Task 3 depends on the adapter packages existing (imports registry + normalizer, but tests use inline mock adapters).
Task 4 depends on Tasks 1-3.
Task 5 depends on Task 4.
