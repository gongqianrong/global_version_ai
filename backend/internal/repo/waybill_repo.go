package repo

import (
	"context"
	"fmt"

	"github.com/rakutao/collection-gateway/internal/db"
	"github.com/rakutao/collection-gateway/internal/domain"
)

// WaybillRepo handles waybill persistence in PostgreSQL.
// It implements api.WaybillStore and api.OrderWarehouseStore.
type WaybillRepo struct {
	db *db.DB
}

// NewWaybillRepo creates a WaybillRepo.
func NewWaybillRepo(d *db.DB) *WaybillRepo {
	return &WaybillRepo{db: d}
}

// Create inserts a waybill and links the given order numbers to it in a transaction.
func (r *WaybillRepo) Create(ctx context.Context, waybill *domain.Waybill, orderNumbers []string) (*domain.Waybill, error) {
	tx, err := r.db.Pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("waybill_repo: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO waybills (waybill_no, user_id, state, shipping_fee_jpy, remark)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, updated_at`,
		waybill.WaybillNo, waybill.UserID, waybill.State, waybill.ShippingFeeJPY, waybill.Remark,
	).Scan(&waybill.ID, &waybill.CreatedAt, &waybill.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("waybill_repo: insert waybill: %w", err)
	}

	for _, orderNo := range orderNumbers {
		_, err = tx.Exec(ctx,
			`INSERT INTO waybill_orders (waybill_id, waybill_no, order_number) VALUES ($1, $2, $3)`,
			waybill.ID, waybill.WaybillNo, orderNo,
		)
		if err != nil {
			return nil, fmt.Errorf("waybill_repo: link order %s: %w", orderNo, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("waybill_repo: commit: %w", err)
	}
	return waybill, nil
}

// GetByWaybillNo returns a waybill and its linked orders by waybill number.
func (r *WaybillRepo) GetByWaybillNo(ctx context.Context, waybillNo string) (*domain.Waybill, []domain.WaybillOrder, error) {
	var w domain.Waybill
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, waybill_no, user_id, state, shipping_fee_jpy,
		  carrier, tracking_no, tracking_url, wms_waybill_no, remark,
		  created_at, updated_at
		 FROM waybills WHERE waybill_no = $1`,
		waybillNo,
	).Scan(&w.ID, &w.WaybillNo, &w.UserID, &w.State, &w.ShippingFeeJPY,
		&w.Carrier, &w.TrackingNo, &w.TrackingURL, &w.WmsWaybillNo, &w.Remark,
		&w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, nil, fmt.Errorf("waybill_repo: get waybill: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT waybill_id, waybill_no, order_number FROM waybill_orders WHERE waybill_no = $1 ORDER BY id`,
		waybillNo,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("waybill_repo: get waybill orders: %w", err)
	}
	defer rows.Close()

	var orders []domain.WaybillOrder
	for rows.Next() {
		var wo domain.WaybillOrder
		if err := rows.Scan(&wo.WaybillID, &wo.WaybillNo, &wo.OrderNumber); err != nil {
			return nil, nil, fmt.Errorf("waybill_repo: scan waybill order: %w", err)
		}
		orders = append(orders, wo)
	}
	return &w, orders, rows.Err()
}

// ListByUser returns paginated waybills for a user, newest first.
func (r *WaybillRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]domain.Waybill, int64, error) {
	var total int64
	if err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM waybills WHERE user_id = $1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("waybill_repo: count: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, waybill_no, user_id, state, shipping_fee_jpy,
		  carrier, tracking_no, tracking_url, wms_waybill_no, remark,
		  created_at, updated_at
		 FROM waybills WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("waybill_repo: list: %w", err)
	}
	defer rows.Close()

	var waybills []domain.Waybill
	for rows.Next() {
		var w domain.Waybill
		if err := rows.Scan(&w.ID, &w.WaybillNo, &w.UserID, &w.State, &w.ShippingFeeJPY,
			&w.Carrier, &w.TrackingNo, &w.TrackingURL, &w.WmsWaybillNo, &w.Remark,
			&w.CreatedAt, &w.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("waybill_repo: scan: %w", err)
		}
		waybills = append(waybills, w)
	}
	return waybills, total, rows.Err()
}

// UpdateState updates a waybill's state.
func (r *WaybillRepo) UpdateState(ctx context.Context, waybillNo string, state int) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE waybills SET state = $1, updated_at = NOW() WHERE waybill_no = $2`,
		state, waybillNo,
	)
	if err != nil {
		return fmt.Errorf("waybill_repo: update_state: %w", err)
	}
	return nil
}

// SetShippingFee sets the international shipping fee for a waybill.
func (r *WaybillRepo) SetShippingFee(ctx context.Context, waybillNo string, feeJPY int64) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE waybills SET shipping_fee_jpy = $1, updated_at = NOW() WHERE waybill_no = $2`,
		feeJPY, waybillNo,
	)
	if err != nil {
		return fmt.Errorf("waybill_repo: set_shipping_fee: %w", err)
	}
	return nil
}

// SetShippingInfo fills carrier, tracking number and tracking URL.
func (r *WaybillRepo) SetShippingInfo(ctx context.Context, waybillNo, carrier, trackingNo, trackingURL, wmsWaybillNo string) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE waybills
		 SET carrier = $1, tracking_no = $2, tracking_url = $3, wms_waybill_no = $4, updated_at = NOW()
		 WHERE waybill_no = $5`,
		carrier, trackingNo, trackingURL, wmsWaybillNo, waybillNo,
	)
	if err != nil {
		return fmt.Errorf("waybill_repo: set_shipping_info: %w", err)
	}
	return nil
}

// UpdateLinkedOrderStates updates all orders linked to the waybill to the given order state.
func (r *WaybillRepo) UpdateLinkedOrderStates(ctx context.Context, waybillNo string, orderState int) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE orders SET order_state = $1, updated_at = NOW()
		 WHERE order_number IN (
		   SELECT order_number FROM waybill_orders WHERE waybill_no = $2
		 )`,
		orderState, waybillNo,
	)
	if err != nil {
		return fmt.Errorf("waybill_repo: update_linked_order_states: %w", err)
	}
	return nil
}

// ListWarehoused returns all warehoused orders for a user (implements OrderWarehouseStore).
func (r *WaybillRepo) ListWarehoused(ctx context.Context, userID int64) ([]domain.Order, error) {
	rows, err := r.db.Pool.Query(ctx,
		`SELECT id, order_number, user_id, order_state, order_total_jp, commission_fee_jp,
		  shipping_fee_jp, order_inprice_jp, order_paytype, order_remark, order_purchase_type,
		  created_at, updated_at
		 FROM orders WHERE user_id = $1 AND order_state = $2
		 ORDER BY created_at DESC`,
		userID, domain.OrderStateWarehoused,
	)
	if err != nil {
		return nil, fmt.Errorf("waybill_repo: list_warehoused: %w", err)
	}
	defer rows.Close()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := rows.Scan(&o.ID, &o.OrderNumber, &o.UserID, &o.OrderState, &o.OrderTotalJp,
			&o.CommissionFeeJp, &o.ShippingFeeJp, &o.OrderInpriceJp, &o.OrderPaytype,
			&o.OrderRemark, &o.OrderPurchaseType, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, fmt.Errorf("waybill_repo: scan warehoused order: %w", err)
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}
