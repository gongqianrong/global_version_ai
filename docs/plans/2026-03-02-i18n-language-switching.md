# I18N Language Switching Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** All API response text (error messages, status/condition labels, product content) switches language based on `Accept-Language` header.

**Architecture:** New `internal/i18n` package holds language constants, context helpers, and Go-map-based translation tables. A new middleware parses `Accept-Language` and injects lang into context. `ErrorWithCode` auto-translates error messages by error code. Product content uses existing `TitleTranslated`/`DescTranslated` maps. A `translate` package wraps a free translation API for pre-translating at index time.

**Tech Stack:** Go 1.22, chi router, no external i18n libraries

---

### Task 1: Create `internal/i18n` package — language constants and context

**Files:**
- Create: `backend/internal/i18n/lang.go`
- Test: `backend/internal/i18n/lang_test.go`

**Step 1: Write the test**

```go
// backend/internal/i18n/lang_test.go
package i18n

import (
	"context"
	"net/http"
	"testing"
)

func TestParseAcceptLanguage(t *testing.T) {
	tests := []struct {
		header string
		want   Lang
	}{
		{"", DefaultLang},
		{"ja", LangJA},
		{"en", LangEN},
		{"zh-TW", LangZhTW},
		{"zh-TW,en;q=0.9", LangZhTW},
		{"en-US,en;q=0.9", LangEN},
		{"ja-JP", LangJA},
		{"fr", DefaultLang}, // unsupported falls back to default
	}
	for _, tt := range tests {
		got := ParseAcceptLanguage(tt.header)
		if got != tt.want {
			t.Errorf("ParseAcceptLanguage(%q) = %q, want %q", tt.header, got, tt.want)
		}
	}
}

func TestContextLang(t *testing.T) {
	ctx := context.Background()
	if got := FromContext(ctx); got != DefaultLang {
		t.Errorf("FromContext(empty) = %q, want %q", got, DefaultLang)
	}

	ctx = WithLang(ctx, LangJA)
	if got := FromContext(ctx); got != LangJA {
		t.Errorf("FromContext(ja) = %q, want %q", got, LangJA)
	}
}

func TestFromRequest(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "ja")
	if got := FromRequest(req); got != LangJA {
		t.Errorf("FromRequest(ja) = %q, want %q", got, LangJA)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/i18n/ -v`
Expected: FAIL — package does not exist

**Step 3: Write the implementation**

```go
// backend/internal/i18n/lang.go
package i18n

import (
	"context"
	"net/http"
	"strings"
)

// Lang represents a supported language.
type Lang string

const (
	LangZhTW Lang = "zh-TW"
	LangJA   Lang = "ja"
	LangEN   Lang = "en"
)

// DefaultLang is the fallback language.
const DefaultLang = LangZhTW

// SupportedLangs lists all supported languages.
var SupportedLangs = []Lang{LangZhTW, LangJA, LangEN}

type langKey struct{}

// WithLang returns a new context with the given language.
func WithLang(ctx context.Context, lang Lang) context.Context {
	return context.WithValue(ctx, langKey{}, lang)
}

// FromContext extracts the language from context, or returns DefaultLang.
func FromContext(ctx context.Context) Lang {
	if l, ok := ctx.Value(langKey{}).(Lang); ok {
		return l
	}
	return DefaultLang
}

// FromRequest extracts language from the request's Accept-Language header.
func FromRequest(r *http.Request) Lang {
	return ParseAcceptLanguage(r.Header.Get("Accept-Language"))
}

// ParseAcceptLanguage parses an Accept-Language header value and returns the
// best matching supported language. Falls back to DefaultLang.
func ParseAcceptLanguage(header string) Lang {
	if header == "" {
		return DefaultLang
	}
	// Split by comma, take first (highest priority) segments.
	for _, part := range strings.Split(header, ",") {
		tag := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		if lang := matchLang(tag); lang != "" {
			return lang
		}
	}
	return DefaultLang
}

// matchLang matches a BCP47-like tag to a supported Lang.
func matchLang(tag string) Lang {
	tag = strings.TrimSpace(tag)
	lower := strings.ToLower(tag)

	switch {
	case lower == "zh-tw" || lower == "zh-hant" || lower == "zh":
		return LangZhTW
	case lower == "ja" || strings.HasPrefix(lower, "ja-"):
		return LangJA
	case lower == "en" || strings.HasPrefix(lower, "en-"):
		return LangEN
	}
	return ""
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/i18n/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/i18n/
git commit -m "feat(i18n): add language constants, context helpers, and Accept-Language parser"
```

