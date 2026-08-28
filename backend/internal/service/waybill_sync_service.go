package service

import (
	"context"
	"fmt"
	"log"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// WaybillSyncStore 运单同步数据访问接口
type WaybillSyncStore interface {
	// GetByWaybillNo 根据运单号获取运单
	GetByWaybillNo(ctx context.Context, waybillNo string) (*domain.Waybill, []domain.WaybillOrder, error)
	
	// GetByWmsWaybillNo 根据WMS运单号获取运单
	GetByWmsWaybillNo(ctx context.Context, wmsWaybillNo string) (*domain.Waybill, error)
	
	// UpdateFromSync 从同步接口更新运单信息
	UpdateFromSync(ctx context.Context, req *domain.WaybillSyncRequest) error
	
	// QueryWaybills 查询运单列表
	QueryWaybills(ctx context.Context, req *domain.WaybillQueryRequest) ([]domain.WaybillDetailInfo, int64, error)
	
	// GetWaybillStates 批量获取运单状态
	GetWaybillStates(ctx context.Context, waybillNos []string) ([]domain.WaybillStateInfo, error)
	
	// RecordSyncLog 记录同步日志（用于幂等性检查）
	RecordSyncLog(ctx context.Context, requestID, waybillNo string, success bool, message string) error
	
	// CheckSyncLog 检查同步日志是否已存在
	CheckSyncLog(ctx context.Context, requestID string) (*domain.WaybillSyncResponse, error)
}

// WaybillSyncService 运单同步服务
type WaybillSyncService struct {
	store WaybillSyncStore
}

// NewWaybillSyncService 创建运单同步服务
func NewWaybillSyncService(store WaybillSyncStore) *WaybillSyncService {
	return &WaybillSyncService{store: store}
}

// SyncWaybill 同步运单数据（从国内管理端推送）
func (s *WaybillSyncService) SyncWaybill(ctx context.Context, req *domain.WaybillSyncRequest) (*domain.WaybillSyncResponse, error) {
	// 1. 幂等性检查
	if existingResp, err := s.store.CheckSyncLog(ctx, req.RequestID); err == nil && existingResp != nil {
		log.Printf("WaybillSync: duplicate request %s, returning cached response", req.RequestID)
		existingResp.Idempotent = true
		return existingResp, nil
	}

	// 2. 验证运单号
	if req.WaybillNo == "" {
		return s.errorResponse(req.WaybillNo, "waybill_no is required"), nil
	}

	// 3. 检查运单是否存在
	waybill, _, err := s.store.GetByWaybillNo(ctx, req.WaybillNo)
	if err != nil {
		msg := fmt.Sprintf("waybill %s not found", req.WaybillNo)
		_ = s.store.RecordSyncLog(ctx, req.RequestID, req.WaybillNo, false, msg)
		return s.errorResponse(req.WaybillNo, msg), nil
	}

	// 4. 验证状态转换是否合法
	if !domain.IsValidWaybillTransition(waybill.State, req.State) {
		msg := fmt.Sprintf("invalid state transition: %d → %d", waybill.State, req.State)
		_ = s.store.RecordSyncLog(ctx, req.RequestID, req.WaybillNo, false, msg)
		return s.errorResponse(req.WaybillNo, msg), nil
	}

	// 5. 更新运单信息
	if err := s.store.UpdateFromSync(ctx, req); err != nil {
		msg := fmt.Sprintf("failed to update waybill: %v", err)
		_ = s.store.RecordSyncLog(ctx, req.RequestID, req.WaybillNo, false, msg)
		return nil, fmt.Errorf("waybill_sync: %w", err)
	}

	// 6. 记录同步日志
	_ = s.store.RecordSyncLog(ctx, req.RequestID, req.WaybillNo, true, "success")

	// 7. 返回成功响应
	stateLabel := domain.WaybillStateLabel(req.State)
	return &domain.WaybillSyncResponse{
		Success:    true,
		Idempotent: false,
		Message:    "waybill synced successfully",
		WaybillNo:  req.WaybillNo,
		State:      &req.State,
		StateLabel: &stateLabel,
	}, nil
}

// QueryWaybills 查询运单列表
func (s *WaybillSyncService) QueryWaybills(ctx context.Context, req *domain.WaybillQueryRequest) (*domain.WaybillQueryResponse, error) {
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	// 查询运单
	items, total, err := s.store.QueryWaybills(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("query_waybills: %w", err)
	}

	// 填充状态标签
	for i := range items {
		items[i].StateLabel = domain.WaybillStateLabel(items[i].State)
	}

	return &domain.WaybillQueryResponse{
		Items:    items,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetWaybillStates 批量获取运单状态
func (s *WaybillSyncService) GetWaybillStates(ctx context.Context, req *domain.WaybillStatesRequest) (*domain.WaybillStatesResponse, error) {
	if len(req.WaybillNos) == 0 {
		return &domain.WaybillStatesResponse{Items: []domain.WaybillStateInfo{}}, nil
	}

	// 限制批量查询数量
	if len(req.WaybillNos) > 100 {
		return nil, fmt.Errorf("too many waybill_nos: max 100")
	}

	items, err := s.store.GetWaybillStates(ctx, req.WaybillNos)
	if err != nil {
		return nil, fmt.Errorf("get_waybill_states: %w", err)
	}

	// 填充状态标签
	for i := range items {
		items[i].StateLabel = domain.WaybillStateLabel(items[i].State)
	}

	return &domain.WaybillStatesResponse{Items: items}, nil
}

// errorResponse 创建错误响应
func (s *WaybillSyncService) errorResponse(waybillNo, message string) *domain.WaybillSyncResponse {
	return &domain.WaybillSyncResponse{
		Success:    false,
		Idempotent: false,
		Message:    message,
		WaybillNo:  waybillNo,
		State:      nil,
		StateLabel: nil,
	}
}
