package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/service"
)

// --- mocks for OrderService dependencies ---

type orderMockWallet struct {
	wallet    *domain.Wallet
	adjustTx  *domain.WalletTransaction
	adjustErr error
}

func (m *orderMockWallet) GetOrCreateWallet(_ context.Context, _ int64) (*domain.Wallet, error) {
	return m.wallet, nil
}
func (m *orderMockWallet) Adjust(_ context.Context, _ int64, _ int64, _, _ string, _ *string) (*domain.WalletTransaction, error) {
	if m.adjustErr != nil {
		return nil, m.adjustErr
	}
	return m.adjustTx, nil
}

type mockOrderRepo struct {
	createdOrder *domain.Order
	createErr    error
	order        *domain.Order
	details      []domain.OrderDetail
	getErr       error
	updateStateErr error
	orders       []domain.Order
	total        int64
	listErr      error
}

func (m *mockOrderRepo) CreateOrder(_ context.Context, order *domain.Order, _ []domain.OrderDetail) (*domain.Order, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	order.ID = 1
	m.createdOrder = order
	return order, nil
}

func (m *mockOrderRepo) GetByOrderNumber(_ context.Context, _ string) (*domain.Order, []domain.OrderDetail, error) {
	if m.getErr != nil {
		return nil, nil, m.getErr
	}
	return m.order, m.details, nil
}

func (m *mockOrderRepo) UpdateState(_ context.Context, _ string, _, _ int) error {
	return m.updateStateErr
}

func (m *mockOrderRepo) ListByUser(_ context.Context, _ int64, _, _, _ int) ([]domain.Order, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	return m.orders, m.total, nil
}

func (m *mockOrderRepo) GetDetailsByOrderIDs(_ context.Context, _ []int64) (map[int64][]domain.OrderDetail, error) {
	return nil, nil
}

type mockCartRemover struct {
	removed []string
}

func (m *mockCartRemover) Remove(_ context.Context, _ int64, productID string) error {
	m.removed = append(m.removed, productID)
	return nil
}

func orderReq(method, url, body string, userID int64) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	ctx := context.WithValue(r.Context(), userIDCtxKey, userID)
	return r.WithContext(ctx)
}

