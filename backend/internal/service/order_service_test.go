package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// --- Mock implementations ---

// mockProductFetcher implements ProductFetcher.
type mockProductFetcher struct {
	products map[string]*domain.UnifiedProduct
	err      error
}

func (m *mockProductFetcher) GetProduct(_ context.Context, id string) (*domain.UnifiedProduct, error) {
	if m.err != nil {
		return nil, m.err
	}
	p, ok := m.products[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return p, nil
}

// mockWalletService implements WalletService.
type mockWalletService struct {
	wallet    *domain.Wallet
	walletErr error
	adjustTx  *domain.WalletTransaction
	adjustErr error
}

func (m *mockWalletService) GetOrCreateWallet(_ context.Context, _ int64) (*domain.Wallet, error) {
	if m.walletErr != nil {
		return nil, m.walletErr
	}
	return m.wallet, nil
}

func (m *mockWalletService) Adjust(_ context.Context, _ int64, _ int64, _, _ string, _ *string) (*domain.WalletTransaction, error) {
	if m.adjustErr != nil {
		return nil, m.adjustErr
	}
	return m.adjustTx, nil
}

// mockOrderRepository implements OrderRepository.
type mockOrderRepository struct {
	createdOrder   *domain.Order
	createErr      error
	order          *domain.Order
	details        []domain.OrderDetail
	getErr         error
	updateStateErr error
	orders         []domain.Order
	total          int64
	listErr        error
	detailsMap     map[int64][]domain.OrderDetail
	detailsMapErr  error
}

func (m *mockOrderRepository) CreateOrder(_ context.Context, order *domain.Order, _ []domain.OrderDetail) (*domain.Order, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	order.ID = 1
	m.createdOrder = order
	return order, nil
}

func (m *mockOrderRepository) GetByOrderNumber(_ context.Context, _ string) (*domain.Order, []domain.OrderDetail, error) {
	if m.getErr != nil {
		return nil, nil, m.getErr
	}
	return m.order, m.details, nil
}

func (m *mockOrderRepository) UpdateState(_ context.Context, _ string, _, _ int) error {
	return m.updateStateErr
}

func (m *mockOrderRepository) ListByUser(_ context.Context, _ int64, _, _, _ int) ([]domain.Order, int64, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	return m.orders, m.total, nil
}

func (m *mockOrderRepository) GetDetailsByOrderIDs(_ context.Context, _ []int64) (map[int64][]domain.OrderDetail, error) {
	if m.detailsMapErr != nil {
		return nil, m.detailsMapErr
	}
	if m.detailsMap != nil {
		return m.detailsMap, nil
	}
	return map[int64][]domain.OrderDetail{}, nil
}

// mockCartRemover implements CartRemover.
type mockCartRemover struct {
	removed  []string
	removeErr error
}

func (m *mockCartRemover) Remove(_ context.Context, _ int64, productID string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removed = append(m.removed, productID)
	return nil
}

// --- Helper ---

func makeProduct(id string, priceJPY, shippingJPY int64) *domain.UnifiedProduct {
	return &domain.UnifiedProduct{
		ID:             id,
		Title:          "Test Product",
		PriceJPY:       priceJPY,
		ShippingFeeJPY: shippingJPY,
		Status:         domain.StatusAvailable,
		SourcePlatform: "surugaya",
		SourceURL:      "https://surugaya.com/001",
		Condition:      "used",
		Images:         []string{"https://img.com/a.jpg"},
		Seller:         domain.SellerInfo{SellerID: "s1", SellerName: "Shop A"},
	}
}

// --- Settlement tests ---

func TestSettlement_Success(t *testing.T) {
	ctx := context.Background()

	fetcher := &mockProductFetcher{
		products: map[string]*domain.UnifiedProduct{
			"surugaya_001": makeProduct("surugaya_001", 10000, 500),
		},
	}
	walletSvc := &mockWalletService{wallet: &domain.Wallet{Balance: 100000}}
	svc := NewOrderService(fetcher, walletSvc, nil, nil)

	result, err := svc.Settlement(ctx, 10, []OrderItem{{ProductID: "surugaya_001", Quantity: 1}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// commission = 10000 * 0.10 = 1000, shipping = 500, total = 11500
	expectedCommission := int64(1000)
	expectedShipping := int64(500)
	expectedTotal := int64(11500)

	if result.CommissionFeeJp != expectedCommission {
		t.Errorf("CommissionFeeJp = %d, want %d", result.CommissionFeeJp, expectedCommission)
	}
	if result.TotalShippingFee != expectedShipping {
		t.Errorf("TotalShippingFee = %d, want %d", result.TotalShippingFee, expectedShipping)
	}
	if result.OrderTotalJp != expectedTotal {
		t.Errorf("OrderTotalJp = %d, want %d", result.OrderTotalJp, expectedTotal)
	}
	if result.WalletBalance != 100000 {
		t.Errorf("WalletBalance = %d, want 100000", result.WalletBalance)
	}
	if len(result.OrderDetailList) != 1 {
		t.Errorf("len(OrderDetailList) = %d, want 1", len(result.OrderDetailList))
	}
}

func TestSettlement_ProductNotFound(t *testing.T) {
	ctx := context.Background()

	fetcher := &mockProductFetcher{err: errors.New("not found")}
	walletSvc := &mockWalletService{wallet: &domain.Wallet{Balance: 100000}}
	svc := NewOrderService(fetcher, walletSvc, nil, nil)

	_, err := svc.Settlement(ctx, 10, []OrderItem{{ProductID: "bad_id", Quantity: 1}})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrProductUnavailable) {
		t.Errorf("error = %v, want ErrProductUnavailable", err)
	}
}

func TestSettlement_EmptyItems(t *testing.T) {
	ctx := context.Background()
	svc := NewOrderService(nil, nil, nil, nil)

	_, err := svc.Settlement(ctx, 10, []OrderItem{})
	if !errors.Is(err, ErrEmptyItems) {
		t.Errorf("error = %v, want ErrEmptyItems", err)
	}
}

func TestSettlement_MultipleItems_CommissionCalc(t *testing.T) {
	ctx := context.Background()

	fetcher := &mockProductFetcher{
		products: map[string]*domain.UnifiedProduct{
			"item_1": makeProduct("item_1", 5000, 300),
			"item_2": makeProduct("item_2", 8000, 400),
		},
	}
	walletSvc := &mockWalletService{wallet: &domain.Wallet{Balance: 50000}}
	svc := NewOrderService(fetcher, walletSvc, nil, nil)

	items := []OrderItem{
		{ProductID: "item_1", Quantity: 2},
		{ProductID: "item_2", Quantity: 1},
	}
	result, err := svc.Settlement(ctx, 10, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// item_1: price=5000*2=10000, commission=500*2=1000, shipping=300*2=600
	// item_2: price=8000*1=8000, commission=800*1=800, shipping=400*1=400
	// total goods = 18000, commission = 1800, shipping = 1000, grand = 20800
	if result.OrderTotalJp != 20800 {
		t.Errorf("OrderTotalJp = %d, want 20800", result.OrderTotalJp)
	}
	if result.CommissionFeeJp != 1800 {
		t.Errorf("CommissionFeeJp = %d, want 1800", result.CommissionFeeJp)
	}
}

// --- Confirm tests ---

func TestConfirm_Success(t *testing.T) {
	ctx := context.Background()

	fetcher := &mockProductFetcher{
		products: map[string]*domain.UnifiedProduct{
			"surugaya_001": makeProduct("surugaya_001", 10000, 500),
		},
	}
	orderRepo := &mockOrderRepository{}
	cartRemover := &mockCartRemover{}
	svc := NewOrderService(fetcher, nil, orderRepo, cartRemover)

	result, err := svc.Confirm(ctx, 10, []OrderItem{{ProductID: "surugaya_001", Quantity: 1}}, 0, "test remark", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OrderState != domain.OrderStatePending {
		t.Errorf("OrderState = %d, want %d (Pending)", result.OrderState, domain.OrderStatePending)
	}
	if result.OrderTotalJp != 11500 {
		t.Errorf("OrderTotalJp = %d, want 11500", result.OrderTotalJp)
	}
	if result.OrderNumber == "" {
		t.Error("OrderNumber should not be empty")
	}
	// Verify cart item was removed
	if len(cartRemover.removed) != 1 || cartRemover.removed[0] != "surugaya_001" {
		t.Errorf("cart removed = %v, want [surugaya_001]", cartRemover.removed)
	}
	// Verify order was persisted
	if orderRepo.createdOrder == nil {
		t.Fatal("order should have been created in repo")
	}
	if orderRepo.createdOrder.UserID != 10 {
		t.Errorf("created order UserID = %d, want 10", orderRepo.createdOrder.UserID)
	}
}

func TestConfirm_ProductNotFound(t *testing.T) {
	ctx := context.Background()

	fetcher := &mockProductFetcher{err: errors.New("not found")}
	svc := NewOrderService(fetcher, nil, &mockOrderRepository{}, nil)

	_, err := svc.Confirm(ctx, 10, []OrderItem{{ProductID: "bad_id", Quantity: 1}}, 0, "", 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrProductUnavailable) {
		t.Errorf("error = %v, want ErrProductUnavailable", err)
	}
}

func TestConfirm_EmptyItems(t *testing.T) {
	ctx := context.Background()
	svc := NewOrderService(nil, nil, nil, nil)

	_, err := svc.Confirm(ctx, 10, []OrderItem{}, 0, "", 0)
	if !errors.Is(err, ErrEmptyItems) {
		t.Errorf("error = %v, want ErrEmptyItems", err)
	}
}

// --- Pay tests ---

func TestPay_Success(t *testing.T) {
	ctx := context.Background()

	walletSvc := &mockWalletService{
		wallet:   &domain.Wallet{Balance: 100000},
		adjustTx: &domain.WalletTransaction{ID: 1, Amount: -11500, BalanceAfter: 88500},
	}
	orderRepo := &mockOrderRepository{
		order: &domain.Order{
			ID:             1,
			OrderNumber:    "RO20260304TEST",
			UserID:         10,
			OrderState:     domain.OrderStatePending,
			OrderInpriceJp: 11500,
			CreatedAt:      time.Now(),
		},
	}
	svc := NewOrderService(nil, walletSvc, orderRepo, nil)

	result, err := svc.Pay(ctx, 10, "RO20260304TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.OrderState != domain.OrderStatePaid {
		t.Errorf("OrderState = %d, want %d (Paid)", result.OrderState, domain.OrderStatePaid)
	}
	if result.PaidAmount != 11500 {
		t.Errorf("PaidAmount = %d, want 11500", result.PaidAmount)
	}
	if result.BalanceAfter != 88500 {
		t.Errorf("BalanceAfter = %d, want 88500", result.BalanceAfter)
	}
	if result.OrderNumber != "RO20260304TEST" {
		t.Errorf("OrderNumber = %s, want RO20260304TEST", result.OrderNumber)
	}
}

func TestPay_InsufficientBalance(t *testing.T) {
	ctx := context.Background()

	walletSvc := &mockWalletService{wallet: &domain.Wallet{Balance: 5000}}
	orderRepo := &mockOrderRepository{
		order: &domain.Order{
			ID:             1,
			OrderNumber:    "RO20260304TEST",
			UserID:         10,
			OrderState:     domain.OrderStatePending,
			OrderInpriceJp: 11500,
			CreatedAt:      time.Now(),
		},
	}
	svc := NewOrderService(nil, walletSvc, orderRepo, nil)

	_, err := svc.Pay(ctx, 10, "RO20260304TEST")
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("error = %v, want ErrInsufficientBalance", err)
	}
}

func TestPay_OrderNotFound(t *testing.T) {
	ctx := context.Background()

	orderRepo := &mockOrderRepository{getErr: errors.New("not found")}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	_, err := svc.Pay(ctx, 10, "NO_SUCH_ORDER")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Errorf("error = %v, want ErrOrderNotFound", err)
	}
}

func TestPay_OrderNotPayable_AlreadyPaid(t *testing.T) {
	ctx := context.Background()

	orderRepo := &mockOrderRepository{
		order: &domain.Order{
			ID:          1,
			OrderNumber: "RO_PAID",
			UserID:      10,
			OrderState:  domain.OrderStatePaid,
			CreatedAt:   time.Now(),
		},
	}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	_, err := svc.Pay(ctx, 10, "RO_PAID")
	if !errors.Is(err, ErrOrderNotPayable) {
		t.Errorf("error = %v, want ErrOrderNotPayable", err)
	}
}

// --- GetDetail tests ---

func TestGetDetail_Success(t *testing.T) {
	ctx := context.Background()

	orderRepo := &mockOrderRepository{
		order: &domain.Order{
			ID:              1,
			OrderNumber:     "RO_TEST",
			UserID:          10,
			OrderState:      domain.OrderStatePaid,
			OrderTotalJp:    11500,
			CommissionFeeJp: 1000,
			ShippingFeeJp:   500,
			OrderInpriceJp:  11500,
			CreatedAt:       time.Now(),
		},
		details: []domain.OrderDetail{
			{
				ID:            1,
				OrderID:       1,
				GoodsMid:      "surugaya_001",
				GoodsName:     "Test Product",
				GoodsNum:      1,
				GoodsImg:      "https://img.com/a.jpg",
				GoodsAmountJp: 10000,
			},
		},
	}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	resp, err := svc.GetDetail(ctx, 10, "RO_TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.OrderNumber != "RO_TEST" {
		t.Errorf("OrderNumber = %s, want RO_TEST", resp.OrderNumber)
	}
	if resp.OrderState != domain.OrderStatePaid {
		t.Errorf("OrderState = %d, want %d (Paid)", resp.OrderState, domain.OrderStatePaid)
	}
	if len(resp.OrderDetailList) != 1 {
		t.Errorf("len(OrderDetailList) = %d, want 1", len(resp.OrderDetailList))
	}
	if resp.FirstGoodsImg != "https://img.com/a.jpg" {
		t.Errorf("FirstGoodsImg = %s, want https://img.com/a.jpg", resp.FirstGoodsImg)
	}
	if resp.OrderTotalJp != 11500 {
		t.Errorf("OrderTotalJp = %d, want 11500", resp.OrderTotalJp)
	}
}

func TestGetDetail_OrderNotFound(t *testing.T) {
	ctx := context.Background()

	orderRepo := &mockOrderRepository{getErr: errors.New("not found")}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	_, err := svc.GetDetail(ctx, 10, "NO_SUCH_ORDER")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Errorf("error = %v, want ErrOrderNotFound", err)
	}
}

func TestGetDetail_WrongUser(t *testing.T) {
	ctx := context.Background()

	// Order belongs to user 99, but we request with user 10
	orderRepo := &mockOrderRepository{
		order: &domain.Order{
			ID:          1,
			OrderNumber: "RO_TEST",
			UserID:      99,
			OrderState:  domain.OrderStatePaid,
			CreatedAt:   time.Now(),
		},
	}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	_, err := svc.GetDetail(ctx, 10, "RO_TEST")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Errorf("error = %v, want ErrOrderNotFound (wrong user)", err)
	}
}

