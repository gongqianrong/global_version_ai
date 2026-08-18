package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/rakutao/collection-gateway/internal/domain"
)

var (
	ErrSyncParamsEmpty          = errors.New("同步参数不能为空")
	ErrRequestIDEmpty           = errors.New("requestId不能为空")
	ErrGlobalOrderNumberEmpty   = errors.New("globalOrderNumber不能为空")
	ErrAccountInfoIDEmpty       = errors.New("accountInfoId不能为空")
	ErrGlobalOrderPayTypeEmpty  = errors.New("globalOrderPayType不能为空")
	ErrDetailListEmpty          = errors.New("订单明细不能为空")
	ErrPayEffectiveTimeEmpty    = errors.New("payEffectiveTime不能为空")
	ErrDetailNumberEmpty        = errors.New("globalOrderDetailNumber不能为空")
	ErrGoodsMidEmpty            = errors.New("goodsMid不能为空")
	ErrPlatformEmpty            = errors.New("platform不能为空")
	ErrPaymentNumberEmpty       = errors.New("paymentNumber不能为空")
	ErrPayChannelEmpty          = errors.New("payChannel不能为空")
	ErrPayCurrencyEmpty         = errors.New("payCurrency不能为空")
	ErrPayAmountInvalid         = errors.New("payAmount必须大于0")
	ErrPayTimeEmpty             = errors.New("payTime不能为空")
	ErrOrderNotSynced           = errors.New("国际版订单尚未同步，请先调用订单同步接口")
	ErrNotGlobalOrder           = errors.New("订单不是国际版同步订单")
	ErrPayTypeMismatch          = errors.New("globalOrderPayType与订单不一致")
	ErrAlreadyPaid              = errors.New("订单已支付成功")
	ErrPaymentNumberMismatch    = errors.New("支付流水号与已有记录不一致")
	ErrPaymentSyncException     = errors.New("支付同步状态异常，需要人工处理")
	ErrOrderNotBEPAY            = errors.New("订单状态不是待支付")
	ErrAmountMismatch           = errors.New("支付金额与订单金额不一致")
)

// GlobalOrderRepository defines the interface for global order persistence.
type GlobalOrderRepository interface {
	CreateGlobalOrder(ctx context.Context, req *domain.GlobalOrderSyncRequest) (*domain.Order, *domain.GlobalOrderRecord, []domain.OrderDetail, error)
	GetByRequestID(ctx context.Context, requestID string) (*domain.GlobalOrderRecord, error)
	GetByGlobalOrderNumber(ctx context.Context, globalOrderNumber string) (*domain.GlobalOrderRecord, error)
	GetPaymentByOrderID(ctx context.Context, orderID int64) (*domain.GlobalOrderPayment, error)
	GetPaymentByPaymentNumber(ctx context.Context, paymentNumber string) (*domain.GlobalOrderPayment, error)
	UpdatePaymentSuccess(ctx context.Context, globalOrderNumber string, paymentReq *domain.GlobalPaymentSyncRequest) error
	MarkPaymentException(ctx context.Context, globalOrderNumber string, reason string) error
}

// LocalOrderRepository defines the interface for local order queries.
type LocalOrderRepository interface {
	GetByOrderNumber(ctx context.Context, orderNumber string) (*domain.Order, []domain.OrderDetail, error)
}

// GlobalOrderService handles global order synchronization.
type GlobalOrderService struct {
	globalRepo GlobalOrderRepository
	localRepo  LocalOrderRepository
}

// NewGlobalOrderService creates a GlobalOrderService.
func NewGlobalOrderService(globalRepo GlobalOrderRepository, localRepo LocalOrderRepository) *GlobalOrderService {
	return &GlobalOrderService{
		globalRepo: globalRepo,
		localRepo:  localRepo,
	}
}

// ValidateSyncRequest validates the sync request parameters.
func ValidateSyncRequest(req *domain.GlobalOrderSyncRequest) error {
	if req == nil {
		return ErrSyncParamsEmpty
	}
	if req.RequestID == "" {
		return ErrRequestIDEmpty
	}
	if req.GlobalOrderNumber == "" {
		return ErrGlobalOrderNumberEmpty
	}
	if req.AccountInfoID == "" {
		return ErrAccountInfoIDEmpty
	}
	if req.GlobalOrderPayType == 0 {
		return ErrGlobalOrderPayTypeEmpty
	}
	if len(req.DetailList) == 0 {
		return ErrDetailListEmpty
	}
	if req.PayEffectiveTime.IsZero() {
		return ErrPayEffectiveTimeEmpty
	}

	// Validate each detail
	for i, detail := range req.DetailList {
		if detail.GlobalOrderDetailNumber == "" {
			return fmt.Errorf("明细[%d]: %w", i, ErrDetailNumberEmpty)
		}
		if detail.GoodsMid == "" {
			return fmt.Errorf("明细[%d]: %w", i, ErrGoodsMidEmpty)
		}
		if detail.Platform == 0 {
			return fmt.Errorf("明细[%d]: %w", i, ErrPlatformEmpty)
		}
	}

	return nil
}

