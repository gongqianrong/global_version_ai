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

type mockRechargeStore struct {
	createFn      func(ctx context.Context, order *domain.RechargeOrder) (*domain.RechargeOrder, error)
	getByNoFn     func(ctx context.Context, rechargeNo string) (*domain.RechargeOrder, error)
	updateStateFn func(ctx context.Context, rechargeNo string, state int) error
}

func (m *mockRechargeStore) Create(ctx context.Context, order *domain.RechargeOrder) (*domain.RechargeOrder, error) {
	return m.createFn(ctx, order)
}
func (m *mockRechargeStore) GetByRechargeNo(ctx context.Context, rechargeNo string) (*domain.RechargeOrder, error) {
	return m.getByNoFn(ctx, rechargeNo)
}
func (m *mockRechargeStore) UpdateState(ctx context.Context, rechargeNo string, state int) error {
	return m.updateStateFn(ctx, rechargeNo, state)
}

type mockPaymentGateway struct {
	createFn   func(ctx context.Context, order *domain.RechargeOrder) (string, error)
	verifyFn   func(r *http.Request) (string, bool, error)
}

func (m *mockPaymentGateway) CreatePayment(ctx context.Context, order *domain.RechargeOrder) (string, error) {
	return m.createFn(ctx, order)
}
func (m *mockPaymentGateway) VerifyCallback(r *http.Request) (string, bool, error) {
	return m.verifyFn(r)
}

