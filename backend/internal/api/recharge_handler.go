package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// Recharge amount limits (JPY).
const (
	MinRechargeAmountJPY int64 = 1000
	MaxRechargeAmountJPY int64 = 500000
)

// RechargeStore abstracts recharge order persistence.
type RechargeStore interface {
	Create(ctx context.Context, order *domain.RechargeOrder) (*domain.RechargeOrder, error)
	GetByRechargeNo(ctx context.Context, rechargeNo string) (*domain.RechargeOrder, error)
	UpdateState(ctx context.Context, rechargeNo string, state int) error
}

// PaymentGateway abstracts payment provider integration.
// Each payment method (WechatPay, Alipay, etc.) implements this interface.
// When a payment method is integrated, register its implementation in the gateways map.
type PaymentGateway interface {
	// CreatePayment initiates a payment and returns a pay URL or QR code content for the user.
	CreatePayment(ctx context.Context, order *domain.RechargeOrder) (payURL string, err error)
	// VerifyCallback verifies the payment provider's callback notification.
	// Returns rechargeNo, whether payment succeeded, and any error.
	VerifyCallback(r *http.Request) (rechargeNo string, success bool, err error)
}

// RechargeHandler handles wallet top-up endpoints.
type RechargeHandler struct {
	recharges RechargeStore
	wallets   WalletStore
	gateways  map[string]PaymentGateway // key = pay_method constant
}

// NewRechargeHandler creates a RechargeHandler.
// gateways may be empty; register each PaymentGateway when the method is integrated.
func NewRechargeHandler(recharges RechargeStore, wallets WalletStore, gateways map[string]PaymentGateway) *RechargeHandler {
	if gateways == nil {
		gateways = map[string]PaymentGateway{}
	}
	return &RechargeHandler{recharges: recharges, wallets: wallets, gateways: gateways}
}

// HandleGetPayMethods handles GET /api/v1/wallet/pay-methods.
// Returns all supported payment methods and their availability.
func (h *RechargeHandler) HandleGetPayMethods(w http.ResponseWriter, r *http.Request) {
	Success(w, r, map[string]interface{}{
		"methods": domain.SupportedPayMethods,
	})
}

// HandleCreateRecharge handles POST /api/v1/wallet/recharge.
// Creates a recharge order and (if the gateway is integrated) returns a pay URL.
func (h *RechargeHandler) HandleCreateRecharge(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req struct {
		AmountJPY int64  `json:"amount_jpy"`
		PayMethod string `json:"pay_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.AmountJPY < MinRechargeAmountJPY || req.AmountJPY > MaxRechargeAmountJPY {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002,
			fmt.Sprintf("amount must be between %d and %d JPY", MinRechargeAmountJPY, MaxRechargeAmountJPY))
		return
	}
	if !domain.IsSupportedPayMethod(req.PayMethod) {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "unsupported payment method")
		return
	}

	order := &domain.RechargeOrder{
		RechargeNo: domain.GenerateRechargeNo(),
		UserID:     userID,
		AmountJPY:  req.AmountJPY,
		PayMethod:  req.PayMethod,
		State:      domain.RechargeStatePending,
	}

	created, err := h.recharges.Create(r.Context(), order)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to create recharge order")
		return
	}

	// Call payment gateway if integrated; silently skip if not yet implemented.
	payURL := ""
	if gw, ok := h.gateways[req.PayMethod]; ok {
		payURL, err = gw.CreatePayment(r.Context(), created)
		if err != nil {
			log.Printf("[recharge] create payment %s via %s: %v", created.RechargeNo, req.PayMethod, err)
			// Gateway errors don't fail the request — client can query status later.
		}
	}

	Success(w, r, map[string]interface{}{
		"recharge_no": created.RechargeNo,
		"amount_jpy":  created.AmountJPY,
		"pay_method":  created.PayMethod,
		"pay_url":     payURL,
		"state":       created.State,
	})
}

// HandleRechargeCallback handles POST /api/v1/wallet/recharge/callback/{method}.
// This is a public webhook endpoint called by payment providers.
// It verifies the callback, credits the wallet, and marks the order as paid.
func (h *RechargeHandler) HandleRechargeCallback(w http.ResponseWriter, r *http.Request) {
	method := chi.URLParam(r, "method")

	gw, ok := h.gateways[method]
	if !ok {
		// Unknown or unintegrated method — ACK to prevent provider retry storms.
		w.WriteHeader(http.StatusOK)
		return
	}

	rechargeNo, success, err := gw.VerifyCallback(r)
	if err != nil {
		log.Printf("[recharge] callback verify error for %s: %v", method, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	if !success {
		if rechargeNo != "" {
			if updateErr := h.recharges.UpdateState(r.Context(), rechargeNo, domain.RechargeStateFailed); updateErr != nil {
				log.Printf("[recharge] mark failed %s: %v", rechargeNo, updateErr)
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	order, err := h.recharges.GetByRechargeNo(r.Context(), rechargeNo)
	if err != nil {
		log.Printf("[recharge] callback get order %s: %v", rechargeNo, err)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Idempotent: skip if already processed.
	if order.State != domain.RechargeStatePending {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Credit wallet.
	desc := fmt.Sprintf("Recharge via %s (order %s)", method, rechargeNo)
	if _, err := h.wallets.Adjust(r.Context(), order.UserID, order.AmountJPY, domain.TxTypeRecharge, desc, nil); err != nil {
		log.Printf("[recharge] credit wallet for %s: %v", rechargeNo, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Mark order as paid.
	if err := h.recharges.UpdateState(r.Context(), rechargeNo, domain.RechargeStatePaid); err != nil {
		log.Printf("[recharge] update state paid for %s: %v", rechargeNo, err)
	}

	w.WriteHeader(http.StatusOK)
}

// HandleGetRechargeStatus handles GET /api/v1/wallet/recharge/{rechargeNo}.
// Allows the client to poll for payment completion.
func (h *RechargeHandler) HandleGetRechargeStatus(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	rechargeNo := chi.URLParam(r, "rechargeNo")

	order, err := h.recharges.GetByRechargeNo(r.Context(), rechargeNo)
	if err != nil || order.UserID != userID {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "recharge order not found")
		return
	}

	Success(w, r, map[string]interface{}{
		"recharge_no": order.RechargeNo,
		"amount_jpy":  order.AmountJPY,
		"pay_method":  order.PayMethod,
		"state":       order.State,
		"created_at":  order.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}