func orderReqWithParam(method, url string, userID int64, params map[string]string) *http.Request {
	r := httptest.NewRequest(method, url, nil)
	ctx := context.WithValue(r.Context(), userIDCtxKey, userID)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx = context.WithValue(ctx, chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

// --- Settlement tests ---

func TestHandleSettlement_Success(t *testing.T) {
	fetcher := &mapProductFetcher{
		products: map[string]*domain.UnifiedProduct{
			"surugaya_001": {
				ID: "surugaya_001", Title: "商品A", PriceJPY: 10000,
				Status: domain.StatusAvailable, SourcePlatform: "surugaya",
				SourceURL: "https://surugaya.com/001", ShippingFeeJPY: 500,
				Images: []string{"https://img.com/a.jpg"}, Condition: "used",
				Seller: domain.SellerInfo{SellerID: "s1", SellerName: "Shop A"},
			},
		},
	}
	walletStore := &orderMockWallet{wallet: &domain.Wallet{Balance: 100000}}
	svc := service.NewOrderService(fetcher, walletStore, nil, nil)
	h := NewOrderHandler(svc, nil)

	req := orderReq(http.MethodPost, "/order/settlement", `{"items":[{"product_id":"surugaya_001","quantity":1}]}`, 10)
	rec := httptest.NewRecorder()
	h.HandleSettlement(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if int64(data["orderTotalJp"].(float64)) != 11500 {
		t.Errorf("orderTotalJp = %v, want 11500", data["orderTotalJp"])
	}
}

func TestHandleSettlement_EmptyItems(t *testing.T) {
	svc := service.NewOrderService(nil, nil, nil, nil)
	h := NewOrderHandler(svc, nil)
	req := orderReq(http.MethodPost, "/order/settlement", `{"items":[]}`, 10)
	rec := httptest.NewRecorder()
	h.HandleSettlement(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleSettlement_ProductNotFound(t *testing.T) {
	fetcher := &mockProductFetcher{err: fmt.Errorf("not found")}
	walletStore := &orderMockWallet{wallet: &domain.Wallet{Balance: 100000}}
	svc := service.NewOrderService(fetcher, walletStore, nil, nil)
	h := NewOrderHandler(svc, nil)
	req := orderReq(http.MethodPost, "/order/settlement", `{"items":[{"product_id":"bad_id","quantity":1}]}`, 10)
	rec := httptest.NewRecorder()
	h.HandleSettlement(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// --- Confirm tests ---

func TestHandleConfirm_Success(t *testing.T) {
	fetcher := &mapProductFetcher{
		products: map[string]*domain.UnifiedProduct{
			"surugaya_001": {
				ID: "surugaya_001", Title: "商品A", PriceJPY: 10000,
				Status: domain.StatusAvailable, SourcePlatform: "surugaya",
				ShippingFeeJPY: 500, Images: []string{"https://img.com/a.jpg"},
				Condition: "used", Seller: domain.SellerInfo{SellerID: "s1", SellerName: "Shop A"},
			},
		},
	}
	orderRepo := &mockOrderRepo{}
	cartRemover := &mockCartRemover{}
	svc := service.NewOrderService(fetcher, nil, orderRepo, cartRemover)
	h := NewOrderHandler(svc, nil)

	req := orderReq(http.MethodPost, "/order/confirm", `{"items":[{"product_id":"surugaya_001","quantity":1}]}`, 10)
	rec := httptest.NewRecorder()
	h.HandleConfirm(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if int(data["orderState"].(float64)) != domain.OrderStatePending {
		t.Errorf("orderState = %v, want 0 (Pending)", data["orderState"])
	}
	if len(cartRemover.removed) != 1 {
		t.Errorf("cart removed = %v, want [surugaya_001]", cartRemover.removed)
	}
}

// --- Pay tests ---

func TestHandlePay_Success(t *testing.T) {
	walletStore := &orderMockWallet{
		wallet:   &domain.Wallet{Balance: 100000},
		adjustTx: &domain.WalletTransaction{ID: 1, Amount: -11500, BalanceAfter: 88500},
	}
	orderRepo := &mockOrderRepo{
		order: &domain.Order{
			ID: 1, OrderNumber: "RO20260304TEST", UserID: 10,
			OrderState: domain.OrderStatePending, OrderInpriceJp: 11500,
		},
	}
	svc := service.NewOrderService(nil, walletStore, orderRepo, nil)
	h := NewOrderHandler(svc, nil)

	req := orderReq(http.MethodPost, "/order/pay", `{"order_number":"RO20260304TEST"}`, 10)
	rec := httptest.NewRecorder()
	h.HandlePay(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if int(data["orderState"].(float64)) != domain.OrderStatePaid {
		t.Errorf("orderState = %v, want 1 (Paid)", data["orderState"])
	}
	if int64(data["balanceAfter"].(float64)) != 88500 {
		t.Errorf("balanceAfter = %v, want 88500", data["balanceAfter"])
	}
}

func TestHandlePay_InsufficientBalance(t *testing.T) {
	walletStore := &orderMockWallet{wallet: &domain.Wallet{Balance: 5000}}
	orderRepo := &mockOrderRepo{
		order: &domain.Order{
			ID: 1, OrderNumber: "RO20260304TEST", UserID: 10,
			OrderState: domain.OrderStatePending, OrderInpriceJp: 11500,
		},
	}
	svc := service.NewOrderService(nil, walletStore, orderRepo, nil)
	h := NewOrderHandler(svc, nil)

	req := orderReq(http.MethodPost, "/order/pay", `{"order_number":"RO20260304TEST"}`, 10)
	rec := httptest.NewRecorder()
	h.HandlePay(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandlePay_AlreadyPaid(t *testing.T) {
	orderRepo := &mockOrderRepo{
		order: &domain.Order{ID: 1, OrderNumber: "RO_TEST", UserID: 10, OrderState: domain.OrderStatePaid},
	}
	svc := service.NewOrderService(nil, nil, orderRepo, nil)
	h := NewOrderHandler(svc, nil)

	req := orderReq(http.MethodPost, "/order/pay", `{"order_number":"RO_TEST"}`, 10)
	rec := httptest.NewRecorder()
	h.HandlePay(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// --- GetOrder tests ---

func TestHandleGetOrder_Success(t *testing.T) {
	orderRepo := &mockOrderRepo{
		order: &domain.Order{
			ID: 1, OrderNumber: "RO_TEST", UserID: 10, OrderState: 1,
			OrderTotalJp: 11500, CommissionFeeJp: 1000, ShippingFeeJp: 500,
			OrderInpriceJp: 11500, CreatedAt: time.Now(),
		},
		details: []domain.OrderDetail{
			{ID: 1, OrderID: 1, GoodsMid: "surugaya_001", GoodsName: "商品A", GoodsNum: 1, GoodsAmountJp: 10000},
		},
	}
	svc := service.NewOrderService(nil, nil, orderRepo, nil)
	h := NewOrderHandler(svc, nil)

	req := orderReqWithParam(http.MethodGet, "/order/RO_TEST", 10, map[string]string{"orderNumber": "RO_TEST"})
	rec := httptest.NewRecorder()
	h.HandleGetOrder(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if data["orderNumber"] != "RO_TEST" {
		t.Errorf("orderNumber = %v, want RO_TEST", data["orderNumber"])
	}
	details := data["orderDetailList"].([]interface{})
	if len(details) != 1 {
		t.Errorf("details len = %d, want 1", len(details))
	}
}

func TestHandleGetOrder_WrongUser(t *testing.T) {
	orderRepo := &mockOrderRepo{
		order: &domain.Order{ID: 1, OrderNumber: "RO_TEST", UserID: 99, CreatedAt: time.Now()},
	}
	svc := service.NewOrderService(nil, nil, orderRepo, nil)
	h := NewOrderHandler(svc, nil)

	req := orderReqWithParam(http.MethodGet, "/order/RO_TEST", 10, map[string]string{"orderNumber": "RO_TEST"})
	rec := httptest.NewRecorder()
	h.HandleGetOrder(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

// --- ListOrders tests ---

func TestHandleListOrders_Success(t *testing.T) {
	orderRepo := &mockOrderRepo{
		orders: []domain.Order{
			{OrderNumber: "RO_001", OrderState: 1, OrderTotalJp: 11500, CreatedAt: time.Now()},
			{OrderNumber: "RO_002", OrderState: 0, OrderTotalJp: 5000, CreatedAt: time.Now()},
		},
		total: 2,
	}
	svc := service.NewOrderService(nil, nil, orderRepo, nil)
	h := NewOrderHandler(svc, nil)

	req := orderReq(http.MethodGet, "/orders?page=1&page_size=20", "", 10)
	rec := httptest.NewRecorder()
	h.HandleListOrders(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if int64(data["total"].(float64)) != 2 {
		t.Errorf("total = %v, want 2", data["total"])
	}
	items := data["items"].([]interface{})
	if len(items) != 2 {
		t.Errorf("items len = %d, want 2", len(items))
	}
}

func TestHandleListOrders_Empty(t *testing.T) {
	orderRepo := &mockOrderRepo{orders: nil, total: 0}
	svc := service.NewOrderService(nil, nil, orderRepo, nil)
	h := NewOrderHandler(svc, nil)

	req := orderReq(http.MethodGet, "/orders", "", 10)
	rec := httptest.NewRecorder()
	h.HandleListOrders(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	items := data["items"].([]interface{})
	if len(items) != 0 {
		t.Errorf("items len = %d, want 0", len(items))
	}
}

// --- Cancel tests ---

func TestHandleCancelOrder_Success(t *testing.T) {
	orderRepo := &mockOrderRepo{
		order: &domain.Order{ID: 1, OrderNumber: "RO_TEST", UserID: 10, OrderState: domain.OrderStatePending, CreatedAt: time.Now()},
	}
	svc := service.NewOrderService(nil, nil, orderRepo, nil)
	h := NewOrderHandler(svc, nil)

	req := orderReq(http.MethodPost, "/order/cancel", `{"order_number":"RO_TEST"}`, 10)
	rec := httptest.NewRecorder()
	h.HandleCancelOrder(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	if int(data["orderState"].(float64)) != domain.OrderStateCancelled {
		t.Errorf("orderState = %v, want 8 (Cancelled)", data["orderState"])
	}
}

func TestHandleCancelOrder_AlreadyPaid(t *testing.T) {
	orderRepo := &mockOrderRepo{
		order: &domain.Order{ID: 1, OrderNumber: "RO_TEST", UserID: 10, OrderState: domain.OrderStatePaid, CreatedAt: time.Now()},
	}
	svc := service.NewOrderService(nil, nil, orderRepo, nil)
	h := NewOrderHandler(svc, nil)

	req := orderReq(http.MethodPost, "/order/cancel", `{"order_number":"RO_TEST"}`, 10)
	rec := httptest.NewRecorder()
	h.HandleCancelOrder(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (already paid, not cancellable)", rec.Code)
	}
}
