package api

import (
	"context"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/normalizer"
	"github.com/rakutao/collection-gateway/internal/registry"
)

// testAdapter is a minimal PlatformAdapter for testing PlatformSearchService.
type testAdapter struct {
	id     string
	result *domain.SearchResult
	err    error
}

func (a *testAdapter) PlatformID() string { return a.id }
func (a *testAdapter) Capabilities() domain.AdapterCaps {
	return domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: true}
}
func (a *testAdapter) Search(_ context.Context, _ domain.SearchQuery) (*domain.SearchResult, error) {
	return a.result, a.err
}
func (a *testAdapter) GetProduct(_ context.Context, _ string) (*domain.RawProduct, error) {
	return nil, nil
}
func (a *testAdapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) {
	return nil, nil
}
func (a *testAdapter) HealthCheck(_ context.Context) domain.HealthStatus {
	return domain.HealthStatus{Status: "healthy"}
}

func TestPlatformSearchService_SearchPlatform(t *testing.T) {
	reg := registry.New()
	adapter := &testAdapter{
		id: "yahoo_auction",
		result: &domain.SearchResult{
			Products: []domain.RawProduct{
				{
					Platform: "yahoo_auction",
					RawID:    "ya-001",
					RawData: map[string]interface{}{
						"title": "グッチ バッグ",
						"price": float64(15000),
					},
				},
			},
			Total: 1,
		},
	}
	reg.Register(registry.PlatformMeta{
		ID:     "yahoo_auction",
		Name:   "ヤフオク",
		Status: registry.StatusActive,
		Caps:   adapter.Capabilities(),
	}, adapter)

	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	summaries, total, err := svc.SearchPlatform(context.Background(), "yahoo_auction", domain.SearchQuery{
		Keyword:  "gucci",
		Page:     1,
		PageSize: 20,
	})
	if err != nil {
		t.Fatalf("SearchPlatform error: %v", err)
	}

	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(summaries) != 1 {
		t.Fatalf("len(summaries) = %d, want 1", len(summaries))
	}
	if summaries[0].ID != "yahoo_auction_ya-001" {
		t.Errorf("ID = %q, want %q", summaries[0].ID, "yahoo_auction_ya-001")
	}
	if summaries[0].Title != "グッチ バッグ" {
		t.Errorf("Title = %q", summaries[0].Title)
	}
	// Price should include 10% service fee: 15000 + 1500 = 16500
	if summaries[0].PriceJPY != 16500 {
		t.Errorf("PriceJPY = %d, want 16500 (15000 + 10%% fee)", summaries[0].PriceJPY)
	}
	if summaries[0].Platform != "yahoo_auction" {
		t.Errorf("Platform = %q", summaries[0].Platform)
	}
}

func TestPlatformSearchService_SearchPlatform_NotFound(t *testing.T) {
	reg := registry.New()
	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	_, _, err := svc.SearchPlatform(context.Background(), "nonexistent", domain.SearchQuery{})
	if err == nil {
		t.Fatal("expected error for unregistered platform")
	}
}