---

### Task 2: Create error message translations

**Files:**
- Create: `backend/internal/i18n/messages.go`
- Test: `backend/internal/i18n/messages_test.go`

**Step 1: Write the test**

```go
// backend/internal/i18n/messages_test.go
package i18n

import "testing"

func TestErrorMessage(t *testing.T) {
	tests := []struct {
		code int
		lang Lang
		want string
	}{
		{40002, LangEN, "missing required parameter"},
		{40002, LangJA, "必須パラメータが不足しています"},
		{40002, LangZhTW, "缺少必要參數"},
		{40401, LangEN, "product not found"},
		{40401, LangJA, "商品が見つかりません"},
		{40401, LangZhTW, "找不到商品"},
		{50001, LangEN, "internal server error"},
		{50003, LangEN, "service unavailable"},
		{40001, LangEN, "keyword is blocked by content policy"},
		{99999, LangEN, "unknown error"}, // unmapped code
	}
	for _, tt := range tests {
		got := ErrorMessage(tt.code, tt.lang)
		if got != tt.want {
			t.Errorf("ErrorMessage(%d, %q) = %q, want %q", tt.code, tt.lang, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/i18n/ -v -run TestErrorMessage`
Expected: FAIL

**Step 3: Write the implementation**

```go
// backend/internal/i18n/messages.go
package i18n

// errorMessages maps error code → lang → translated message.
var errorMessages = map[int]map[Lang]string{
	40001: {
		LangEN:   "keyword is blocked by content policy",
		LangJA:   "キーワードはコンテンツポリシーにより制限されています",
		LangZhTW: "關鍵字被內容政策封鎖",
	},
	40002: {
		LangEN:   "missing required parameter",
		LangJA:   "必須パラメータが不足しています",
		LangZhTW: "缺少必要參數",
	},
	40401: {
		LangEN:   "product not found",
		LangJA:   "商品が見つかりません",
		LangZhTW: "找不到商品",
	},
	50001: {
		LangEN:   "internal server error",
		LangJA:   "内部サーバーエラー",
		LangZhTW: "內部伺服器錯誤",
	},
	50003: {
		LangEN:   "service unavailable",
		LangJA:   "サービスが利用できません",
		LangZhTW: "服務暫時無法使用",
	},
}

// unknownError is the fallback for unmapped error codes.
var unknownError = map[Lang]string{
	LangEN:   "unknown error",
	LangJA:   "不明なエラー",
	LangZhTW: "未知錯誤",
}

// ErrorMessage returns the translated error message for the given code and language.
func ErrorMessage(code int, lang Lang) string {
	if msgs, ok := errorMessages[code]; ok {
		if msg, ok := msgs[lang]; ok {
			return msg
		}
	}
	if msg, ok := unknownError[lang]; ok {
		return msg
	}
	return unknownError[LangEN]
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/i18n/ -v -run TestErrorMessage`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/i18n/messages.go backend/internal/i18n/messages_test.go
git commit -m "feat(i18n): add error message translation table"
```

---

### Task 3: Create status/condition label translations

**Files:**
- Create: `backend/internal/i18n/labels.go`
- Test: `backend/internal/i18n/labels_test.go`

**Step 1: Write the test**

```go
// backend/internal/i18n/labels_test.go
package i18n

import "testing"

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		status string
		lang   Lang
		want   string
	}{
		{"available", LangEN, "Available"},
		{"available", LangJA, "販売中"},
		{"available", LangZhTW, "可購買"},
		{"sold", LangEN, "Sold"},
		{"sold", LangJA, "売り切れ"},
		{"unknown_status", LangEN, "unknown_status"}, // passthrough
	}
	for _, tt := range tests {
		got := StatusLabel(tt.status, tt.lang)
		if got != tt.want {
			t.Errorf("StatusLabel(%q, %q) = %q, want %q", tt.status, tt.lang, got, tt.want)
		}
	}
}

