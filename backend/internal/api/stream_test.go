package api

import (
	"testing"
	"time"

	"github.com/rakutao/collection-gateway/internal/domain"
)

func TestStreamManager_Create(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	query := domain.SearchQuery{Keyword: "gucci"}
	id := sm.Create(query)

	if id == "" {
		t.Fatal("expected non-empty stream ID")
	}
	if sm.Count() != 1 {
		t.Errorf("count = %d, want 1", sm.Count())
	}
}

func TestStreamManager_Get_Found(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	query := domain.SearchQuery{Keyword: "gucci"}
	id := sm.Create(query)

	stream, ok := sm.Get(id)
	if !ok {
		t.Fatal("expected stream to be found")
	}
	if stream.Query.Keyword != "gucci" {
		t.Errorf("keyword = %q, want %q", stream.Query.Keyword, "gucci")
	}
}

func TestStreamManager_Get_NotFound(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	_, ok := sm.Get("nonexistent")
	if ok {
		t.Error("expected stream not to be found")
	}
}

func TestStreamManager_Remove(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	id := sm.Create(domain.SearchQuery{Keyword: "test"})
	sm.Remove(id)

	if sm.Count() != 0 {
		t.Errorf("count = %d, want 0 after remove", sm.Count())
	}
}

func TestStreamManager_UniqueIDs(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := sm.Create(domain.SearchQuery{Keyword: "test"})
		if ids[id] {
			t.Fatalf("duplicate stream ID: %s", id)
		}
		ids[id] = true
	}
}

func TestStream_Claim_FirstTime(t *testing.T) {
	s := &Stream{}
	if !s.Claim() {
		t.Error("expected first Claim() to return true")
	}
}

func TestStream_Claim_AlreadyClaimed(t *testing.T) {
	s := &Stream{}
	s.Claim()
	if s.Claim() {
		t.Error("expected second Claim() to return false")
	}
}

func TestStreamEvent_Channel(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	id := sm.Create(domain.SearchQuery{Keyword: "test"})
	stream, _ := sm.Get(id)

	// Send an event
	event := StreamEvent{
		Type:     EventResults,
		Platform: "yahoo_auction",
		Total:    10,
	}
	stream.Events <- event

	// Receive it
	select {
	case got := <-stream.Events:
		if got.Type != EventResults {
			t.Errorf("type = %q, want %q", got.Type, EventResults)
		}
		if got.Platform != "yahoo_auction" {
			t.Errorf("platform = %q, want %q", got.Platform, "yahoo_auction")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}
