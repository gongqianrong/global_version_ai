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

// --- Translator interface (search.Translator) ---

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

// --- AIExtractor interface (brand.AIExtractor) ---

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
