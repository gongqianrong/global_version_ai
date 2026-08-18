package domain

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Order state constants matching the domestic backend.
const (
	OrderStatePending    = 0
	OrderStatePaid       = 1
	OrderStatePurchasing = 2
	OrderStateWarehoused = 3
	OrderStatePacking    = 4
	OrderStatePacked     = 5
	OrderStateShipped    = 6
	OrderStateFulfilled  = 7
	OrderStateCancelled  = 8
	OrderStateRefunded   = 9
)

// Order purchase type constants.
const (
	OrderPurchaseTypeNormal    = 1
	OrderPurchaseTypeOrderLink = 2
)

// Order represents a row in the orders table.
type Order struct {
	ID                int64     `json:"id"`
	OrderNumber       string    `json:"order_number"`
	UserID            int64     `json:"user_id"`
	OrderState        int       `json:"order_state"`
	OrderTotalJp      int64     `json:"order_total_jp"`
	CommissionFeeJp   int64     `json:"commission_fee_jp"`
	ShippingFeeJp     int64     `json:"shipping_fee_jp"`
	OrderInpriceJp    int64     `json:"order_inprice_jp"`
	OrderPaytype      int       `json:"order_paytype"`
	OrderRemark       string    `json:"order_remark"`
	OrderPurchaseType int       `json:"order_purchase_type"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// OrderDetail represents a row in the order_details table.
type OrderDetail struct {
	ID              int64     `json:"id"`
	OrderID         int64     `json:"order_id"`
	GoodsMid        string    `json:"goods_mid"`
	GoodsName       string    `json:"goods_name"`
	GoodsNum        int       `json:"goods_num"`
	GoodsImg        string    `json:"goods_img"`
	GoodsUrl        string    `json:"goods_url"`
	GoodsAmountJp   int64     `json:"goods_amount_jp"`
	CommissionFeeJp int64     `json:"commission_fee_jp"`
	ShippingFeeJp   int64     `json:"shipping_fee_jp"`
	SellerID        string    `json:"seller_id"`
	SellerName      string    `json:"seller_name"`
	Platform        string    `json:"platform"`
	Condition       string    `json:"condition"`
	State           int       `json:"state"`
	CreatedAt       time.Time `json:"created_at"`
}

// GenerateOrderNumber creates a unique order number: RO + timestamp (microsecond) + 8 random hex chars.
// Format: RO20260102150405123456ABCD1234 (prefix + YYYYMMDDHHmmssμs + random)
// Microsecond precision + 4 random bytes = extremely low collision risk.
func GenerateOrderNumber() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback: use nanosecond time as additional entropy
		nanos := time.Now().UnixNano()
		b[0] = byte(nanos >> 24)
		b[1] = byte(nanos >> 16)
		b[2] = byte(nanos >> 8)
		b[3] = byte(nanos)
	}
	now := time.Now()
	// Use microsecond precision to reduce same-second collisions
	return fmt.Sprintf("RO%s%06d%X", 
		now.Format("20060102150405"), 
		now.Nanosecond()/1000, // microseconds
		b)
}
