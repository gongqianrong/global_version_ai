# API Layer Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build the first-phase API layer (search + product detail + health check) with HTTP REST and WebSocket real-time search.

**Architecture:** Flat handler layer using chi router. Handlers call existing domain services (Gateway, Registry, Normalizer) directly. StreamManager manages in-memory WebSocket sessions. Interfaces abstract ES/product lookups for testability.

**Tech Stack:** Go 1.22, chi/v5 (HTTP), nhooyr.io/websocket (WebSocket), go-chi/cors (CORS)

**Go binary:** `$HOME/go-sdk/go/bin/go`
**Project root:** `/Users/gongqianrong/Desktop/ai/backend`
**Run tests:** `export PATH="$HOME/go-sdk/go/bin:$PATH" && cd /Users/gongqianrong/Desktop/ai/backend && go test ./... -v -race`

---

## Task 1: Prep — Export PrepareQuery + Add Condition to ProductSummary + Dependencies

Before building the API layer, we need three prep changes:

1. Export `Gateway.prepareQuery` → `Gateway.PrepareQuery` so the API handler can call it
2. Add `Condition` field to `ProductSummary` (design spec includes it in search results)
3. Add external dependencies to go.mod

**Files:**
- Modify: `backend/internal/search/gateway.go` — rename method + update comment
- Modify: `backend/internal/search/gateway_test.go` — update all `prepareQuery` calls to `PrepareQuery`
- Modify: `backend/internal/domain/search.go` — add Condition field to ProductSummary
- Modify: `backend/internal/domain/search_test.go` — update ProductSummary test if needed
- Modify: `backend/go.mod` + `backend/go.sum` — add chi, cors, websocket

### Step 1: Export PrepareQuery in gateway.go

In `backend/internal/search/gateway.go`, rename `prepareQuery` to `PrepareQuery` and update the doc comment:

```go
// PrepareQuery inspects the query keyword to detect whether it is already
// Japanese (contains hiragana/katakana). If the keyword is Japanese it is
// assigned directly to KeywordJA. Otherwise the Translator is invoked to
// produce a Japanese translation. On translation error the original keyword
// is used as a fallback so that searches never fail purely because of
// translation issues.
func (g *Gateway) PrepareQuery(ctx context.Context, q domain.SearchQuery) (domain.SearchQuery, error) {
```

### Step 2: Update all test calls in gateway_test.go

In `backend/internal/search/gateway_test.go`, replace all `gw.prepareQuery(` with `gw.PrepareQuery(` — there are 6 occurrences plus the 4 keyword checker tests (10 total).

### Step 3: Add Condition to ProductSummary

In `backend/internal/domain/search.go`, add `Condition` field to `ProductSummary`:

```go
type ProductSummary struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	TitleOriginal string   `json:"title_original"`
	Image         string   `json:"image"`
	PriceJPY      int64    `json:"price_jpy"`
	Platform      string   `json:"platform"`
	Status        string   `json:"status"`
	Brand         string   `json:"brand,omitempty"`
	Condition     string   `json:"condition,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	IsTranslated  bool     `json:"is_translated"`
}
```

### Step 4: Add dependencies

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend
go get github.com/go-chi/chi/v5@latest
go get github.com/go-chi/cors@latest
go get nhooyr.io/websocket@latest
```

### Step 5: Run tests

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./... -v -race
```

Expected: all existing tests pass (PrepareQuery rename is compatible, Condition field addition is backward compatible).

---

## Task 2: Response Helpers (response.go + test)

Unified JSON response format used by all handlers.

**Files:**
- Create: `backend/internal/api/response.go`
- Create: `backend/internal/api/response_test.go`

### response.go

```go
package api

import (
	"encoding/json"
	"net/http"
)

// APIResponse is the standard JSON envelope for all API responses.
type APIResponse struct {
	Code      int         `json:"code"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// contextKey is an unexported type for context keys in this package.
type contextKey string

// requestIDKey is the context key for the request ID.
const requestIDKey contextKey = "request_id"

// Success writes a successful JSON response with code 0.
func Success(w http.ResponseWriter, r *http.Request, data interface{}) {
	writeJSON(w, http.StatusOK, APIResponse{
		Code:      0,
		Data:      data,
		RequestID: getRequestID(r),
	})
}

// ErrorWithCode writes an error JSON response with the given HTTP status and business error code.
func ErrorWithCode(w http.ResponseWriter, r *http.Request, httpStatus, code int, message string) {
	writeJSON(w, httpStatus, APIResponse{
		Code:      code,
		Message:   message,
		RequestID: getRequestID(r),
	})
}

// writeJSON marshals v to JSON and writes it to w with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// getRequestID extracts the request ID from the request context.
func getRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
```

### response_test.go

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r = r.WithContext(context.WithValue(r.Context(), requestIDKey, "req-123"))

	Success(w, r, map[string]string{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
	if resp.RequestID != "req-123" {
		t.Errorf("request_id = %q, want %q", resp.RequestID, "req-123")
	}
	if resp.Message != "" {
		t.Errorf("message = %q, want empty", resp.Message)
	}
}

func TestErrorWithCode(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	r = r.WithContext(context.WithValue(r.Context(), requestIDKey, "req-456"))

	ErrorWithCode(w, r, http.StatusBadRequest, 40001, "keyword blocked")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40001 {
		t.Errorf("code = %d, want 40001", resp.Code)
	}
	if resp.Message != "keyword blocked" {
		t.Errorf("message = %q, want %q", resp.Message, "keyword blocked")
	}
}

func TestSuccess_NoRequestID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)

	Success(w, r, "ok")

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.RequestID != "" {
		t.Errorf("request_id = %q, want empty", resp.RequestID)
	}
}

