package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// --- mocks ---

type mockWaybillStore struct {
	createFn                  func(ctx context.Context, w *domain.Waybill, orderNumbers []string) (*domain.Waybill, error)
	getByNoFn                 func(ctx context.Context, waybillNo string) (*domain.Waybill, []domain.WaybillOrder, error)
	listByUserFn              func(ctx context.Context, userID int64, limit, offset int) ([]domain.Waybill, int64, error)
	updateStateFn             func(ctx context.Context, waybillNo string, state int) error
	setShippingFeeFn          func(ctx context.Context, waybillNo string, feeJPY int64) error
	setShippingInfoFn         func(ctx context.Context, waybillNo, carrier, trackingNo, trackingURL, wmsNo string) error
	updateLinkedOrderStatesFn func(ctx context.Context, waybillNo string, orderState int) error
}

func (m *mockWaybillStore) Create(ctx context.Context, w *domain.Waybill, orderNumbers []string) (*domain.Waybill, error) {
	return m.createFn(ctx, w, orderNumbers)
}
func (m *mockWaybillStore) GetByWaybillNo(ctx context.Context, waybillNo string) (*domain.Waybill, []domain.WaybillOrder, error) {
	return m.getByNoFn(ctx, waybillNo)
}
func (m *mockWaybillStore) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Waybill, int64, error) {
	return m.listByUserFn(ctx, userID, limit, offset)
}
func (m *mockWaybillStore) UpdateState(ctx context.Context, waybillNo string, state int) error {
	return m.updateStateFn(ctx, waybillNo, state)
}
func (m *mockWaybillStore) SetShippingFee(ctx context.Context, waybillNo string, feeJPY int64) error {
	return m.setShippingFeeFn(ctx, waybillNo, feeJPY)
}
func (m *mockWaybillStore) SetShippingInfo(ctx context.Context, waybillNo, carrier, trackingNo, trackingURL, wmsNo string) error {
	return m.setShippingInfoFn(ctx, waybillNo, carrier, trackingNo, trackingURL, wmsNo)
}
func (m *mockWaybillStore) UpdateLinkedOrderStates(ctx context.Context, waybillNo string, orderState int) error {
	if m.updateLinkedOrderStatesFn != nil {
		return m.updateLinkedOrderStatesFn(ctx, waybillNo, orderState)
	}
	return nil
}

type mockOrderWarehouseStore struct {
	listFn func(ctx context.Context, userID int64) ([]domain.Order, error)
}

func (m *mockOrderWarehouseStore) ListWarehoused(ctx context.Context, userID int64) ([]domain.Order, error) {
	return m.listFn(ctx, userID)
}

func waybillReq(method, url, body string, userID int64) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	ctx := context.WithValue(r.Context(), userIDCtxKey, userID)
	return r.WithContext(ctx)
}

