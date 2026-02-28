package api

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/rakutao/collection-gateway/internal/domain"
)

const (
	// StreamTTL is how long a stream remains valid after creation.
	StreamTTL = 30 * time.Second
	// StreamSearchTimeout is the max time for all platform searches.
	StreamSearchTimeout = 5 * time.Second
	// StreamEventBufferSize is the channel buffer size for stream events.
	StreamEventBufferSize = 50
)

// StreamEventType identifies the kind of event sent over a search stream.
type StreamEventType string

const (
	EventResults StreamEventType = "results"
	EventDone    StreamEventType = "done"
	EventError   StreamEventType = "error"
)

// StreamEvent is a single event pushed to a WebSocket client during real-time search.
type StreamEvent struct {
	Type      StreamEventType         `json:"type"`
	Platform  string                  `json:"platform,omitempty"`
	Products  []domain.ProductSummary `json:"products,omitempty"`
	Total     int64                   `json:"total,omitempty"`
	Platforms []string                `json:"platforms_searched,omitempty"`
	Message   string                  `json:"message,omitempty"`
}

// Stream represents an active real-time search session.
type Stream struct {
	ID        string
	Query     domain.SearchQuery
	Events    chan StreamEvent
	CreatedAt time.Time
	claimed   bool
	mu        sync.Mutex
}

// Claim marks the stream as connected by a WebSocket client.
// Returns false if already claimed.
func (s *Stream) Claim() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.claimed {
		return false
	}
	s.claimed = true
	return true
}

// StreamManager manages active real-time search streams.
type StreamManager struct {
	mu      sync.RWMutex
	streams map[string]*Stream
	ttl     time.Duration
	stopCh  chan struct{}
}

// NewStreamManager creates a StreamManager and starts background cleanup.
func NewStreamManager() *StreamManager {
	sm := &StreamManager{
		streams: make(map[string]*Stream),
		ttl:     StreamTTL,
		stopCh:  make(chan struct{}),
	}
	go sm.cleanup()
	return sm
}

// Create creates a new stream for the given search query and returns its ID.
func (sm *StreamManager) Create(query domain.SearchQuery) string {
	id := "stream_" + generateStreamID()
	stream := &Stream{
		ID:        id,
		Query:     query,
		Events:    make(chan StreamEvent, StreamEventBufferSize),
		CreatedAt: time.Now(),
	}

	sm.mu.Lock()
	sm.streams[id] = stream
	sm.mu.Unlock()

	return id
}

// Get retrieves a stream by ID. Returns nil, false if not found or expired.
func (sm *StreamManager) Get(streamID string) (*Stream, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	s, ok := sm.streams[streamID]
	if !ok {
		return nil, false
	}
	if time.Since(s.CreatedAt) > sm.ttl {
		return nil, false
	}
	return s, true
}

// Remove deletes a stream by ID.
func (sm *StreamManager) Remove(streamID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.streams, streamID)
}

// Count returns the number of active streams (for testing).
func (sm *StreamManager) Count() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.streams)
}

// Stop signals the background cleanup goroutine to exit.
func (sm *StreamManager) Stop() {
	close(sm.stopCh)
}

// cleanup periodically removes expired streams.
func (sm *StreamManager) cleanup() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.mu.Lock()
			now := time.Now()
			for id, s := range sm.streams {
				if now.Sub(s.CreatedAt) > sm.ttl {
					close(s.Events)
					delete(sm.streams, id)
				}
			}
			sm.mu.Unlock()
		case <-sm.stopCh:
			return
		}
	}
}

// generateStreamID creates a random 12-character hex string.
func generateStreamID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