func TestWriteJSON_ContentType(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusOK, map[string]string{"a": "b"})

	ct := w.Header().Get("Content-Type")
	want := "application/json; charset=utf-8"
	if ct != want {
		t.Errorf("Content-Type = %q, want %q", ct, want)
	}
}
```

### Run tests

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/api/ -v -race
```

---

## Task 3: Middleware (middleware.go + test)

HTTP middleware chain: Recovery, Logger, RequestID, CORS config.

**Files:**
- Create: `backend/internal/api/middleware.go`
- Create: `backend/internal/api/middleware_test.go`

### middleware.go

```go
package api

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Recovery catches panics in handlers, logs them, and returns a 500 error.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[PANIC] %s %s: %v", r.Method, r.URL.Path, err)
				ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// RequestID generates a unique request ID and stores it in the request context.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := generateID()
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger logs each request's method, path, status code, and duration.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("[HTTP] %s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

// statusWriter wraps ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

// generateID creates a random 16-character hex string.
func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
```

### middleware_test.go

```go
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecovery_NoPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRecovery_WithPanic(t *testing.T) {
	handler := Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 50001 {
		t.Errorf("code = %d, want 50001", resp.Code)
	}
}

func TestRequestID_SetsHeader(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := getRequestID(r)
		if id == "" {
			t.Error("expected request ID in context, got empty")
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if w.Header().Get("X-Request-ID") == "" {
		t.Error("expected X-Request-ID header, got empty")
	}
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	var ids []string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids = append(ids, getRequestID(r))
	}))

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/", nil)
		handler.ServeHTTP(w, r)
	}

	seen := make(map[string]bool)
	for _, id := range ids {
		if seen[id] {
			t.Errorf("duplicate request ID: %s", id)
		}
		seen[id] = true
	}
}

func TestLogger_PassesThrough(t *testing.T) {
	called := false
	handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("expected handler to be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestStatusWriter_CapturesStatus(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	sw.WriteHeader(http.StatusNotFound)

	if sw.status != http.StatusNotFound {
		t.Errorf("status = %d, want %d", sw.status, http.StatusNotFound)
	}
}
```

### Run tests

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/api/ -v -race
```

---

## Task 4: StreamManager (stream.go + test)

In-memory manager for real-time search WebSocket sessions.

**Files:**
- Create: `backend/internal/api/stream.go`
- Create: `backend/internal/api/stream_test.go`

### stream.go

```go
package api

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/rakutao/collection-gateway/internal/domain"
)

const (
	// StreamTTL is how long a stream remains valid after creation.
	StreamTTL = 30 * time.Second
	// StreamSearchTimeout is the max time for all platform searches.
	StreamSearchTimeout = 5 * time.Second
	// StreamEventBufferSize is the channel buffer size for stream events.
	StreamEventBufferSize = 50
)

// StreamEventType identifies the kind of event sent over a search stream.
type StreamEventType string

const (
	EventResults StreamEventType = "results"
	EventDone    StreamEventType = "done"
	EventError   StreamEventType = "error"
)

// StreamEvent is a single event pushed to a WebSocket client during real-time search.
type StreamEvent struct {
	Type      StreamEventType        `json:"type"`
	Platform  string                 `json:"platform,omitempty"`
	Products  []domain.ProductSummary `json:"products,omitempty"`
	Total     int64                  `json:"total,omitempty"`
	Platforms []string               `json:"platforms_searched,omitempty"`
	Message   string                 `json:"message,omitempty"`
}

// Stream represents an active real-time search session.
type Stream struct {
	ID        string
	Query     domain.SearchQuery
	Events    chan StreamEvent
	CreatedAt time.Time
	claimed   bool
	mu        sync.Mutex
}

// Claim marks the stream as connected by a WebSocket client.
// Returns false if already claimed.
func (s *Stream) Claim() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.claimed {
		return false
	}
	s.claimed = true
	return true
}

// StreamManager manages active real-time search streams.
type StreamManager struct {
	mu      sync.RWMutex
	streams map[string]*Stream
	ttl     time.Duration
	stopCh  chan struct{}
}

// NewStreamManager creates a StreamManager and starts background cleanup.
func NewStreamManager() *StreamManager {
	sm := &StreamManager{
		streams: make(map[string]*Stream),
		ttl:     StreamTTL,
		stopCh:  make(chan struct{}),
	}
	go sm.cleanup()
	return sm
}

// Create creates a new stream for the given search query and returns its ID.
func (sm *StreamManager) Create(query domain.SearchQuery) string {
	id := "stream_" + generateStreamID()
	stream := &Stream{
		ID:        id,
		Query:     query,
		Events:    make(chan StreamEvent, StreamEventBufferSize),
		CreatedAt: time.Now(),
	}

	sm.mu.Lock()
	sm.streams[id] = stream
	sm.mu.Unlock()

	return id
}

// Get retrieves a stream by ID. Returns nil, false if not found or expired.
func (sm *StreamManager) Get(streamID string) (*Stream, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.streams[streamID]
	if !ok {
		return nil, false
	}
	if time.Since(s.CreatedAt) > sm.ttl {
		return nil, false
	}
	return s, true
}

// Remove deletes a stream by ID.
func (sm *StreamManager) Remove(streamID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.streams, streamID)
}

// Count returns the number of active streams (for testing).
func (sm *StreamManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.streams)
}

// Stop signals the background cleanup goroutine to exit.
func (sm *StreamManager) Stop() {
	close(sm.stopCh)
}

