# AI Service Integration Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Wire the Python AI service (translation + brand extraction) to the Go backend via HTTP, enabling keyword translation and AI-powered brand identification.

**Architecture:** Python FastAPI endpoints expose existing `TranslationService` and `BrandExtractor` classes. Go `aiclient` package implements `search.Translator` and `brand.AIExtractor` interfaces via HTTP calls. `main.go` wires everything together.

**Tech Stack:** Python 3.12 + FastAPI + uvicorn, Go 1.22 + net/http

---

### Task 1: Python — Install FastAPI + Create HTTP Endpoints

**Files:**
- Modify: `ai-service/setup.py`
- Create: `ai-service/app/main.py`
- Modify: `ai-service/Dockerfile`

**Step 1: Update `setup.py` to add FastAPI + uvicorn dependencies**

```python
from setuptools import setup, find_packages

setup(
    name="rakutao-ai-service",
    version="0.1.0",
    packages=find_packages(),
    install_requires=[
        "fastapi>=0.115.0",
        "uvicorn[standard]>=0.30.0",
    ],
    extras_require={"dev": ["pytest", "httpx"]},
)
```

**Step 2: Create `app/main.py` with FastAPI endpoints**

```python
from __future__ import annotations

from typing import Optional

from fastapi import FastAPI
from pydantic import BaseModel

from app.brand_extractor import BrandExtractor
from app.translation import TranslationService

app = FastAPI(title="Rakutao AI Service", version="0.1.0")

# Singletons
_translation_service = TranslationService(
    brand_mappings={
        "gucci": "グッチ",
        "chanel": "シャネル",
        "louis vuitton": "ルイ・ヴィトン",
        "prada": "プラダ",
        "hermes": "エルメス",
        "burberry": "バーバリー",
        "coach": "コーチ",
        "nike": "ナイキ",
        "adidas": "アディダス",
        "supreme": "シュプリーム",
        "balenciaga": "バレンシアガ",
        "dior": "ディオール",
        "fendi": "フェンディ",
        "celine": "セリーヌ",
    }
)
_brand_extractor = BrandExtractor()


# --- Request/Response models ---

class TranslateRequest(BaseModel):
    keyword: str
    source_lang: Optional[str] = None


class TranslateResponse(BaseModel):
    keyword_ja: str
    original: str
    source_lang: str
    was_translated: bool


class ExtractBrandRequest(BaseModel):
    title: str
    description: str = ""
    category: str = ""


class ExtractBrandResponse(BaseModel):
    brand_name: Optional[str] = None
    confidence: float = 0.0


class HealthResponse(BaseModel):
    status: str


# --- Endpoints ---

@app.post("/translate", response_model=TranslateResponse)
def translate(req: TranslateRequest) -> TranslateResponse:
    result = _translation_service.translate_keyword(
        keyword=req.keyword,
        source_lang=req.source_lang,
    )
    return TranslateResponse(
        keyword_ja=result.keyword_ja,
        original=result.original,
        source_lang=result.source_lang,
        was_translated=result.was_translated,
    )


@app.post("/extract-brand", response_model=ExtractBrandResponse)
def extract_brand(req: ExtractBrandRequest) -> ExtractBrandResponse:
    result = _brand_extractor.extract(
        title=req.title,
        description=req.description,
        category=req.category,
    )
    if result is None:
        return ExtractBrandResponse(brand_name=None, confidence=0.0)
    return ExtractBrandResponse(
        brand_name=result.brand_name,
        confidence=result.confidence,
    )


@app.get("/health", response_model=HealthResponse)
def health() -> HealthResponse:
    return HealthResponse(status="healthy")
```

**Step 3: Update `Dockerfile`**

```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY setup.py .
COPY app/ app/
RUN pip install --no-cache-dir .
EXPOSE 8000
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8000"]
```

**Verify:**
```bash
cd /Users/gongqianrong/Desktop/ai/ai-service && pip3 install -e ".[dev]" && python3 -m pytest tests/ -v
```

Expected: 25 existing tests PASS.

---

### Task 2: Python — Add FastAPI Endpoint Tests

**Files:**
- Create: `ai-service/tests/test_api.py`

**Step 1: Create endpoint tests using FastAPI TestClient**

