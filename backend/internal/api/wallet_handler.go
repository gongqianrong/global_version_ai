package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// WalletStore abstracts wallet persistence for testing.
type WalletStore interface {
	GetOrCreateWallet(ctx context.Context, userID int64) (*domain.Wallet, error)
	Adjust(ctx context.Context, userID int64, amount int64, txType, description string, relatedOrder *string) (*domain.WalletTransaction, error)
	ListTransactions(ctx context.Context, userID int64, limit, offset int) ([]domain.WalletTransaction, int64, error)
}

// WalletHandler handles wallet endpoints.
type WalletHandler struct {
	store WalletStore
}

// NewWalletHandler creates a WalletHandler.
func NewWalletHandler(store WalletStore) *WalletHandler {
	return &WalletHandler{store: store}
}

// HandleGetBalance handles GET /api/v1/wallet/balance.
func (h *WalletHandler) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	wallet, err := h.store.GetOrCreateWallet(r.Context(), userID)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to get wallet")
		return
	}

	Success(w, r, map[string]interface{}{
		"balance":  wallet.Balance,
		"currency": "JPY",
	})
}

// HandleListTransactions handles GET /api/v1/wallet/transactions?page=1&page_size=20.
func (h *WalletHandler) HandleListTransactions(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	items, total, err := h.store.ListTransactions(r.Context(), userID, pageSize, offset)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to list transactions")
		return
	}

	if items == nil {
		items = []domain.WalletTransaction{}
	}

	Success(w, r, map[string]interface{}{
		"items": items,
		"total": total,
		"page":  page,
	})
}

type adminAdjustRequest struct {
	UserID      int64   `json:"user_id"`
	Amount      int64   `json:"amount"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	RelatedOrder *string `json:"related_order,omitempty"`
}

// HandleAdminAdjust handles POST /api/v1/admin/wallet/adjust.
func (h *WalletHandler) HandleAdminAdjust(w http.ResponseWriter, r *http.Request) {
	var req adminAdjustRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}

	if req.UserID <= 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "user_id is required")
		return
	}
	if req.Amount == 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "amount cannot be zero")
		return
	}

	// Validate transaction type.
	switch req.Type {
	case domain.TxTypeRecharge, domain.TxTypeAdjustment, domain.TxTypeRefund:
		// allowed for admin
	default:
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "type must be recharge, adjustment, or refund")
		return
	}

	txn, err := h.store.Adjust(r.Context(), req.UserID, req.Amount, req.Type, req.Description, req.RelatedOrder)
	if err != nil {
		if err.Error() == "insufficient balance" {
			ErrorWithCode(w, r, http.StatusBadRequest, 40009, "insufficient balance")
			return
		}
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to adjust wallet")
		return
	}

	Success(w, r, map[string]interface{}{
		"balance":        txn.BalanceAfter,
		"transaction_id": txn.ID,
	})
}