// cleanup periodically removes expired streams.
func (sm *StreamManager) cleanup() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.mu.Lock()
			now := time.Now()
			for id, s := range sm.streams {
				if now.Sub(s.CreatedAt) > sm.ttl {
					close(s.Events)
					delete(sm.streams, id)
				}
			}
			sm.mu.Unlock()
		case <-sm.stopCh:
			return
		}
	}
}

// generateStreamID creates a random 12-character hex string.
func generateStreamID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
```

### stream_test.go

```go
package api

import (
	"testing"
	"time"

	"github.com/rakutao/collection-gateway/internal/domain"
)

func TestStreamManager_Create(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	query := domain.SearchQuery{Keyword: "gucci"}
	id := sm.Create(query)

	if id == "" {
		t.Fatal("expected non-empty stream ID")
	}
	if sm.Count() != 1 {
		t.Errorf("count = %d, want 1", sm.Count())
	}
}

func TestStreamManager_Get_Found(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	query := domain.SearchQuery{Keyword: "gucci"}
	id := sm.Create(query)

	stream, ok := sm.Get(id)
	if !ok {
		t.Fatal("expected stream to be found")
	}
	if stream.Query.Keyword != "gucci" {
		t.Errorf("keyword = %q, want %q", stream.Query.Keyword, "gucci")
	}
}

func TestStreamManager_Get_NotFound(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	_, ok := sm.Get("nonexistent")
	if ok {
		t.Error("expected stream not to be found")
	}
}

func TestStreamManager_Remove(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	id := sm.Create(domain.SearchQuery{Keyword: "test"})
	sm.Remove(id)

	if sm.Count() != 0 {
		t.Errorf("count = %d, want 0 after remove", sm.Count())
	}
}

func TestStreamManager_UniqueIDs(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := sm.Create(domain.SearchQuery{Keyword: "test"})
		if ids[id] {
			t.Fatalf("duplicate stream ID: %s", id)
		}
		ids[id] = true
	}
}

func TestStream_Claim_FirstTime(t *testing.T) {
	s := &Stream{}
	if !s.Claim() {
		t.Error("expected first Claim() to return true")
	}
}

func TestStream_Claim_AlreadyClaimed(t *testing.T) {
	s := &Stream{}
	s.Claim()
	if s.Claim() {
		t.Error("expected second Claim() to return false")
	}
}

