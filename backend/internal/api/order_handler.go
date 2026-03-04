package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/service"
)

// OrderHandler handles order endpoints.
type OrderHandler struct {
	svc *service.OrderService
}

// NewOrderHandler creates an OrderHandler.
func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

type settlementRequest struct {
	Items []service.OrderItem `json:"items"`
}

type confirmRequest struct {
	Items             []service.OrderItem `json:"items"`
	OrderPaytype      int                 `json:"order_paytype"`
	OrderRemark       string              `json:"order_remark"`
	OrderPurchaseType int                 `json:"order_purchase_type"`
}

type payRequest struct {
	OrderNumber string `json:"order_number"`
}

type cancelRequest struct {
	OrderNumber string `json:"order_number"`
}

// HandleSettlement handles POST /api/v1/order/settlement.
func (h *OrderHandler) HandleSettlement(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req settlementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "items is required")
		return
	}

	result, err := h.svc.Settlement(r.Context(), userID, req.Items)
	if err != nil {
		if errors.Is(err, service.ErrProductUnavailable) {
			ErrorWithCode(w, r, http.StatusBadRequest, 40010, err.Error())
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "settlement failed")
		return
	}

	Success(w, r, result)
}

// HandleConfirm handles POST /api/v1/order/confirm.
// Creates order in Pending state, removes cart items, does NOT deduct wallet.
func (h *OrderHandler) HandleConfirm(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req confirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "items is required")
		return
	}

	if req.OrderPurchaseType == 0 {
		req.OrderPurchaseType = 1
	}

	result, err := h.svc.Confirm(r.Context(), userID, req.Items, req.OrderPaytype, req.OrderRemark, req.OrderPurchaseType)
	if err != nil {
		if errors.Is(err, service.ErrProductUnavailable) {
			ErrorWithCode(w, r, http.StatusBadRequest, 40010, err.Error())
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "order confirm failed")
		return
	}

	Success(w, r, result)
}

// HandlePay handles POST /api/v1/order/pay.
// Deducts wallet balance and transitions order from Pending to Paid.
func (h *OrderHandler) HandlePay(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req payRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.OrderNumber == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "order_number is required")
		return
	}

	result, err := h.svc.Pay(r.Context(), userID, req.OrderNumber)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			ErrorWithCode(w, r, http.StatusNotFound, 40401, "order not found")
			return
		}
		if errors.Is(err, service.ErrOrderNotPayable) {
			ErrorWithCode(w, r, http.StatusBadRequest, 40012, "order is not payable")
			return
		}
		if errors.Is(err, service.ErrInsufficientBalance) {
			ErrorWithCode(w, r, http.StatusBadRequest, 40011, "insufficient balance")
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "payment failed")
		return
	}

	Success(w, r, result)
}

// HandleGetOrder handles GET /api/v1/order/{orderNumber}.
func (h *OrderHandler) HandleGetOrder(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	orderNumber := chi.URLParam(r, "orderNumber")

	result, err := h.svc.GetDetail(r.Context(), userID, orderNumber)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			ErrorWithCode(w, r, http.StatusNotFound, 40401, "order not found")
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to get order")
		return
	}

	Success(w, r, result)
}

// HandleListOrders handles GET /api/v1/orders?page=1&page_size=20&state=-1.
func (h *OrderHandler) HandleListOrders(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	state := -1 // default: all states
	if s := r.URL.Query().Get("state"); s != "" {
		state, _ = strconv.Atoi(s)
	}

	orders, total, err := h.svc.ListOrders(r.Context(), userID, state, page, pageSize)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to list orders")
		return
	}

	if orders == nil {
		orders = []service.OrderResponse{}
	}

	Success(w, r, map[string]interface{}{
		"items": orders,
		"total": total,
		"page":  page,
	})
}

// HandleCancelOrder handles POST /api/v1/order/cancel.
func (h *OrderHandler) HandleCancelOrder(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req cancelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.OrderNumber == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "order_number is required")
		return
	}

	err := h.svc.Cancel(r.Context(), userID, req.OrderNumber)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			ErrorWithCode(w, r, http.StatusNotFound, 40401, "order not found")
			return
		}
		if errors.Is(err, service.ErrOrderNotCancellable) {
			ErrorWithCode(w, r, http.StatusBadRequest, 40013, "order cannot be cancelled")
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "cancel failed")
		return
	}

	Success(w, r, map[string]interface{}{
		"orderNumber": req.OrderNumber,
		"orderState":  8, // Cancelled
	})
}
