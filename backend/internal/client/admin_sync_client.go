package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/rakutao/collection-gateway/internal/domain"
)

// JavaCompatibleTime wraps time.Time to marshal in Java-compatible format.
type JavaCompatibleTime struct {
	time.Time
}

// MarshalJSON formats time in Java-compatible format: yyyy-MM-dd HH:mm:ss (UTC+8).
func (t JavaCompatibleTime) MarshalJSON() ([]byte, error) {
	// Format: 2026-08-21 15:14:38 (no timezone info, assumes UTC+8)
	formatted := t.Time.Format("2006-01-02 15:04:05")
	return json.Marshal(formatted)
}

// AdminSyncClient handles synchronization with the admin backend.
type AdminSyncClient struct {
	baseURL    string
	httpClient *http.Client
	enabled    bool
}

// NewAdminSyncClient creates a new admin sync client.
func NewAdminSyncClient(baseURL string) *AdminSyncClient {
	enabled := baseURL != ""
	if !enabled {
		log.Println("[AdminSync] Admin sync disabled (no ADMIN_SYNC_URL configured)")
	} else {
		log.Printf("[AdminSync] Admin sync enabled: %s", baseURL)
	}
	
	return &AdminSyncClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		enabled: enabled,
	}
}

// SyncOrderRequest matches the admin API's order sync request.
type SyncOrderRequest struct {
	RequestID            string                   `json:"requestId"`
	GlobalOrderNumber    string                   `json:"globalOrderNumber"`
	GlobalAccountID      string                   `json:"globalAccountId,omitempty"`
	AccountInfoID        string                   `json:"accountInfoId"`
	AccountAddressID     string                   `json:"accountAddressId,omitempty"`
	OrderAddtime         *JavaCompatibleTime      `json:"orderAddtime,omitempty"`
	PayEffectiveTime     JavaCompatibleTime       `json:"payEffectiveTime"`
	OrderTotalJp         float64                  `json:"orderTotalJp"`
	OrderTotalCn         float64                  `json:"orderTotalCn"`
	CommissionFeeJp      float64                  `json:"commissionFeeJp"`
	CommissionFeeCn      float64                  `json:"commissionFeeCn"`
	HandlingFeeJp        float64                  `json:"handlingFeeJp"`
	HandlingFeeCn        float64                  `json:"handlingFeeCn"`
	OrderInpriceJp       float64                  `json:"orderInpriceJp"`
	OrderInpriceCn       float64                  `json:"orderInpriceCn"`
	OrderRate            float64                  `json:"orderRate"`
	TotalShippingFee     float64                  `json:"totalShippingFee"`
	TotalShippingFeeCn   float64                  `json:"totalShippingFeeCn"`
	OrderType            int                      `json:"orderType,omitempty"`
	OrderPurchaseType    int                      `json:"orderPurchaseType,omitempty"`
	OrderMode            int                      `json:"orderMode,omitempty"`
	OrderRemark          string                   `json:"orderRemark,omitempty"`
	Operator             string                   `json:"operator,omitempty"`
	GlobalOrderPayType   int                      `json:"globalOrderPayType"`
	DetailList           []SyncOrderDetailRequest `json:"detailList"`
}

// SyncOrderDetailRequest matches the admin API's order detail sync request.
type SyncOrderDetailRequest struct {
	GlobalOrderDetailNumber string  `json:"globalOrderDetailNumber"`
	Platform                int     `json:"platform"`
	GoodsMid                string  `json:"goodsMid"`
	GoodsImg                string  `json:"goodsImg,omitempty"`
	GoodsName               string  `json:"goodsName,omitempty"`
	GoodsNum                int     `json:"goodsNum,omitempty"`
	GoodsAmountJp           float64 `json:"goodsAmountJp"`
	GoodsAmountCn           float64 `json:"goodsAmountCn"`
	CommissionFeeJp         float64 `json:"commissionFeeJp"`
	CommissionFeeCn         float64 `json:"commissionFeeCn"`
	HandlingFeeJp           float64 `json:"handlingFeeJp"`
	HandlingFeeCn           float64 `json:"handlingFeeCn"`
	GoodsUrl                string  `json:"goodsUrl,omitempty"`
	SellerID                string  `json:"sellerId,omitempty"`
	ShippingFeeJp           float64 `json:"shippingFeeJp"`
	ShippingFeeCn           float64 `json:"shippingFeeCn"`
	OrderPurchaseType       int     `json:"orderPurchaseType,omitempty"`
	PurchaseDirect          int     `json:"purchaseDirect,omitempty"`
	DiscountType            int     `json:"discountType,omitempty"`
}

