package domain

import (
	"time"
)

// GlobalOrderSyncRequest represents the request to sync an international order.
type GlobalOrderSyncRequest struct {
	RequestID           string                      `json:"requestId"`
	GlobalOrderNumber   string                      `json:"globalOrderNumber"`
	GlobalAccountID     *string                     `json:"globalAccountId"`
	AccountInfoID       string                      `json:"accountInfoId"`
	AccountAddressID    *string                     `json:"accountAddressId"`
	OrderAddtime        *time.Time                  `json:"orderAddtime"`
	PayEffectiveTime    time.Time                   `json:"payEffectiveTime"`
	OrderTotalJp        float64                     `json:"orderTotalJp"`
	OrderTotalCn        float64                     `json:"orderTotalCn"`
	CommissionFeeJp     float64                     `json:"commissionFeeJp"`
	CommissionFeeCn     float64                     `json:"commissionFeeCn"`
	HandlingFeeJp       float64                     `json:"handlingFeeJp"`
	HandlingFeeCn       float64                     `json:"handlingFeeCn"`
	OrderInpriceJp      float64                     `json:"orderInpriceJp"`
	OrderInpriceCn      float64                     `json:"orderInpriceCn"`
	OrderRate           float64                     `json:"orderRate"`
	TotalShippingFee    float64                     `json:"totalShippingFee"`
	TotalShippingFeeCn  float64                     `json:"totalShippingFeeCn"`
	OrderType           *int                        `json:"orderType"`
	OrderPurchaseType   *int                        `json:"orderPurchaseType"`
	OrderMode           *int                        `json:"orderMode"`
	OrderRemark         *string                     `json:"orderRemark"`
	Operator            *string                     `json:"operator"`
	GlobalOrderPayType  int                         `json:"globalOrderPayType"`
	DetailList          []GlobalOrderDetailRequest  `json:"detailList"`
}

// GlobalOrderDetailRequest represents a detail item in the sync request.
type GlobalOrderDetailRequest struct {
	GlobalOrderDetailNumber string   `json:"globalOrderDetailNumber"`
	Platform                int      `json:"platform"`
	GoodsMid                string   `json:"goodsMid"`
	GoodsImg                *string  `json:"goodsImg"`
	GoodsName               *string  `json:"goodsName"`
	GoodsNum                *int     `json:"goodsNum"`
	GoodsAmountJp           float64  `json:"goodsAmountJp"`
	GoodsAmountCn           float64  `json:"goodsAmountCn"`
	CommissionFeeJp         float64  `json:"commissionFeeJp"`
	CommissionFeeCn         float64  `json:"commissionFeeCn"`
	HandlingFeeJp           float64  `json:"handlingFeeJp"`
	HandlingFeeCn           float64  `json:"handlingFeeCn"`
	GoodsUrl                *string  `json:"goodsUrl"`
	SellerID                *string  `json:"sellerId"`
	ShippingFeeJp           float64  `json:"shippingFeeJp"`
	ShippingFeeCn           float64  `json:"shippingFeeCn"`
	OrderPurchaseType       *int     `json:"orderPurchaseType"`
	PurchaseDirect          *int     `json:"purchaseDirect"`
	DiscountType            *int     `json:"discountType"`
}

// GlobalOrderSyncResponse represents the response of order sync.
type GlobalOrderSyncResponse struct {
	Success           bool    `json:"success"`
	Idempotent        bool    `json:"idempotent"`
	Message           string  `json:"message"`
	OrderInfoID       *string `json:"orderInfoId"`
	OrderNumber       *string `json:"orderNumber"`
	GlobalOrderNumber string  `json:"globalOrderNumber"`
	OrderState        *int    `json:"orderState"`
}

// GlobalPaymentSyncRequest represents the request to sync payment success.
type GlobalPaymentSyncRequest struct {
	RequestID          string      `json:"requestId"`
	GlobalOrderNumber  string      `json:"globalOrderNumber"`
	PaymentNumber      string      `json:"paymentNumber"`
	PayChannel         string      `json:"payChannel"`
	PayWay             *int        `json:"payWay"`
	PayAmountJp        float64     `json:"payAmountJp"`
	PayAmountCn        float64     `json:"payAmountCn"`
	PaySeccussTime     time.Time   `json:"paySeccussTime"`
	Operator           *string     `json:"operator"`
}

// GlobalPaymentSyncResponse represents the response of payment sync.
type GlobalPaymentSyncResponse struct {
	Success           bool    `json:"success"`
	Idempotent        bool    `json:"idempotent"`
	Message           string  `json:"message"`
	OrderInfoID       *string `json:"orderInfoId"`
	OrderNumber       *string `json:"orderNumber"`
	GlobalOrderNumber string  `json:"globalOrderNumber"`
	PaymentNumber     string  `json:"paymentNumber"`
	OrderState        *int    `json:"orderState"`
}

// GlobalOrderRecord stores the mapping between global order and local order.
type GlobalOrderRecord struct {
	ID                  int64     `json:"id"`
	RequestID           string    `json:"request_id"`
	GlobalOrderNumber   string    `json:"global_order_number"`
	GlobalAccountID     *string   `json:"global_account_id"`
	OrderID             int64     `json:"order_id"`
	OrderNumber         string    `json:"order_number"`
	GlobalOrderPayType  int       `json:"global_order_pay_type"`
	SyncTime            time.Time `json:"sync_time"`
	PaymentSyncState    int       `json:"payment_sync_state"` // 0=not paid, 1=paid, 2=exception
	PaymentNumber       *string   `json:"payment_number"`
	PaymentRequestID    *string   `json:"payment_request_id"`
	PaymentSyncTime     *time.Time `json:"payment_sync_time"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// GlobalOrderPayment stores additional payment information for global orders.
type GlobalOrderPayment struct {
	ID                int64      `json:"id"`
	OrderID           int64      `json:"order_id"`
	PaymentNumber     string     `json:"payment_number"`
	PayChannel        string     `json:"pay_channel"`
	PayCurrency       string     `json:"pay_currency"`
	PayAmount         int64      `json:"pay_amount"` // stored in cents/smallest unit
	PayTime           time.Time  `json:"pay_time"`
	Operator          *string    `json:"operator"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// Order state for payment sync
const (
	OrderStateBEPAY    = 0 // Pending payment
	OrderStatePAID     = 1 // Paid
	OrderStateACCEPTED = 2 // Accepted (for purchasing)
)

// Payment sync states
const (
	PaymentSyncStateNotPaid   = 0
	PaymentSyncStatePaid      = 1
	PaymentSyncStateException = 2
)