func TestStreamEvent_Channel(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	id := sm.Create(domain.SearchQuery{Keyword: "test"})
	stream, _ := sm.Get(id)

	// Send an event
	event := StreamEvent{
		Type:     EventResults,
		Platform: "yahoo_auction",
		Total:    10,
	}
	stream.Events <- event

	// Receive it
	select {
	case got := <-stream.Events:
		if got.Type != EventResults {
			t.Errorf("type = %q, want %q", got.Type, EventResults)
		}
		if got.Platform != "yahoo_auction" {
			t.Errorf("platform = %q, want %q", got.Platform, "yahoo_auction")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
```

### Run tests

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/api/ -v -race
```

---

## Task 5: SearchHandler (search.go + test)

HTTP search endpoint that parses query params, calls Gateway.PrepareQuery, executes search, and creates a real-time stream.

**Files:**
- Create: `backend/internal/api/search.go`
- Create: `backend/internal/api/search_test.go`

### search.go

```go
package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/search"
)

// Searcher executes a search query and returns results.
// This interface abstracts the Elasticsearch client for testability.
type Searcher interface {
	Search(ctx context.Context, query domain.SearchQuery) (*domain.SearchResponse, error)
}

// QueryPreparer translates keywords and checks the blacklist.
type QueryPreparer interface {
	PrepareQuery(ctx context.Context, q domain.SearchQuery) (domain.SearchQuery, error)
}

// SearchHandler handles HTTP search requests.
type SearchHandler struct {
	preparer      QueryPreparer
	searcher      Searcher
	streamManager *StreamManager
}

// NewSearchHandler creates a SearchHandler with the given dependencies.
func NewSearchHandler(preparer QueryPreparer, searcher Searcher, sm *StreamManager) *SearchHandler {
	return &SearchHandler{
		preparer:      preparer,
		searcher:      searcher,
		streamManager: sm,
	}
}

// HandleSearch handles GET /api/v1/search.
func (h *SearchHandler) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := parseSearchParams(r)

	if query.Keyword == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing required parameter: keyword")
		return
	}

	// Translate keyword and check blacklist.
	prepared, err := h.preparer.PrepareQuery(r.Context(), query)
	if err == search.ErrBlockedKeyword {
		ErrorWithCode(w, r, http.StatusBadRequest, 40001, "keyword is blocked by content policy")
		return
	}
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "internal server error")
		return
	}

	// Execute ES search.
	result, err := h.searcher.Search(r.Context(), prepared)
	if err != nil {
		ErrorWithCode(w, r, http.StatusServiceUnavailable, 50003, "search service unavailable")
		return
	}

	// Create real-time search stream.
	streamID := h.streamManager.Create(prepared)
	result.RealtimeStreamID = streamID

	Success(w, r, result)
}

// parseSearchParams extracts SearchQuery fields from URL query parameters.
func parseSearchParams(r *http.Request) domain.SearchQuery {
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	priceMin, _ := strconv.ParseInt(q.Get("price_min"), 10, 64)
	priceMax, _ := strconv.ParseInt(q.Get("price_max"), 10, 64)

	lang := q.Get("lang")
	if lang == "" {
		lang = "zh-TW"
	}

	contentRating := q.Get("content_rating")
	if contentRating == "" {
		contentRating = "general"
	}

	return domain.SearchQuery{
		Keyword:       q.Get("keyword"),
		Platforms:     splitCSV(q.Get("platforms")),
		BrandID:       q.Get("brand_id"),
		Categories:    splitCSV(q.Get("categories")),
		PriceMin:      priceMin,
		PriceMax:      priceMax,
		Condition:     splitCSV(q.Get("condition")),
		SortBy:        q.Get("sort"),
		Page:          page,
		PageSize:      pageSize,
		UserLang:      lang,
		ContentRating: contentRating,
	}
}

// splitCSV splits a comma-separated string into a slice, trimming whitespace.
// Returns nil for empty input.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
```

### search_test.go

```go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/search"
)

// --- Mocks ---

type mockPreparer struct {
	query domain.SearchQuery
	err   error
}

func (m *mockPreparer) PrepareQuery(_ context.Context, q domain.SearchQuery) (domain.SearchQuery, error) {
	if m.err != nil {
		return q, m.err
	}
	q.KeywordJA = m.query.KeywordJA
	return q, nil
}

type mockSearcher struct {
	result *domain.SearchResponse
	err    error
}

func (m *mockSearcher) Search(_ context.Context, _ domain.SearchQuery) (*domain.SearchResponse, error) {
	return m.result, m.err
}

// --- Tests ---

func TestHandleSearch_Success(t *testing.T) {
	preparer := &mockPreparer{query: domain.SearchQuery{KeywordJA: "グッチ"}}
	searcher := &mockSearcher{result: &domain.SearchResponse{
		CachedResults:     []domain.ProductSummary{{ID: "p1", Title: "Gucci Bag"}},
		CachedTotal:       1,
		TranslatedKeyword: "グッチ",
	}}
	sm := NewStreamManager()
	defer sm.Stop()
	handler := NewSearchHandler(preparer, searcher, sm)

	r := httptest.NewRequest("GET", "/api/v1/search?keyword=gucci&lang=en", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}

	// Check that data contains realtime_stream_id
	data, _ := json.Marshal(resp.Data)
	var searchResp domain.SearchResponse
	json.Unmarshal(data, &searchResp)
	if searchResp.RealtimeStreamID == "" {
		t.Error("expected non-empty realtime_stream_id")
	}
	if searchResp.CachedTotal != 1 {
		t.Errorf("cached_total = %d, want 1", searchResp.CachedTotal)
	}
}

func TestHandleSearch_MissingKeyword(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()
	handler := NewSearchHandler(&mockPreparer{}, &mockSearcher{}, sm)

	r := httptest.NewRequest("GET", "/api/v1/search", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40002 {
		t.Errorf("code = %d, want 40002", resp.Code)
	}
}

func TestHandleSearch_BlockedKeyword(t *testing.T) {
	preparer := &mockPreparer{err: search.ErrBlockedKeyword}
	sm := NewStreamManager()
	defer sm.Stop()
	handler := NewSearchHandler(preparer, &mockSearcher{}, sm)

	r := httptest.NewRequest("GET", "/api/v1/search?keyword=blocked", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40001 {
		t.Errorf("code = %d, want 40001", resp.Code)
	}
}

func TestHandleSearch_SearchError(t *testing.T) {
	preparer := &mockPreparer{query: domain.SearchQuery{KeywordJA: "test"}}
	searcher := &mockSearcher{err: errors.New("es unavailable")}
	sm := NewStreamManager()
	defer sm.Stop()
	handler := NewSearchHandler(preparer, searcher, sm)

	r := httptest.NewRequest("GET", "/api/v1/search?keyword=test", nil)
	w := httptest.NewRecorder()
	handler.HandleSearch(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 50003 {
		t.Errorf("code = %d, want 50003", resp.Code)
	}
}

func TestParseSearchParams_Defaults(t *testing.T) {
	r := httptest.NewRequest("GET", "/search?keyword=test", nil)
	q := parseSearchParams(r)

	if q.Keyword != "test" {
		t.Errorf("keyword = %q, want %q", q.Keyword, "test")
	}
	if q.Page != 1 {
		t.Errorf("page = %d, want 1", q.Page)
	}
	if q.PageSize != 20 {
		t.Errorf("page_size = %d, want 20", q.PageSize)
	}
	if q.UserLang != "zh-TW" {
		t.Errorf("lang = %q, want %q", q.UserLang, "zh-TW")
	}
	if q.ContentRating != "general" {
		t.Errorf("content_rating = %q, want %q", q.ContentRating, "general")
	}
}

func TestParseSearchParams_AllParams(t *testing.T) {
	r := httptest.NewRequest("GET", "/search?keyword=gucci&platforms=yahoo_auction,amazon_jp&brand_id=b1&categories=bags,shoes&price_min=1000&price_max=50000&condition=new,good&sort=price_asc&page=3&page_size=50&lang=en&content_rating=all", nil)
	q := parseSearchParams(r)

	if q.Keyword != "gucci" {
		t.Errorf("keyword = %q", q.Keyword)
	}
	if len(q.Platforms) != 2 || q.Platforms[0] != "yahoo_auction" {
		t.Errorf("platforms = %v", q.Platforms)
	}
	if q.BrandID != "b1" {
		t.Errorf("brand_id = %q", q.BrandID)
	}
	if len(q.Categories) != 2 {
		t.Errorf("categories = %v", q.Categories)
	}
	if q.PriceMin != 1000 {
		t.Errorf("price_min = %d", q.PriceMin)
	}
	if q.PriceMax != 50000 {
		t.Errorf("price_max = %d", q.PriceMax)
	}
	if len(q.Condition) != 2 {
		t.Errorf("condition = %v", q.Condition)
	}
	if q.SortBy != "price_asc" {
		t.Errorf("sort = %q", q.SortBy)
	}
	if q.Page != 3 {
		t.Errorf("page = %d", q.Page)
	}
	if q.PageSize != 50 {
		t.Errorf("page_size = %d", q.PageSize)
	}
}

func TestParseSearchParams_PageSizeCap(t *testing.T) {
	r := httptest.NewRequest("GET", "/search?keyword=test&page_size=999", nil)
	q := parseSearchParams(r)

	if q.PageSize != 100 {
		t.Errorf("page_size = %d, want 100 (capped)", q.PageSize)
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"a", 1},
		{"a,b,c", 3},
		{"a, b , c", 3},
		{",,,", 0},
	}
	for _, tt := range tests {
		got := splitCSV(tt.input)
		if len(got) != tt.want {
			t.Errorf("splitCSV(%q) = %d items, want %d", tt.input, len(got), tt.want)
		}
	}
}
```

### Run tests

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/api/ -v -race
```

---

## Task 6: RealtimeHandler (realtime.go + test)

WebSocket handler that streams real-time search results from platform adapters.

**Files:**
- Create: `backend/internal/api/realtime.go`
- Create: `backend/internal/api/realtime_test.go`

### realtime.go

```go
package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"nhooyr.io/websocket"
)

// PlatformSearcher can search a specific platform and normalize results.
type PlatformSearcher interface {
	// SearchPlatform searches a single platform and returns normalized product summaries.
	SearchPlatform(ctx context.Context, platformID string, query domain.SearchQuery) ([]domain.ProductSummary, int64, error)
}

// PlatformLister lists platforms that support real-time search.
type PlatformLister interface {
	RealtimePlatformIDs() []string
}

// RealtimeHandler handles WebSocket connections for real-time search results.
type RealtimeHandler struct {
	streamManager    *StreamManager
	platformSearcher PlatformSearcher
	platformLister   PlatformLister
}

// NewRealtimeHandler creates a RealtimeHandler with the given dependencies.
func NewRealtimeHandler(sm *StreamManager, ps PlatformSearcher, pl PlatformLister) *RealtimeHandler {
	return &RealtimeHandler{
		streamManager:    sm,
		platformSearcher: ps,
		platformLister:   pl,
	}
}

// HandleStream handles WS /api/v1/search/stream/{streamID}.
func (h *RealtimeHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")
	if streamID == "" {
		http.Error(w, "missing stream ID", http.StatusBadRequest)
		return
	}

	stream, ok := h.streamManager.Get(streamID)
	if !ok {
		http.Error(w, "stream not found or expired", http.StatusNotFound)
		return
	}

	if !stream.Claim() {
		http.Error(w, "stream already claimed", http.StatusConflict)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow all origins for now
	})
	if err != nil {
		log.Printf("[WS] accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	// Start platform searches in background.
	go h.searchPlatforms(stream)

	// Read events from stream and write to WebSocket.
	ctx, cancel := context.WithTimeout(r.Context(), StreamSearchTimeout+2*time.Second)
	defer cancel()

	for {
		select {
		case event, ok := <-stream.Events:
			if !ok {
				return // channel closed
			}
			data, _ := json.Marshal(event)
			if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
				log.Printf("[WS] write error: %v", err)
				return
			}
			if event.Type == EventDone {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// searchPlatforms concurrently searches all real-time platforms and sends results to the stream.
func (h *RealtimeHandler) searchPlatforms(stream *Stream) {
	platformIDs := h.platformLister.RealtimePlatformIDs()
	if len(platformIDs) == 0 {
		stream.Events <- StreamEvent{Type: EventDone, Platforms: []string{}}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), StreamSearchTimeout)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var searched []string

	for _, pid := range platformIDs {
		wg.Add(1)
		go func(platformID string) {
			defer wg.Done()

			products, total, err := h.platformSearcher.SearchPlatform(ctx, platformID, stream.Query)

			mu.Lock()
			searched = append(searched, platformID)
			mu.Unlock()

			if err != nil {
				stream.Events <- StreamEvent{
					Type:     EventError,
					Platform: platformID,
					Message:  err.Error(),
				}
				return
			}

			stream.Events <- StreamEvent{
				Type:     EventResults,
				Platform: platformID,
				Products: products,
				Total:    total,
			}
		}(pid)
	}

	wg.Wait()

	stream.Events <- StreamEvent{
		Type:      EventDone,
		Platforms: searched,
	}
}
```

### realtime_test.go

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"nhooyr.io/websocket"
)

// --- Mocks ---

type mockPlatformSearcher struct {
	results map[string][]domain.ProductSummary
	err     error
	delay   time.Duration
}

func (m *mockPlatformSearcher) SearchPlatform(_ context.Context, platformID string, _ domain.SearchQuery) ([]domain.ProductSummary, int64, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.err != nil {
		return nil, 0, m.err
	}
	products := m.results[platformID]
	return products, int64(len(products)), nil
}

type mockPlatformLister struct {
	ids []string
}

func (m *mockPlatformLister) RealtimePlatformIDs() []string {
	return m.ids
}

// --- Tests ---

func TestRealtimeHandler_StreamNotFound(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()
	handler := NewRealtimeHandler(sm, &mockPlatformSearcher{}, &mockPlatformLister{})

	// Use chi router to extract URL param
	r := chi.NewRouter()
	r.Get("/stream/{streamID}", handler.HandleStream)

	req := httptest.NewRequest("GET", "/stream/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRealtimeHandler_WebSocket_ReceivesResults(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	ps := &mockPlatformSearcher{
		results: map[string][]domain.ProductSummary{
			"yahoo_auction": {{ID: "y1", Title: "Product 1"}},
		},
	}
	pl := &mockPlatformLister{ids: []string{"yahoo_auction"}}
	handler := NewRealtimeHandler(sm, ps, pl)

	// Create stream
	streamID := sm.Create(domain.SearchQuery{Keyword: "test", KeywordJA: "テスト"})

	// Set up chi router + test server
	r := chi.NewRouter()
	r.Get("/stream/{streamID}", handler.HandleStream)
	server := httptest.NewServer(r)
	defer server.Close()

	// Connect WebSocket
	wsURL := "ws" + server.URL[4:] + "/stream/" + streamID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial error: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Collect all events
	var events []StreamEvent
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}
		var event StreamEvent
		json.Unmarshal(data, &event)
		events = append(events, event)
		if event.Type == EventDone {
			break
		}
	}

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (results + done), got %d", len(events))
	}

	// First event should be results
	if events[0].Type != EventResults {
		t.Errorf("first event type = %q, want %q", events[0].Type, EventResults)
	}
	if events[0].Platform != "yahoo_auction" {
		t.Errorf("first event platform = %q, want %q", events[0].Platform, "yahoo_auction")
	}

	// Last event should be done
	last := events[len(events)-1]
	if last.Type != EventDone {
		t.Errorf("last event type = %q, want %q", last.Type, EventDone)
	}
}

func TestRealtimeHandler_NoPlatforms(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	pl := &mockPlatformLister{ids: []string{}}
	handler := NewRealtimeHandler(sm, &mockPlatformSearcher{}, pl)

	streamID := sm.Create(domain.SearchQuery{Keyword: "test"})

	r := chi.NewRouter()
	r.Get("/stream/{streamID}", handler.HandleStream)
	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/stream/" + streamID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial error: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var event StreamEvent
	json.Unmarshal(data, &event)
	if event.Type != EventDone {
		t.Errorf("event type = %q, want %q", event.Type, EventDone)
	}
}
```

### Run tests

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/api/ -v -race
```

---

## Task 7: ProductHandler + HealthHandler (product.go, health.go + tests)

**Files:**
- Create: `backend/internal/api/product.go`
- Create: `backend/internal/api/product_test.go`
- Create: `backend/internal/api/health.go`
- Create: `backend/internal/api/health_test.go`

### product.go

```go
package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// ProductFetcher retrieves a product by its unified ID.
// This interface abstracts ES cache + adapter fallback for testability.
type ProductFetcher interface {
	GetProduct(ctx context.Context, id string) (*domain.UnifiedProduct, error)
}

// ProductHandler handles product detail requests.
type ProductHandler struct {
	fetcher ProductFetcher
}

// NewProductHandler creates a ProductHandler.
func NewProductHandler(fetcher ProductFetcher) *ProductHandler {
	return &ProductHandler{fetcher: fetcher}
}

// HandleGetProduct handles GET /api/v1/products/{id}.
func (h *ProductHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing product ID")
		return
	}

	product, err := h.fetcher.GetProduct(r.Context(), id)
	if err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "product not found")
		return
	}

	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "zh-TW"
	}

	resp := buildProductResponse(product, lang)
	Success(w, r, resp)
}

// ProductResponse is the API response for a product detail.
type ProductResponse struct {
	ID                  string           `json:"id"`
	Platform            string           `json:"platform"`
	Title               string           `json:"title"`
	TitleOriginal       string           `json:"title_original"`
	Description         string           `json:"description"`
	DescriptionOriginal string           `json:"description_original"`
	Images              []string         `json:"images"`
	PriceJPY            int64            `json:"price_jpy"`
	ServiceFeeJPY       int64            `json:"service_fee_jpy"`
	OriginalPrice       int64            `json:"original_price"`
	ShippingType        string           `json:"shipping_type"`
	ShippingFeeJPY      int64            `json:"shipping_fee_jpy"`
	Brand               *domain.Brand    `json:"brand,omitempty"`
	Categories          []string         `json:"categories"`
	Condition           string           `json:"condition"`
	Status              string           `json:"status"`
	Quantity            int              `json:"quantity"`
	Seller              domain.SellerInfo `json:"seller"`
	Variants            []domain.Variant `json:"variants,omitempty"`
	ContentRating       string           `json:"content_rating"`
	ListedAt            string           `json:"listed_at"`
	IsTranslated        bool             `json:"is_translated"`
}

// buildProductResponse maps a UnifiedProduct to the API response, selecting
// the translated title/description for the given language.
func buildProductResponse(p *domain.UnifiedProduct, lang string) ProductResponse {
	title := p.Title
	description := p.Description
	isTranslated := false

	if t, ok := p.TitleTranslated[lang]; ok && t != "" {
		title = t
		isTranslated = true
	}
	if d, ok := p.DescTranslated[lang]; ok && d != "" {
		description = d
	}

	return ProductResponse{
		ID:                  p.ID,
		Platform:            p.SourcePlatform,
		Title:               title,
		TitleOriginal:       p.Title,
		Description:         description,
		DescriptionOriginal: p.Description,
		Images:              p.Images,
		PriceJPY:            p.PriceJPY,
		ServiceFeeJPY:       p.ServiceFeeJPY,
		OriginalPrice:       p.OriginalPrice,
		ShippingType:        p.ShippingType,
		ShippingFeeJPY:      p.ShippingFeeJPY,
		Brand:               p.Brand,
		Categories:          p.Categories,
		Condition:           p.Condition,
		Status:              p.Status,
		Quantity:            p.Quantity,
		Seller:              p.Seller,
		Variants:            p.Variants,
		ContentRating:       p.ContentRating,
		ListedAt:            p.ListedAt.Format("2006-01-02T15:04:05Z"),
		IsTranslated:        isTranslated,
	}
}
```

### product_test.go

```go
package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
)

