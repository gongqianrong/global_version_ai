package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// GlobalOrderRepo handles global order sync persistence.
type GlobalOrderRepo struct {
	db *db.DB
}

// NewGlobalOrderRepo creates a GlobalOrderRepo.
func NewGlobalOrderRepo(d *db.DB) *GlobalOrderRepo {
	return &GlobalOrderRepo{db: d}
}

// Helper function to convert float64 to int64 cents (JPY)
func float64ToInt64Cents(f float64) int64 {
	// Multiply by 100 to convert to cents, then convert to int64
	cents := f * 100
	return int64(cents)
}

// Helper function to get int value with default
func intOrDefault(val *int, defaultVal int) int {
	if val == nil {
		return defaultVal
	}
	return *val
}

// Helper function to get string value with default
func stringOrDefault(val *string, defaultVal string) string {
	if val == nil {
		return defaultVal
	}
	return *val
}

// CreateGlobalOrder creates both the order and global order record atomically.
// Returns the created order, global record, and details.
func (r *GlobalOrderRepo) CreateGlobalOrder(
	ctx context.Context,
	req *domain.GlobalOrderSyncRequest,
) (*domain.Order, *domain.GlobalOrderRecord, []domain.OrderDetail, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("global_order_repo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Map globalAccountId to local accountInfoId (user_id)
	// TODO: This should call mall-userms service to map/create user
	// For now, use a simple hash mapping or direct conversion
	var userID int64
	// Try to find existing mapping or create new one
	// Simplified: hash globalAccountId to generate userID or query from mapping table
	// For MVP, we'll use a simple approach
	userID = hashGlobalAccountID(req.GlobalAccountID)

	// Generate local order number
	orderNumber := domain.GenerateOrderNumber()

	// Parse order add time
	var orderAddtime time.Time
	if req.OrderAddtime != nil && !req.OrderAddtime.IsZero() {
		orderAddtime = req.OrderAddtime.Time
	} else {
		orderAddtime = time.Now()
	}

	// Create main order
	order := &domain.Order{
		OrderNumber:       orderNumber,
		UserID:            userID,
		OrderState:        domain.OrderStateBEPAY, // BEPAY = waiting for payment
		OrderTotalJp:      float64ToInt64Cents(req.OrderTotalJp),
		CommissionFeeJp:   float64ToInt64Cents(req.CommissionFeeJp),
		ShippingFeeJp:     float64ToInt64Cents(req.TotalShippingFee),
		OrderInpriceJp:    float64ToInt64Cents(req.OrderInpriceJp),
		OrderPaytype:      req.GlobalOrderPayType,
		OrderRemark:       stringOrDefault(req.OrderRemark, ""),
		OrderPurchaseType: intOrDefault(req.OrderPurchaseType, domain.OrderPurchaseTypeNormal),
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO orders (order_number, user_id, order_state, order_total_jp, commission_fee_jp,
		  shipping_fee_jp, order_inprice_jp, order_paytype, order_remark, order_purchase_type, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		 RETURNING id, updated_at`,
		order.OrderNumber, order.UserID, order.OrderState, order.OrderTotalJp,
		order.CommissionFeeJp, order.ShippingFeeJp, order.OrderInpriceJp,
		order.OrderPaytype, order.OrderRemark, order.OrderPurchaseType, orderAddtime,
	).Scan(&order.ID, &order.UpdatedAt)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("global_order_repo: insert order: %w", err)
	}
	order.CreatedAt = orderAddtime

	// Create order details
	var details []domain.OrderDetail
	for _, detailReq := range req.DetailList {
		detail := domain.OrderDetail{
			OrderID:         order.ID,
			GoodsMid:        detailReq.GoodsMid,
			GoodsName:       detailReq.GoodsName,
			GoodsNum:        intOrDefault(detailReq.GoodsNum, 1),
			GoodsImg:        stringOrDefault(detailReq.GoodsImg, ""),
			GoodsUrl:        detailReq.GoodsUrl,
			GoodsAmountJp:   float64ToInt64Cents(detailReq.GoodsAmountJp),
			CommissionFeeJp: float64ToInt64Cents(detailReq.CommissionFeeJp),
			ShippingFeeJp:   float64ToInt64Cents(detailReq.ShippingFeeJp),
			SellerID:        stringOrDefault(detailReq.SellerID, ""),
			SellerName:      "", // Not provided in request
			Platform:        fmt.Sprintf("%d", detailReq.Platform),
			Condition:       "", // Not provided in request
			State:           0,  // Default state
		}

		err = tx.QueryRow(ctx,
			`INSERT INTO order_details (order_id, goods_mid, goods_name, goods_num, goods_img,
			  goods_url, goods_amount_jp, commission_fee_jp, shipping_fee_jp,
			  seller_id, seller_name, platform, condition, state)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
			 RETURNING id, created_at`,
			detail.OrderID, detail.GoodsMid, detail.GoodsName,
			detail.GoodsNum, detail.GoodsImg, detail.GoodsUrl,
			detail.GoodsAmountJp, detail.CommissionFeeJp, detail.ShippingFeeJp,
			detail.SellerID, detail.SellerName, detail.Platform,
			detail.Condition, detail.State,
		).Scan(&detail.ID, &detail.CreatedAt)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("global_order_repo: insert detail: %w", err)
		}
		details = append(details, detail)
	}

	// Create global order record
	globalRecord := &domain.GlobalOrderRecord{
		RequestID:          req.RequestID,
		GlobalOrderNumber:  req.GlobalOrderNumber,
		GlobalAccountID:    req.GlobalAccountID,
		OrderID:            order.ID,
		OrderNumber:        orderNumber,
		GlobalOrderPayType: req.GlobalOrderPayType,
		PaymentSyncState:   domain.PaymentSyncStateNotPaid,
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO global_order_records 
		 (request_id, global_order_number, global_account_id, order_id, order_number, 
		  global_order_pay_type, payment_sync_state, sync_time)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,NOW())
		 RETURNING id, created_at, updated_at`,
		globalRecord.RequestID, globalRecord.GlobalOrderNumber, globalRecord.GlobalAccountID,
		globalRecord.OrderID, globalRecord.OrderNumber, globalRecord.GlobalOrderPayType,
		globalRecord.PaymentSyncState,
	).Scan(&globalRecord.ID, &globalRecord.CreatedAt, &globalRecord.UpdatedAt)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("global_order_repo: insert global record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, nil, fmt.Errorf("global_order_repo: commit: %w", err)
	}

	return order, globalRecord, details, nil
}

// hashGlobalAccountID generates a local user ID from global account ID
// TODO: Replace with actual user mapping service call
func hashGlobalAccountID(globalAccountID string) int64 {
	// Simple hash for MVP - should be replaced with proper user service mapping
	var hash uint64
	for i := 0; i < len(globalAccountID); i++ {
		hash = hash*31 + uint64(globalAccountID[i])
	}
	// Ensure positive int64
	return int64(hash % 1000000000)
}

// GetByRequestID retrieves global order record by request ID.
func (r *GlobalOrderRepo) GetByRequestID(ctx context.Context, requestID string) (*domain.GlobalOrderRecord, error) {
	var record domain.GlobalOrderRecord
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, request_id, global_order_number, global_account_id, order_id, order_number,
		  global_order_pay_type, payment_sync_state, payment_number, payment_request_id,
		  payment_sync_time, sync_time, created_at, updated_at
		 FROM global_order_records WHERE request_id = $1`,
		requestID,
	).Scan(&record.ID, &record.RequestID, &record.GlobalOrderNumber, &record.GlobalAccountID,
		&record.OrderID, &record.OrderNumber, &record.GlobalOrderPayType, &record.PaymentSyncState,
		&record.PaymentNumber, &record.PaymentRequestID, &record.PaymentSyncTime, &record.SyncTime,
		&record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("global_order_repo: get by request_id: %w", err)
	}
	return &record, nil
}

// GetByGlobalOrderNumber retrieves global order record by global order number.
func (r *GlobalOrderRepo) GetByGlobalOrderNumber(ctx context.Context, globalOrderNumber string) (*domain.GlobalOrderRecord, error) {
	var record domain.GlobalOrderRecord
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, request_id, global_order_number, global_account_id, order_id, order_number,
		  global_order_pay_type, payment_sync_state, payment_number, payment_request_id,
		  payment_sync_time, sync_time, created_at, updated_at
		 FROM global_order_records WHERE global_order_number = $1`,
		globalOrderNumber,
	).Scan(&record.ID, &record.RequestID, &record.GlobalOrderNumber, &record.GlobalAccountID,
		&record.OrderID, &record.OrderNumber, &record.GlobalOrderPayType, &record.PaymentSyncState,
		&record.PaymentNumber, &record.PaymentRequestID, &record.PaymentSyncTime, &record.SyncTime,
		&record.CreatedAt, &record.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("global_order_repo: get by global_order_number: %w", err)
	}
	return &record, nil
}

// GetPaymentByOrderID retrieves payment record by order ID.
func (r *GlobalOrderRepo) GetPaymentByOrderID(ctx context.Context, orderID int64) (*domain.GlobalOrderPayment, error) {
	var payment domain.GlobalOrderPayment
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, order_id, payment_number, pay_channel, pay_currency, pay_amount,
		  pay_time, operator, created_at, updated_at
		 FROM global_order_payments WHERE order_id = $1`,
		orderID,
	).Scan(&payment.ID, &payment.OrderID, &payment.PaymentNumber, &payment.PayChannel,
		&payment.PayCurrency, &payment.PayAmount, &payment.PayTime, &payment.Operator,
		&payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("global_order_repo: get payment: %w", err)
	}
	return &payment, nil
}

// GetPaymentByPaymentNumber retrieves payment record by payment number.
func (r *GlobalOrderRepo) GetPaymentByPaymentNumber(ctx context.Context, paymentNumber string) (*domain.GlobalOrderPayment, error) {
	var payment domain.GlobalOrderPayment
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, order_id, payment_number, pay_channel, pay_currency, pay_amount,
		  pay_time, operator, created_at, updated_at
		 FROM global_order_payments WHERE payment_number = $1`,
		paymentNumber,
	).Scan(&payment.ID, &payment.OrderID, &payment.PaymentNumber, &payment.PayChannel,
		&payment.PayCurrency, &payment.PayAmount, &payment.PayTime, &payment.Operator,
		&payment.CreatedAt, &payment.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("global_order_repo: get payment by number: %w", err)
	}
	return &payment, nil
}