```python
from __future__ import annotations

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)


class TestTranslateEndpoint:
    def test_translate_english_keyword(self) -> None:
        resp = client.post("/translate", json={
            "keyword": "gucci bag",
            "source_lang": "en",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["original"] == "gucci bag"
        assert data["source_lang"] == "en"
        assert data["was_translated"] is True
        assert "グッチ" in data["keyword_ja"]

    def test_translate_japanese_keyword(self) -> None:
        resp = client.post("/translate", json={
            "keyword": "グッチ バッグ",
            "source_lang": "ja",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["was_translated"] is False
        assert data["keyword_ja"] == "グッチ バッグ"

    def test_translate_auto_detect(self) -> None:
        resp = client.post("/translate", json={
            "keyword": "グッチ バッグ",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["source_lang"] == "ja"
        assert data["was_translated"] is False

    def test_translate_missing_keyword(self) -> None:
        resp = client.post("/translate", json={})
        assert resp.status_code == 422


class TestExtractBrandEndpoint:
    def test_extract_brand_from_title(self) -> None:
        resp = client.post("/extract-brand", json={
            "title": "GUCCI バッグ 新品",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["brand_name"] == "GUCCI"
        assert data["confidence"] == 0.95

    def test_extract_brand_from_description(self) -> None:
        resp = client.post("/extract-brand", json={
            "title": "高級バッグ",
            "description": "CHANEL製",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["brand_name"] == "CHANEL"
        assert data["confidence"] == 0.75

    def test_extract_brand_no_match(self) -> None:
        resp = client.post("/extract-brand", json={
            "title": "手作りバッグ",
        })
        assert resp.status_code == 200
        data = resp.json()
        assert data["brand_name"] is None
        assert data["confidence"] == 0.0


class TestHealthEndpoint:
    def test_health(self) -> None:
        resp = client.get("/health")
        assert resp.status_code == 200
        assert resp.json() == {"status": "healthy"}
```

**Verify:**
```bash
cd /Users/gongqianrong/Desktop/ai/ai-service && python3 -m pytest tests/ -v
```

Expected: 25 existing + 8 new = 33 tests PASS.

---

### Task 3: Go — Create aiclient Package

**Files:**
- Create: `backend/internal/aiclient/client.go`
- Create: `backend/internal/aiclient/client_test.go`

**Step 1: Create `client.go`**

```go
package aiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/rakutao/collection-gateway/internal/brand"
)

// Client is an HTTP client for the Rakutao AI service.
// It implements search.Translator and brand.AIExtractor interfaces.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// New creates an AI service client. If httpClient is nil, a default
// client with a 10-second timeout is used.
func New(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Client{baseURL: baseURL, httpClient: httpClient}
}

// --- Translator interface ---

type translateRequest struct {
	Keyword    string `json:"keyword"`
	SourceLang string `json:"source_lang,omitempty"`
}

type translateResponse struct {
	KeywordJA     string `json:"keyword_ja"`
	Original      string `json:"original"`
	SourceLang    string `json:"source_lang"`
	WasTranslated bool   `json:"was_translated"`
}

// Translate sends a keyword to the AI service for translation to Japanese.
func (c *Client) Translate(ctx context.Context, keyword, sourceLang string) (string, error) {
	reqBody := translateRequest{
		Keyword:    keyword,
		SourceLang: sourceLang,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("aiclient: marshal translate request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/translate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("aiclient: create translate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("aiclient: translate request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("aiclient: translate status %d", resp.StatusCode)
	}

	var result translateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("aiclient: decode translate response: %w", err)
	}

	return result.KeywordJA, nil
}

// --- AIExtractor interface ---

type extractBrandRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

type extractBrandResponse struct {
	BrandName  *string `json:"brand_name"`
	Confidence float64 `json:"confidence"`
}

// Extract sends product text to the AI service for brand identification.
func (c *Client) Extract(title, description, category string) (*brand.AIExtractionResult, error) {
	reqBody := extractBrandRequest{
		Title:       title,
		Description: description,
		Category:    category,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("aiclient: marshal extract request: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.baseURL+"/extract-brand", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("aiclient: create extract request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aiclient: extract request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aiclient: extract status %d", resp.StatusCode)
	}

	var result extractBrandResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("aiclient: decode extract response: %w", err)
	}

	if result.BrandName == nil {
		return nil, nil
	}

	return &brand.AIExtractionResult{
		BrandName:  *result.BrandName,
		Confidence: result.Confidence,
	}, nil
}
```