func rechargeReq(method, url, body string, userID int64) *http.Request {
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

func rechargeReqWithChi(method, url, body string, userID int64, params map[string]string) *http.Request {
	r := rechargeReq(method, url, body, userID)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func newRechargeStore(order *domain.RechargeOrder) *mockRechargeStore {
	return &mockRechargeStore{
		createFn: func(_ context.Context, o *domain.RechargeOrder) (*domain.RechargeOrder, error) {
			o.ID = 1
			o.CreatedAt = time.Now()
			return o, nil
		},
		getByNoFn: func(_ context.Context, _ string) (*domain.RechargeOrder, error) {
			if order == nil {
				return nil, errors.New("not found")
			}
			return order, nil
		},
		updateStateFn: func(_ context.Context, _ string, _ int) error { return nil },
	}
}

// --- HandleGetPayMethods ---

func TestHandleGetPayMethods(t *testing.T) {
	h := NewRechargeHandler(nil, nil, nil)
	req := rechargeReq(http.MethodGet, "/wallet/pay-methods", "", 1)
	rec := httptest.NewRecorder()
	h.HandleGetPayMethods(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	methods := data["methods"].([]interface{})
	if len(methods) == 0 {
		t.Error("expected at least one payment method")
	}
}

// --- HandleCreateRecharge ---

func TestHandleCreateRecharge_Success(t *testing.T) {
	store := newRechargeStore(nil)
	h := NewRechargeHandler(store, nil, nil)
	body := `{"amount_jpy":10000,"pay_method":"wechat_pay"}`
	req := rechargeReq(http.MethodPost, "/wallet/recharge", body, 1)
	rec := httptest.NewRecorder()
	h.HandleCreateRecharge(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if data["recharge_no"] == "" {
		t.Error("expected recharge_no in response")
	}
	if data["amount_jpy"].(float64) != 10000 {
		t.Errorf("amount_jpy = %v, want 10000", data["amount_jpy"])
	}
	if data["pay_method"] != "wechat_pay" {
		t.Errorf("pay_method = %v, want wechat_pay", data["pay_method"])
	}
	if data["state"].(float64) != float64(domain.RechargeStatePending) {
		t.Errorf("state = %v, want %d", data["state"], domain.RechargeStatePending)
	}
}

func TestHandleCreateRecharge_WithGateway_ReturnsPayURL(t *testing.T) {
	store := newRechargeStore(nil)
	gw := &mockPaymentGateway{
		createFn: func(_ context.Context, _ *domain.RechargeOrder) (string, error) {
			return "https://pay.example.com/qr/abc123", nil
		},
	}
	h := NewRechargeHandler(store, nil, map[string]PaymentGateway{domain.PayMethodWechat: gw})
	body := `{"amount_jpy":5000,"pay_method":"wechat_pay"}`
	req := rechargeReq(http.MethodPost, "/wallet/recharge", body, 1)
	rec := httptest.NewRecorder()
	h.HandleCreateRecharge(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if data["pay_url"] != "https://pay.example.com/qr/abc123" {
		t.Errorf("pay_url = %v, want URL", data["pay_url"])
	}
}

func TestHandleCreateRecharge_AmountTooLow(t *testing.T) {
	h := NewRechargeHandler(newRechargeStore(nil), nil, nil)
	body := `{"amount_jpy":500,"pay_method":"wechat_pay"}`
	req := rechargeReq(http.MethodPost, "/wallet/recharge", body, 1)
	rec := httptest.NewRecorder()
	h.HandleCreateRecharge(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCreateRecharge_AmountTooHigh(t *testing.T) {
	h := NewRechargeHandler(newRechargeStore(nil), nil, nil)
	body := `{"amount_jpy":999999,"pay_method":"alipay"}`
	req := rechargeReq(http.MethodPost, "/wallet/recharge", body, 1)
	rec := httptest.NewRecorder()
	h.HandleCreateRecharge(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCreateRecharge_UnsupportedMethod(t *testing.T) {
	h := NewRechargeHandler(newRechargeStore(nil), nil, nil)
	body := `{"amount_jpy":10000,"pay_method":"bitcoin"}`
	req := rechargeReq(http.MethodPost, "/wallet/recharge", body, 1)
	rec := httptest.NewRecorder()
	h.HandleCreateRecharge(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCreateRecharge_InvalidBody(t *testing.T) {
	h := NewRechargeHandler(newRechargeStore(nil), nil, nil)
	req := rechargeReq(http.MethodPost, "/wallet/recharge", `{bad`, 1)
	rec := httptest.NewRecorder()
	h.HandleCreateRecharge(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCreateRecharge_StoreError(t *testing.T) {
	store := &mockRechargeStore{
		createFn: func(_ context.Context, _ *domain.RechargeOrder) (*domain.RechargeOrder, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewRechargeHandler(store, nil, nil)
	body := `{"amount_jpy":10000,"pay_method":"alipay"}`
	req := rechargeReq(http.MethodPost, "/wallet/recharge", body, 1)
	rec := httptest.NewRecorder()
	h.HandleCreateRecharge(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

// --- HandleRechargeCallback ---

func TestHandleRechargeCallback_UnknownMethod(t *testing.T) {
	h := NewRechargeHandler(nil, nil, nil)
	req := rechargeReqWithChi(http.MethodPost, "/wallet/recharge/callback/bitcoin", "", 0, map[string]string{"method": "bitcoin"})
	rec := httptest.NewRecorder()
	h.HandleRechargeCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (ACK unknown)", rec.Code)
	}
}

func TestHandleRechargeCallback_Success(t *testing.T) {
	order := &domain.RechargeOrder{
		ID: 1, RechargeNo: "RC20260312ABCD", UserID: 5,
		AmountJPY: 10000, PayMethod: domain.PayMethodWechat,
		State: domain.RechargeStatePending, CreatedAt: time.Now(),
	}
	store := newRechargeStore(order)
	walletStore := &mockWalletStore{
		adjustFn: func(_ context.Context, uid int64, amount int64, txType, desc string, _ *string) (*domain.WalletTransaction, error) {
			return &domain.WalletTransaction{ID: 1, UserID: uid, Amount: amount, BalanceAfter: amount}, nil
		},
	}
	gw := &mockPaymentGateway{
		verifyFn: func(_ *http.Request) (string, bool, error) {
			return "RC20260312ABCD", true, nil
		},
	}
	h := NewRechargeHandler(store, walletStore, map[string]PaymentGateway{domain.PayMethodWechat: gw})
	req := rechargeReqWithChi(http.MethodPost, "/wallet/recharge/callback/wechat_pay", "", 0, map[string]string{"method": "wechat_pay"})
	rec := httptest.NewRecorder()
	h.HandleRechargeCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleRechargeCallback_AlreadyProcessed(t *testing.T) {
	order := &domain.RechargeOrder{
		ID: 1, RechargeNo: "RC20260312ABCD", UserID: 5,
		AmountJPY: 10000, State: domain.RechargeStatePaid, // already paid
	}
	store := newRechargeStore(order)
	gw := &mockPaymentGateway{
		verifyFn: func(_ *http.Request) (string, bool, error) {
			return "RC20260312ABCD", true, nil
		},
	}
	h := NewRechargeHandler(store, nil, map[string]PaymentGateway{domain.PayMethodWechat: gw})
	req := rechargeReqWithChi(http.MethodPost, "/wallet/recharge/callback/wechat_pay", "", 0, map[string]string{"method": "wechat_pay"})
	rec := httptest.NewRecorder()
	h.HandleRechargeCallback(rec, req)

	// Should ACK without double-crediting
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestHandleRechargeCallback_PaymentFailed(t *testing.T) {
	updated := false
	store := &mockRechargeStore{
		updateStateFn: func(_ context.Context, _ string, state int) error {
			if state == domain.RechargeStateFailed {
				updated = true
			}
			return nil
		},
	}
	gw := &mockPaymentGateway{
		verifyFn: func(_ *http.Request) (string, bool, error) {
			return "RC20260312ABCD", false, nil
		},
	}
	h := NewRechargeHandler(store, nil, map[string]PaymentGateway{domain.PayMethodWechat: gw})
	req := rechargeReqWithChi(http.MethodPost, "/wallet/recharge/callback/wechat_pay", "", 0, map[string]string{"method": "wechat_pay"})
	rec := httptest.NewRecorder()
	h.HandleRechargeCallback(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !updated {
		t.Error("expected recharge state to be updated to failed")
	}
}

// --- HandleGetRechargeStatus ---

func TestHandleGetRechargeStatus_Success(t *testing.T) {
	order := &domain.RechargeOrder{
		ID: 1, RechargeNo: "RC20260312ABCD", UserID: 1,
		AmountJPY: 10000, PayMethod: domain.PayMethodAlipay,
		State: domain.RechargeStatePaid, CreatedAt: time.Now(),
	}
	h := NewRechargeHandler(newRechargeStore(order), nil, nil)
	req := rechargeReqWithChi(http.MethodGet, "/wallet/recharge/RC20260312ABCD", "", 1, map[string]string{"rechargeNo": "RC20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandleGetRechargeStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if data["recharge_no"] != "RC20260312ABCD" {
		t.Errorf("recharge_no = %v, want RC20260312ABCD", data["recharge_no"])
	}
	if data["state"].(float64) != float64(domain.RechargeStatePaid) {
		t.Errorf("state = %v, want %d", data["state"], domain.RechargeStatePaid)
	}
}

func TestHandleGetRechargeStatus_NotFound(t *testing.T) {
	h := NewRechargeHandler(newRechargeStore(nil), nil, nil)
	req := rechargeReqWithChi(http.MethodGet, "/wallet/recharge/NOTEXIST", "", 1, map[string]string{"rechargeNo": "NOTEXIST"})
	rec := httptest.NewRecorder()
	h.HandleGetRechargeStatus(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleGetRechargeStatus_WrongUser(t *testing.T) {
	order := &domain.RechargeOrder{
		ID: 1, RechargeNo: "RC20260312ABCD", UserID: 99, // belongs to user 99
		AmountJPY: 10000, State: domain.RechargeStatePending, CreatedAt: time.Now(),
	}
	h := NewRechargeHandler(newRechargeStore(order), nil, nil)
	// Request from user 1, but order belongs to user 99
	req := rechargeReqWithChi(http.MethodGet, "/wallet/recharge/RC20260312ABCD", "", 1, map[string]string{"rechargeNo": "RC20260312ABCD"})
	rec := httptest.NewRecorder()
	h.HandleGetRechargeStatus(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}
