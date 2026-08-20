package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/service"
)

// GlobalOrderService defines the interface for global order operations.
type GlobalOrderService interface {
	SyncGlobalOrder(ctx context.Context, req *domain.GlobalOrderSyncRequest) (*domain.GlobalOrderSyncResponse, error)
	SyncGlobalPayment(ctx context.Context, req *domain.GlobalPaymentSyncRequest) (*domain.GlobalPaymentSyncResponse, error)
}

// GlobalOrderHandler handles international order sync endpoints.
type GlobalOrderHandler struct {
	svc *service.GlobalOrderService
}

// NewGlobalOrderHandler creates a GlobalOrderHandler.
func NewGlobalOrderHandler(svc *service.GlobalOrderService) *GlobalOrderHandler {
	return &GlobalOrderHandler{svc: svc}
}

// HandleSync handles POST /internal/global/order/sync.
// Creates local order in BEPAY (pending payment) state.
func (h *GlobalOrderHandler) HandleSync(w http.ResponseWriter, r *http.Request) {
	var req domain.GlobalOrderSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding sync request: %v", err)
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}

	resp, err := h.svc.SyncGlobalOrder(r.Context(), &req)
	if err != nil {
		log.Printf("Error syncing global order: %v", err)
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "order sync failed")
		return
	}

	// Check business result
	if !resp.Success {
		log.Printf("Order sync validation failed: %s", resp.Message)
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, resp.Message)
		return
	}

	// Return success
	Success(w, r, resp)
}

// HandlePaymentSuccess handles POST /internal/global/order/payment-success.
// Updates order from BEPAY to PAID and records payment info.
func (h *GlobalOrderHandler) HandlePaymentSuccess(w http.ResponseWriter, r *http.Request) {
	var req domain.GlobalPaymentSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding payment request: %v", err)
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}

	resp, err := h.svc.SyncGlobalPayment(r.Context(), &req)
	if err != nil {
		log.Printf("Error syncing global payment: %v", err)
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "payment sync failed")
		return
	}

	// Check business result
	if !resp.Success {
		log.Printf("Payment sync validation failed: %s", resp.Message)
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, resp.Message)
		return
	}

	// Return success
	Success(w, r, resp)
}