**Step 2: Create `client_test.go`**

```go
package aiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranslate_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/translate" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req translateRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Keyword != "gucci bag" {
			t.Errorf("keyword = %q, want %q", req.Keyword, "gucci bag")
		}
		if req.SourceLang != "en" {
			t.Errorf("source_lang = %q, want %q", req.SourceLang, "en")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(translateResponse{
			KeywordJA:     "グッチ bag",
			Original:      "gucci bag",
			SourceLang:    "en",
			WasTranslated: true,
		})
	}))
	defer srv.Close()

	client := New(srv.URL, nil)
	result, err := client.Translate(context.Background(), "gucci bag", "en")
	if err != nil {
		t.Fatalf("Translate error: %v", err)
	}
	if result != "グッチ bag" {
		t.Errorf("result = %q, want %q", result, "グッチ bag")
	}
}

func TestTranslate_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, nil)
	_, err := client.Translate(context.Background(), "test", "en")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestTranslate_ConnectionError(t *testing.T) {
	client := New("http://localhost:1", nil)
	_, err := client.Translate(context.Background(), "test", "en")
	if err == nil {
		t.Fatal("expected error for connection failure")
	}
}

func TestExtract_BrandFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/extract-brand" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		var req extractBrandRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Title != "GUCCI バッグ" {
			t.Errorf("title = %q", req.Title)
		}

		brandName := "GUCCI"
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(extractBrandResponse{
			BrandName:  &brandName,
			Confidence: 0.95,
		})
	}))
	defer srv.Close()

	client := New(srv.URL, nil)
	result, err := client.Extract("GUCCI バッグ", "新品", "ファッション")
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.BrandName != "GUCCI" {
		t.Errorf("BrandName = %q", result.BrandName)
	}
	if result.Confidence != 0.95 {
		t.Errorf("Confidence = %f", result.Confidence)
	}
}

func TestExtract_NoBrand(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(extractBrandResponse{
			BrandName:  nil,
			Confidence: 0.0,
		})
	}))
	defer srv.Close()

	client := New(srv.URL, nil)
	result, err := client.Extract("手作りバッグ", "", "")
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for no brand, got %+v", result)
	}
}

func TestExtract_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := New(srv.URL, nil)
	_, err := client.Extract("test", "", "")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
```

**Verify:**
```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./internal/aiclient/... -v
```

Expected: 6 tests PASS.

---

### Task 4: Go — Wire AI Client in main.go

**Files:**
- Modify: `backend/cmd/gateway/main.go`

**Step 1: Update main.go to wire AI client**

Add imports:
```go
"github.com/rakutao/collection-gateway/internal/aiclient"
"github.com/rakutao/collection-gateway/internal/brand"
```

Add AI client initialization after keyword filter setup:

```go
// AI service client (translation + brand extraction).
aiServiceURL := os.Getenv("AI_SERVICE_URL")
if aiServiceURL == "" {
    aiServiceURL = "http://localhost:8000"
}
aiClient := aiclient.New(aiServiceURL, nil)
```

Update Gateway to use aiClient:
```go
gateway := search.NewGateway(aiClient, keywordFilter)
```

Add brand pipeline wiring:
```go
// Brand pipeline (rule-based + AI extraction).
brandRegistry := brand.DefaultRegistry()
brandPipeline := brand.NewPipeline(brandRegistry, aiClient)
brandAdapter := brand.NewPipelineAdapter(brandPipeline)

// Normalizer with brand extraction.
norm := normalizer.New(brandAdapter)
```

Add log line:
```go
log.Printf("  AI Service:        %s", aiServiceURL)
```

**Verify:**
```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go build ./...
```

Expected: compiles cleanly.

---

### Task 5: Full Verification

**Step 1: Run all Go tests**

```bash
export PATH="$HOME/go-sdk/go/bin:$PATH"
cd /Users/gongqianrong/Desktop/ai/backend && go test ./... -v -race
```

Expected: all packages PASS (including new aiclient package).

**Step 2: Run all Python tests**

```bash
cd /Users/gongqianrong/Desktop/ai/ai-service && python3 -m pytest tests/ -v
```

Expected: 33 tests PASS (25 existing + 8 new).
