package output

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/rakutao/collection-gateway/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockESClient(statusCode int, body string) *elasticsearch.Client {
	client, _ := elasticsearch.NewClient(elasticsearch.Config{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: statusCode,
				Header: http.Header{
					"Content-Type":      []string{"application/json"},
					"X-Elastic-Product": []string{"Elasticsearch"},
				},
				Body: io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	})
	return client
}

func TestESSink_Write_Success(t *testing.T) {
	client := newMockESClient(200, `{"errors": false, "items": []}`)
	sink := NewESSink(client, "test_index")

	products := []domain.UnifiedProduct{
		{ID: "p1", Title: "Product 1", PriceJPY: 1000},
		{ID: "p2", Title: "Product 2", PriceJPY: 2000},
	}

	err := sink.Write(context.Background(), products)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
}

func TestESSink_Write_Empty(t *testing.T) {
	// Should not make any HTTP call for empty products
	sink := NewESSink(nil, "test_index")

	err := sink.Write(context.Background(), nil)
	if err != nil {
		t.Fatalf("Write error for empty: %v", err)
	}

	err = sink.Write(context.Background(), []domain.UnifiedProduct{})
	if err != nil {
		t.Fatalf("Write error for empty slice: %v", err)
	}
}

func TestESSink_Write_NilClient(t *testing.T) {
	sink := NewESSink(nil, "test_index")

	products := []domain.UnifiedProduct{
		{ID: "p1", Title: "Product 1"},
	}

	err := sink.Write(context.Background(), products)
	if err != nil {
		t.Fatalf("Write with nil client should be no-op, got error: %v", err)
	}
}

func TestESSink_Write_BulkError(t *testing.T) {
	client := newMockESClient(200, `{"errors": true, "items": [{"index": {"status": 400}}]}`)
	sink := NewESSink(client, "test_index")

	products := []domain.UnifiedProduct{
		{ID: "p1", Title: "Product 1"},
	}

	err := sink.Write(context.Background(), products)
	if err == nil {
		t.Fatal("expected error for bulk errors")
	}
}

func TestESSink_Write_ESError(t *testing.T) {
	client := newMockESClient(500, `{"error": "internal server error"}`)
	sink := NewESSink(client, "test_index")

	products := []domain.UnifiedProduct{
		{ID: "p1", Title: "Product 1"},
	}

	err := sink.Write(context.Background(), products)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestESSink_Name(t *testing.T) {
	sink := NewESSink(nil, "test_index")
	if sink.Name() != "elasticsearch" {
		t.Errorf("Name() = %q, want %q", sink.Name(), "elasticsearch")
	}
}
