package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Google implements Translator using Google Cloud Translation API v2.
type Google struct {
	client *http.Client
	apiKey string
}

// NewGoogle creates a Google Translate translator.
// apiKey is the Google Cloud API key with Translation API enabled.
func NewGoogle(apiKey string, client *http.Client) *Google {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &Google{client: client, apiKey: apiKey}
}

const googleTranslateURL = "https://translation.googleapis.com/language/translate/v2"

type googleRequest struct {
	Q      []string `json:"q"`
	Source string   `json:"source"`
	Target string   `json:"target"`
	Format string   `json:"format"`
}

type googleResponse struct {
	Data struct {
		Translations []struct {
			TranslatedText string `json:"translatedText"`
		} `json:"translations"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

// Translate translates a single text from sourceLang to targetLang.
func (g *Google) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}
	results, err := g.BatchTranslate(ctx, []string{text}, sourceLang, targetLang)
	if err != nil {
		return "", err
	}
	return results[0], nil
}

// BatchTranslate translates multiple texts in a single API call.
func (g *Google) BatchTranslate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Filter out empty texts, remember positions.
	var nonEmpty []string
	indices := make([]int, 0, len(texts))
	for i, t := range texts {
		if t != "" {
			nonEmpty = append(nonEmpty, t)
			indices = append(indices, i)
		}
	}
	if len(nonEmpty) == 0 {
		return make([]string, len(texts)), nil
	}

	body, err := json.Marshal(googleRequest{
		Q:      nonEmpty,
		Source: sourceLang,
		Target: targetLang,
		Format: "text",
	})
	if err != nil {
		return nil, fmt.Errorf("google translate: marshal: %w", err)
	}

	url := googleTranslateURL + "?key=" + g.apiKey
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("google translate: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google translate: request: %w", err)
	}
	defer resp.Body.Close()

	var result googleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("google translate: decode: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("google translate: %s (code %d)", result.Error.Message, result.Error.Code)
	}

	if len(result.Data.Translations) != len(nonEmpty) {
		return nil, fmt.Errorf("google translate: expected %d results, got %d", len(nonEmpty), len(result.Data.Translations))
	}

	// Map results back to original positions.
	out := make([]string, len(texts))
	for i, idx := range indices {
		out[idx] = result.Data.Translations[i].TranslatedText
	}
	return out, nil
}
