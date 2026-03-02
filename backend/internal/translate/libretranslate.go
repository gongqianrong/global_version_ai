package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// LibreTranslate implements Translator using a self-hosted LibreTranslate API.
type LibreTranslate struct {
	client  *http.Client
	baseURL string
}

// NewLibreTranslate creates a LibreTranslate translator.
// baseURL is the LibreTranslate server address, e.g. "http://localhost:5000".
func NewLibreTranslate(baseURL string, client *http.Client) *LibreTranslate {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return &LibreTranslate{client: client, baseURL: baseURL}
}

type libreRequest struct {
	Q      string `json:"q"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type libreResponse struct {
	TranslatedText string `json:"translatedText"`
	Error          string `json:"error,omitempty"`
}

// langMap maps our internal language codes to LibreTranslate language codes.
var langMap = map[string]string{
	"ja":    "ja",
	"en":    "en",
	"zh-TW": "zh-Hans",
}

func mapLang(lang string) string {
	if mapped, ok := langMap[lang]; ok {
		return mapped
	}
	return lang
}

// Translate translates a single text from sourceLang to targetLang.
func (lt *LibreTranslate) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	body, err := json.Marshal(libreRequest{
		Q:      text,
		Source: mapLang(sourceLang),
		Target: mapLang(targetLang),
	})
	if err != nil {
		return "", fmt.Errorf("libretranslate: marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, lt.baseURL+"/translate", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("libretranslate: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := lt.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("libretranslate: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("libretranslate: status %d", resp.StatusCode)
	}

	var result libreResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("libretranslate: decode: %w", err)
	}

	if result.Error != "" {
		return "", fmt.Errorf("libretranslate: %s", result.Error)
	}

	return result.TranslatedText, nil
}

// BatchTranslate translates multiple texts sequentially.
func (lt *LibreTranslate) BatchTranslate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	results := make([]string, len(texts))
	for i, text := range texts {
		translated, err := lt.Translate(ctx, text, sourceLang, targetLang)
		if err != nil {
			return nil, fmt.Errorf("libretranslate: batch item %d: %w", i, err)
		}
		results[i] = translated
	}
	return results, nil
}
