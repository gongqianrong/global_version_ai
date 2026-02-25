package registry

import (
	"context"
	"testing"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// mockAdapter is a minimal stub that satisfies domain.PlatformAdapter.
type mockAdapter struct {
	id   string
	caps domain.AdapterCaps
}

func (m *mockAdapter) PlatformID() string                { return m.id }
func (m *mockAdapter) Capabilities() domain.AdapterCaps  { return m.caps }
func (m *mockAdapter) Search(_ context.Context, _ domain.SearchQuery) (*domain.SearchResult, error) {
	return &domain.SearchResult{}, nil
}
func (m *mockAdapter) GetProduct(_ context.Context, _ string) (*domain.RawProduct, error) {
	return nil, nil
}
func (m *mockAdapter) BatchCollect(_ context.Context, _ domain.CollectParams) (<-chan domain.RawProduct, error) {
	return nil, nil
}
func (m *mockAdapter) HealthCheck(_ context.Context) domain.HealthStatus {
	return domain.HealthStatus{Status: "healthy"}
}

func TestRegisterAndGetAdapter(t *testing.T) {
	r := New()

	adapter := &mockAdapter{id: "mercari"}
	meta := PlatformMeta{
		ID:     "mercari",
		Name:   "メルカリ",
		NameEN: "Mercari",
		Icon:   "mercari.png",
		Type:   TypeDomesticProxy,
		Status: StatusActive,
		Caps: domain.AdapterCaps{
			SupportsSearch:   true,
			SupportsRealtime: true,
			MaxQPS:           10,
		},
	}

	r.Register(meta, adapter)

	got, err := r.GetAdapter("mercari")
	if err != nil {
		t.Fatalf("GetAdapter returned unexpected error: %v", err)
	}
	if got.PlatformID() != "mercari" {
		t.Errorf("PlatformID() = %q, want %q", got.PlatformID(), "mercari")
	}
}

func TestGetAdapterNotFound(t *testing.T) {
	r := New()

	_, err := r.GetAdapter("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent platform, got nil")
	}
}

func TestActivePlatformsFiltersOffline(t *testing.T) {
	r := New()

	r.Register(PlatformMeta{
		ID:     "mercari",
		Name:   "メルカリ",
		NameEN: "Mercari",
		Type:   TypeDomesticProxy,
		Status: StatusActive,
	}, &mockAdapter{id: "mercari"})

	r.Register(PlatformMeta{
		ID:     "yahoo",
		Name:   "ヤフオク",
		NameEN: "Yahoo Auctions",
		Type:   TypeDomesticProxy,
		Status: StatusOffline,
	}, &mockAdapter{id: "yahoo"})

	r.Register(PlatformMeta{
		ID:     "rakuten",
		Name:   "楽天",
		NameEN: "Rakuten",
		Type:   TypeSelfCrawler,
		Status: StatusDegraded,
	}, &mockAdapter{id: "rakuten"})

	active := r.ActivePlatforms()

	// Only platforms with StatusActive should be returned.
	if len(active) != 1 {
		t.Fatalf("ActivePlatforms() returned %d entries, want 1", len(active))
	}
	if active[0].ID != "mercari" {
		t.Errorf("ActivePlatforms()[0].ID = %q, want %q", active[0].ID, "mercari")
	}
}

func TestRealtimeSearchable(t *testing.T) {
	r := New()

	// Active + realtime capable
	r.Register(PlatformMeta{
		ID:     "mercari",
		Name:   "メルカリ",
		NameEN: "Mercari",
		Type:   TypeDomesticProxy,
		Status: StatusActive,
		Caps: domain.AdapterCaps{
			SupportsSearch:   true,
			SupportsRealtime: true,
		},
	}, &mockAdapter{id: "mercari", caps: domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: true}})

	// Active but NOT realtime capable (SupportsRealtime is false)
	r.Register(PlatformMeta{
		ID:     "rakuten",
		Name:   "楽天",
		NameEN: "Rakuten",
		Type:   TypeSelfCrawler,
		Status: StatusActive,
		Caps: domain.AdapterCaps{
			SupportsSearch:   true,
			SupportsRealtime: false,
		},
	}, &mockAdapter{id: "rakuten", caps: domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: false}})

	// Offline + realtime capable (should be excluded because offline)
	r.Register(PlatformMeta{
		ID:     "yahoo",
		Name:   "ヤフオク",
		NameEN: "Yahoo Auctions",
		Type:   TypeDomesticProxy,
		Status: StatusOffline,
		Caps: domain.AdapterCaps{
			SupportsSearch:   true,
			SupportsRealtime: true,
		},
	}, &mockAdapter{id: "yahoo", caps: domain.AdapterCaps{SupportsSearch: true, SupportsRealtime: true}})

	results := r.RealtimeSearchable()

	if len(results) != 1 {
		t.Fatalf("RealtimeSearchable() returned %d entries, want 1", len(results))
	}
	if results[0].ID != "mercari" {
		t.Errorf("RealtimeSearchable()[0].ID = %q, want %q", results[0].ID, "mercari")
	}
}
