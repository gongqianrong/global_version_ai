# 国际版订单同步接口测试 - Curl 命令集合

## 快速测试命令

### 1. 订单同步（Order Sync）

```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "TEST-001",
    "globalOrderNumber": "G2026082001",
    "globalAccountId": "GU100001",
    "accountInfoId": "100001",
    "accountAddressId": "ADDR001",
    "orderAddtime": "2026-08-20T10:00:00+08:00",
    "payEffectiveTime": "2026-08-20T11:00:00+08:00",
    "orderTotalJp": 10000,
    "orderTotalCn": 500,
    "commissionFeeJp": 1000,
    "commissionFeeCn": 50,
    "handlingFeeJp": 0,
    "handlingFeeCn": 0,
    "orderInpriceJp": 11000,
    "orderInpriceCn": 550,
    "orderRate": 0.05,
    "totalShippingFee": 0,
    "totalShippingFeeCn": 0,
    "orderType": 1,
    "orderPurchaseType": 1,
    "orderMode": 1,
    "orderRemark": "测试订单",
    "operator": "TEST",
    "globalOrderPayType": 100,
    "detailList": [
      {
        "globalOrderDetailNumber": "GD2026082001",
        "platform": 1,
        "goodsMid": "test_item_001",
        "goodsImg": "https://example.com/item.jpg",
        "goodsName": "测试商品",
        "goodsNum": 1,
        "goodsAmountJp": 10000,
        "goodsAmountCn": 500,
        "commissionFeeJp": 1000,
        "commissionFeeCn": 50,
        "handlingFeeJp": 0,
        "handlingFeeCn": 0,
        "goodsUrl": "https://example.com/item",
        "sellerId": "test_seller",
        "shippingFeeJp": 0,
        "shippingFeeCn": 0,
        "orderPurchaseType": 1,
        "purchaseDirect": 0,
        "discountType": 0
      }
    ]
  }'
```

**期望响应**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "success": true,
    "idempotent": false,
    "message": "国际版订单同步成功",
    "orderInfoId": "12345",
    "orderNumber": "RO20260820...",
    "globalOrderNumber": "G2026082001",
    "orderState": 0
  }
}
```

### 2. 支付同步（Payment Success）

```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/payment-success \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "PAY-TEST-001",
    "globalOrderNumber": "G2026082001",
    "paymentNumber": "PAY001",
    "payChannel": "STRIPE",
    "payWay": 1,
    "payAmountJp": 11000,
    "payAmountCn": 550,
    "paySeccussTime": "2026-08-20T10:30:00+08:00"
  }'
```

**期望响应**：
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "success": true,
    "idempotent": false,
    "message": "支付同步成功",
    "orderNumber": "RO20260820...",
    "orderState": 3,
    "paymentNumber": "PAY001"
  }
}
```

## 测试场景

### 场景 1：幂等性测试

再次发送相同的 requestId：

```bash
# 订单幂等性
curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \
  -H "Content-Type: application/json" \
  -d '{"requestId": "TEST-001", ...}' # 相同的 requestId

# 期望: idempotent=true
```

```bash
# 支付幂等性
curl -X POST http://localhost:8080/api/v1/internal/global/order/payment-success \
  -H "Content-Type: application/json" \
  -d '{"requestId": "PAY-TEST-001", ...}' # 相同的 requestId

# 期望: idempotent=true
```

### 场景 2：金额不匹配（应失败）

```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/payment-success \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "PAY-WRONG-001",
    "globalOrderNumber": "G2026082001",
    "paymentNumber": "PAY-WRONG-001",
    "payChannel": "STRIPE",
    "payWay": 1,
    "payAmountJp": 9999,
    "payAmountCn": 500,
    "paySeccussTime": "2026-08-20T10:35:00+08:00"
  }'

# 期望: success=false, message包含"金额不匹配"
```

### 场景 3：不存在的订单（应失败）

```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/payment-success \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "PAY-NOTFOUND-001",
    "globalOrderNumber": "G99999999",
    "paymentNumber": "PAY-NOTFOUND-001",
    "payChannel": "STRIPE",
    "payWay": 1,
    "payAmountJp": 11000,
    "payAmountCn": 550,
    "paySeccussTime": "2026-08-20T10:40:00+08:00"
  }'

# 期望: success=false, message包含"未找到"或"不存在"
```

### 场景 4：缺少必填字段（应失败）

```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "TEST-INVALID-001"
  }'

# 期望: code != 200, 返回错误信息
```

## 数据库验证查询

### 查看国际版订单记录

```bash
docker-compose exec postgres psql -U postgres -d rakutao -c "
SELECT * FROM global_order_records 
WHERE global_order_number = 'G2026082001';
"
```

### 查看支付记录

```bash
docker-compose exec postgres psql -U postgres -d rakutao -c "
SELECT * FROM global_order_payments 
WHERE global_order_number = 'G2026082001';
"
```

### 查看对应本地订单

```bash
docker-compose exec postgres psql -U postgres -d rakutao -c "
SELECT o.*, gor.global_order_number, gor.pay_sync_state
FROM orders o
JOIN global_order_records gor ON o.order_number = gor.order_number
WHERE gor.global_order_number = 'G2026082001';
"
```

### 检查数据一致性

```bash
docker-compose exec postgres psql -U postgres -d rakutao -c "
SELECT 
  gor.global_order_number,
  gor.order_number,
  gor.order_state as recorded_state,
  o.order_state as actual_state,
  gor.pay_sync_state,
  CASE 
    WHEN gor.order_state = o.order_state THEN '一致'
    ELSE '不一致'
  END as consistency
FROM global_order_records gor
JOIN orders o ON gor.order_number = o.order_number
WHERE gor.global_order_number = 'G2026082001';
"
```

## 服务器环境变量

如果在服务器上测试，替换 localhost：

```bash
# 使用服务器 IP
export BASE_URL="http://52.195.4.10:8080"

curl -X POST $BASE_URL/api/v1/internal/global/order/sync \
  -H "Content-Type: application/json" \
  -d '...'
```

## 健康检查

```bash
# 检查服务是否运行
curl http://localhost:8080/health

# 检查 Swagger UI
curl http://localhost:8080/swagger/

# 检查数据库连接
docker-compose exec postgres psql -U postgres -d rakutao -c "SELECT 1;"
```

## 故障排查

### 服务未响应

```bash
# 检查服务状态
docker-compose ps

# 查看日志
docker-compose logs gateway

# 重启服务
docker-compose restart gateway
```

### 路由 404

```bash
# 检查路由注册
docker-compose logs gateway | grep "global"
docker-compose logs gateway | grep "router"

# 查看 router.go 配置
cat backend/internal/api/router.go | grep -A 5 "GlobalOrderHandler"
```

### 数据库错误

```bash
# 检查表是否存在
docker-compose exec postgres psql -U postgres -d rakutao -c "\dt global*"

# 手动运行迁移
docker-compose exec gateway sh
psql $DATABASE_URL -f /app/migrations/007_create_global_order_tables.sql
```

---

**提示**：更多详细测试请使用自动化测试脚本：
- `bash test_global_order_sync.sh` - 完整测试套件
- `bash quick_test_global_order.sh` - 快速测试
- `bash verify_global_order_data.sh` - 数据库验证