// PaymentSyncRequest matches the admin API's payment sync request.
type PaymentSyncRequest struct {
	RequestID          string             `json:"requestId"`
	GlobalOrderNumber  string             `json:"globalOrderNumber"`
	PaymentNumber      string             `json:"paymentNumber"`
	PayChannel         string             `json:"payChannel"`
	GlobalOrderPayType int                `json:"globalOrderPayType"`
	PayCurrency        string             `json:"payCurrency"`
	PayAmount          float64            `json:"payAmount"`
	PayTime            JavaCompatibleTime `json:"payTime"`
	Operator           string             `json:"operator,omitempty"`
}

// SyncOrderResponse is the admin API's order sync response.
type SyncOrderResponse struct {
	Success           bool   `json:"success"`
	Idempotent        bool   `json:"idempotent"`
	Message           string `json:"message"`
	OrderInfoID       string `json:"orderInfoId"`
	OrderNumber       string `json:"orderNumber"`
	GlobalOrderNumber string `json:"globalOrderNumber"`
	OrderState        int    `json:"orderState"`
}

// PaymentSyncResponse is the admin API's payment sync response.
type PaymentSyncResponse struct {
	Success           bool   `json:"success"`
	Idempotent        bool   `json:"idempotent"`
	Message           string `json:"message"`
	OrderInfoID       string `json:"orderInfoId"`
	OrderNumber       string `json:"orderNumber"`
	GlobalOrderNumber string `json:"globalOrderNumber"`
	PaymentNumber     string `json:"paymentNumber"`
	OrderState        int    `json:"orderState"`
}

// SyncOrder synchronizes an order to the admin backend.
func (c *AdminSyncClient) SyncOrder(ctx context.Context, req *SyncOrderRequest) (*SyncOrderResponse, error) {
	url := fmt.Sprintf("%s/internal/global/order/sync", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	log.Printf("[AdminSync] Syncing order %s to admin: %s", req.GlobalOrderNumber, url)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result SyncOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("admin API returned status %d: %s", resp.StatusCode, result.Message)
	}

	if !result.Success {
		return &result, fmt.Errorf("admin API rejected sync: %s", result.Message)
	}

	log.Printf("[AdminSync] Order %s synced successfully (idempotent=%v)", req.GlobalOrderNumber, result.Idempotent)
	return &result, nil
}

// SyncPayment synchronizes payment success to the admin backend.
func (c *AdminSyncClient) SyncPayment(ctx context.Context, req *PaymentSyncRequest) (*PaymentSyncResponse, error) {
	url := fmt.Sprintf("%s/internal/global/order/payment-success", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	log.Printf("[AdminSync] Syncing payment for order %s to admin: %s", req.GlobalOrderNumber, url)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result PaymentSyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &result, fmt.Errorf("admin API returned status %d: %s", resp.StatusCode, result.Message)
	}

	if !result.Success {
		return &result, fmt.Errorf("admin API rejected payment sync: %s", result.Message)
	}

	log.Printf("[AdminSync] Payment for order %s synced successfully (idempotent=%v)", req.GlobalOrderNumber, result.Idempotent)
	return &result, nil
}

