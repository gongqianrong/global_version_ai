package repo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// WaybillSyncRepo 运单同步数据访问层
type WaybillSyncRepo struct {
	db *db.DB
}

// NewWaybillSyncRepo 创建运单同步仓储
func NewWaybillSyncRepo(d *db.DB) *WaybillSyncRepo {
	return &WaybillSyncRepo{db: d}
}

// GetByWaybillNo 根据运单号获取运单
func (r *WaybillSyncRepo) GetByWaybillNo(ctx context.Context, waybillNo string) (*domain.Waybill, []domain.WaybillOrder, error) {
	repo := NewWaybillRepo(r.db)
	return repo.GetByWaybillNo(ctx, waybillNo)
}

// GetByWmsWaybillNo 根据WMS运单号获取运单
func (r *WaybillSyncRepo) GetByWmsWaybillNo(ctx context.Context, wmsWaybillNo string) (*domain.Waybill, error) {
	var w domain.Waybill
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, waybill_no, user_id, state, shipping_fee_jpy,
		  carrier, tracking_no, tracking_url, wms_waybill_no, remark,
		  created_at, updated_at
		 FROM waybills WHERE wms_waybill_no = $1`,
		wmsWaybillNo,
	).Scan(&w.ID, &w.WaybillNo, &w.UserID, &w.State, &w.ShippingFeeJPY,
		&w.Carrier, &w.TrackingNo, &w.TrackingURL, &w.WmsWaybillNo, &w.Remark,
		&w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("waybill_sync_repo: get by wms_waybill_no: %w", err)
	}
	return &w, nil
}

// UpdateFromSync 从同步接口更新运单信息
func (r *WaybillSyncRepo) UpdateFromSync(ctx context.Context, req *domain.WaybillSyncRequest) error {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 构建动态更新SQL
	query := `UPDATE waybills SET state = $1, updated_at = NOW()`
	args := []interface{}{req.State}
	argIdx := 2

	if req.WmsWaybillNo != "" {
		query += fmt.Sprintf(`, wms_waybill_no = $%d`, argIdx)
		args = append(args, req.WmsWaybillNo)
		argIdx++
	}

	if req.ShippingFeeJPY != nil {
		query += fmt.Sprintf(`, shipping_fee_jpy = $%d`, argIdx)
		args = append(args, *req.ShippingFeeJPY)
		argIdx++
	}

	if req.Carrier != nil {
		query += fmt.Sprintf(`, carrier = $%d`, argIdx)
		args = append(args, *req.Carrier)
		argIdx++
	}

	if req.TrackingNo != nil {
		query += fmt.Sprintf(`, tracking_no = $%d`, argIdx)
		args = append(args, *req.TrackingNo)
		argIdx++
	}

	if req.TrackingURL != nil {
		query += fmt.Sprintf(`, tracking_url = $%d`, argIdx)
		args = append(args, *req.TrackingURL)
		argIdx++
	}

	if req.Remark != nil {
		query += fmt.Sprintf(`, remark = $%d`, argIdx)
		args = append(args, *req.Remark)
		argIdx++
	}

	query += fmt.Sprintf(` WHERE waybill_no = $%d`, argIdx)
	args = append(args, req.WaybillNo)

	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update waybill: %w", err)
	}

	// 根据状态同步订单状态
	var orderState *int
	switch req.State {
	case domain.WaybillStatePendingPayment:
		// 待支付 → 订单进入 Packed 状态
		state := domain.OrderStatePacked
		orderState = &state
	case domain.WaybillStateShipped:
		// 已发货 → 订单进入 Shipped 状态
		state := domain.OrderStateShipped
		orderState = &state
	case domain.WaybillStateDelivered:
		// 已收货 → 订单进入 Fulfilled 状态
		state := domain.OrderStateFulfilled
		orderState = &state
	}

	if orderState != nil {
		_, err = tx.Exec(ctx,
			`UPDATE orders SET order_state = $1, updated_at = NOW()
			 WHERE order_number IN (
			   SELECT order_number FROM waybill_orders WHERE waybill_no = $2
			 )`,
			*orderState, req.WaybillNo,
		)
		if err != nil {
			return fmt.Errorf("update linked orders: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// QueryWaybills 查询运单列表
func (r *WaybillSyncRepo) QueryWaybills(ctx context.Context, req *domain.WaybillQueryRequest) ([]domain.WaybillDetailInfo, int64, error) {
	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if req.WaybillNo != nil {
		whereClause += fmt.Sprintf(" AND w.waybill_no = $%d", argIdx)
		args = append(args, *req.WaybillNo)
		argIdx++
	}

	if req.WmsWaybillNo != nil {
		whereClause += fmt.Sprintf(" AND w.wms_waybill_no = $%d", argIdx)
		args = append(args, *req.WmsWaybillNo)
		argIdx++
	}

	if req.UserID != nil {
		whereClause += fmt.Sprintf(" AND w.user_id = $%d", argIdx)
		args = append(args, *req.UserID)
		argIdx++
	}

	if req.State != nil {
		whereClause += fmt.Sprintf(" AND w.state = $%d", argIdx)
		args = append(args, *req.State)
		argIdx++
	}

	if req.DateFrom != nil {
		whereClause += fmt.Sprintf(" AND w.created_at >= $%d", argIdx)
		args = append(args, *req.DateFrom)
		argIdx++
	}

	if req.DateTo != nil {
		whereClause += fmt.Sprintf(" AND w.created_at <= $%d", argIdx)
		args = append(args, *req.DateTo)
		argIdx++
	}

	// 查询总数
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM waybills w %s", whereClause)
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count waybills: %w", err)
	}

	// 查询运单列表
	limit := req.PageSize
	offset := (req.Page - 1) * req.PageSize
	args = append(args, limit, offset)

	query := fmt.Sprintf(`
		SELECT w.id, w.waybill_no, w.user_id, w.state, w.shipping_fee_jpy,
		       w.carrier, w.tracking_no, w.tracking_url, w.wms_waybill_no, w.remark,
		       w.created_at, w.updated_at
		FROM waybills w %s
		ORDER BY w.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query waybills: %w", err)
	}
	defer rows.Close()

	var results []domain.WaybillDetailInfo
	for rows.Next() {
		var info domain.WaybillDetailInfo
		if err := rows.Scan(&info.ID, &info.WaybillNo, &info.UserID, &info.State, &info.ShippingFeeJPY,
			&info.Carrier, &info.TrackingNo, &info.TrackingURL, &info.WmsWaybillNo, &info.Remark,
			&info.CreatedAt, &info.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan waybill: %w", err)
		}

		// 查询关联订单
		orders, err := r.getWaybillOrders(ctx, info.WaybillNo)
		if err != nil {
			return nil, 0, fmt.Errorf("get waybill orders: %w", err)
		}
		info.Orders = orders

		results = append(results, info)
	}

	return results, total, rows.Err()
}

