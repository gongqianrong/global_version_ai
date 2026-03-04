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

// GenerateOrderNumber creates a unique order number: RO + timestamp + 4 random hex chars.
func GenerateOrderNumber() string {
	b := make([]byte, 2)
	rand.Read(b)
	return fmt.Sprintf("RO%s%X", time.Now().Format("20060102150405"), b)
}
