package service

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// CommissionRate is the purchasing commission rate (10%).
const CommissionRate = 0.10

// Errors returned by the order service.
var (
	ErrEmptyItems          = errors.New("items list is empty")
	ErrProductUnavailable  = errors.New("product unavailable")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrOrderNotFound       = errors.New("order not found")
	ErrOrderNotPayable     = errors.New("order is not in pending state")
	ErrOrderNotCancellable = errors.New("order is not cancellable")
)

// OrderItem is a single item in a settlement/confirm request.
type OrderItem struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// SettlementDetail holds per-item detail for the settlement response.
type SettlementDetail struct {
	GoodsMid      string `json:"goodsMid"`
	GoodsName     string `json:"goodsName"`
	GoodsNum      int    `json:"goodsNum"`
	GoodsImg      string `json:"goodsImg"`
	GoodsUrl      string `json:"goodsUrl"`
	GoodsSellerId string `json:"goodsSellerId"`
	GoodsAmountJp int64  `json:"goodsAmountJp"`
	CommissionFee int64  `json:"commissionFeeJp"`
	ShippingFee   int64  `json:"shippingFee"`
	Platform      string `json:"platform"`
	Condition     string `json:"condition"`
	SellerID      string `json:"sellerId"`
	State         int    `json:"state"`
	BlockLevel    int    `json:"blockLevel"`
}

// SettlementResult is the response for the settlement (preview) endpoint.
type SettlementResult struct {
	OrderTotalJp     int64              `json:"orderTotalJp"`
	CommissionFeeJp  int64              `json:"commissionFeeJp"`
	OrderInpriceJp   int64              `json:"orderInpriceJp"`
	OrderRate        int                `json:"orderRate"`
	OrderPaytype     int                `json:"orderPaytype"`
	OrderType        int                `json:"orderType"`
	IsChange         int                `json:"isChange"`
	OrderDetailList  []SettlementDetail `json:"orderDetailList"`
	WalletBalance    int64              `json:"walletBalance"`
	TotalShippingFee int64              `json:"totalShippingFee"`
}

// ConfirmResult is the response for the order confirm endpoint.
type ConfirmResult struct {
	OrderNumber      string             `json:"orderNumber"`
	OrderState       int                `json:"orderState"`
	OrderTotalJp     int64              `json:"orderTotalJp"`
	CommissionFeeJp  int64              `json:"commissionFeeJp"`
	OrderInpriceJp   int64              `json:"orderInpriceJp"`
	OrderPaytype     int                `json:"orderPaytype"`
	OrderDetailList  []SettlementDetail `json:"orderDetailList"`
	TotalShippingFee int64              `json:"totalShippingFee"`
}

// PayResult is the response for the order pay endpoint.
type PayResult struct {
	OrderNumber  string `json:"orderNumber"`
	OrderState   int    `json:"orderState"`
	PaidAmount   int64  `json:"paidAmount"`
	BalanceAfter int64  `json:"balanceAfter"`
}

// ProductFetcher retrieves a product by its unified ID.
type ProductFetcher interface {
	GetProduct(ctx context.Context, id string) (*domain.UnifiedProduct, error)
}

// WalletService handles wallet operations.
type WalletService interface {
	GetOrCreateWallet(ctx context.Context, userID int64) (*domain.Wallet, error)
	Adjust(ctx context.Context, userID int64, amount int64, txType, description string, relatedOrder *string) (*domain.WalletTransaction, error)
}

// OrderRepository persists orders.
type OrderRepository interface {
	CreateOrder(ctx context.Context, order *domain.Order, details []domain.OrderDetail) (*domain.Order, error)
	GetByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, []domain.OrderDetail, error)
	UpdateState(ctx context.Context, orderNumber string, fromState, toState int) error
	ListByUser(ctx context.Context, userID int64, state, limit, offset int) ([]domain.Order, int64, error)
	GetDetailsByOrderIDs(ctx context.Context, orderIDs []int64) (map[int64][]domain.OrderDetail, error)
}

