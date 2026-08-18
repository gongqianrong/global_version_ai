package domain

import (
	"crypto/rand"
	"fmt"
	"time"
)

// OrderLink state constants.
const (
	OrderLinkStatePending   = 0 // 待报价
	OrderLinkStateQuoted    = 1 // 已报价
	OrderLinkStatePaid      = 2 // 已支付
	OrderLinkStateCancelled = 3 // 已取消
)

// OrderLinkStateLabel returns a human-readable label for the state.
func OrderLinkStateLabel(state int) string {
	switch state {
	case OrderLinkStatePending:
		return "待报价"
	case OrderLinkStateQuoted:
		return "已报价"
	case OrderLinkStatePaid:
		return "已支付"
	case OrderLinkStateCancelled:
		return "已取消"
	default:
		return "未知"
	}
}

// IsValidOrderLinkTransition checks whether a state transition is allowed.
func IsValidOrderLinkTransition(from, to int) bool {
	switch from {
	case OrderLinkStatePending:
		return to == OrderLinkStateQuoted || to == OrderLinkStateCancelled
	case OrderLinkStateQuoted:
		return to == OrderLinkStatePaid || to == OrderLinkStateCancelled
	default:
		return false
	}
}

// OrderLink represents a row in the order_links table.
type OrderLink struct {
	ID          int64     `json:"id"`
	LinkNo      string    `json:"link_no"`
	UserID      int64     `json:"user_id"`
	State       int       `json:"state"`
	TotalAmount int64     `json:"total_amount"`
	OrderNumber *string   `json:"order_number,omitempty"`
	Remark      string    `json:"remark"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Items       []OrderLinkItem `json:"items,omitempty"`
}

// OrderLinkItem represents a row in the order_link_items table.
type OrderLinkItem struct {
	ID          int64     `json:"id"`
	OrderLinkID int64     `json:"order_link_id"`
	GoodsURL    string    `json:"goods_url"`
	GoodsName   string    `json:"goods_name"`
	GoodsImg    string    `json:"goods_img"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
	Remark      string    `json:"remark"`
	CreatedAt   time.Time `json:"created_at"`
}

// GenerateLinkNo creates a unique link number: OL + timestamp (microsecond) + 8 random hex chars.
// Format: OL20260102150405123456ABCD1234 (prefix + YYYYMMDDHHmmssμs + random)
func GenerateLinkNo() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		nanos := time.Now().UnixNano()
		b[0] = byte(nanos >> 24)
		b[1] = byte(nanos >> 16)
		b[2] = byte(nanos >> 8)
		b[3] = byte(nanos)
	}
	now := time.Now()
	return fmt.Sprintf("OL%s%06d%X", 
		now.Format("20060102150405"), 
		now.Nanosecond()/1000,
		b)
}
