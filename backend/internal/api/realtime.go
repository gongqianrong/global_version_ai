package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"nhooyr.io/websocket"
)

// PlatformSearcher can search a specific platform and normalize results.
type PlatformSearcher interface {
	// SearchPlatform searches a single platform and returns normalized product summaries.
	SearchPlatform(ctx context.Context, platformID string, query domain.SearchQuery) ([]domain.ProductSummary, int64, error)
}

// PlatformLister lists platforms that support real-time search.
type PlatformLister interface {
	RealtimePlatformIDs() []string
}

// RealtimeHandler handles WebSocket connections for real-time search results.
type RealtimeHandler struct {
	streamManager    *StreamManager
	platformSearcher PlatformSearcher
	platformLister   PlatformLister
}

// NewRealtimeHandler creates a RealtimeHandler with the given dependencies.
func NewRealtimeHandler(sm *StreamManager, ps PlatformSearcher, pl PlatformLister) *RealtimeHandler {
	return &RealtimeHandler{
		streamManager:    sm,
		platformSearcher: ps,
		platformLister:   pl,
	}
}

// HandleStream handles WS /api/v1/search/stream/{streamID}.
func (h *RealtimeHandler) HandleStream(w http.ResponseWriter, r *http.Request) {
	streamID := chi.URLParam(r, "streamID")
	if streamID == "" {
		http.Error(w, "missing stream ID", http.StatusBadRequest)
		return
	}

	stream, ok := h.streamManager.Get(streamID)
	if !ok {
		http.Error(w, "stream not found or expired", http.StatusNotFound)
		return
	}

	if !stream.Claim() {
		http.Error(w, "stream already claimed", http.StatusConflict)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // Allow all origins for now
	})
	if err != nil {
		log.Printf("[WS] accept error: %v", err)
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	// Start platform searches in background.
	go h.searchPlatforms(stream)

	// Read events from stream and write to WebSocket.
	ctx, cancel := context.WithTimeout(r.Context(), StreamSearchTimeout+2*time.Second)
	defer cancel()

	for {
		select {
		case event, ok := <-stream.Events:
			if !ok {
				return // channel closed
			}
			data, _ := json.Marshal(event)
			if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
				log.Printf("[WS] write error: %v", err)
				return
			}
			if event.Type == EventDone {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// searchPlatforms concurrently searches all real-time platforms and sends results to the stream.
func (h *RealtimeHandler) searchPlatforms(stream *Stream) {
	platformIDs := h.platformLister.RealtimePlatformIDs()
	if len(platformIDs) == 0 {
		stream.Events <- StreamEvent{Type: EventDone, Platforms: []string{}}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), StreamSearchTimeout)
	defer cancel()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var searched []string

	for _, pid := range platformIDs {
		wg.Add(1)
		go func(platformID string) {
			defer wg.Done()

			products, total, err := h.platformSearcher.SearchPlatform(ctx, platformID, stream.Query)

			mu.Lock()
			searched = append(searched, platformID)
			mu.Unlock()

			if err != nil {
				stream.Events <- StreamEvent{
					Type:     EventError,
					Platform: platformID,
					Message:  err.Error(),
				}
				return
			}

			stream.Events <- StreamEvent{
				Type:     EventResults,
				Platform: platformID,
				Products: products,
				Total:    total,
			}
		}(pid)
	}

	wg.Wait()

	stream.Events <- StreamEvent{
		Type:      EventDone,
		Platforms: searched,
	}
}