// OrderDetailResponse is a single order detail item in API responses.
type OrderDetailResponse struct {
	GoodsMid      string `json:"goodsMid"`
	GoodsName     string `json:"goodsName"`
	GoodsNum      int    `json:"goodsNum"`
	GoodsImg      string `json:"goodsImg"`
	GoodsUrl      string `json:"goodsUrl"`
	GoodsAmountJp int64  `json:"goodsAmountJp"`
	CommissionFee int64  `json:"commissionFeeJp"`
	ShippingFee   int64  `json:"shippingFee"`
	SellerID      string `json:"sellerId"`
	SellerName    string `json:"sellerName"`
	Platform      string `json:"platform"`
	Condition     string `json:"condition"`
	State         int    `json:"state"`
}

// OrderResponse is the full order response for detail and list endpoints.
type OrderResponse struct {
	OrderNumber      string                `json:"orderNumber"`
	OrderState       int                   `json:"orderState"`
	OrderTotalJp     int64                 `json:"orderTotalJp"`
	CommissionFeeJp  int64                 `json:"commissionFeeJp"`
	ShippingFeeJp    int64                 `json:"shippingFeeJp"`
	OrderInpriceJp   int64                 `json:"orderInpriceJp"`
	OrderPaytype     int                   `json:"orderPaytype"`
	OrderRemark      string                `json:"orderRemark"`
	FirstGoodsImg    string                `json:"firstGoodsImg"`
	OrderDetailList  []OrderDetailResponse `json:"orderDetailList,omitempty"`
	CreatedAt        string                `json:"createdAt"`
}

// CartRemover removes items from the cart.
type CartRemover interface {
	Remove(ctx context.Context, userID int64, productID string) error
}

// AdminSyncClient handles syncing to admin backend.
type AdminSyncClient interface {
	SyncOrderAsync(ctx context.Context, order *domain.Order, details []domain.OrderDetail, userID string)
	SyncPaymentAsync(ctx context.Context, orderNumber string, paymentAmount int64, userID string)
}

// OrderService handles order settlement, confirmation, and payment logic.
type OrderService struct {
	products    ProductFetcher
	wallet      WalletService
	orders      OrderRepository
	orderRepo   interface{} // Concrete *OrderRepo for PayOrderAtomic
	walletRepo  interface{} // Concrete *WalletRepo for atomic transactions  
	cart        CartRemover
	adminSync   AdminSyncClient // Optional: sync to admin backend
}

// NewOrderService creates an OrderService.
func NewOrderService(pf ProductFetcher, ws WalletService, or OrderRepository, cr CartRemover) *OrderService {
	return &OrderService{
		products:    pf,
		wallet:      ws,
		orders:      or,
		orderRepo:   or,  // Same instance, used for concrete methods
		walletRepo:  ws,  // Same instance, used for atomic transactions
		cart:        cr,
		adminSync:   nil, // Set via SetAdminSync if needed
	}
}

// SetAdminSync sets the admin sync client (optional).
func (s *OrderService) SetAdminSync(client AdminSyncClient) {
	s.adminSync = client
}

// Settlement computes an order preview without persisting anything.
func (s *OrderService) Settlement(ctx context.Context, userID int64, items []OrderItem) (*SettlementResult, error) {
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}

	details, err := s.buildDetails(ctx, items)
	if err != nil {
		return nil, err
	}

	var totalGoods, totalCommission, totalShipping int64
	for _, d := range details {
		totalGoods += d.GoodsAmountJp * int64(d.GoodsNum)
		totalCommission += d.CommissionFee * int64(d.GoodsNum)
		totalShipping += d.ShippingFee * int64(d.GoodsNum)
	}

	orderTotal := totalGoods + totalCommission + totalShipping

	wallet, err := s.wallet.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("order_service: get wallet: %w", err)
	}

	return &SettlementResult{
		OrderTotalJp:     orderTotal,
		CommissionFeeJp:  totalCommission,
		OrderInpriceJp:   orderTotal,
		OrderRate:        1,
		OrderPaytype:     0,
		OrderType:        0,
		IsChange:         0,
		OrderDetailList:  details,
		WalletBalance:    wallet.Balance,
		TotalShippingFee: totalShipping,
	}, nil
}