func TestPlatformSearchService_SearchPlatform_SkipsBadProducts(t *testing.T) {
	reg := registry.New()
	adapter := &testAdapter{
		id: "yahoo_auction",
		result: &domain.SearchResult{
			Products: []domain.RawProduct{
				{Platform: "yahoo_auction", RawID: "good", RawData: map[string]interface{}{"title": "Valid", "price": float64(1000)}},
				{Platform: "yahoo_auction", RawID: "bad", RawData: map[string]interface{}{"title": ""}}, // missing title
				{Platform: "yahoo_auction", RawID: "good2", RawData: map[string]interface{}{"title": "Also Valid", "price": float64(2000)}},
			},
			Total: 3,
		},
	}
	reg.Register(registry.PlatformMeta{
		ID:     "yahoo_auction",
		Status: registry.StatusActive,
		Caps:   adapter.Capabilities(),
	}, adapter)

	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	summaries, _, err := svc.SearchPlatform(context.Background(), "yahoo_auction", domain.SearchQuery{Keyword: "test", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("len(summaries) = %d, want 2 (bad product should be skipped)", len(summaries))
	}
}

func TestPlatformSearchService_GetProduct_Success(t *testing.T) {
	reg := registry.New()
	adapter := &testAdapter{
		id: "surugaya",
		result: &domain.SearchResult{
			Products: []domain.RawProduct{
				{Platform: "surugaya", RawID: "663043159", RawData: map[string]interface{}{"title": "Test", "price": float64(4950)}},
			},
			Total: 1,
		},
	}
	// Override GetProduct to return a valid product.
	productAdapter := &testAdapterWithProduct{
		testAdapter: *adapter,
		product: &domain.RawProduct{
			Platform: "surugaya",
			RawID:    "663043159",
			RawData: map[string]interface{}{
				"title": "FW GUNDAM CONVERGE SB ニカーヤ",
				"price": float64(4950),
			},
		},
	}
	reg.Register(registry.PlatformMeta{
		ID:     "surugaya",
		Status: registry.StatusActive,
		Caps:   adapter.Capabilities(),
	}, productAdapter)

	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	product, err := svc.GetProduct(context.Background(), "surugaya_663043159")
	if err != nil {
		t.Fatalf("GetProduct error: %v", err)
	}
	if product.ID != "surugaya_663043159" {
		t.Errorf("ID = %q, want %q", product.ID, "surugaya_663043159")
	}
	if product.Title != "FW GUNDAM CONVERGE SB ニカーヤ" {
		t.Errorf("Title = %q", product.Title)
	}
	// 4950 + 10% = 5445
	if product.PriceJPY != 5445 {
		t.Errorf("PriceJPY = %d, want 5445", product.PriceJPY)
	}
}

func TestPlatformSearchService_GetProduct_InvalidID(t *testing.T) {
	reg := registry.New()
	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	_, err := svc.GetProduct(context.Background(), "nounderscore")
	if err == nil {
		t.Fatal("expected error for invalid ID format")
	}
}

func TestPlatformSearchService_GetProduct_PlatformNotFound(t *testing.T) {
	reg := registry.New()
	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	_, err := svc.GetProduct(context.Background(), "nonexistent_123")
	if err == nil {
		t.Fatal("expected error for unregistered platform")
	}
}

// testAdapterWithProduct extends testAdapter with a GetProduct implementation.
type testAdapterWithProduct struct {
	testAdapter
	product *domain.RawProduct
	err     error
}

func (a *testAdapterWithProduct) GetProduct(_ context.Context, _ string) (*domain.RawProduct, error) {
	return a.product, a.err
}

func TestPlatformSearchService_RealtimePlatformIDs(t *testing.T) {
	reg := registry.New()

	// Register two adapters: one realtime-capable, one not
	realtimeAdapter := &testAdapter{id: "yahoo_auction"}
	nonRealtimeAdapter := &testAdapter{id: "static_catalog"}

	reg.Register(registry.PlatformMeta{
		ID:     "yahoo_auction",
		Status: registry.StatusActive,
		Caps:   domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: true},
	}, realtimeAdapter)

	reg.Register(registry.PlatformMeta{
		ID:     "static_catalog",
		Status: registry.StatusActive,
		Caps:   domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: false},
	}, nonRealtimeAdapter)

	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	ids := svc.RealtimePlatformIDs()
	if len(ids) != 1 {
		t.Fatalf("len(ids) = %d, want 1", len(ids))
	}
	if ids[0] != "yahoo_auction" {
		t.Errorf("ids[0] = %q, want %q", ids[0], "yahoo_auction")
	}
}

func TestPlatformSearchService_RealtimePlatformIDs_Empty(t *testing.T) {
	reg := registry.New()
	norm := normalizer.New(nil)
	svc := NewPlatformSearchService(reg, norm)

	ids := svc.RealtimePlatformIDs()
	if len(ids) != 0 {
		t.Errorf("len(ids) = %d, want 0", len(ids))
	}
}
