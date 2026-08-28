package domain

import (
	"fmt"
	"strings"
	"time"
)

// CustomTime 自定义时间类型，支持 "yyyy-MM-dd HH:mm:ss" 格式
type CustomTime struct {
	time.Time
}

// UnmarshalJSON 自定义JSON解析，支持 "yyyy-MM-dd HH:mm:ss" 格式
func (ct *CustomTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		ct.Time = time.Time{}
		return nil
	}
	
	// 支持格式: yyyy-MM-dd HH:mm:ss
	t, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		return fmt.Errorf("日期格式错误，应为 yyyy-MM-dd HH:mm:ss: %w", err)
	}
	ct.Time = t
	return nil
}

// MarshalJSON 自定义JSON序列化
func (ct CustomTime) MarshalJSON() ([]byte, error) {
	if ct.Time.IsZero() {
		return []byte("null"), nil
	}
	formatted := fmt.Sprintf("\"%s\"", ct.Time.Format("2006-01-02 15:04:05"))
	return []byte(formatted), nil
}

// IsZero 判断时间是否为零值
func (ct CustomTime) IsZero() bool {
	return ct.Time.IsZero()
}

// GlobalOrderSyncRequest represents the request to sync an international order.
type GlobalOrderSyncRequest struct {
	RequestID           string                      `json:"requestId"`
	GlobalOrderNumber   string                      `json:"globalOrderNumber"`
	GlobalAccountID     string                      `json:"globalAccountId"` // 必填，国际版用户ID
	AccountAddressID    *string                     `json:"accountAddressId"`
	OrderAddtime        *CustomTime                 `json:"orderAddtime"`
	PayEffectiveTime    CustomTime                  `json:"payEffectiveTime"`
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
	GoodsName               string   `json:"goodsName"` // 必填，数据库不允许为空
	GoodsNum                *int     `json:"goodsNum"`
	GoodsAmountJp           float64  `json:"goodsAmountJp"`
	GoodsAmountCn           float64  `json:"goodsAmountCn"`
	CommissionFeeJp         float64  `json:"commissionFeeJp"`
	CommissionFeeCn         float64  `json:"commissionFeeCn"`
	HandlingFeeJp           float64  `json:"handlingFeeJp"`
	HandlingFeeCn           float64  `json:"handlingFeeCn"`
	GoodsUrl                string   `json:"goodsUrl"` // 必填，数据库不允许为空
	SellerID                *string  `json:"sellerId"`
	ShippingFeeJp           float64  `json:"shippingFeeJp"`
	ShippingFeeCn           float64  `json:"shippingFeeCn"`
	OrderPurchaseType       *int     `json:"orderPurchaseType"`
	PurchaseDirect          *int     `json:"purchaseDirect"`
	DiscountType            int      `json:"discountType"` // 必填，无折扣传0
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
	GlobalOrderPayType int         `json:"globalOrderPayType"` // 必填，必须与订单同步时一致
	PayCurrency        string      `json:"payCurrency"`        // 必填，实际支付币种，例如JPY、USD
	PayAmount          float64     `json:"payAmount"`          // 必填，实际支付金额，必须大于0
	PayTime            CustomTime  `json:"payTime"`            // 必填，支付成功时间
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
	ID                  int64      `json:"id"`
	RequestID           string     `json:"request_id"`
	GlobalOrderNumber   string     `json:"global_order_number"`
	GlobalAccountID     string     `json:"global_account_id"` // 国际版用户ID
	OrderID             int64      `json:"order_id"`
	OrderNumber         string     `json:"order_number"`
	GlobalOrderPayType  int        `json:"global_order_pay_type"`
	SyncTime            time.Time  `json:"sync_time"`
	PaymentSyncState    int        `json:"payment_sync_state"` // 0=not paid, 1=paid, 2=exception
	PaymentNumber       *string    `json:"payment_number"`
	PaymentRequestID    *string    `json:"payment_request_id"`
	PaymentSyncTime     *time.Time `json:"payment_sync_time"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
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
