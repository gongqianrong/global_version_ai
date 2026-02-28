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
