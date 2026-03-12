package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// WaybillStore abstracts waybill persistence.
type WaybillStore interface {
	// Create inserts a waybill and links the given orders to it.
	// Updates linked orders state to Packing (4).
	Create(ctx context.Context, waybill *domain.Waybill, orderNumbers []string) (*domain.Waybill, error)
	// GetByWaybillNo returns a waybill and its linked order numbers.
	GetByWaybillNo(ctx context.Context, waybillNo string) (*domain.Waybill, []domain.WaybillOrder, error)
	// ListByUser returns paginated waybills for a user.
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Waybill, int64, error)
	// UpdateState transitions a waybill to the given state.
	UpdateState(ctx context.Context, waybillNo string, state int) error
	// SetShippingFee sets the international shipping fee (filled by WMS after packing).
	SetShippingFee(ctx context.Context, waybillNo string, feeJPY int64) error
	// SetShippingInfo fills carrier, tracking number and tracking URL (filled by WMS on dispatch).
	SetShippingInfo(ctx context.Context, waybillNo string, carrier, trackingNo, trackingURL, wmsWaybillNo string) error
	// UpdateLinkedOrderStates updates all orders linked to the waybill to the given order state.
	UpdateLinkedOrderStates(ctx context.Context, waybillNo string, orderState int) error
}

// OrderWarehouseStore reads warehoused orders for waybill creation validation.
type OrderWarehouseStore interface {
	ListWarehoused(ctx context.Context, userID int64) ([]domain.Order, error)
}

// WaybillHandler handles international shipment waybill endpoints.
type WaybillHandler struct {
	waybills WaybillStore
	orders   OrderWarehouseStore
	wallets  WalletStore // for shipping fee payment
}

// NewWaybillHandler creates a WaybillHandler.
func NewWaybillHandler(waybills WaybillStore, orders OrderWarehouseStore, wallets WalletStore) *WaybillHandler {
	return &WaybillHandler{waybills: waybills, orders: orders, wallets: wallets}
}

// HandleApplyShipment handles POST /api/v1/waybill/apply.
// User selects one or more warehoused orders to apply for international shipment.
// Creates a waybill in 待合单 state and transitions orders to Packing.
func (h *WaybillHandler) HandleApplyShipment(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req struct {
		OrderNumbers []string `json:"order_numbers"`
		Remark       string   `json:"remark"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if len(req.OrderNumbers) == 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "order_numbers cannot be empty")
		return
	}

	// Validate all requested orders are warehoused and belong to this user.
	warehoused, err := h.orders.ListWarehoused(r.Context(), userID)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to fetch warehoused orders")
		return
	}
	warehousedSet := make(map[string]bool, len(warehoused))
	for _, o := range warehoused {
		warehousedSet[o.OrderNumber] = true
	}
	for _, no := range req.OrderNumbers {
		if !warehousedSet[no] {
			ErrorWithCode(w, r, http.StatusBadRequest, 40003,
				fmt.Sprintf("order %s is not warehoused or does not belong to you", no))
			return
		}
	}

	waybill := &domain.Waybill{
		WaybillNo: domain.GenerateWaybillNo(),
		UserID:    userID,
		State:     domain.WaybillStatePendingConsolidation,
		Remark:    req.Remark,
	}

	created, err := h.waybills.Create(r.Context(), waybill, req.OrderNumbers)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to create waybill")
		return
	}
	created.StateLabel = domain.WaybillStateLabel(created.State)

	Success(w, r, created)
}

// HandleListWaybills handles GET /api/v1/waybills?page=1&page_size=20.
func (h *WaybillHandler) HandleListWaybills(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	waybills, total, err := h.waybills.ListByUser(r.Context(), userID, pageSize, (page-1)*pageSize)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to list waybills")
		return
	}
	if waybills == nil {
		waybills = []domain.Waybill{}
	}
	for i := range waybills {
		waybills[i].StateLabel = domain.WaybillStateLabel(waybills[i].State)
	}

	Success(w, r, map[string]interface{}{
		"items": waybills,
		"total": total,
		"page":  page,
	})
}

// HandleGetWaybill handles GET /api/v1/waybill/{waybillNo}.
func (h *WaybillHandler) HandleGetWaybill(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	waybillNo := chi.URLParam(r, "waybillNo")

	waybill, orders, err := h.waybills.GetByWaybillNo(r.Context(), waybillNo)
	if err != nil || waybill.UserID != userID {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "waybill not found")
		return
	}
	waybill.StateLabel = domain.WaybillStateLabel(waybill.State)

	Success(w, r, map[string]interface{}{
		"waybill": waybill,
		"orders":  orders,
	})
}

// HandlePayShippingFee handles POST /api/v1/waybill/{waybillNo}/pay-shipping.
// User pays the international shipping fee from wallet; transitions waybill 待支付→待出库.
func (h *WaybillHandler) HandlePayShippingFee(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	waybillNo := chi.URLParam(r, "waybillNo")

	waybill, _, err := h.waybills.GetByWaybillNo(r.Context(), waybillNo)
	if err != nil || waybill.UserID != userID {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "waybill not found")
		return
	}
	if waybill.State != domain.WaybillStatePendingPayment {
		ErrorWithCode(w, r, http.StatusBadRequest, 40014, "waybill is not pending payment")
		return
	}
	if waybill.ShippingFeeJPY <= 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40015, "shipping fee not yet set by warehouse")
		return
	}

	// Deduct wallet.
	desc := fmt.Sprintf("International shipping fee for waybill %s", waybillNo)
	wtx, err := h.wallets.Adjust(r.Context(), userID, -waybill.ShippingFeeJPY, domain.TxTypePurchase, desc, nil)
	if err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40011, "insufficient balance")
		return
	}

	// Transition waybill state.
	if err := h.waybills.UpdateState(r.Context(), waybillNo, domain.WaybillStatePendingDispatch); err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to update waybill state")
		return
	}

	Success(w, r, map[string]interface{}{
		"waybill_no":    waybillNo,
		"state":         domain.WaybillStatePendingDispatch,
		"state_label":   domain.WaybillStateLabel(domain.WaybillStatePendingDispatch),
		"balance_after": wtx.BalanceAfter,
	})
}

// HandleConfirmReceipt handles POST /api/v1/waybill/{waybillNo}/confirm-receipt.
// User confirms receipt; transitions waybill 已发货→已收货, orders → Fulfilled.
func (h *WaybillHandler) HandleConfirmReceipt(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	waybillNo := chi.URLParam(r, "waybillNo")

	waybill, _, err := h.waybills.GetByWaybillNo(r.Context(), waybillNo)
	if err != nil || waybill.UserID != userID {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "waybill not found")
		return
	}
	if waybill.State != domain.WaybillStateShipped {
		ErrorWithCode(w, r, http.StatusBadRequest, 40016, "waybill has not been shipped yet")
		return
	}

	if err := h.waybills.UpdateState(r.Context(), waybillNo, domain.WaybillStateDelivered); err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to update waybill state")
		return
	}

	// Sync linked orders to Fulfilled.
	if err := h.waybills.UpdateLinkedOrderStates(r.Context(), waybillNo, domain.OrderStateFulfilled); err != nil {
		// Log but don't fail the response — order sync can be retried.
		_ = err
	}

	Success(w, r, map[string]interface{}{
		"waybill_no":  waybillNo,
		"state":       domain.WaybillStateDelivered,
		"state_label": domain.WaybillStateLabel(domain.WaybillStateDelivered),
	})
}

// --- Admin endpoints ---

// HandleAdminUpdateWaybillState handles POST /api/v1/admin/waybill/{waybillNo}/state.
// Used by WMS to drive waybill lifecycle (合单→打包→待支付→出库→发货).
func (h *WaybillHandler) HandleAdminUpdateWaybillState(w http.ResponseWriter, r *http.Request) {
	waybillNo := chi.URLParam(r, "waybillNo")

	var req struct {
		State int `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}

	waybill, _, err := h.waybills.GetByWaybillNo(r.Context(), waybillNo)
	if err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "waybill not found")
		return
	}

	if !domain.IsValidWaybillTransition(waybill.State, req.State) {
		ErrorWithCode(w, r, http.StatusBadRequest, 40017,
			fmt.Sprintf("invalid state transition: %d → %d", waybill.State, req.State))
		return
	}

	if err := h.waybills.UpdateState(r.Context(), waybillNo, req.State); err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to update waybill state")
		return
	}

	// Sync order states for key transitions.
	switch req.State {
	case domain.WaybillStatePendingPayment:
		// 打包完成 → 订单进入 Packed 状态（等待用户支付运费）
		_ = h.waybills.UpdateLinkedOrderStates(r.Context(), waybillNo, domain.OrderStatePacked)
	case domain.WaybillStateShipped:
		// 已发货 → 订单进入 Shipped 状态
		_ = h.waybills.UpdateLinkedOrderStates(r.Context(), waybillNo, domain.OrderStateShipped)
	}

	Success(w, r, map[string]interface{}{
		"waybill_no":  waybillNo,
		"state":       req.State,
		"state_label": domain.WaybillStateLabel(req.State),
	})
}