// getWaybillOrders 获取运单关联的订单详情
func (r *WaybillSyncRepo) getWaybillOrders(ctx context.Context, waybillNo string) ([]domain.WaybillOrderDetail, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT o.order_number, o.order_state, o.order_total_jp, o.commission_fee_jp,
		       o.shipping_fee_jp, o.order_inprice_jp, o.order_remark, o.created_at
		FROM orders o
		INNER JOIN waybill_orders wo ON o.order_number = wo.order_number
		WHERE wo.waybill_no = $1
		ORDER BY o.created_at DESC
	`, waybillNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []domain.WaybillOrderDetail
	for rows.Next() {
		var order domain.WaybillOrderDetail
		if err := rows.Scan(&order.OrderNumber, &order.OrderState, &order.OrderTotalJp, &order.CommissionFeeJp,
			&order.ShippingFeeJp, &order.OrderInpriceJp, &order.OrderRemark, &order.CreatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return orders, rows.Err()
}

// GetWaybillStates 批量获取运单状态
func (r *WaybillSyncRepo) GetWaybillStates(ctx context.Context, waybillNos []string) ([]domain.WaybillStateInfo, error) {
	if len(waybillNos) == 0 {
		return []domain.WaybillStateInfo{}, nil
	}

	// 构建 IN 子句
	query := `
		SELECT waybill_no, state, wms_waybill_no, tracking_no, updated_at
		FROM waybills
		WHERE waybill_no = ANY($1)
		ORDER BY updated_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, waybillNos)
	if err != nil {
		return nil, fmt.Errorf("query waybill states: %w", err)
	}
	defer rows.Close()

	var results []domain.WaybillStateInfo
	for rows.Next() {
		var info domain.WaybillStateInfo
		var wmsWaybillNo, trackingNo sql.NullString
		if err := rows.Scan(&info.WaybillNo, &info.State, &wmsWaybillNo, &trackingNo, &info.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan waybill state: %w", err)
		}
		info.WmsWaybillNo = wmsWaybillNo.String
		info.TrackingNo = trackingNo.String
		results = append(results, info)
	}

	return results, rows.Err()
}

// RecordSyncLog 记录同步日志
func (r *WaybillSyncRepo) RecordSyncLog(ctx context.Context, requestID, waybillNo string, success bool, message string) error {
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO waybill_sync_logs (request_id, waybill_no, success, message, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (request_id) DO NOTHING
	`, requestID, waybillNo, success, message)
	return err
}

// CheckSyncLog 检查同步日志是否已存在
func (r *WaybillSyncRepo) CheckSyncLog(ctx context.Context, requestID string) (*domain.WaybillSyncResponse, error) {
	var waybillNo, message string
	var success bool
	var state sql.NullInt64

	err := r.db.Pool.QueryRow(ctx, `
		SELECT wsl.waybill_no, wsl.success, wsl.message, w.state
		FROM waybill_sync_logs wsl
		LEFT JOIN waybills w ON wsl.waybill_no = w.waybill_no
		WHERE wsl.request_id = $1
	`, requestID).Scan(&waybillNo, &success, &message, &state)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("check sync log: %w", err)
	}

	resp := &domain.WaybillSyncResponse{
		Success:   success,
		Message:   message,
		WaybillNo: waybillNo,
	}

	if state.Valid {
		stateInt := int(state.Int64)
		stateLabel := domain.WaybillStateLabel(stateInt)
		resp.State = &stateInt
		resp.StateLabel = &stateLabel
	}

	return resp, nil
}