// --- ListOrders tests ---

func TestListOrders_Success(t *testing.T) {
	ctx := context.Background()

	now := time.Now()
	orders := []domain.Order{
		{ID: 1, OrderNumber: "RO_001", OrderState: domain.OrderStatePaid, OrderTotalJp: 11500, UserID: 10, CreatedAt: now},
		{ID: 2, OrderNumber: "RO_002", OrderState: domain.OrderStatePending, OrderTotalJp: 5000, UserID: 10, CreatedAt: now},
	}
	orderRepo := &mockOrderRepository{
		orders: orders,
		total:  2,
		detailsMap: map[int64][]domain.OrderDetail{
			1: {{ID: 1, OrderID: 1, GoodsMid: "surugaya_001", GoodsImg: "https://img.com/a.jpg", GoodsNum: 1, GoodsAmountJp: 10000}},
			2: {},
		},
	}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	list, total, err := svc.ListOrders(ctx, 10, 0, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(list) != 2 {
		t.Errorf("len(list) = %d, want 2", len(list))
	}

	// First order should have detail and firstGoodsImg set
	if list[0].OrderNumber != "RO_001" {
		t.Errorf("list[0].OrderNumber = %s, want RO_001", list[0].OrderNumber)
	}
	if list[0].FirstGoodsImg != "https://img.com/a.jpg" {
		t.Errorf("list[0].FirstGoodsImg = %s, want https://img.com/a.jpg", list[0].FirstGoodsImg)
	}
	if len(list[0].OrderDetailList) != 1 {
		t.Errorf("len(list[0].OrderDetailList) = %d, want 1", len(list[0].OrderDetailList))
	}
}

func TestListOrders_EmptyList(t *testing.T) {
	ctx := context.Background()

	orderRepo := &mockOrderRepository{orders: nil, total: 0}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	list, total, err := svc.ListOrders(ctx, 10, 0, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(list) != 0 {
		t.Errorf("len(list) = %d, want 0", len(list))
	}
}

// --- Cancel tests ---

func TestCancel_Success(t *testing.T) {
	ctx := context.Background()

	orderRepo := &mockOrderRepository{
		order: &domain.Order{
			ID:          1,
			OrderNumber: "RO_TEST",
			UserID:      10,
			OrderState:  domain.OrderStatePending,
			CreatedAt:   time.Now(),
		},
	}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	err := svc.Cancel(ctx, 10, "RO_TEST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancel_OrderNotPayable_AlreadyPaid(t *testing.T) {
	ctx := context.Background()

	orderRepo := &mockOrderRepository{
		order: &domain.Order{
			ID:          1,
			OrderNumber: "RO_PAID",
			UserID:      10,
			OrderState:  domain.OrderStatePaid,
			CreatedAt:   time.Now(),
		},
	}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	err := svc.Cancel(ctx, 10, "RO_PAID")
	if !errors.Is(err, ErrOrderNotCancellable) {
		t.Errorf("error = %v, want ErrOrderNotCancellable", err)
	}
}

func TestCancel_OrderNotFound(t *testing.T) {
	ctx := context.Background()

	orderRepo := &mockOrderRepository{getErr: errors.New("not found")}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	err := svc.Cancel(ctx, 10, "NO_SUCH_ORDER")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Errorf("error = %v, want ErrOrderNotFound", err)
	}
}

func TestCancel_WrongUser(t *testing.T) {
	ctx := context.Background()

	orderRepo := &mockOrderRepository{
		order: &domain.Order{
			ID:          1,
			OrderNumber: "RO_TEST",
			UserID:      99,
			OrderState:  domain.OrderStatePending,
			CreatedAt:   time.Now(),
		},
	}
	svc := NewOrderService(nil, nil, orderRepo, nil)

	err := svc.Cancel(ctx, 10, "RO_TEST")
	if !errors.Is(err, ErrOrderNotFound) {
		t.Errorf("error = %v, want ErrOrderNotFound (wrong user)", err)
	}
}