// HandleAdminSetShippingFee handles PUT /api/v1/admin/waybill/{waybillNo}/shipping-fee.
// WMS fills in the international shipping fee after packing is complete.
func (h *WaybillHandler) HandleAdminSetShippingFee(w http.ResponseWriter, r *http.Request) {
	waybillNo := chi.URLParam(r, "waybillNo")

	var req struct {
		ShippingFeeJPY int64 `json:"shipping_fee_jpy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.ShippingFeeJPY <= 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "shipping_fee_jpy must be positive")
		return
	}

	if err := h.waybills.SetShippingFee(r.Context(), waybillNo, req.ShippingFeeJPY); err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to set shipping fee")
		return
	}

	Success(w, r, map[string]interface{}{
		"waybill_no":       waybillNo,
		"shipping_fee_jpy": req.ShippingFeeJPY,
	})
}

// HandleAdminSetShippingInfo handles PUT /api/v1/admin/waybill/{waybillNo}/shipping-info.
// WMS fills in carrier, tracking number, and WMS waybill no when dispatching.
func (h *WaybillHandler) HandleAdminSetShippingInfo(w http.ResponseWriter, r *http.Request) {
	waybillNo := chi.URLParam(r, "waybillNo")

	var req struct {
		Carrier      string `json:"carrier"`
		TrackingNo   string `json:"tracking_no"`
		TrackingURL  string `json:"tracking_url"`
		WmsWaybillNo string `json:"wms_waybill_no"` // WMS 运单号，预留对接
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.Carrier == "" || req.TrackingNo == "" {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "carrier and tracking_no are required")
		return
	}

	if err := h.waybills.SetShippingInfo(r.Context(), waybillNo, req.Carrier, req.TrackingNo, req.TrackingURL, req.WmsWaybillNo); err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to set shipping info")
		return
	}

	Success(w, r, map[string]interface{}{
		"waybill_no": waybillNo,
		"carrier":    req.Carrier,
		"tracking_no": req.TrackingNo,
	})
}
