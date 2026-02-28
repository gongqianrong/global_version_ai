package api

import (
	"context"
	"net/http"
	"time"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// HealthChecker checks the health of a platform.
type HealthChecker interface {
	GetAdapter(platformID string) (domain.PlatformAdapter, error)
	AllPlatformIDs() []string
}

// HealthHandler handles health check requests.
type HealthHandler struct {
	checker HealthChecker
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(checker HealthChecker) *HealthHandler {
	return &HealthHandler{checker: checker}
}

// PlatformHealth represents the health status of a single platform.
type PlatformHealth struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// HandleHealth handles GET /health.
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	platforms := make(map[string]PlatformHealth)

	ids := h.checker.AllPlatformIDs()
	for _, id := range ids {
		adapter, err := h.checker.GetAdapter(id)
		if err != nil {
			platforms[id] = PlatformHealth{Status: "unknown", Message: err.Error()}
			continue
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		health := adapter.HealthCheck(ctx)
		cancel()

		platforms[id] = PlatformHealth{
			Status:  health.Status,
			Message: health.Message,
		}
	}

	status := "ok"
	for _, ph := range platforms {
		if ph.Status != "healthy" {
			status = "degraded"
			break
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    status,
		"platforms": platforms,
	})
}
