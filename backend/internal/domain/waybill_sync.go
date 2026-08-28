package domain

import "time"

// WaybillSyncRequest 运单数据同步请求（从国内管理端推送）
type WaybillSyncRequest struct {
	RequestID          string    `json:"requestId"`          // 请求唯一ID（幂等性）
	WaybillNo          string    `json:"waybillNo"`          // 运单号
	WmsWaybillNo       string    `json:"wmsWaybillNo"`       // WMS系统运单号
	State              int       `json:"state"`              // 运单状态
	ShippingFeeJPY     *int64    `json:"shippingFeeJpy"`     // 国际运费（日元）
	Carrier            *string   `json:"carrier"`            // 承运商
	TrackingNo         *string   `json:"trackingNo"`         // 物流单号
	TrackingURL        *string   `json:"trackingUrl"`        // 物流追踪URL
	Remark             *string   `json:"remark"`             // 备注
	PackedTime         *time.Time `json:"packedTime"`        // 打包完成时间
	ShippedTime        *time.Time `json:"shippedTime"`       // 发货时间
	Operator           *string   `json:"operator"`           // 操作员
}

// WaybillSyncResponse 运单数据同步响应
type WaybillSyncResponse struct {
	Success      bool    `json:"success"`      // 是否成功
	Idempotent   bool    `json:"idempotent"`   // 是否幂等（重复请求）
	Message      string  `json:"message"`      // 消息
	WaybillNo    string  `json:"waybillNo"`    // 运单号
	State        *int    `json:"state"`        // 当前状态
	StateLabel   *string `json:"stateLabel"`   // 状态标签
}

// WaybillQueryRequest 运单查询请求（国内管理端查询）
type WaybillQueryRequest struct {
	WaybillNo    *string   `json:"waybillNo"`    // 按运单号查询
	WmsWaybillNo *string   `json:"wmsWaybillNo"` // 按WMS运单号查询
	UserID       *int64    `json:"userId"`       // 按用户ID查询
	State        *int      `json:"state"`        // 按状态查询
	DateFrom     *time.Time `json:"dateFrom"`    // 起始日期
	DateTo       *time.Time `json:"dateTo"`      // 结束日期
	Page         int       `json:"page"`         // 页码
	PageSize     int       `json:"pageSize"`     // 每页数量
}

// WaybillQueryResponse 运单查询响应
type WaybillQueryResponse struct {
	Items      []WaybillDetailInfo `json:"items"`      // 运单列表
	Total      int64               `json:"total"`      // 总数
	Page       int                 `json:"page"`       // 当前页
	PageSize   int                 `json:"pageSize"`   // 每页数量
}

// WaybillDetailInfo 运单详细信息（含订单列表）
type WaybillDetailInfo struct {
	Waybill
	Orders []WaybillOrderDetail `json:"orders"` // 关联订单列表
}

// WaybillOrderDetail 运单关联订单详情
type WaybillOrderDetail struct {
	OrderNumber       string    `json:"orderNumber"`       // 订单号
	OrderState        int       `json:"orderState"`        // 订单状态
	OrderTotalJp      int64     `json:"orderTotalJp"`      // 订单总价
	CommissionFeeJp   int64     `json:"commissionFeeJp"`   // 手续费
	ShippingFeeJp     int64     `json:"shippingFeeJp"`     // 国内运费
	OrderInpriceJp    int64     `json:"orderInpriceJp"`    // 实付金额
	OrderRemark       string    `json:"orderRemark"`       // 订单备注
	CreatedAt         time.Time `json:"createdAt"`         // 创建时间
}

// WaybillStatesRequest 批量查询运单状态请求
type WaybillStatesRequest struct {
	WaybillNos []string `json:"waybillNos"` // 运单号列表
}

// WaybillStatesResponse 批量查询运单状态响应
type WaybillStatesResponse struct {
	Items []WaybillStateInfo `json:"items"` // 状态列表
}

// WaybillStateInfo 运单状态信息
type WaybillStateInfo struct {
	WaybillNo    string  `json:"waybillNo"`    // 运单号
	State        int     `json:"state"`        // 状态
	StateLabel   string  `json:"stateLabel"`   // 状态标签
	WmsWaybillNo string  `json:"wmsWaybillNo"` // WMS运单号
	TrackingNo   string  `json:"trackingNo"`   // 物流单号
	UpdatedAt    time.Time `json:"updatedAt"`  // 更新时间
}