// UpdatePaymentSuccess updates order to PAID and records payment info atomically.
func (r *GlobalOrderRepo) UpdatePaymentSuccess(
	ctx context.Context,
	globalOrderNumber string,
	paymentReq *domain.GlobalPaymentSyncRequest,
) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("global_order_repo: begin payment tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get global order record
	var record domain.GlobalOrderRecord
	err = tx.QueryRow(ctx,
		`SELECT id, order_id, order_number, payment_sync_state
		 FROM global_order_records WHERE global_order_number = $1 FOR UPDATE`,
		globalOrderNumber,
	).Scan(&record.ID, &record.OrderID, &record.OrderNumber, &record.PaymentSyncState)
	if err != nil {
		return fmt.Errorf("global_order_repo: get record for payment: %w", err)
	}

	// Update order state from BEPAY to PAID
	tag, err := tx.Exec(ctx,
		`UPDATE orders SET order_state = $1, updated_at = NOW()
		 WHERE id = $2 AND order_state = $3`,
		domain.OrderStatePAID, record.OrderID, domain.OrderStateBEPAY,
	)
	if err != nil {
		return fmt.Errorf("global_order_repo: update order state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("global_order_repo: order not in BEPAY state")
	}

	// Convert payment amount to cents
	payAmount := float64ToInt64Cents(paymentReq.PayAmount)
	
	// Insert payment record
	_, err = tx.Exec(ctx,
		`INSERT INTO global_order_payments 
		 (order_id, payment_number, pay_channel, pay_currency, pay_amount, pay_time, operator)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		record.OrderID, paymentReq.PaymentNumber, paymentReq.PayChannel,
		paymentReq.PayCurrency, payAmount, paymentReq.PayTime.Time, paymentReq.Operator,
	)
	if err != nil {
		return fmt.Errorf("global_order_repo: insert payment: %w", err)
	}

	// Update global record
	_, err = tx.Exec(ctx,
		`UPDATE global_order_records 
		 SET payment_sync_state = $1, payment_number = $2, payment_request_id = $3, 
		     payment_sync_time = NOW(), updated_at = NOW()
		 WHERE id = $4`,
		domain.PaymentSyncStatePaid, paymentReq.PaymentNumber, paymentReq.RequestID, record.ID,
	)
	if err != nil {
		return fmt.Errorf("global_order_repo: update global record: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("global_order_repo: commit payment: %w", err)
	}

	return nil
}

// MarkPaymentException marks payment sync as exception.
func (r *GlobalOrderRepo) MarkPaymentException(ctx context.Context, globalOrderNumber string, reason string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE global_order_records 
		 SET payment_sync_state = $1, updated_at = NOW()
		 WHERE global_order_number = $2`,
		domain.PaymentSyncStateException, globalOrderNumber,
	)
	if err != nil {
		return fmt.Errorf("global_order_repo: mark exception: %w", err)
	}
	return nil
}