func waybillReqWithChi(method, url, body string, userID int64, params map[string]string) *http.Request {
	r := waybillReq(method, url, body, userID)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func sampleWaybill(state int, userID int64) *domain.Waybill {
	return &domain.Waybill{
		ID: 1, WaybillNo: "LO20260312ABCD", UserID: userID,
		State: state, ShippingFeeJPY: 3000,
		Carrier: "EMS", TrackingNo: "EJ000000000JP",
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

// --- HandleApplyShipment ---

func TestHandleApplyShipment_Success(t *testing.T) {
	orders := &mockOrderWarehouseStore{
		listFn: func(_ context.Context, _ int64) ([]domain.Order, error) {
			return []domain.Order{
				{OrderNumber: "RO001", OrderState: domain.OrderStateWarehoused},
			}, nil
		},
	}
	waybills := &mockWaybillStore{
		createFn: func(_ context.Context, w *domain.Waybill, _ []string) (*domain.Waybill, error) {
			w.ID = 1
			w.CreatedAt = time.Now()
			return w, nil
		},
	}
	h := NewWaybillHandler(waybills, orders, nil)
	body := `{"order_numbers":["RO001"],"remark":"please consolidate"}`
	req := waybillReq(http.MethodPost, "/waybill/apply", body, 1)
	rec := httptest.NewRecorder()
	h.HandleApplyShipment(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if data["waybill_no"] == "" {
		t.Error("expected waybill_no in response")
	}
	if data["state_label"] != "待合单" {
		t.Errorf("state_label = %v, want 待合单", data["state_label"])
	}
}

func TestHandleApplyShipment_EmptyOrders(t *testing.T) {
	h := NewWaybillHandler(nil, nil, nil)
	req := waybillReq(http.MethodPost, "/waybill/apply", `{"order_numbers":[]}`, 1)
	rec := httptest.NewRecorder()
	h.HandleApplyShipment(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleApplyShipment_OrderNotWarehoused(t *testing.T) {
	orders := &mockOrderWarehouseStore{
		listFn: func(_ context.Context, _ int64) ([]domain.Order, error) {
			return []domain.Order{}, nil // no warehoused orders
		},
	}
	h := NewWaybillHandler(nil, orders, nil)
	body := `{"order_numbers":["RO001"]}`
	req := waybillReq(http.MethodPost, "/waybill/apply", body, 1)
	rec := httptest.NewRecorder()
	h.HandleApplyShipment(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- HandleListWaybills ---

func TestHandleListWaybills_Success(t *testing.T) {
	waybills := &mockWaybillStore{
		listByUserFn: func(_ context.Context, _ int64, _, _ int) ([]domain.Waybill, int64, error) {
			return []domain.Waybill{*sampleWaybill(domain.WaybillStateShipped, 1)}, 1, nil
		},
	}
	h := NewWaybillHandler(waybills, nil, nil)
	req := waybillReq(http.MethodGet, "/waybills", "", 1)
	rec := httptest.NewRecorder()
	h.HandleListWaybills(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
}

// --- HandleGetWaybill ---

func TestHandleGetWaybill_Success(t *testing.T) {
	waybills := &mockWaybillStore{
		getByNoFn: func(_ context.Context, _ string) (*domain.Waybill, []domain.WaybillOrder, error) {
			return sampleWaybill(domain.WaybillStatePendingConsolidation, 1),
				[]domain.WaybillOrder{{WaybillNo: "LO20260312ABCD", OrderNumber: "RO001"}}, nil
		},
	}
	h := NewWaybillHandler(waybills, nil, nil)
	req := waybillReqWithChi(http.MethodGet, "/waybill/LO20260312ABCD", "", 1, map[string]string{"waybillNo": "LO20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandleGetWaybill(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleGetWaybill_WrongUser(t *testing.T) {
	waybills := &mockWaybillStore{
		getByNoFn: func(_ context.Context, _ string) (*domain.Waybill, []domain.WaybillOrder, error) {
			return sampleWaybill(domain.WaybillStatePendingConsolidation, 99), nil, nil // belongs to user 99
		},
	}
	h := NewWaybillHandler(waybills, nil, nil)
	req := waybillReqWithChi(http.MethodGet, "/waybill/LO20260312ABCD", "", 1, map[string]string{"waybillNo": "LO20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandleGetWaybill(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- HandlePayShippingFee ---

func TestHandlePayShippingFee_Success(t *testing.T) {
	waybills := &mockWaybillStore{
		getByNoFn: func(_ context.Context, _ string) (*domain.Waybill, []domain.WaybillOrder, error) {
			return sampleWaybill(domain.WaybillStatePendingPayment, 1), nil, nil
		},
		updateStateFn: func(_ context.Context, _ string, _ int) error { return nil },
	}
	wallet := &mockWalletStore{
		adjustFn: func(_ context.Context, _ int64, _ int64, _, _ string, _ *string) (*domain.WalletTransaction, error) {
			return &domain.WalletTransaction{BalanceAfter: 50000}, nil
		},
	}
	h := NewWaybillHandler(waybills, nil, wallet)
	req := waybillReqWithChi(http.MethodPost, "/waybill/LO20260312ABCD/pay-shipping", "", 1, map[string]string{"waybillNo": "LO20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandlePayShippingFee(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if data["state_label"] != "待出库" {
		t.Errorf("state_label = %v, want 待出库", data["state_label"])
	}
}

func TestHandlePayShippingFee_WrongState(t *testing.T) {
	waybills := &mockWaybillStore{
		getByNoFn: func(_ context.Context, _ string) (*domain.Waybill, []domain.WaybillOrder, error) {
			return sampleWaybill(domain.WaybillStatePendingConsolidation, 1), nil, nil
		},
	}
	h := NewWaybillHandler(waybills, nil, nil)
	req := waybillReqWithChi(http.MethodPost, "/waybill/LO20260312ABCD/pay-shipping", "", 1, map[string]string{"waybillNo": "LO20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandlePayShippingFee(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- HandleConfirmReceipt ---

func TestHandleConfirmReceipt_Success(t *testing.T) {
	orderStateUpdated := 0
	waybills := &mockWaybillStore{
		getByNoFn: func(_ context.Context, _ string) (*domain.Waybill, []domain.WaybillOrder, error) {
			return sampleWaybill(domain.WaybillStateShipped, 1), nil, nil
		},
		updateStateFn: func(_ context.Context, _ string, _ int) error { return nil },
		updateLinkedOrderStatesFn: func(_ context.Context, _ string, state int) error {
			orderStateUpdated = state
			return nil
		},
	}
	h := NewWaybillHandler(waybills, nil, nil)
	req := waybillReqWithChi(http.MethodPost, "/waybill/LO20260312ABCD/confirm-receipt", "", 1, map[string]string{"waybillNo": "LO20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandleConfirmReceipt(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if orderStateUpdated != domain.OrderStateFulfilled {
		t.Errorf("linked orders should be updated to Fulfilled(%d), got %d", domain.OrderStateFulfilled, orderStateUpdated)
	}
}

func TestHandleConfirmReceipt_WrongState(t *testing.T) {
	waybills := &mockWaybillStore{
		getByNoFn: func(_ context.Context, _ string) (*domain.Waybill, []domain.WaybillOrder, error) {
			return sampleWaybill(domain.WaybillStatePendingDispatch, 1), nil, nil
		},
	}
	h := NewWaybillHandler(waybills, nil, nil)
	req := waybillReqWithChi(http.MethodPost, "/waybill/LO20260312ABCD/confirm-receipt", "", 1, map[string]string{"waybillNo": "LO20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandleConfirmReceipt(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- HandleAdminUpdateWaybillState ---

func TestHandleAdminUpdateWaybillState_ValidTransition(t *testing.T) {
	linkedUpdated := 0
	waybills := &mockWaybillStore{
		getByNoFn: func(_ context.Context, _ string) (*domain.Waybill, []domain.WaybillOrder, error) {
			return sampleWaybill(domain.WaybillStatePendingPacking, 1), nil, nil
		},
		updateStateFn: func(_ context.Context, _ string, _ int) error { return nil },
		updateLinkedOrderStatesFn: func(_ context.Context, _ string, state int) error {
			linkedUpdated = state
			return nil
		},
	}
	h := NewWaybillHandler(waybills, nil, nil)
	body := `{"state":2}` // 待打包 → 待支付
	req := waybillReqWithChi(http.MethodPost, "/admin/waybill/LO/state", body, 0, map[string]string{"waybillNo": "LO20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandleAdminUpdateWaybillState(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	// Packed state should be synced to orders
	if linkedUpdated != domain.OrderStatePacked {
		t.Errorf("linked orders = %d, want OrderStatePacked(%d)", linkedUpdated, domain.OrderStatePacked)
	}
}

func TestHandleAdminUpdateWaybillState_InvalidTransition(t *testing.T) {
	waybills := &mockWaybillStore{
		getByNoFn: func(_ context.Context, _ string) (*domain.Waybill, []domain.WaybillOrder, error) {
			return sampleWaybill(domain.WaybillStatePendingConsolidation, 1), nil, nil
		},
	}
	h := NewWaybillHandler(waybills, nil, nil)
	body := `{"state":4}` // 待合单 → 已发货 (invalid)
	req := waybillReqWithChi(http.MethodPost, "/admin/waybill/LO/state", body, 0, map[string]string{"waybillNo": "LO20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandleAdminUpdateWaybillState(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- HandleAdminSetShippingFee ---

func TestHandleAdminSetShippingFee_Success(t *testing.T) {
	waybills := &mockWaybillStore{
		setShippingFeeFn: func(_ context.Context, _ string, _ int64) error { return nil },
	}
	h := NewWaybillHandler(waybills, nil, nil)
	body := `{"shipping_fee_jpy":4500}`
	req := waybillReqWithChi(http.MethodPut, "/admin/waybill/LO/shipping-fee", body, 0, map[string]string{"waybillNo": "LO20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandleAdminSetShippingFee(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleAdminSetShippingFee_ZeroAmount(t *testing.T) {
	h := NewWaybillHandler(&mockWaybillStore{}, nil, nil)
	body := `{"shipping_fee_jpy":0}`
	req := waybillReqWithChi(http.MethodPut, "/admin/waybill/LO/shipping-fee", body, 0, map[string]string{"waybillNo": "LO"})
	rec := httptest.NewRecorder()
	h.HandleAdminSetShippingFee(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- HandleAdminSetShippingInfo ---

func TestHandleAdminSetShippingInfo_Success(t *testing.T) {
	waybills := &mockWaybillStore{
		setShippingInfoFn: func(_ context.Context, _, _, _, _, _ string) error { return nil },
	}
	h := NewWaybillHandler(waybills, nil, nil)
	body := `{"carrier":"EMS","tracking_no":"EJ000000000JP","tracking_url":"https://track.example.com","wms_waybill_no":"WMS-001"}`
	req := waybillReqWithChi(http.MethodPut, "/admin/waybill/LO/shipping-info", body, 0, map[string]string{"waybillNo": "LO20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandleAdminSetShippingInfo(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleAdminSetShippingInfo_MissingFields(t *testing.T) {
	h := NewWaybillHandler(&mockWaybillStore{}, nil, nil)
	body := `{"carrier":"EMS"}` // missing tracking_no
	req := waybillReqWithChi(http.MethodPut, "/admin/waybill/LO/shipping-info", body, 0, map[string]string{"waybillNo": "LO"})
	rec := httptest.NewRecorder()
	h.HandleAdminSetShippingInfo(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- Domain: state transition validation ---

func TestIsValidWaybillTransition(t *testing.T) {
	cases := []struct {
		from, to int
		want     bool
	}{
		{domain.WaybillStatePendingConsolidation, domain.WaybillStatePendingPacking, true},
		{domain.WaybillStatePendingPacking, domain.WaybillStatePendingPayment, true},
		{domain.WaybillStatePendingPayment, domain.WaybillStatePendingDispatch, true},
		{domain.WaybillStatePendingDispatch, domain.WaybillStateShipped, true},
		{domain.WaybillStateShipped, domain.WaybillStateDelivered, true},
		{domain.WaybillStatePendingConsolidation, domain.WaybillStateShipped, false}, // skip states
		{domain.WaybillStateDelivered, domain.WaybillStatePendingConsolidation, false}, // backwards
	}
	for _, c := range cases {
		if got := domain.IsValidWaybillTransition(c.from, c.to); got != c.want {
			t.Errorf("IsValidWaybillTransition(%d→%d) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestIsValidOrderTransition(t *testing.T) {
	if !domain.IsValidOrderTransition(domain.OrderStatePaid, domain.OrderStatePurchasing) {
		t.Error("Paid→Purchasing should be valid")
	}
	if !domain.IsValidOrderTransition(domain.OrderStatePurchasing, domain.OrderStateWarehoused) {
		t.Error("Purchasing→Warehoused should be valid")
	}
	if domain.IsValidOrderTransition(domain.OrderStatePaid, domain.OrderStateWarehoused) {
		t.Error("Paid→Warehoused should be invalid (must go through Purchasing)")
	}
	if domain.IsValidOrderTransition(domain.OrderStateWarehoused, domain.OrderStatePaid) {
		t.Error("backwards transition should be invalid")
	}
}

// --- HandleAdminUpdateOrderState ---

func TestHandleAdminUpdateOrderState_Success(t *testing.T) {
	updater := &mockOrderStateUpdater{
		getFn: func(_ context.Context, orderNumber string) (*domain.Order, []domain.OrderDetail, error) {
			return &domain.Order{OrderNumber: orderNumber, OrderState: domain.OrderStatePaid}, nil, nil
		},
		updateFn: func(_ context.Context, _ string, _, _ int) error { return nil },
	}
	h := NewOrderHandler(nil, updater)
	body := `{"state":2}` // Paid → Purchasing
	req := waybillReqWithChi(http.MethodPost, "/admin/orders/RO001/state", body, 0, map[string]string{"orderNumber": "RO001"})
	rec := httptest.NewRecorder()
	h.HandleAdminUpdateOrderState(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAdminUpdateOrderState_InvalidTransition(t *testing.T) {
	updater := &mockOrderStateUpdater{
		getFn: func(_ context.Context, orderNumber string) (*domain.Order, []domain.OrderDetail, error) {
			return &domain.Order{OrderNumber: orderNumber, OrderState: domain.OrderStatePaid}, nil, nil
		},
	}
	h := NewOrderHandler(nil, updater)
	body := `{"state":3}` // Paid → Warehoused (invalid, must go through Purchasing)
	req := waybillReqWithChi(http.MethodPost, "/admin/orders/RO001/state", body, 0, map[string]string{"orderNumber": "RO001"})
	rec := httptest.NewRecorder()
	h.HandleAdminUpdateOrderState(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAdminUpdateOrderState_OrderNotFound(t *testing.T) {
	updater := &mockOrderStateUpdater{
		getFn: func(_ context.Context, _ string) (*domain.Order, []domain.OrderDetail, error) {
			return nil, nil, errors.New("not found")
		},
	}
	h := NewOrderHandler(nil, updater)
	body := `{"state":2}`
	req := waybillReqWithChi(http.MethodPost, "/admin/orders/NOTEXIST/state", body, 0, map[string]string{"orderNumber": "NOTEXIST"})
	rec := httptest.NewRecorder()
	h.HandleAdminUpdateOrderState(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// mockOrderStateUpdater implements OrderStateUpdater for testing
type mockOrderStateUpdater struct {
	getFn    func(ctx context.Context, orderNumber string) (*domain.Order, []domain.OrderDetail, error)
	updateFn func(ctx context.Context, orderNumber string, fromState, toState int) error
}

func (m *mockOrderStateUpdater) GetByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, []domain.OrderDetail, error) {
	return m.getFn(ctx, orderNumber)
}
func (m *mockOrderStateUpdater) UpdateState(ctx context.Context, orderNumber string, fromState, toState int) error {
	return m.updateFn(ctx, orderNumber, fromState, toState)
}