// Confirm creates an order in Pending state and removes cart items. Does NOT deduct wallet.
func (s *OrderService) Confirm(ctx context.Context, userID int64, items []OrderItem, paytype int, remark string, purchaseType int) (*ConfirmResult, error) {
	if len(items) == 0 {
		return nil, ErrEmptyItems
	}

	details, err := s.buildDetails(ctx, items)
	if err != nil {
		return nil, err
	}

	var totalGoods, totalCommission, totalShipping int64
	for _, d := range details {
		totalGoods += d.GoodsAmountJp * int64(d.GoodsNum)
		totalCommission += d.CommissionFee * int64(d.GoodsNum)
		totalShipping += d.ShippingFee * int64(d.GoodsNum)
	}
	orderTotal := totalGoods + totalCommission + totalShipping

	orderNumber := domain.GenerateOrderNumber()

	order := &domain.Order{
		OrderNumber:       orderNumber,
		UserID:            userID,
		OrderState:        domain.OrderStatePending,
		OrderTotalJp:      orderTotal,
		CommissionFeeJp:   totalCommission,
		ShippingFeeJp:     totalShipping,
		OrderInpriceJp:    orderTotal,
		OrderPaytype:      paytype,
		OrderRemark:       remark,
		OrderPurchaseType: purchaseType,
	}

	var orderDetails []domain.OrderDetail
	for _, d := range details {
		orderDetails = append(orderDetails, domain.OrderDetail{
			GoodsMid:        d.GoodsMid,
			GoodsName:       d.GoodsName,
			GoodsNum:        d.GoodsNum,
			GoodsImg:        d.GoodsImg,
			GoodsUrl:        d.GoodsUrl,
			GoodsAmountJp:   d.GoodsAmountJp,
			CommissionFeeJp: d.CommissionFee,
			ShippingFeeJp:   d.ShippingFee,
			SellerID:        d.SellerID,
			SellerName:      d.GoodsSellerId,
			Platform:        d.Platform,
			Condition:       d.Condition,
			State:           0,
		})
	}

	// Persist order with Pending state.
	order, err = s.orders.CreateOrder(ctx, order, orderDetails)
	if err != nil {
		return nil, fmt.Errorf("order_service: create order: %w", err)
	}

	// Sync to admin backend (async, best effort)
	if s.adminSync != nil {
		userIDStr := fmt.Sprintf("%d", userID)
		go s.adminSync.SyncOrderAsync(context.Background(), order, orderDetails, userIDStr)
	}

	// Remove items from cart (best effort).
	for _, item := range items {
		if removeErr := s.cart.Remove(ctx, userID, item.ProductID); removeErr != nil {
			log.Printf("[order] failed to remove cart item %s: %v", item.ProductID, removeErr)
		}
	}

	return &ConfirmResult{
		OrderNumber:      orderNumber,
		OrderState:       domain.OrderStatePending,
		OrderTotalJp:     orderTotal,
		CommissionFeeJp:  totalCommission,
		OrderInpriceJp:   orderTotal,
		OrderPaytype:     paytype,
		OrderDetailList:  details,
		TotalShippingFee: totalShipping,
	}, nil
}

