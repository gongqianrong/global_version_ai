package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rakutao/collection-gateway/internal/domain"
	"github.com/rakutao/collection-gateway/internal/repo"
)

// OrderLinkStore abstracts order_link persistence.
type OrderLinkStore interface {
	Create(ctx context.Context, ol *domain.OrderLink) error
	GetByLinkNo(ctx context.Context, linkNo string) (*domain.OrderLink, error)
	ListByUser(ctx context.Context, userID int64, state, limit, offset int) ([]domain.OrderLink, int64, error)
	UpdateState(ctx context.Context, linkNo string, fromState, toState int) error
	Quote(ctx context.Context, linkNo string, items []repo.QuoteItem, totalAmount int64) error
	SetOrderNumber(ctx context.Context, linkNo string, orderNumber string) error
}

// OrderCreator abstracts creating real orders (used by order link pay flow).
type OrderCreator interface {
	CreateOrder(ctx context.Context, order *domain.Order, details []domain.OrderDetail) (*domain.Order, error)
}

// OrderLinkHandler handles order-link endpoints.
type OrderLinkHandler struct {
	store      OrderLinkStore
	wallet     WalletStore
	orderStore OrderCreator
}

// NewOrderLinkHandler creates an OrderLinkHandler.
func NewOrderLinkHandler(store OrderLinkStore, wallet WalletStore, orderStore OrderCreator) *OrderLinkHandler {
	return &OrderLinkHandler{store: store, wallet: wallet, orderStore: orderStore}
}

// --- Request types ---

type submitOrderLinkRequest struct {
	Items  []submitOrderLinkItem `json:"items"`
	Remark string                `json:"remark"`
}

type submitOrderLinkItem struct {
	GoodsURL  string `json:"goods_url"`
	GoodsName string `json:"goods_name"`
	GoodsImg  string `json:"goods_img"`
	Quantity  int    `json:"quantity"`
	Remark    string `json:"remark"`
}

type quoteOrderLinkRequest struct {
	Items       []repo.QuoteItem `json:"items"`
	TotalAmount int64            `json:"total_amount"`
}

// HandleSubmitOrderLink handles POST /api/v1/order-link/submit.
func (h *OrderLinkHandler) HandleSubmitOrderLink(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	var req submitOrderLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if len(req.Items) == 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "items is required")
		return
	}

	ol := &domain.OrderLink{
		LinkNo: domain.GenerateLinkNo(),
		UserID: userID,
		State:  domain.OrderLinkStatePending,
		Remark: req.Remark,
	}
	for _, item := range req.Items {
		if item.GoodsURL == "" {
			ErrorWithCode(w, r, http.StatusBadRequest, 40002, "goods_url is required for each item")
			return
		}
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}
		ol.Items = append(ol.Items, domain.OrderLinkItem{
			GoodsURL:  item.GoodsURL,
			GoodsName: item.GoodsName,
			GoodsImg:  item.GoodsImg,
			Quantity:  qty,
			Remark:    item.Remark,
		})
	}

	if err := h.store.Create(r.Context(), ol); err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to create order link")
		return
	}

	Success(w, r, ol)
}

// HandleListOrderLinks handles GET /api/v1/order-links?page=1&page_size=20&state=-1.
func (h *OrderLinkHandler) HandleListOrderLinks(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	state := -1
	if s := r.URL.Query().Get("state"); s != "" {
		state, _ = strconv.Atoi(s)
	}
	offset := (page - 1) * pageSize

	items, total, err := h.store.ListByUser(r.Context(), userID, state, pageSize, offset)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to list order links")
		return
	}
	if items == nil {
		items = []domain.OrderLink{}
	}

	Success(w, r, map[string]interface{}{
		"items": items,
		"total": total,
		"page":  page,
	})
}

// HandleGetOrderLink handles GET /api/v1/order-link/{linkNo}.
func (h *OrderLinkHandler) HandleGetOrderLink(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	linkNo := chi.URLParam(r, "linkNo")

	ol, err := h.store.GetByLinkNo(r.Context(), linkNo)
	if err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "order link not found")
		return
	}
	if ol.UserID != userID {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "order link not found")
		return
	}

	Success(w, r, ol)
}

