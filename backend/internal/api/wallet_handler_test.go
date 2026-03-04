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

	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/repo"
)

// --- mock ---

type mockWalletStore struct {
	getOrCreateFn    func(ctx context.Context, userID int64) (*domain.Wallet, error)
	adjustFn         func(ctx context.Context, userID int64, amount int64, txType, desc string, relatedOrder *string) (*domain.WalletTransaction, error)
	listTransFn      func(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int64, error)
}

func (m *mockWalletStore) GetOrCreateWallet(ctx context.Context, userID int64) (*domain.Wallet, error) {
	return m.getOrCreateFn(ctx, userID)
}
func (m *mockWalletStore) Adjust(ctx context.Context, userID int64, amount int64, txType, desc string, relatedOrder *string) (*domain.WalletTransaction, error) {
	return m.adjustFn(ctx, userID, amount, txType, desc, relatedOrder)
}
func (m *mockWalletStore) ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int64, error) {
	return m.listTransFn(ctx, userID, limit, offset)
}

func walletReq(method, url string, body string, userID int64) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, url, strings.NewReader(body))
	} else {
		r = httptest.NewRequest(method, url, nil)
	}
	ctx := context.WithValue(r.Context(), userIDCtxKey, userID)
	return r.WithContext(ctx)
}

// --- tests ---

func TestHandleGetBalance_Success(t *testing.T) {
	store := &mockWalletStore{
		getOrCreateFn: func(_ context.Context, uid int64) (*domain.Wallet, error) {
			return &domain.Wallet{ID: 1, UserID: uid, Balance: 105600, CreatedAt: time.Now(), UpdatedAt: time.Now()}, nil
		},
	}
	h := NewWalletHandler(store)
	req := walletReq(http.MethodGet, "/wallet/balance", "", 10)
	rec := httptest.NewRecorder()
	h.HandleGetBalance(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["balance"].(float64) != 105600 {
		t.Errorf("balance = %v, want 105600", data["balance"])
	}
	if data["currency"] != "JPY" {
		t.Errorf("currency = %v, want JPY", data["currency"])
	}
}

func TestHandleGetBalance_Error(t *testing.T) {
	store := &mockWalletStore{
		getOrCreateFn: func(_ context.Context, _ int64) (*domain.Wallet, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewWalletHandler(store)
	req := walletReq(http.MethodGet, "/wallet/balance", "", 10)
	rec := httptest.NewRecorder()
	h.HandleGetBalance(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

func TestHandleListTransactions_Empty(t *testing.T) {
	store := &mockWalletStore{
		listTransFn: func(_ context.Context, _ int64, _, _ int) ([]domain.WalletTransaction, int64, error) {
			return nil, 0, nil
		},
	}
	h := NewWalletHandler(store)
	req := walletReq(http.MethodGet, "/wallet/transactions", "", 10)
	rec := httptest.NewRecorder()
	h.HandleListTransactions(rec, req)

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

func TestHandleListTransactions_WithItems(t *testing.T) {
	store := &mockWalletStore{
		listTransFn: func(_ context.Context, _ int64, _, _ int) ([]domain.WalletTransaction, int64, error) {
			return []domain.WalletTransaction{
				{ID: 1, UserID: 10, Type: domain.TxTypeRecharge, Amount: 10000, BalanceAfter: 10000, Description: "initial", CreatedAt: time.Now()},
				{ID: 2, UserID: 10, Type: domain.TxTypePurchase, Amount: -3000, BalanceAfter: 7000, Description: "order", CreatedAt: time.Now()},
			}, 2, nil
		},
	}
	h := NewWalletHandler(store)
	req := walletReq(http.MethodGet, "/wallet/transactions?page=1&page_size=20", "", 10)
	rec := httptest.NewRecorder()
	h.HandleListTransactions(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	data := resp.Data.(map[string]interface{})
	total := data["total"].(float64)
	if total != 2 {
		t.Errorf("total = %v, want 2", total)
	}
}

func TestHandleAdminAdjust_Success(t *testing.T) {
	store := &mockWalletStore{
		adjustFn: func(_ context.Context, uid int64, amount int64, txType, desc string, _ *string) (*domain.WalletTransaction, error) {
			return &domain.WalletTransaction{
				ID: 12, UserID: uid, Type: txType, Amount: amount,
				BalanceAfter: 105600, Description: desc, CreatedAt: time.Now(),
			}, nil
		},
	}
	h := NewWalletHandler(store)
	body := `{"user_id":5,"amount":10000,"type":"recharge","description":"test"}`
	req := walletReq(http.MethodPost, "/admin/wallet/adjust", body, 1)
	rec := httptest.NewRecorder()
	h.HandleAdminAdjust(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp APIResponse
	json.NewDecoder(rec.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["transaction_id"].(float64) != 12 {
		t.Errorf("transaction_id = %v, want 12", data["transaction_id"])
	}
	if data["balance"].(float64) != 105600 {
		t.Errorf("balance = %v, want 105600", data["balance"])
	}
}

func TestHandleAdminAdjust_InvalidBody(t *testing.T) {
	h := NewWalletHandler(&mockWalletStore{})
	req := walletReq(http.MethodPost, "/admin/wallet/adjust", "not json", 1)
	rec := httptest.NewRecorder()
	h.HandleAdminAdjust(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAdminAdjust_MissingUserID(t *testing.T) {
	h := NewWalletHandler(&mockWalletStore{})
	body := `{"amount":10000,"type":"recharge"}`
	req := walletReq(http.MethodPost, "/admin/wallet/adjust", body, 1)
	rec := httptest.NewRecorder()
	h.HandleAdminAdjust(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAdminAdjust_ZeroAmount(t *testing.T) {
	h := NewWalletHandler(&mockWalletStore{})
	body := `{"user_id":5,"amount":0,"type":"recharge"}`
	req := walletReq(http.MethodPost, "/admin/wallet/adjust", body, 1)
	rec := httptest.NewRecorder()
	h.HandleAdminAdjust(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAdminAdjust_InvalidType(t *testing.T) {
	h := NewWalletHandler(&mockWalletStore{})
	body := `{"user_id":5,"amount":10000,"type":"purchase"}`
	req := walletReq(http.MethodPost, "/admin/wallet/adjust", body, 1)
	rec := httptest.NewRecorder()
	h.HandleAdminAdjust(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleAdminAdjust_InsufficientBalance(t *testing.T) {
	store := &mockWalletStore{
		adjustFn: func(_ context.Context, _ int64, _ int64, _, _ string, _ *string) (*domain.WalletTransaction, error) {
			return nil, repo.ErrInsufficientBalance
		},
	}
	h := NewWalletHandler(store)
	body := `{"user_id":5,"amount":-99999,"type":"adjustment","description":"deduct"}`
	req := walletReq(http.MethodPost, "/admin/wallet/adjust", body, 1)
	rec := httptest.NewRecorder()
	h.HandleAdminAdjust(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