// Pay deducts wallet balance for a pending order and transitions it to Paid state.
// FIXED: Now uses atomic transaction to ensure consistency.
func (s *OrderService) Pay(ctx context.Context, userID int64, orderNumber string) (*PayResult, error) {
	// Fetch order and verify ownership + state.
	order, _, err := s.orders.GetByOrderNumber(ctx, orderNumber)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.UserID != userID {
		return nil, ErrOrderNotFound
	}
	if order.OrderState != domain.OrderStatePending {
		return nil, ErrOrderNotPayable
	}

	// Check wallet balance before attempting payment.
	wallet, err := s.wallet.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("order_service: get wallet: %w", err)
	}
	if wallet.Balance < order.OrderInpriceJp {
		return nil, ErrInsufficientBalance
	}

	// CRITICAL FIX: Atomic payment using PayOrderAtomic
	// This ensures wallet deduction and order state update happen in ONE transaction.
	// If either fails, both are rolled back - no partial success.
	
	// Type assert to concrete types for atomic payment
	type orderRepoWithAtomic interface {
		PayOrderAtomic(ctx context.Context, orderNumber string, userID int64, amount int64, walletRepo interface{}) (*domain.WalletTransaction, error)
	}
	
	orderRepo, ok := s.orderRepo.(orderRepoWithAtomic)
	if !ok {
		return nil, fmt.Errorf("order_service: order repository does not support atomic payment")
	}
	
	wtx, err := orderRepo.PayOrderAtomic(ctx, orderNumber, userID, order.OrderInpriceJp, s.walletRepo)
	if err != nil {
		return nil, fmt.Errorf("order_service: atomic payment failed: %w", err)
	}

	// Sync payment success to admin backend (async, best effort)
	if s.adminSync != nil {
		userIDStr := fmt.Sprintf("%d", userID)
		go s.adminSync.SyncPaymentAsync(context.Background(), orderNumber, order.OrderInpriceJp, userIDStr)
	}

	return &PayResult{
		OrderNumber:  orderNumber,
		OrderState:   domain.OrderStatePaid,
		PaidAmount:   order.OrderInpriceJp,
		BalanceAfter: wtx.BalanceAfter,
	}, nil
}

// GetDetail returns order detail for a specific order owned by the user.
func (s *OrderService) GetDetail(ctx context.Context, userID int64, orderNumber string) (*OrderResponse, error) {
	order, details, err := s.orders.GetByOrderNumber(ctx, orderNumber)
	if err != nil {
		return nil, ErrOrderNotFound
	}
	if order.UserID != userID {
		return nil, ErrOrderNotFound
	}

	var detailList []OrderDetailResponse
	for _, d := range details {
		detailList = append(detailList, OrderDetailResponse{
			GoodsMid:      d.GoodsMid,
			GoodsName:     d.GoodsName,
			GoodsNum:      d.GoodsNum,
			GoodsImg:      d.GoodsImg,
			GoodsUrl:      d.GoodsUrl,
			GoodsAmountJp: d.GoodsAmountJp,
			CommissionFee: d.CommissionFeeJp,
			ShippingFee:   d.ShippingFeeJp,
			SellerID:      d.SellerID,
			SellerName:    d.SellerName,
			Platform:      d.Platform,
			Condition:     d.Condition,
			State:         d.State,
		})
	}

	firstImg := ""
	if len(detailList) > 0 {
		firstImg = detailList[0].GoodsImg
	}

	return &OrderResponse{
		OrderNumber:     order.OrderNumber,
		OrderState:      order.OrderState,
		OrderTotalJp:    order.OrderTotalJp,
		CommissionFeeJp: order.CommissionFeeJp,
		ShippingFeeJp:   order.ShippingFeeJp,
		OrderInpriceJp:  order.OrderInpriceJp,
		OrderPaytype:    order.OrderPaytype,
		OrderRemark:     order.OrderRemark,
		FirstGoodsImg:   firstImg,
		OrderDetailList: detailList,
		CreatedAt:       order.CreatedAt.Format("2006-01-02 15:04:05"),
	}, nil
}

