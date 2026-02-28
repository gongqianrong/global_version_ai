package output

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// ESSink writes UnifiedProduct documents to Elasticsearch using the Bulk API.
type ESSink struct {
	client    *elasticsearch.Client
	indexName string
}

// NewESSink creates an ESSink targeting the given index.
// If client is nil, Write is a no-op (useful for testing or when ES is not configured).
func NewESSink(client *elasticsearch.Client, indexName string) *ESSink {
	return &ESSink{client: client, indexName: indexName}
}

// Name returns the sink identifier.
func (s *ESSink) Name() string { return "elasticsearch" }

// Write bulk-indexes the given products into Elasticsearch.
func (s *ESSink) Write(ctx context.Context, products []domain.UnifiedProduct) error {
	if len(products) == 0 {
		return nil
	}
	if s.client == nil {
		return nil
	}

	var buf bytes.Buffer
	for _, p := range products {
		meta := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": s.indexName,
				"_id":    p.ID,
			},
		}
		metaLine, _ := json.Marshal(meta)
		buf.Write(metaLine)
		buf.WriteByte('\n')
		docLine, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("es sink: marshal product %s: %w", p.ID, err)
		}
		buf.Write(docLine)
		buf.WriteByte('\n')
	}

	res, err := s.client.Bulk(
		bytes.NewReader(buf.Bytes()),
		s.client.Bulk.WithContext(ctx),
		s.client.Bulk.WithIndex(s.indexName),
	)
	if err != nil {
		return fmt.Errorf("es sink: bulk request: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("es sink: bulk response: %s", res.Status())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("es sink: read bulk response: %w", err)
	}

	var bulkResp bulkResponse
	if err := json.Unmarshal(body, &bulkResp); err != nil {
		return fmt.Errorf("es sink: decode bulk response: %w", err)
	}

	if bulkResp.Errors {
		return fmt.Errorf("es sink: bulk indexing had errors")
	}

	return nil
}

type bulkResponse struct {
	Errors bool `json:"errors"`
}
