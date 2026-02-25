package output

import (
	"context"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// RedisSink is a stub Redis output sink. The Write method is a no-op for now
// and will be wired to a real Redis client in a later task.
type RedisSink struct{}

// Name returns the sink identifier.
func (s *RedisSink) Name() string { return "redis" }

// Write persists products to Redis. Currently a no-op stub.
func (s *RedisSink) Write(ctx context.Context, products []domain.UnifiedProduct) error {
	return nil
}
