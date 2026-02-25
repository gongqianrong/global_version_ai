// Package output provides a fan-out router that dispatches normalised products
// to one or more output sinks (Elasticsearch, Redis, future WMS, etc.).
package output

import (
	"context"
	"sync"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// Sink is the interface that output destinations must implement.
type Sink interface {
	// Name returns a human-readable identifier for the sink (e.g. "elasticsearch").
	Name() string

	// Write persists the given products to the output destination.
	Write(ctx context.Context, products []domain.UnifiedProduct) error
}

// SinkError pairs a sink name with the error it produced during dispatch.
type SinkError struct {
	Sink string
	Err  error
}

// Router fans out products to all registered sinks concurrently.
type Router struct {
	mu    sync.RWMutex
	sinks []Sink
}

// NewRouter creates a Router pre-loaded with the given sinks.
func NewRouter(sinks ...Sink) *Router {
	return &Router{
		sinks: sinks,
	}
}

// AddSink registers an additional sink for dynamic extension (e.g. future WMS).
func (r *Router) AddSink(sink Sink) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sinks = append(r.sinks, sink)
}

// Dispatch sends the products to every registered sink concurrently.
// It collects and returns any errors that occur.
func (r *Router) Dispatch(ctx context.Context, products []domain.UnifiedProduct) []SinkError {
	r.mu.RLock()
	sinks := make([]Sink, len(r.sinks))
	copy(sinks, r.sinks)
	r.mu.RUnlock()

	if len(sinks) == 0 || len(products) == 0 {
		return nil
	}

	var (
		mu     sync.Mutex
		errors []SinkError
		wg     sync.WaitGroup
	)

	for _, s := range sinks {
		wg.Add(1)
		go func(sink Sink) {
			defer wg.Done()
			if err := sink.Write(ctx, products); err != nil {
				mu.Lock()
				errors = append(errors, SinkError{
					Sink: sink.Name(),
					Err:  err,
				})
				mu.Unlock()
			}
		}(s)
	}

	wg.Wait()
	return errors
}