// ValidatePaymentRequest validates the payment sync request parameters.
func ValidatePaymentRequest(req *domain.GlobalPaymentSyncRequest) error {
	if req == nil {
		return ErrSyncParamsEmpty
	}
	if req.RequestID == "" {
		return ErrRequestIDEmpty
	}
	if req.GlobalOrderNumber == "" {
		return ErrGlobalOrderNumberEmpty
	}
	if req.PaymentNumber == "" {
		return ErrPaymentNumberEmpty
	}
	if req.PayChannel == "" {
		return ErrPayChannelEmpty
	}
	if req.GlobalOrderPayType == 0 {
		return ErrGlobalOrderPayTypeEmpty
	}
	if req.PayCurrency == "" {
		return ErrPayCurrencyEmpty
	}
	if req.PayAmount == nil || req.PayAmount.Cmp(big.NewFloat(0)) <= 0 {
		return ErrPayAmountInvalid
	}
	if req.PayTime.IsZero() {
		return ErrPayTimeEmpty
	}

	return nil
}

// SyncGlobalOrder syncs an international order to the local system.
func (s *GlobalOrderService) SyncGlobalOrder(ctx context.Context, req *domain.GlobalOrderSyncRequest) (*domain.GlobalOrderSyncResponse, error) {
	// Validate request
	if err := ValidateSyncRequest(req); err != nil {
		return &domain.GlobalOrderSyncResponse{
			Success:           false,
			Idempotent:        false,
			Message:           err.Error(),
			GlobalOrderNumber: req.GlobalOrderNumber,
		}, nil
	}

	// Check idempotency by requestId
	existingByRequestID, err := s.globalRepo.GetByRequestID(ctx, req.RequestID)
	if err != nil {
		log.Printf("Error checking requestId: %v", err)
		return nil, fmt.Errorf("检查requestId失败: %w", err)
	}
	if existingByRequestID != nil {
		// Return existing order (idempotent)
		order, _, err := s.localRepo.GetByOrderNumber(ctx, existingByRequestID.OrderNumber)
		if err != nil {
			return nil, fmt.Errorf("获取已存在订单失败: %w", err)
		}

		orderIDStr := fmt.Sprintf("%d", order.ID)
		return &domain.GlobalOrderSyncResponse{
			Success:           true,
			Idempotent:        true,
			Message:           "国际版订单已同步（幂等返回）",
			OrderInfoID:       &orderIDStr,
			OrderNumber:       &order.OrderNumber,
			GlobalOrderNumber: req.GlobalOrderNumber,
			OrderState:        &order.OrderState,
		}, nil
	}

	// Check idempotency by globalOrderNumber
	existingByGlobalNumber, err := s.globalRepo.GetByGlobalOrderNumber(ctx, req.GlobalOrderNumber)
	if err != nil {
		log.Printf("Error checking globalOrderNumber: %v", err)
		return nil, fmt.Errorf("检查globalOrderNumber失败: %w", err)
	}
	if existingByGlobalNumber != nil {
		// Return existing order (idempotent)
		order, _, err := s.localRepo.GetByOrderNumber(ctx, existingByGlobalNumber.OrderNumber)
		if err != nil {
			return nil, fmt.Errorf("获取已存在订单失败: %w", err)
		}

		orderIDStr := fmt.Sprintf("%d", order.ID)
		return &domain.GlobalOrderSyncResponse{
			Success:           true,
			Idempotent:        true,
			Message:           "国际版订单已同步（幂等返回）",
			OrderInfoID:       &orderIDStr,
			OrderNumber:       &order.OrderNumber,
			GlobalOrderNumber: req.GlobalOrderNumber,
			OrderState:        &order.OrderState,
		}, nil
	}

	// Create new global order
	order, globalRecord, _, err := s.globalRepo.CreateGlobalOrder(ctx, req)
	if err != nil {
		log.Printf("Error creating global order: %v", err)
		return nil, fmt.Errorf("创建国际版订单失败: %w", err)
	}

	orderIDStr := fmt.Sprintf("%d", order.ID)
	return &domain.GlobalOrderSyncResponse{
		Success:           true,
		Idempotent:        false,
		Message:           "国际版订单同步成功",
		OrderInfoID:       &orderIDStr,
		OrderNumber:       &order.OrderNumber,
		GlobalOrderNumber: globalRecord.GlobalOrderNumber,
		OrderState:        &order.OrderState,
	}, nil
}

