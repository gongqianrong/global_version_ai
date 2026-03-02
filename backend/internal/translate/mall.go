package translate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/longbridgeapp/opencc"
)

// Mall implements Translator using the internal mall-translate service (AWS Translate).
type Mall struct {
	client  *http.Client
	baseURL string
	s2t     *opencc.OpenCC
}

// NewMall creates a Mall translator.
// baseURL example: "http://cloud.gamestudio.tech/mall-translate"
func NewMall(baseURL string, client *http.Client) *Mall {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	s2t, _ := opencc.New("s2t")
	return &Mall{client: client, baseURL: baseURL, s2t: s2t}
}

type mallResponse struct {
	Code string `json:"code"`
	Data string `json:"data"`
	Msg  string `json:"msg"`
}

// mapTarget maps our internal language codes to mall-translate target codes.
// zh-TW is translated via zh-CN + opencc s2t conversion.
func mallTarget(lang string) string {
	switch lang {
	case "zh-TW":
		return "zh-CN"
	default:
		return lang
	}
}

// Translate translates a single text from sourceLang to targetLang.
func (m *Mall) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	endpoint := m.baseURL + "/admin-api/transResultByAdmin"
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("mall translate: parse url: %w", err)
	}

	q := u.Query()
	q.Set("sourceText", text)
	q.Set("target", mallTarget(targetLang))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("mall translate: build request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("mall translate: request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mall translate: status %d", resp.StatusCode)
	}

	var result mallResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("mall translate: decode: %w", err)
	}

	if result.Code != "000000" {
		return "", fmt.Errorf("mall translate: %s (code %s)", result.Msg, result.Code)
	}

	// Convert simplified Chinese to traditional Chinese for zh-TW.
	if targetLang == "zh-TW" && m.s2t != nil {
		converted, err := m.s2t.Convert(result.Data)
		if err == nil {
			return converted, nil
		}
	}

	return result.Data, nil
}

// BatchTranslate translates multiple texts sequentially.
func (m *Mall) BatchTranslate(ctx context.Context, texts []string, sourceLang, targetLang string) ([]string, error) {
	results := make([]string, len(texts))
	for i, text := range texts {
		translated, err := m.Translate(ctx, text, sourceLang, targetLang)
		if err != nil {
			return nil, fmt.Errorf("mall translate: batch item %d: %w", i, err)
		}
		results[i] = translated
	}
	return results, nil
}
