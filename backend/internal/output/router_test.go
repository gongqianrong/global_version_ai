package output

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// --- Mock Sink ---

type mockSink struct {
	name     string
	mu       sync.Mutex
	received []domain.UnifiedProduct
	err      error
}

func newMockSink(name string, err error) *mockSink {
	return &mockSink{name: name, err: err}
}

func (m *mockSink) Name() string { return m.name }

func (m *mockSink) Write(ctx context.Context, products []domain.UnifiedProduct) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.received = append(m.received, products...)
	return m.err
}

func (m *mockSink) getReceived() []domain.UnifiedProduct {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.received
}

// --- Test 1: Router fans out to 2 sinks, both receive data ---

func TestRouter_FanOut_BothReceive(t *testing.T) {
	sink1 := newMockSink("sink1", nil)
	sink2 := newMockSink("sink2", nil)

	router := NewRouter(sink1, sink2)

	products := []domain.UnifiedProduct{
		{ID: "mercari_001", Title: "Product 1", PriceJPY: 1000},
		{ID: "mercari_002", Title: "Product 2", PriceJPY: 2000},
	}

	errs := router.Dispatch(context.Background(), products)

	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}

	got1 := sink1.getReceived()
	if len(got1) != 2 {
		t.Errorf("sink1 received %d products, want 2", len(got1))
	}

	got2 := sink2.getReceived()
	if len(got2) != 2 {
		t.Errorf("sink2 received %d products, want 2", len(got2))
	}
}

// --- Test 2: One sink fails, other still receives ---

func TestRouter_OneSinkFails_OtherReceives(t *testing.T) {
	sink1 := newMockSink("good_sink", nil)
	sink2 := newMockSink("bad_sink", errors.New("write failed"))

	router := NewRouter(sink1, sink2)

	products := []domain.UnifiedProduct{
		{ID: "mercari_001", Title: "Product 1", PriceJPY: 1000},
	}

	errs := router.Dispatch(context.Background(), products)

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Sink != "bad_sink" {
		t.Errorf("error sink = %q, want %q", errs[0].Sink, "bad_sink")
	}
	if errs[0].Err.Error() != "write failed" {
		t.Errorf("error message = %q, want %q", errs[0].Err.Error(), "write failed")
	}

	got1 := sink1.getReceived()
	if len(got1) != 1 {
		t.Errorf("good_sink received %d products, want 1", len(got1))
	}
}

// --- Test 3: AddSink for dynamic extension ---

func TestRouter_AddSink(t *testing.T) {
	sink1 := newMockSink("initial", nil)
	router := NewRouter(sink1)

	sink2 := newMockSink("dynamic", nil)
	router.AddSink(sink2)

	products := []domain.UnifiedProduct{
		{ID: "test_001", Title: "Test", PriceJPY: 500},
	}

	errs := router.Dispatch(context.Background(), products)

	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}

	got1 := sink1.getReceived()
	got2 := sink2.getReceived()
	if len(got1) != 1 || len(got2) != 1 {
		t.Errorf("both sinks should receive 1 product each; got sink1=%d, sink2=%d", len(got1), len(got2))
	}
}

// --- Test 4: Empty product slice ---

func TestRouter_EmptyProducts(t *testing.T) {
	sink1 := newMockSink("sink1", nil)
	router := NewRouter(sink1)

	errs := router.Dispatch(context.Background(), nil)
	if len(errs) != 0 {
		t.Errorf("expected 0 errors, got %d", len(errs))
	}
}

// --- Test 5: ESSink and RedisSink stubs ---

func TestESSink_Stub(t *testing.T) {
	es := &ESSink{}
	if es.Name() != "elasticsearch" {
		t.Errorf("Name() = %q, want %q", es.Name(), "elasticsearch")
	}
	err := es.Write(context.Background(), []domain.UnifiedProduct{{ID: "test"}})
	if err != nil {
		t.Errorf("Write() returned unexpected error: %v", err)
	}
}

func TestRedisSink_Stub(t *testing.T) {
	rs := &RedisSink{}
	if rs.Name() != "redis" {
		t.Errorf("Name() = %q, want %q", rs.Name(), "redis")
	}
	err := rs.Write(context.Background(), []domain.UnifiedProduct{{ID: "test"}})
	if err != nil {
		t.Errorf("Write() returned unexpected error: %v", err)
	}
}