// SyncGlobalPayment syncs payment success for an international order.
func (s *GlobalOrderService) SyncGlobalPayment(ctx context.Context, req *domain.GlobalPaymentSyncRequest) (*domain.GlobalPaymentSyncResponse, error) {
	// Validate request
	if err := ValidatePaymentRequest(req); err != nil {
		return &domain.GlobalPaymentSyncResponse{
			Success:           false,
			Idempotent:        false,
			Message:           err.Error(),
			GlobalOrderNumber: req.GlobalOrderNumber,
			PaymentNumber:     req.PaymentNumber,
		}, nil
	}

	// Get global order record
	globalRecord, err := s.globalRepo.GetByGlobalOrderNumber(ctx, req.GlobalOrderNumber)
	if err != nil {
		log.Printf("Error getting global order: %v", err)
		return nil, fmt.Errorf("获取国际版订单失败: %w", err)
	}
	if globalRecord == nil {
		return &domain.GlobalPaymentSyncResponse{
			Success:           false,
			Idempotent:        false,
			Message:           ErrOrderNotSynced.Error(),
			GlobalOrderNumber: req.GlobalOrderNumber,
			PaymentNumber:     req.PaymentNumber,
		}, nil
	}

	// Check globalOrderPayType matches
	if globalRecord.GlobalOrderPayType != req.GlobalOrderPayType {
		s.globalRepo.MarkPaymentException(ctx, req.GlobalOrderNumber, "PayType mismatch")
		return &domain.GlobalPaymentSyncResponse{
			Success:           false,
			Idempotent:        false,
			Message:           ErrPayTypeMismatch.Error(),
			GlobalOrderNumber: req.GlobalOrderNumber,
			PaymentNumber:     req.PaymentNumber,
		}, nil
	}

	// Check if already in exception state
	if globalRecord.PaymentSyncState == domain.PaymentSyncStateException {
		return &domain.GlobalPaymentSyncResponse{
			Success:           false,
			Idempotent:        false,
			Message:           ErrPaymentSyncException.Error(),
			GlobalOrderNumber: req.GlobalOrderNumber,
			PaymentNumber:     req.PaymentNumber,
		}, nil
	}

	// Check if already paid
	if globalRecord.PaymentSyncState == domain.PaymentSyncStatePaid {
		// Check if same payment number (idempotent)
		if globalRecord.PaymentNumber != nil && *globalRecord.PaymentNumber == req.PaymentNumber {
			order, _, err := s.localRepo.GetByOrderNumber(ctx, globalRecord.OrderNumber)
			if err != nil {
				return nil, fmt.Errorf("获取订单失败: %w", err)
			}

			orderIDStr := fmt.Sprintf("%d", order.ID)
			return &domain.GlobalPaymentSyncResponse{
				Success:           true,
				Idempotent:        true,
				Message:           "国际版支付状态已同步（幂等返回）",
				OrderInfoID:       &orderIDStr,
				OrderNumber:       &order.OrderNumber,
				GlobalOrderNumber: req.GlobalOrderNumber,
				PaymentNumber:     req.PaymentNumber,
				OrderState:        &order.OrderState,
			}, nil
		} else {
			// Different payment number - not allowed
			return &domain.GlobalPaymentSyncResponse{
				Success:           false,
				Idempotent:        false,
				Message:           ErrPaymentNumberMismatch.Error(),
				GlobalOrderNumber: req.GlobalOrderNumber,
				PaymentNumber:     req.PaymentNumber,
			}, nil
		}
	}

	// Check payment amount for JPY currency
	if req.PayCurrency == "JPY" {
		order, _, err := s.localRepo.GetByOrderNumber(ctx, globalRecord.OrderNumber)
		if err != nil {
			return nil, fmt.Errorf("获取订单失败: %w", err)
		}

		// Convert payment amount to cents
		payAmountCents := new(big.Float).Mul(req.PayAmount, big.NewFloat(100))
		payAmountInt64, _ := payAmountCents.Int64()

		if payAmountInt64 != order.OrderInpriceJp {
			s.globalRepo.MarkPaymentException(ctx, req.GlobalOrderNumber, "Amount mismatch")
			return &domain.GlobalPaymentSyncResponse{
				Success:           false,
				Idempotent:        false,
				Message:           fmt.Sprintf("%s: 期望 %d, 实际 %d", ErrAmountMismatch.Error(), order.OrderInpriceJp, payAmountInt64),
				GlobalOrderNumber: req.GlobalOrderNumber,
				PaymentNumber:     req.PaymentNumber,
			}, nil
		}
	}

	// Update payment success
	err = s.globalRepo.UpdatePaymentSuccess(ctx, req.GlobalOrderNumber, req)
	if err != nil {
		if err.Error() == "global_order_repo: order not in BEPAY state" {
			return &domain.GlobalPaymentSyncResponse{
				Success:           false,
				Idempotent:        false,
				Message:           ErrOrderNotBEPAY.Error(),
				GlobalOrderNumber: req.GlobalOrderNumber,
				PaymentNumber:     req.PaymentNumber,
			}, nil
		}
		log.Printf("Error updating payment success: %v", err)
		return nil, fmt.Errorf("更新支付状态失败: %w", err)
	}

	// Get updated order
	order, _, err := s.localRepo.GetByOrderNumber(ctx, globalRecord.OrderNumber)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}

	orderIDStr := fmt.Sprintf("%d", order.ID)
	return &domain.GlobalPaymentSyncResponse{
		Success:           true,
		Idempotent:        false,
		Message:           "国际版支付状态同步成功",
		OrderInfoID:       &orderIDStr,
		OrderNumber:       &order.OrderNumber,
		GlobalOrderNumber: req.GlobalOrderNumber,
		PaymentNumber:     req.PaymentNumber,
		OrderState:        &order.OrderState,
	}, nil
}
