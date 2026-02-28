package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"nhooyr.io/websocket"
)

// --- Mocks ---

type mockPlatformSearcher struct {
	results map[string][]domain.ProductSummary
	err     error
	delay   time.Duration
}

func (m *mockPlatformSearcher) SearchPlatform(_ context.Context, platformID string, _ domain.SearchQuery) ([]domain.ProductSummary, int64, error) {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
	if m.err != nil {
		return nil, 0, m.err
	}
	products := m.results[platformID]
	return products, int64(len(products)), nil
}

type mockPlatformLister struct {
	ids []string
}

func (m *mockPlatformLister) RealtimePlatformIDs() []string {
	return m.ids
}

// --- Tests ---

func TestRealtimeHandler_StreamNotFound(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()
	handler := NewRealtimeHandler(sm, &mockPlatformSearcher{}, &mockPlatformLister{})

	// Use chi router to extract URL param
	r := chi.NewRouter()
	r.Get("/stream/{streamID}", handler.HandleStream)

	req := httptest.NewRequest("GET", "/stream/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestRealtimeHandler_WebSocket_ReceivesResults(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	ps := &mockPlatformSearcher{
		results: map[string][]domain.ProductSummary{
			"yahoo_auction": {{ID: "y1", Title: "Product 1"}},
		},
	}
	pl := &mockPlatformLister{ids: []string{"yahoo_auction"}}
	handler := NewRealtimeHandler(sm, ps, pl)

	// Create stream
	streamID := sm.Create(domain.SearchQuery{Keyword: "test", KeywordJA: "テスト"})

	// Set up chi router + test server
	r := chi.NewRouter()
	r.Get("/stream/{streamID}", handler.HandleStream)
	server := httptest.NewServer(r)
	defer server.Close()

	// Connect WebSocket
	wsURL := "ws" + server.URL[4:] + "/stream/" + streamID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial error: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	// Collect all events
	var events []StreamEvent
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}
		var event StreamEvent
		json.Unmarshal(data, &event)
		events = append(events, event)
		if event.Type == EventDone {
			break
		}
	}

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events (results + done), got %d", len(events))
	}

	// First event should be results
	if events[0].Type != EventResults {
		t.Errorf("first event type = %q, want %q", events[0].Type, EventResults)
	}
	if events[0].Platform != "yahoo_auction" {
		t.Errorf("first event platform = %q, want %q", events[0].Platform, "yahoo_auction")
	}

	// Last event should be done
	last := events[len(events)-1]
	if last.Type != EventDone {
		t.Errorf("last event type = %q, want %q", last.Type, EventDone)
	}
}

func TestRealtimeHandler_NoPlatforms(t *testing.T) {
	sm := NewStreamManager()
	defer sm.Stop()

	pl := &mockPlatformLister{ids: []string{}}
	handler := NewRealtimeHandler(sm, &mockPlatformSearcher{}, pl)

	streamID := sm.Create(domain.SearchQuery{Keyword: "test"})

	r := chi.NewRouter()
	r.Get("/stream/{streamID}", handler.HandleStream)
	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] + "/stream/" + streamID
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("websocket dial error: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	var event StreamEvent
	json.Unmarshal(data, &event)
	if event.Type != EventDone {
		t.Errorf("event type = %q, want %q", event.Type, EventDone)
	}
}