// ListOrders returns paginated orders for a user, optionally filtered by state.
func (s *OrderService) ListOrders(ctx context.Context, userID int64, state, page, pageSize int) ([]OrderResponse, int64, error) {
	offset := (page - 1) * pageSize
	orders, total, err := s.orders.ListByUser(ctx, userID, state, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("order_service: list: %w", err)
	}

	// Batch fetch order details for all orders.
	var orderIDs []int64
	for _, o := range orders {
		orderIDs = append(orderIDs, o.ID)
	}
	detailsMap, err := s.orders.GetDetailsByOrderIDs(ctx, orderIDs)
	if err != nil {
		return nil, 0, fmt.Errorf("order_service: list details: %w", err)
	}

	var list []OrderResponse
	for _, o := range orders {
		var detailList []OrderDetailResponse
		for _, d := range detailsMap[o.ID] {
			detailList = append(detailList, OrderDetailResponse{
				GoodsMid:      d.GoodsMid,
				GoodsName:     d.GoodsName,
				GoodsNum:      d.GoodsNum,
				GoodsImg:      d.GoodsImg,
				GoodsUrl:      d.GoodsUrl,
				GoodsAmountJp: d.GoodsAmountJp,
				CommissionFee: d.CommissionFeeJp,
				ShippingFee:   d.ShippingFeeJp,
				SellerID:      d.SellerID,
				SellerName:    d.SellerName,
				Platform:      d.Platform,
				Condition:     d.Condition,
				State:         d.State,
			})
		}
		firstImg := ""
		if len(detailList) > 0 {
			firstImg = detailList[0].GoodsImg
		}
		list = append(list, OrderResponse{
			OrderNumber:     o.OrderNumber,
			OrderState:      o.OrderState,
			OrderTotalJp:    o.OrderTotalJp,
			CommissionFeeJp: o.CommissionFeeJp,
			ShippingFeeJp:   o.ShippingFeeJp,
			OrderInpriceJp:  o.OrderInpriceJp,
			OrderPaytype:    o.OrderPaytype,
			OrderRemark:     o.OrderRemark,
			FirstGoodsImg:   firstImg,
			OrderDetailList: detailList,
			CreatedAt:       o.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return list, total, nil
}

// Cancel cancels a pending order. Only orders in Pending state can be cancelled.
func (s *OrderService) Cancel(ctx context.Context, userID int64, orderNumber string) error {
	order, _, err := s.orders.GetByOrderNumber(ctx, orderNumber)
	if err != nil {
		return ErrOrderNotFound
	}
	if order.UserID != userID {
		return ErrOrderNotFound
	}
	if order.OrderState != domain.OrderStatePending {
		return ErrOrderNotCancellable
	}

	if err := s.orders.UpdateState(ctx, orderNumber, domain.OrderStatePending, domain.OrderStateCancelled); err != nil {
		return fmt.Errorf("order_service: cancel: %w", err)
	}
	return nil
}

// buildDetails fetches product data from ES and computes per-item fees.
func (s *OrderService) buildDetails(ctx context.Context, items []OrderItem) ([]SettlementDetail, error) {
	var details []SettlementDetail

	for _, item := range items {
		if item.Quantity <= 0 {
			item.Quantity = 1
		}

		product, err := s.products.GetProduct(ctx, item.ProductID)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrProductUnavailable, item.ProductID)
		}

		state := 0
		if !product.IsAvailable() {
			state = 1 // mark as unavailable
		}

		commissionFee := int64(float64(product.PriceJPY) * CommissionRate)

		img := ""
		if len(product.Images) > 0 {
			img = product.Images[0]
		}

		details = append(details, SettlementDetail{
			GoodsMid:      item.ProductID,
			GoodsName:     product.Title,
			GoodsNum:      item.Quantity,
			GoodsImg:      img,
			GoodsUrl:      product.SourceURL,
			GoodsSellerId: product.Seller.SellerName,
			GoodsAmountJp: product.PriceJPY,
			CommissionFee: commissionFee,
			ShippingFee:   product.ShippingFeeJPY,
			Platform:      product.SourcePlatform,
			Condition:     product.Condition,
			SellerID:      product.Seller.SellerID,
			State:         state,
			BlockLevel:    0,
		})
	}

	return details, nil
}