func TestConditionLabel(t *testing.T) {
	tests := []struct {
		cond string
		lang Lang
		want string
	}{
		{"new", LangEN, "New"},
		{"new", LangJA, "新品"},
		{"new", LangZhTW, "全新"},
		{"like_new", LangEN, "Like New"},
		{"good", LangJA, "良好"},
		{"unknown_cond", LangEN, "unknown_cond"}, // passthrough
	}
	for _, tt := range tests {
		got := ConditionLabel(tt.cond, tt.lang)
		if got != tt.want {
			t.Errorf("ConditionLabel(%q, %q) = %q, want %q", tt.cond, tt.lang, got, tt.want)
		}
	}
}

func TestShippingLabel(t *testing.T) {
	if got := ShippingLabel("free", LangJA); got != "送料無料" {
		t.Errorf("ShippingLabel(free, ja) = %q, want 送料無料", got)
	}
}

func TestContentRatingLabel(t *testing.T) {
	if got := ContentRatingLabel("general", LangZhTW); got != "一般" {
		t.Errorf("ContentRatingLabel(general, zh-TW) = %q, want 一般", got)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/i18n/ -v -run "TestStatusLabel|TestConditionLabel|TestShippingLabel|TestContentRatingLabel"`
Expected: FAIL

**Step 3: Write the implementation**

```go
// backend/internal/i18n/labels.go
package i18n

// label maps are: constant value → lang → display text.

var statusLabels = map[string]map[Lang]string{
	"available": {LangEN: "Available", LangJA: "販売中", LangZhTW: "可購買"},
	"sold":      {LangEN: "Sold", LangJA: "売り切れ", LangZhTW: "已售出"},
	"reserved":  {LangEN: "Reserved", LangJA: "取り置き中", LangZhTW: "已預約"},
	"delisted":  {LangEN: "Delisted", LangJA: "掲載終了", LangZhTW: "已下架"},
}

var conditionLabels = map[string]map[Lang]string{
	"new":      {LangEN: "New", LangJA: "新品", LangZhTW: "全新"},
	"like_new": {LangEN: "Like New", LangJA: "未使用に近い", LangZhTW: "近全新"},
	"good":     {LangEN: "Good", LangJA: "良好", LangZhTW: "良好"},
	"fair":     {LangEN: "Fair", LangJA: "やや傷あり", LangZhTW: "尚可"},
	"poor":     {LangEN: "Poor", LangJA: "傷あり", LangZhTW: "較差"},
}

var shippingLabels = map[string]map[Lang]string{
	"free":       {LangEN: "Free Shipping", LangJA: "送料無料", LangZhTW: "免運費"},
	"buyer_pays": {LangEN: "Buyer Pays", LangJA: "送料購入者負担", LangZhTW: "買家自付"},
	"included":   {LangEN: "Included", LangJA: "送料込み", LangZhTW: "含運費"},
}

var contentRatingLabels = map[string]map[Lang]string{
	"general": {LangEN: "General", LangJA: "一般", LangZhTW: "一般"},
	"r18":     {LangEN: "R18", LangJA: "R18", LangZhTW: "R18"},
}

// labelLookup is the generic lookup for any label map.
func labelLookup(table map[string]map[Lang]string, key string, lang Lang) string {
	if byLang, ok := table[key]; ok {
		if label, ok := byLang[lang]; ok {
			return label
		}
	}
	return key // passthrough if not found
}

// StatusLabel returns the localized display text for a status constant.
func StatusLabel(status string, lang Lang) string {
	return labelLookup(statusLabels, status, lang)
}

// ConditionLabel returns the localized display text for a condition constant.
func ConditionLabel(condition string, lang Lang) string {
	return labelLookup(conditionLabels, condition, lang)
}

// ShippingLabel returns the localized display text for a shipping type constant.
func ShippingLabel(shipping string, lang Lang) string {
	return labelLookup(shippingLabels, shipping, lang)
}

// ContentRatingLabel returns the localized display text for a content rating constant.
func ContentRatingLabel(rating string, lang Lang) string {
	return labelLookup(contentRatingLabels, rating, lang)
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/i18n/ -v -run "TestStatusLabel|TestConditionLabel|TestShippingLabel|TestContentRatingLabel"`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/i18n/labels.go backend/internal/i18n/labels_test.go
git commit -m "feat(i18n): add status, condition, shipping, and content rating label translations"
```

---

### Task 4: Add language middleware

**Files:**
- Modify: `backend/internal/api/middleware.go` (append new middleware)
- Modify: `backend/internal/api/middleware_test.go` (add test)
- Modify: `backend/internal/api/router.go` (wire middleware)

**Step 1: Write the test**

Append to `backend/internal/api/middleware_test.go`:

```go
func TestLanguageMiddleware(t *testing.T) {
	handler := Language(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := i18n.FromContext(r.Context())
		w.Write([]byte(string(lang)))
	}))

	// With Accept-Language header
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Language", "ja")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Body.String() != "ja" {
		t.Errorf("got %q, want %q", w.Body.String(), "ja")
	}

	// Without header — should default to zh-TW
	req2 := httptest.NewRequest("GET", "/", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	if w2.Body.String() != "zh-TW" {
		t.Errorf("got %q, want %q", w2.Body.String(), "zh-TW")
	}
}
```

Add import `"github.com/rakutao/collection-gateway/internal/i18n"` to the test file.

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/api/ -v -run TestLanguageMiddleware`
Expected: FAIL — `Language` undefined

**Step 3: Implement the middleware**

Append to `backend/internal/api/middleware.go`:

```go
import "github.com/rakutao/collection-gateway/internal/i18n"

// Language parses the Accept-Language header and stores the resolved language in the request context.
func Language(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lang := i18n.FromRequest(r)
		ctx := i18n.WithLang(r.Context(), lang)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

Wire in `backend/internal/api/router.go` — add `r.Use(Language)` after `r.Use(RequestID)`.

Also add `"Accept-Language"` to the CORS `AllowedHeaders` list.

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/api/ -v -run TestLanguageMiddleware`
Expected: PASS

**Step 5: Commit**

```bash
git add backend/internal/api/middleware.go backend/internal/api/middleware_test.go backend/internal/api/router.go
git commit -m "feat(i18n): add Language middleware, wire into router"
```

---

### Task 5: Modify `ErrorWithCode` to auto-translate messages

**Files:**
- Modify: `backend/internal/api/response.go`
- Modify: `backend/internal/api/response_test.go`

**Step 1: Write the test**

Add to `backend/internal/api/response_test.go`:

```go
func TestErrorWithCode_I18N(t *testing.T) {
	// Create request with Japanese language context
	req := httptest.NewRequest("GET", "/", nil)
	ctx := i18n.WithLang(req.Context(), i18n.LangJA)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	ErrorWithCode(w, req, http.StatusBadRequest, 40002, "missing required parameter: keyword")

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Message != "必須パラメータが不足しています" {
		t.Errorf("message = %q, want Japanese translation", resp.Message)
	}
}

func TestErrorWithCode_DefaultLang(t *testing.T) {
	// No language context → should use zh-TW
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	ErrorWithCode(w, req, http.StatusNotFound, 40401, "product not found")

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.Message != "找不到商品" {
		t.Errorf("message = %q, want zh-TW translation", resp.Message)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/api/ -v -run "TestErrorWithCode_I18N|TestErrorWithCode_DefaultLang"`
Expected: FAIL — message is still English

**Step 3: Modify `ErrorWithCode` in `response.go`**

```go
import "github.com/rakutao/collection-gateway/internal/i18n"

func ErrorWithCode(w http.ResponseWriter, r *http.Request, httpStatus, code int, message string) {
	lang := i18n.FromContext(r.Context())
	translated := i18n.ErrorMessage(code, lang)
	writeJSON(w, httpStatus, APIResponse{
		Code:      code,
		Message:   translated,
		RequestID: getRequestID(r),
	})
}
```

**Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/api/ -v -run "TestErrorWithCode_I18N|TestErrorWithCode_DefaultLang"`
Expected: PASS

**Step 5: Fix any broken existing tests**

Run: `cd backend && go test ./internal/api/ -v`

Existing tests that check for English error messages (e.g. `resp.Message == "missing required parameter: keyword"`) will now get zh-TW text since no lang context is set. Update these tests to either:
- Set English lang context on the request, OR
- Check for the zh-TW message, OR
- Only check `resp.Code` (not message text)

The simplest fix: in existing tests that don't set Accept-Language, the default is zh-TW, so update message assertions to match zh-TW translations, or simply remove message text assertions and keep code assertions.

**Step 6: Commit**

```bash
git add backend/internal/api/response.go backend/internal/api/response_test.go
git commit -m "feat(i18n): auto-translate error messages in ErrorWithCode"
```

---

### Task 6: Localize product detail response (`product.go`)

**Files:**
- Modify: `backend/internal/api/product.go`
- Modify: `backend/internal/api/product_test.go`

**Step 1: Write the test**

Add to `backend/internal/api/product_test.go`:

```go
func TestBuildProductResponse_I18N_Labels(t *testing.T) {
	p := &domain.UnifiedProduct{
		Status:    "available",
		Condition: "new",
		ShippingType: "free",
		ContentRating: "general",
		Title:     "テスト商品",
		TitleTranslated: map[string]string{"zh-TW": "測試商品", "en": "Test Product"},
		Description: "テスト説明",
		DescTranslated: map[string]string{"zh-TW": "測試說明", "en": "Test Description"},
		Images: []string{"img.jpg"},
	}

	resp := buildProductResponse(p, i18n.LangEN)
	if resp.Status != "Available" {
		t.Errorf("status = %q, want %q", resp.Status, "Available")
	}
	if resp.Condition != "New" {
		t.Errorf("condition = %q, want %q", resp.Condition, "New")
	}
	if resp.Title != "Test Product" {
		t.Errorf("title = %q, want %q", resp.Title, "Test Product")
	}

	respJA := buildProductResponse(p, i18n.LangJA)
	if respJA.Title != "テスト商品" {
		t.Errorf("title(ja) = %q, want original Japanese", respJA.Title)
	}
	if respJA.Status != "販売中" {
		t.Errorf("status(ja) = %q, want %q", respJA.Status, "販売中")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/api/ -v -run TestBuildProductResponse_I18N`
Expected: FAIL — status is still "available" not "Available"

**Step 3: Modify `product.go`**

Change `buildProductResponse` signature to accept `i18n.Lang` instead of `string`:

```go
func buildProductResponse(p *domain.UnifiedProduct, lang i18n.Lang) ProductResponse {
	title := p.Title
	description := p.Description
	isTranslated := false

	if t, ok := p.TitleTranslated[string(lang)]; ok && t != "" {
		title = t
		isTranslated = true
	}
	if d, ok := p.DescTranslated[string(lang)]; ok && d != "" {
		description = d
	}

	return ProductResponse{
		// ... same fields ...
		Status:        i18n.StatusLabel(p.Status, lang),
		Condition:     i18n.ConditionLabel(p.Condition, lang),
		ShippingType:  i18n.ShippingLabel(p.ShippingType, lang),
		ContentRating: i18n.ContentRatingLabel(p.ContentRating, lang),
		// ... rest same ...
		IsTranslated: isTranslated,
	}
}
```

Update `HandleGetProduct` to use `i18n.FromContext(r.Context())` instead of reading `lang` query param:

```go
func (h *ProductHandler) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "missing product ID")
		return
	}

	product, err := h.fetcher.GetProduct(r.Context(), id)
	if err != nil && h.fallback != nil {
		product, err = h.fallback.GetProduct(r.Context(), id)
	}
	if err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "product not found")
		return
	}

	lang := i18n.FromContext(r.Context())
	resp := buildProductResponse(product, lang)
	Success(w, r, resp)
}
```

**Step 4: Run all tests**

Run: `cd backend && go test ./internal/api/ -v`
Expected: PASS (fix any broken product tests)

**Step 5: Commit**

```bash
git add backend/internal/api/product.go backend/internal/api/product_test.go
git commit -m "feat(i18n): localize product detail response (status, condition, title, description)"
```

---

### Task 7: Localize `ProductSummary` in search and platform search

**Files:**
- Modify: `backend/internal/api/platform_service.go`
- Modify: `backend/internal/api/search.go`
- Modify: `backend/internal/api/platform_search_handler.go`

**Step 1: Write test**

Add to `backend/internal/api/platform_service_test.go` (or create if needed):

```go
func TestSearchPlatformFull_LocalizedSummary(t *testing.T) {
	// Setup test with a mock adapter that returns products with translations
	// Verify that summaries have localized status/condition labels
	// based on the query's UserLang
}
```

**Step 2: Implement**

Update `SearchPlatform` and `SearchPlatformFull` in `platform_service.go` to accept a `lang i18n.Lang` parameter (or read from `query.UserLang`) and localize the `ProductSummary` fields:

```go
summary := domain.ProductSummary{
	// ...
	Status:    i18n.StatusLabel(product.Status, lang),
	Condition: i18n.ConditionLabel(product.Condition, lang),
}
```

For title: use `TitleTranslated[string(lang)]` if available, otherwise original title.

Update callers:
- `platform_search_handler.go`: pass `i18n.FromContext(r.Context())`
- `realtime.go`: pass lang from `stream.Query.UserLang`
- `search.go`: pass lang from query

**Step 3: Run all tests**

Run: `cd backend && go test ./internal/api/ -v`
Expected: PASS

**Step 4: Commit**

```bash
git add backend/internal/api/platform_service.go backend/internal/api/platform_search_handler.go backend/internal/api/realtime.go backend/internal/api/search.go
git commit -m "feat(i18n): localize ProductSummary in search results"
```

---

### Task 8: Update `parseSearchParams` to use Accept-Language instead of `?lang=`

**Files:**
- Modify: `backend/internal/api/search.go`
- Modify: `backend/internal/api/search_test.go`

**Step 1: Write test**

```go
func TestParseSearchParams_UsesAcceptLanguage(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/search?keyword=test", nil)
	ctx := i18n.WithLang(req.Context(), i18n.LangJA)
	req = req.WithContext(ctx)

	q := parseSearchParams(req)
	if q.UserLang != "ja" {
		t.Errorf("UserLang = %q, want %q", q.UserLang, "ja")
	}
}
```

**Step 2: Implement**

In `parseSearchParams`, replace the `lang` query param logic with:

```go
lang := i18n.FromContext(r.Context())
// ... in the return:
UserLang: string(lang),
```

Remove the `q.Get("lang")` logic.

**Step 3: Run all tests**

Run: `cd backend && go test ./internal/api/ -v`
Expected: PASS

**Step 4: Commit**

```bash
git add backend/internal/api/search.go backend/internal/api/search_test.go
git commit -m "refactor(i18n): use Accept-Language header instead of ?lang= query param"
```

---

### Task 9: Create `internal/translate` package (free translation API wrapper)

**Files:**
- Create: `backend/internal/translate/translator.go`
- Create: `backend/internal/translate/translator_test.go`

**Step 1: Write the test**

```go
// backend/internal/translate/translator_test.go
package translate

import (
	"context"
	"testing"
)

func TestTranslator_Interface(t *testing.T) {
	// Test with a mock/noop translator
	tr := NewNoop()
	result, err := tr.Translate(context.Background(), "テスト", "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Noop returns original text
	if result != "テスト" {
		t.Errorf("got %q, want %q", result, "テスト")
	}
}

func TestBatchTranslate(t *testing.T) {
	tr := NewNoop()
	results, err := tr.BatchTranslate(context.Background(), []string{"a", "b"}, "ja", "en")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
}
```

**Step 2: Implement**

```go
// backend/internal/translate/translator.go
package translate

import "context"

// Translator translates text between languages.
type Translator interface {
	Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error)
	BatchTranslate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error)
}

// Noop is a no-op translator that returns the original text. Used as placeholder.
type Noop struct{}

func NewNoop() *Noop { return &Noop{} }

func (n *Noop) Translate(_ context.Context, text, _, _ string) (string, error) {
	return text, nil
}

func (n *Noop) BatchTranslate(_ context.Context, texts []string, _, _ string) ([]string, error) {
	result := make([]string, len(texts))
	copy(result, texts)
	return result, nil
}
```

This defines the interface and a noop implementation. The actual free API integration (e.g. Google Translate, LibreTranslate) can be added as a separate concrete implementation later without changing the interface.

**Step 3: Run test**

Run: `cd backend && go test ./internal/translate/ -v`
Expected: PASS

**Step 4: Commit**

```bash
git add backend/internal/translate/
git commit -m "feat(i18n): add Translator interface and noop implementation"
```

---

### Task 10: Wire translator into output pipeline for pre-translation on ES write

**Files:**
- Create: `backend/internal/output/translate_sink.go`
- Create: `backend/internal/output/translate_sink_test.go`
- Modify: `backend/cmd/gateway/main.go`

**Step 1: Write the test**

```go
// backend/internal/output/translate_sink_test.go
package output

import (
	"context"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/i18n"
	"github.com/rakutao/collection-gateway/internal/translate"
)

func TestTranslateSink_PopulatesTranslations(t *testing.T) {
	tr := translate.NewNoop() // noop copies original text
	sink := NewTranslateSink(tr)

	products := []domain.UnifiedProduct{
		{
			ID:    "test_1",
			Title: "テスト商品",
		},
	}

	err := sink.Enrich(context.Background(), products)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With noop translator, TitleTranslated should have entries for non-ja langs
	for _, lang := range i18n.SupportedLangs {
		if lang == i18n.LangJA {
			continue // source lang, no translation needed
		}
		if _, ok := products[0].TitleTranslated[string(lang)]; !ok {
			t.Errorf("missing TitleTranslated for %s", lang)
		}
	}
}
```

**Step 2: Implement**

```go
// backend/internal/output/translate_sink.go
package output

import (
	"context"
	"log"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/i18n"
	"github.com/rakutao/collection-gateway/internal/translate"
)

// TranslateSink enriches products with translations before they are persisted.
// It is NOT a Sink — it's a pre-processing step called before dispatch.
type TranslateSink struct {
	translator translate.Translator
}

func NewTranslateSink(tr translate.Translator) *TranslateSink {
	return &TranslateSink{translator: tr}
}

// Enrich populates TitleTranslated and DescTranslated for all supported languages.
// Products are modified in place. Source language is assumed to be Japanese.
func (s *TranslateSink) Enrich(ctx context.Context, products []domain.UnifiedProduct) error {
	for i := range products {
		p := &products[i]
		if p.TitleTranslated == nil {
			p.TitleTranslated = make(map[string]string)
		}
		if p.DescTranslated == nil {
			p.DescTranslated = make(map[string]string)
		}

		for _, lang := range i18n.SupportedLangs {
			if lang == i18n.LangJA {
				continue // source language
			}
			langStr := string(lang)

			// Skip if already translated
			if _, ok := p.TitleTranslated[langStr]; ok {
				continue
			}

			if p.Title != "" {
				translated, err := s.translator.Translate(ctx, p.Title, "ja", langStr)
				if err != nil {
					log.Printf("[translate] title error for %s→%s: %v", p.ID, langStr, err)
					continue
				}
				p.TitleTranslated[langStr] = translated
			}

			if p.Description != "" {
				translated, err := s.translator.Translate(ctx, p.Description, "ja", langStr)
				if err != nil {
					log.Printf("[translate] desc error for %s→%s: %v", p.ID, langStr, err)
					continue
				}
				p.DescTranslated[langStr] = translated
			}
		}
	}
	return nil
}
```

Wire in `main.go`: create `TranslateSink` and call `Enrich` before dispatch in the `asyncProductWriter`, or integrate it into the output Router as a pre-processing step.

**Step 3: Run tests**

Run: `cd backend && go test ./internal/output/ -v`
Expected: PASS

**Step 4: Commit**

```bash
git add backend/internal/output/translate_sink.go backend/internal/output/translate_sink_test.go backend/cmd/gateway/main.go
git commit -m "feat(i18n): add TranslateSink for pre-translating products before ES write"
```

---

### Task 11: Run full test suite and fix any failures

**Files:**
- Various test files that may need updates

**Step 1: Run all tests**

Run: `cd backend && go test ./... -v`

**Step 2: Fix any failures**

Common issues:
- Tests that assert exact English error messages now get zh-TW (default lang)
- Tests that pass `string` lang to `buildProductResponse` now need `i18n.Lang`
- Import cycles (should not happen since i18n has no internal dependencies)

**Step 3: Commit**

```bash
git add -A
git commit -m "fix: update tests for i18n integration"
```

---

### Task 12: Integration verification

**Step 1:** Manual smoke test with curl:

```bash
# Default (zh-TW)
curl -s http://localhost:8080/api/v1/products/surugaya_663043159

# Japanese
curl -s -H "Accept-Language: ja" http://localhost:8080/api/v1/products/surugaya_663043159

# English
curl -s -H "Accept-Language: en" http://localhost:8080/api/v1/products/surugaya_663043159

# Error in Japanese
curl -s -H "Accept-Language: ja" http://localhost:8080/api/v1/products/

# Search in English
curl -s -H "Accept-Language: en" "http://localhost:8080/api/v1/search?keyword=gundam"
```

**Step 2:** Verify response structure:
- `message` field in errors should be in requested language
- Product `status`, `condition` should be localized display text
- Product `title` should pick from `TitleTranslated` when available
- `is_translated` flag should be correct

**Step 3: Final commit**

```bash
git add -A
git commit -m "feat: complete i18n language switching for all API responses"
```
