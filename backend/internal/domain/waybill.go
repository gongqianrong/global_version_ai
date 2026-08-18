package domain

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Waybill state constants — aligned with WMS system.
// 国际版运单生命周期：待合单 → 待打包 → 待支付 → 待出库 → 已发货 → 已收货
const (
	WaybillStatePendingConsolidation = 0 // 待合单：用户申请发货，仓库待合并订单
	WaybillStatePendingPacking       = 1 // 待打包：合单完成，仓库待打包
	WaybillStatePendingPayment       = 2 // 待支付：打包完成，运费待用户支付
	WaybillStatePendingDispatch      = 3 // 待出库：用户已支付运费，仓库待出库
	WaybillStateShipped              = 4 // 已发货：仓库已出库，物流运输中
	WaybillStateDelivered            = 5 // 已收货：用户确认收货
)

// WaybillStateLabel returns a human-readable label for a waybill state.
func WaybillStateLabel(state int) string {
	labels := map[int]string{
		WaybillStatePendingConsolidation: "待合单",
		WaybillStatePendingPacking:       "待打包",
		WaybillStatePendingPayment:       "待支付",
		WaybillStatePendingDispatch:      "待出库",
		WaybillStateShipped:              "已发货",
		WaybillStateDelivered:            "已收货",
	}
	if l, ok := labels[state]; ok {
		return l
	}
	return "未知"
}

// Waybill represents an international shipment waybill.
// One waybill may consolidate multiple warehoused orders from the same user.
type Waybill struct {
	ID             int64     `json:"id"`
	WaybillNo      string    `json:"waybill_no"`
	UserID         int64     `json:"user_id"`
	State          int       `json:"state"`
	StateLabel     string    `json:"state_label"`
	ShippingFeeJPY int64     `json:"shipping_fee_jpy"` // 国际运费（打包完成后由WMS填写）
	Carrier        string    `json:"carrier"`           // 承运商
	TrackingNo     string    `json:"tracking_no"`       // 国际物流单号
	TrackingURL    string    `json:"tracking_url"`      // 物流追踪链接
	WmsWaybillNo   string    `json:"wms_waybill_no,omitempty"` // WMS系统运单号（预留对接）
	Remark         string    `json:"remark"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// WaybillOrder links an order to a waybill (合单关联).
type WaybillOrder struct {
	WaybillID   int64  `json:"waybill_id"`
	WaybillNo   string `json:"waybill_no"`
	OrderNumber string `json:"order_number"`
}

// GenerateWaybillNo creates a unique waybill number: LO + timestamp (microsecond) + 8 random hex chars.
// Format: LO20260102150405123456ABCD1234 (prefix + YYYYMMDDHHmmssμs + random)
func GenerateWaybillNo() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		nanos := time.Now().UnixNano()
		b[0] = byte(nanos >> 24)
		b[1] = byte(nanos >> 16)
		b[2] = byte(nanos >> 8)
		b[3] = byte(nanos)
	}
	now := time.Now()
	return fmt.Sprintf("LO%s%06d%X", 
		now.Format("20060102150405"), 
		now.Nanosecond()/1000,
		b)
}

// ValidWaybillStateTransitions defines allowed state transitions.
// key = fromState, value = allowed toStates (admin-driven transitions).
var ValidWaybillStateTransitions = map[int][]int{
	WaybillStatePendingConsolidation: {WaybillStatePendingPacking},
	WaybillStatePendingPacking:       {WaybillStatePendingPayment},
	WaybillStatePendingPayment:       {WaybillStatePendingDispatch}, // after user pays
	WaybillStatePendingDispatch:      {WaybillStateShipped},
	WaybillStateShipped:              {WaybillStateDelivered},
}

// IsValidWaybillTransition returns true if the given state transition is allowed.
func IsValidWaybillTransition(from, to int) bool {
	allowed, ok := ValidWaybillStateTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// ValidOrderStateTransitions defines allowed admin-driven order state transitions.
// Paid→Purchasing (automation script), Purchasing→Warehoused (WMS scan-in).
var ValidOrderStateTransitions = map[int][]int{
	OrderStatePaid:       {OrderStatePurchasing},
	OrderStatePurchasing: {OrderStateWarehoused},
}

// IsValidOrderTransition returns true if the given order state transition is allowed.
func IsValidOrderTransition(from, to int) bool {
	allowed, ok := ValidOrderStateTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}