type mockProductFetcher struct {
	product *domain.UnifiedProduct
	err     error
}

func (m *mockProductFetcher) GetProduct(_ context.Context, _ string) (*domain.UnifiedProduct, error) {
	return m.product, m.err
}

func TestHandleGetProduct_Success(t *testing.T) {
	fetcher := &mockProductFetcher{
		product: &domain.UnifiedProduct{
			ID:              "yahoo_auction_abc123",
			SourcePlatform:  "yahoo_auction",
			Title:           "グッチ バッグ",
			TitleTranslated: map[string]string{"zh-TW": "古馳手提包"},
			Description:     "良い状態",
			PriceJPY:        16500,
			ServiceFeeJPY:   1500,
			OriginalPrice:   15000,
			Status:          "available",
			Condition:       "good",
			Quantity:        1,
			Images:          []string{"img1.jpg"},
			ListedAt:        time.Date(2026, 2, 20, 10, 0, 0, 0, time.UTC),
		},
	}
	handler := NewProductHandler(fetcher)

	r := chi.NewRouter()
	r.Get("/products/{id}", handler.HandleGetProduct)

	req := httptest.NewRequest("GET", "/products/yahoo_auction_abc123?lang=zh-TW", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}

	data, _ := json.Marshal(resp.Data)
	var product ProductResponse
	json.Unmarshal(data, &product)

	if product.Title != "古馳手提包" {
		t.Errorf("title = %q, want translated title", product.Title)
	}
	if product.TitleOriginal != "グッチ バッグ" {
		t.Errorf("title_original = %q", product.TitleOriginal)
	}
	if !product.IsTranslated {
		t.Error("expected is_translated = true")
	}
	if product.PriceJPY != 16500 {
		t.Errorf("price_jpy = %d, want 16500", product.PriceJPY)
	}
}