// HandlePayOrderLink handles POST /api/v1/order-link/{linkNo}/pay.
func (h *OrderLinkHandler) HandlePayOrderLink(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	linkNo := chi.URLParam(r, "linkNo")

	ol, err := h.store.GetByLinkNo(r.Context(), linkNo)
	if err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "order link not found")
		return
	}
	if ol.UserID != userID {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "order link not found")
		return
	}
	if ol.State != domain.OrderLinkStateQuoted {
		ErrorWithCode(w, r, http.StatusBadRequest, 40012, "order link is not payable")
		return
	}
	if ol.TotalAmount <= 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40012, "total amount must be greater than 0")
		return
	}

	// Check wallet balance.
	wallet, err := h.wallet.GetOrCreateWallet(r.Context(), userID)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to get wallet")
		return
	}
	if wallet.Balance < ol.TotalAmount {
		ErrorWithCode(w, r, http.StatusBadRequest, 40011, "insufficient balance")
		return
	}

	// Create real order.
	orderNumber := domain.GenerateOrderNumber()
	order := &domain.Order{
		OrderNumber:       orderNumber,
		UserID:            userID,
		OrderState:        domain.OrderStatePaid,
		OrderTotalJp:      ol.TotalAmount,
		OrderPaytype:      0,
		OrderRemark:       ol.Remark,
		OrderPurchaseType: domain.OrderPurchaseTypeOrderLink,
	}

	var details []domain.OrderDetail
	for _, item := range ol.Items {
		details = append(details, domain.OrderDetail{
			GoodsMid:      "",
			GoodsName:     item.GoodsName,
			GoodsNum:      item.Quantity,
			GoodsImg:      item.GoodsImg,
			GoodsUrl:      item.GoodsURL,
			GoodsAmountJp: item.UnitPrice * int64(item.Quantity),
			State:         domain.OrderStatePaid,
		})
	}

	createdOrder, err := h.orderStore.CreateOrder(r.Context(), order, details)
	if err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to create order")
		return
	}

	// Deduct wallet.
	relatedOrder := createdOrder.OrderNumber
	_, err = h.wallet.Adjust(r.Context(), userID, -ol.TotalAmount, domain.TxTypePurchase,
		"指定购买: "+linkNo, &relatedOrder)
	if err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40011, "insufficient balance")
		return
	}

	// Update order link state to Paid and set order_number.
	if err := h.store.SetOrderNumber(r.Context(), linkNo, createdOrder.OrderNumber); err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to update order link")
		return
	}

	Success(w, r, map[string]interface{}{
		"link_no":      linkNo,
		"order_number": createdOrder.OrderNumber,
		"state":        domain.OrderLinkStatePaid,
	})
}

// HandleCancelOrderLink handles POST /api/v1/order-link/{linkNo}/cancel.
func (h *OrderLinkHandler) HandleCancelOrderLink(w http.ResponseWriter, r *http.Request) {
	userID := UserIDFromContext(r.Context())
	linkNo := chi.URLParam(r, "linkNo")

	ol, err := h.store.GetByLinkNo(r.Context(), linkNo)
	if err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "order link not found")
		return
	}
	if ol.UserID != userID {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "order link not found")
		return
	}

	if !domain.IsValidOrderLinkTransition(ol.State, domain.OrderLinkStateCancelled) {
		ErrorWithCode(w, r, http.StatusBadRequest, 40013, "order link cannot be cancelled")
		return
	}

	if err := h.store.UpdateState(r.Context(), linkNo, ol.State, domain.OrderLinkStateCancelled); err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to cancel order link")
		return
	}

	Success(w, r, map[string]interface{}{
		"link_no": linkNo,
		"state":   domain.OrderLinkStateCancelled,
	})
}

// HandleAdminQuoteOrderLink handles POST /api/v1/admin/order-link/{linkNo}/quote.
func (h *OrderLinkHandler) HandleAdminQuoteOrderLink(w http.ResponseWriter, r *http.Request) {
	linkNo := chi.URLParam(r, "linkNo")

	var req quoteOrderLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "invalid request body")
		return
	}
	if req.TotalAmount <= 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "total_amount must be greater than 0")
		return
	}
	if len(req.Items) == 0 {
		ErrorWithCode(w, r, http.StatusBadRequest, 40002, "items is required")
		return
	}

	if err := h.store.Quote(r.Context(), linkNo, req.Items, req.TotalAmount); err != nil {
		ErrorWithCode(w, r, http.StatusBadRequest, 40017, "failed to quote: "+err.Error())
		return
	}

	Success(w, r, map[string]interface{}{
		"link_no": linkNo,
		"state":   domain.OrderLinkStateQuoted,
	})
}

// HandleAdminCancelOrderLink handles POST /api/v1/admin/order-link/{linkNo}/cancel.
func (h *OrderLinkHandler) HandleAdminCancelOrderLink(w http.ResponseWriter, r *http.Request) {
	linkNo := chi.URLParam(r, "linkNo")

	ol, err := h.store.GetByLinkNo(r.Context(), linkNo)
	if err != nil {
		ErrorWithCode(w, r, http.StatusNotFound, 40401, "order link not found")
		return
	}

	if !domain.IsValidOrderLinkTransition(ol.State, domain.OrderLinkStateCancelled) {
		ErrorWithCode(w, r, http.StatusBadRequest, 40013, "order link cannot be cancelled")
		return
	}

	if err := h.store.UpdateState(r.Context(), linkNo, ol.State, domain.OrderLinkStateCancelled); err != nil {
		ErrorWithCode(w, r, http.StatusInternalServerError, 50001, "failed to cancel order link")
		return
	}

	Success(w, r, map[string]interface{}{
		"link_no": linkNo,
		"state":   domain.OrderLinkStateCancelled,
	})
}
