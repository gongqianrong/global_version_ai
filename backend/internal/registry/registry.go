// Package registry provides a thread-safe registry for platform adapters,
// allowing the gateway to look up adapters by platform ID and query which
// platforms are active or support realtime search.
package registry

import (
	"fmt"
	"sync"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// Platform type constants describe how the adapter connects to its source.
const (
	TypeDomesticProxy = "domestic_proxy" // Proxies requests to a domestic version API
	TypeSelfCrawler   = "self_crawler"   // Crawls the platform directly
	TypeSelfAPI       = "self_api"       // Uses the platform's official API
)

// Platform status constants.
const (
	StatusActive   = "active"
	StatusDegraded = "degraded"
	StatusOffline  = "offline"
)

// PlatformMeta holds descriptive metadata about a registered platform.
type PlatformMeta struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	NameEN string           `json:"name_en"`
	Icon   string           `json:"icon"`
	Type   string           `json:"type"`
	Status string           `json:"status"`
	Caps   domain.AdapterCaps `json:"caps"`
}

// entry bundles a platform's metadata with its adapter implementation.
type entry struct {
	meta    PlatformMeta
	adapter domain.PlatformAdapter
}

// Registry is a thread-safe store of platform adapters keyed by platform ID.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]entry
}

// New creates an empty Registry ready for use.
func New() *Registry {
	return &Registry{
		entries: make(map[string]entry),
	}
}

// Register adds a platform adapter to the registry under the given metadata.
// If a platform with the same ID already exists it is silently replaced.
func (r *Registry) Register(meta PlatformMeta, adapter domain.PlatformAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries[meta.ID] = entry{meta: meta, adapter: adapter}
}

// GetAdapter returns the adapter for the given platform ID, or an error if the
// platform is not registered.
func (r *Registry) GetAdapter(platformID string) (domain.PlatformAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.entries[platformID]
	if !ok {
		return nil, fmt.Errorf("registry: platform %q not found", platformID)
	}
	return e.adapter, nil
}

// ActivePlatforms returns metadata for every registered platform whose status
// is StatusActive.
func (r *Registry) ActivePlatforms() []PlatformMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []PlatformMeta
	for _, e := range r.entries {
		if e.meta.Status == StatusActive {
			out = append(out, e.meta)
		}
	}
	return out
}

// RealtimeSearchable returns metadata for every registered platform that is
// active AND whose capabilities indicate realtime search support.
func (r *Registry) RealtimeSearchable() []PlatformMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []PlatformMeta
	for _, e := range r.entries {
		if e.meta.Status == StatusActive && e.meta.Caps.CanRealtimeSearch() {
			out = append(out, e.meta)
		}
	}
	return out
}
