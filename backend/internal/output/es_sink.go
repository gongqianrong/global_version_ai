package output

import (
	"context"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// ESSink is a stub Elasticsearch output sink. The Write method is a no-op
// for now and will be wired to a real ES client in a later task.
type ESSink struct {
	indexName string
}

// NewESSink creates an ESSink that targets the given Elasticsearch index.
func NewESSink(indexName string) *ESSink {
	return &ESSink{indexName: indexName}
}

// Name returns the sink identifier.
func (s *ESSink) Name() string { return "elasticsearch" }

// Write persists products to Elasticsearch. Currently a no-op stub.
func (s *ESSink) Write(ctx context.Context, products []domain.UnifiedProduct) error {
	return nil
}