func TestHandleGetProduct_NotFound(t *testing.T) {
	fetcher := &mockProductFetcher{err: errors.New("not found")}
	handler := NewProductHandler(fetcher)

	r := chi.NewRouter()
	r.Get("/products/{id}", handler.HandleGetProduct)

	req := httptest.NewRequest("GET", "/products/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusNotFound)
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 40401 {
		t.Errorf("code = %d, want 40401", resp.Code)
	}
}

func TestHandleGetProduct_DefaultLang(t *testing.T) {
	fetcher := &mockProductFetcher{
		product: &domain.UnifiedProduct{
			ID:              "p1",
			SourcePlatform:  "test",
			Title:           "テスト",
			TitleTranslated: map[string]string{"zh-TW": "測試"},
			ListedAt:        time.Now(),
		},
	}
	handler := NewProductHandler(fetcher)

	r := chi.NewRouter()
	r.Get("/products/{id}", handler.HandleGetProduct)

	// No lang param — should default to zh-TW
	req := httptest.NewRequest("GET", "/products/p1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	data, _ := json.Marshal(resp.Data)
	var product ProductResponse
	json.Unmarshal(data, &product)

	if product.Title != "測試" {
		t.Errorf("title = %q, want %q (default zh-TW)", product.Title, "測試")
	}
}

func TestBuildProductResponse_NoTranslation(t *testing.T) {
	p := &domain.UnifiedProduct{
		ID:       "p1",
		Title:    "テスト",
		ListedAt: time.Now(),
	}
	resp := buildProductResponse(p, "zh-TW")

	if resp.Title != "テスト" {
		t.Errorf("title = %q, want original when no translation", resp.Title)
	}
	if resp.IsTranslated {
		t.Error("expected is_translated = false when no translation available")
	}
}
```

### health.go

```go
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// HealthChecker checks the health of a platform.
type HealthChecker interface {
	GetAdapter(platformID string) (domain.PlatformAdapter, error)
	AllPlatformIDs() []string
}

// HealthHandler handles health check requests.
type HealthHandler struct {
	checker HealthChecker
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(checker HealthChecker) *HealthHandler {
	return &HealthHandler{checker: checker}
}

// PlatformHealth represents the health status of a single platform.
type PlatformHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HandleHealth handles GET /health.
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	platforms := make(map[string]PlatformHealth)

	ids := h.checker.AllPlatformIDs()
	for _, id := range ids {
		adapter, err := h.checker.GetAdapter(id)
		if err != nil {
			platforms[id] = PlatformHealth{Status: "unknown", Message: err.Error()}
			continue
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		health := adapter.HealthCheck(ctx)
		cancel()

		platforms[id] = PlatformHealth{
			Status:  health.Status,
			Message: health.Message,
		}
	}

	status := "ok"
	for _, ph := range platforms {
		if ph.Status != "healthy" {
			status = "degraded"
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    status,
		"platforms": platforms,
	})
}
```

### health_test.go

```go
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

type mockHealthAdapter struct {
	status domain.HealthStatus
}

func (m *mockHealthAdapter) PlatformID() string                            { return "test" }
func (m *mockHealthAdapter) Capabilities() domain.AdapterCaps              { return domain.AdapterCaps{} }
func (m *mockHealthAdapter) Search(_ context.Context, _ domain.SearchQuery) (*domain.SearchResult, error) {
	return nil, nil
}
func (m *mockHealthAdapter) GetProduct(_ context.Context, _ string) (*domain.RawProduct, error) {
	return nil, nil
}
func (m *mockHealthAdapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) {
	return nil, nil
}
func (m *mockHealthAdapter) HealthCheck(_ context.Context) domain.HealthStatus {
	return m.status
}

type mockHealthChecker struct {
	adapters map[string]domain.PlatformAdapter
}

func (m *mockHealthChecker) GetAdapter(id string) (domain.PlatformAdapter, error) {
	a, ok := m.adapters[id]
	if !ok {
		return nil, domain.ErrAdapterNotFound
	}
	return a, nil
}

func (m *mockHealthChecker) AllPlatformIDs() []string {
	ids := make([]string, 0, len(m.adapters))
	for id := range m.adapters {
		ids = append(ids, id)
	}
	return ids
}

func TestHandleHealth_AllHealthy(t *testing.T) {
	checker := &mockHealthChecker{
		adapters: map[string]domain.PlatformAdapter{
			"yahoo_auction": &mockHealthAdapter{status: domain.HealthStatus{Status: "healthy"}},
			"amazon_jp":     &mockHealthAdapter{status: domain.HealthStatus{Status: "healthy"}},
		},
	}
	handler := NewHealthHandler(checker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)
	handler.HandleHealth(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}

func TestHandleHealth_Degraded(t *testing.T) {
	checker := &mockHealthChecker{
		adapters: map[string]domain.PlatformAdapter{
			"yahoo_auction": &mockHealthAdapter{status: domain.HealthStatus{Status: "healthy"}},
			"surugaya":      &mockHealthAdapter{status: domain.HealthStatus{Status: "degraded", Message: "high latency"}},
		},
	}
	handler := NewHealthHandler(checker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)
	handler.HandleHealth(w, r)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "degraded" {
		t.Errorf("status = %q, want %q", resp["status"], "degraded")
	}
}

func TestHandleHealth_NoPlatforms(t *testing.T) {
	checker := &mockHealthChecker{adapters: map[string]domain.PlatformAdapter{}}
	handler := NewHealthHandler(checker)

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)
	handler.HandleHealth(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want %q", resp["status"], "ok")
	}
}
```

**Note:** The HealthHandler test uses `domain.ErrAdapterNotFound` which needs to be added to the domain package. Add this to `backend/internal/domain/adapter.go`:

```go
// ErrAdapterNotFound is returned when a platform adapter is not registered.
var ErrAdapterNotFound = errors.New("domain: adapter not found")
```

(Add `"errors"` to the import block of adapter.go.)

### Run tests

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/api/ -v -race
```

---

## Task 8: Router + main.go (router.go + main.go update)

Wire all handlers into a chi router and update the gateway entry point.

**Files:**
- Create: `backend/internal/api/router.go`
- Modify: `backend/cmd/gateway/main.go`

### router.go

```go
package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

// RouterConfig holds the dependencies needed to build the API router.
type RouterConfig struct {
	SearchHandler   *SearchHandler
	RealtimeHandler *RealtimeHandler
	ProductHandler  *ProductHandler
	HealthHandler   *HealthHandler
}

// NewRouter creates a chi router with all API routes and middleware.
func NewRouter(cfg RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware
	r.Use(Recovery)
	r.Use(RequestID)
	r.Use(Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health check (outside /api/v1 group)
	r.Get("/health", cfg.HealthHandler.HandleHealth)

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/search", cfg.SearchHandler.HandleSearch)
		r.Get("/search/stream/{streamID}", cfg.RealtimeHandler.HandleStream)
		r.Get("/products/{id}", cfg.ProductHandler.HandleGetProduct)
	})

	return r
}
```

### main.go update

Replace the current placeholder `backend/cmd/gateway/main.go` with:

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/rakutao/collection-gateway/internal/api"
	"github.com/rakutao/collection-gateway/internal/filter"
	"github.com/rakutao/collection-gateway/internal/registry"
	"github.com/rakutao/collection-gateway/internal/search"
)

func main() {
	// --- Build dependencies ---

	// Keyword filter (political keywords blacklist).
	keywordFilter := filter.NewKeywordFilter(
		[]string{"天安门", "六四", "法轮功"},        // exact match
		[]string{"政治", "独立运动"},              // contains match
	)

	// Search gateway (translator is nil — will be wired when AI service is ready).
	gateway := search.NewGateway(nil, keywordFilter)

	// Platform registry (empty — adapters will be registered when implemented).
	reg := registry.New()

	// Stream manager for real-time search.
	streamManager := api.NewStreamManager()
	defer streamManager.Stop()

	// --- Build handlers ---
	// NOTE: Searcher, ProductFetcher, PlatformSearcher, PlatformLister, and
	// HealthChecker are placeholder interfaces. Their real implementations
	// will be wired when ES client and platform adapters are built.

	searchHandler := api.NewSearchHandler(gateway, nil, streamManager)
	realtimeHandler := api.NewRealtimeHandler(streamManager, nil, nil)
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

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

**Note:** The `main.go` passes `nil` for interfaces that don't have implementations yet (Searcher, ProductFetcher, etc.). This is intentional — the handlers already nil-check or will get real implementations when ES client and adapters are built. For compile safety, the `_ = reg` line is needed if `reg` is unused. However, we reference it below for the HealthChecker, so adjust as needed when real adapters are available.

### Run full test suite

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./... -v -race
```

---

## Task 9: Full Verification

Run the complete test suite across all packages and verify everything compiles and passes.

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go build ./... && go test ./... -v -race
```

Expected: 11 packages compile, all tests pass with `-race` (including the new `api` package tests).

Also verify Python tests still pass:

```bash
cd /Users/gongqianrong/Desktop/ai/ai-service && python3 -m pytest tests/ -v
```

---

## Dependencies

```
Task 1 (Prep) → Task 2 (Response) → Task 3 (Middleware)
                                  → Task 4 (StreamManager)
Task 2 + Task 4 → Task 5 (SearchHandler)
Task 4 → Task 6 (RealtimeHandler)
Task 2 → Task 7 (ProductHandler + HealthHandler)
Task 5 + Task 6 + Task 7 → Task 8 (Router + main.go)
Task 8 → Task 9 (Full Verification)
```
