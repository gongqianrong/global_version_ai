package domain

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Transaction type constants.
const (
	TxTypeRecharge   = "recharge"
	TxTypePurchase   = "purchase"
	TxTypeRefund     = "refund"
	TxTypeAdjustment = "adjustment"
)

// Payment method constants.
const (
	PayMethodWechat    = "wechat_pay"  // P0 第一期
	PayMethodAlipay    = "alipay"      // P0 第一期
	PayMethodApplePay  = "apple_pay"   // P1 后续扩展
	PayMethodGooglePay = "google_pay"  // P1 后续扩展
	PayMethodPayPal    = "paypal"      // P1 后续扩展
	PayMethodMoMo      = "momo"        // P2 越南
	PayMethodZaloPay   = "zalopay"     // P2 越南
	PayMethodKakaoPay  = "kakao_pay"   // P2 韩国
)

// RechargeOrder state constants.
const (
	RechargeStatePending = 0
	RechargeStatePaid    = 1
	RechargeStateFailed  = 2
)

// PayMethodInfo describes a payment method and its availability.
type PayMethodInfo struct {
	Method    string `json:"method"`
	Name      string `json:"name"`
	Available bool   `json:"available"` // false = 尚未对接，预留
}

// SupportedPayMethods lists all payment methods. available=false 表示预留待对接。
var SupportedPayMethods = []PayMethodInfo{
	{Method: PayMethodWechat,    Name: "微信支付",    Available: false},
	{Method: PayMethodAlipay,    Name: "支付宝",      Available: false},
	{Method: PayMethodApplePay,  Name: "Apple Pay",  Available: false},
	{Method: PayMethodGooglePay, Name: "Google Pay", Available: false},
	{Method: PayMethodPayPal,    Name: "PayPal",     Available: false},
	{Method: PayMethodMoMo,      Name: "MoMo",       Available: false},
	{Method: PayMethodZaloPay,   Name: "ZaloPay",    Available: false},
}

// IsSupportedPayMethod returns true if the given method is in the supported list.
func IsSupportedPayMethod(method string) bool {
	for _, m := range SupportedPayMethods {
		if m.Method == method {
			return true
		}
	}
	return false
}

// RechargeOrder represents a wallet top-up order.
type RechargeOrder struct {
	ID         int64     `json:"id"`
	RechargeNo string    `json:"recharge_no"`
	UserID     int64     `json:"user_id"`
	AmountJPY  int64     `json:"amount_jpy"`
	PayMethod  string    `json:"pay_method"`
	State      int       `json:"state"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// GenerateRechargeNo creates a unique recharge order number: RC + timestamp (microsecond) + 8 random hex chars.
// Format: RC20260102150405123456ABCD1234 (prefix + YYYYMMDDHHmmssμs + random)
func GenerateRechargeNo() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		nanos := time.Now().UnixNano()
		b[0] = byte(nanos >> 24)
		b[1] = byte(nanos >> 16)
		b[2] = byte(nanos >> 8)
		b[3] = byte(nanos)
	}
	now := time.Now()
	return fmt.Sprintf("RC%s%06d%X", 
		now.Format("20060102150405"), 
		now.Nanosecond()/1000,
		b)
}

// Wallet represents a user's JPY wallet.
type Wallet struct {
	ID        int64     `json:"id"`
	UserID    int64     `json:"user_id"`
	Balance   int64     `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// WalletTransaction represents a single wallet ledger entry.
type WalletTransaction struct {
	ID           int64     `json:"id"`
	UserID       int64     `json:"user_id"`
	Type         string    `json:"type"`
	Amount       int64     `json:"amount"`
	BalanceAfter int64     `json:"balance_after"`
	Description  string    `json:"description"`
	RelatedOrder *string   `json:"related_order,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