// ConvertOrderToSyncRequest converts a local order to admin sync request.
func ConvertOrderToSyncRequest(order *domain.Order, details []domain.OrderDetail, userID string) *SyncOrderRequest {
	// Generate unique request ID
	requestID := fmt.Sprintf("INTL-SYNC-%s", order.OrderNumber)

	// Convert order details
	detailList := make([]SyncOrderDetailRequest, len(details))
	for i, d := range details {
		detailList[i] = SyncOrderDetailRequest{
			GlobalOrderDetailNumber: fmt.Sprintf("%s-D%d", order.OrderNumber, i+1),
			Platform:                1, // TODO: Map platform from product
			GoodsMid:                d.GoodsMid,
			GoodsImg:                d.GoodsImg,
			GoodsName:               d.GoodsName,
			GoodsNum:                d.GoodsNum,
			GoodsAmountJp:           float64(d.GoodsAmountJp) / 100.0,
			GoodsAmountCn:           float64(d.GoodsAmountJp) / 100.0 * 0.05, // TODO: Use actual exchange rate
			CommissionFeeJp:         float64(d.CommissionFeeJp) / 100.0,
			CommissionFeeCn:         float64(d.CommissionFeeJp) / 100.0 * 0.05,
			HandlingFeeJp:           0,
			HandlingFeeCn:           0,
			GoodsUrl:                d.GoodsUrl,
			SellerID:                d.SellerID,
			ShippingFeeJp:           float64(d.ShippingFeeJp) / 100.0,
			ShippingFeeCn:           float64(d.ShippingFeeJp) / 100.0 * 0.05,
			OrderPurchaseType:       order.OrderPurchaseType,
		}
	}

	// Calculate payment effective time (30 minutes from now)
	payEffectiveTime := JavaCompatibleTime{Time: time.Now().Add(30 * time.Minute)}

	return &SyncOrderRequest{
		RequestID:          requestID,
		GlobalOrderNumber:  order.OrderNumber,
		AccountInfoID:      userID,
		PayEffectiveTime:   payEffectiveTime,
		OrderTotalJp:       float64(order.OrderTotalJp) / 100.0,
		OrderTotalCn:       float64(order.OrderTotalJp) / 100.0 * 0.05,
		CommissionFeeJp:    float64(order.CommissionFeeJp) / 100.0,
		CommissionFeeCn:    float64(order.CommissionFeeJp) / 100.0 * 0.05,
		HandlingFeeJp:      0,
		HandlingFeeCn:      0,
		OrderInpriceJp:     float64(order.OrderInpriceJp) / 100.0,
		OrderInpriceCn:     float64(order.OrderInpriceJp) / 100.0 * 0.05,
		OrderRate:          0.05,
		TotalShippingFee:   float64(order.ShippingFeeJp) / 100.0,
		TotalShippingFeeCn: float64(order.ShippingFeeJp) / 100.0 * 0.05,
		OrderType:          1, // Default order type
		OrderPurchaseType:  order.OrderPurchaseType,
		OrderMode:          1, // Auto order
		OrderRemark:        order.OrderRemark,
		Operator:           "SYSTEM_INTL",
		GlobalOrderPayType: order.OrderPaytype,
		DetailList:         detailList,
	}
}

// SyncOrderAsync syncs order creation to admin backend (fire-and-forget).
func (c *AdminSyncClient) SyncOrderAsync(ctx context.Context, order *domain.Order, details []domain.OrderDetail, userID string) {
	if !c.enabled {
		return
	}

	req := ConvertOrderToSyncRequest(order, details, userID)
	
	// Use background context with timeout
	syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	_, err := c.SyncOrder(syncCtx, req)
	if err != nil {
		log.Printf("[AdminSync] Failed to sync order %s: %v", order.OrderNumber, err)
	}
}

// SyncPaymentAsync syncs payment success to admin backend (fire-and-forget).
func (c *AdminSyncClient) SyncPaymentAsync(ctx context.Context, orderNumber string, paymentAmount int64, userID string) {
	if !c.enabled {
		return
	}

	req := &PaymentSyncRequest{
		RequestID:          fmt.Sprintf("INTL-PAY-%s", orderNumber),
		GlobalOrderNumber:  orderNumber,
		PaymentNumber:      fmt.Sprintf("PAY-%s-%d", orderNumber, time.Now().Unix()),
		PayChannel:         "WALLET",
		GlobalOrderPayType: 100, // TODO: Get from order
		PayCurrency:        "JPY",
		PayAmount:          float64(paymentAmount) / 100.0,
		PayTime:            JavaCompatibleTime{Time: time.Now()},
		Operator:           "SYSTEM_INTL",
	}

	// Use background context with timeout
	syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	_, err := c.SyncPayment(syncCtx, req)
	if err != nil {
		log.Printf("[AdminSync] Failed to sync payment for order %s: %v", orderNumber, err)
	}
}

