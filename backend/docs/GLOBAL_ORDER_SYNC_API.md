# 国际版订单同步接口

## 概述

本文档描述国际版订单同步接口的实现，基于 `国际版订单同步接口文档_V1.1_20260818.md`。

## 接口端点

### 1. 订单同步接口

**端点:** `POST /api/v1/internal/global/order/sync`

**用途:** 将国际版订单同步到本地系统，创建待支付订单。

**请求示例:**

```json
{
  "requestId": "GLOBAL-SYNC-20260818-0001",
  "globalOrderNumber": "G202608180001",
  "globalAccountId": "GU100001",
  "accountInfoId": "100001",
  "accountAddressId": "ADDR100001",
  "orderAddtime": "2026-08-18T13:20:00+08:00",
  "payEffectiveTime": "2026-08-18T14:20:00+08:00",
  "orderTotalJp": 10000,
  "orderTotalCn": 500,
  "commissionFeeJp": 500,
  "commissionFeeCn": 25,
  "handlingFeeJp": 0,
  "handlingFeeCn": 0,
  "orderInpriceJp": 10500,
  "orderInpriceCn": 525,
  "orderRate": 0.05,
  "totalShippingFee": 0,
  "totalShippingFeeCn": 0,
  "orderType": 1,
  "orderPurchaseType": 1,
  "orderMode": 1,
  "orderRemark": "international order",
  "operator": "SYSTEM_GLOBAL",
  "globalOrderPayType": 100,
  "detailList": [
    {
      "globalOrderDetailNumber": "GD20260818000101",
      "platform": 1,
      "goodsMid": "m123456789",
      "goodsImg": "https://example.com/item.jpg",
      "goodsName": "sample item",
      "goodsNum": 1,
      "goodsAmountJp": 10000,
      "goodsAmountCn": 500,
      "commissionFeeJp": 500,
      "commissionFeeCn": 25,
      "handlingFeeJp": 0,
      "handlingFeeCn": 0,
      "goodsUrl": "https://example.com/item/123",
      "sellerId": "seller001",
      "shippingFeeJp": 0,
      "shippingFeeCn": 0,
      "orderPurchaseType": 1,
      "purchaseDirect": 0,
      "discountType": 0
    }
  ]
}
```

**响应示例:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "idempotent": false,
    "message": "国际版订单同步成功",
    "orderInfoId": "12345",
    "orderNumber": "RO20260818140500123456ABCD1234",
    "globalOrderNumber": "G202608180001",
    "orderState": 0
  }
}
```

### 2. 支付成功同步接口

**端点:** `POST /api/v1/internal/global/order/payment-success`

**用途:** 将已同步订单从待支付更新为已支付。

**请求示例:**

```json
{
  "requestId": "GLOBAL-PAY-20260818-0001",
  "globalOrderNumber": "G202608180001",
  "paymentNumber": "PAY-STRIPE-20260818-0001",
  "payChannel": "STRIPE",
  "globalOrderPayType": 100,
  "payCurrency": "JPY",
  "payAmount": 10500,
  "payTime": "2026-08-18T13:35:00+08:00",
  "operator": "SYSTEM_GLOBAL"
}
```

**响应示例:**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "success": true,
    "idempotent": false,
    "message": "国际版支付状态同步成功",
    "orderInfoId": "12345",
    "orderNumber": "RO20260818140500123456ABCD1234",
    "globalOrderNumber": "G202608180001",
    "paymentNumber": "PAY-STRIPE-20260818-0001",
    "orderState": 1
  }
}
```

## 幂等性

### 订单同步幂等

1. 根据 `requestId` 检查是否已存在
2. 根据 `globalOrderNumber` 检查是否已存在
3. 如果已存在，返回历史订单信息，`idempotent=true`

### 支付同步幂等

1. 根据 `paymentNumber` + 订单检查是否已存在
2. 如果已存在且相同，返回历史支付信息，`idempotent=true`
3. 如果已存在但不同，返回错误

## 校验规则

### 订单同步校验

- `requestId` 必填
- `globalOrderNumber` 必填
- `accountInfoId` 必填
- `globalOrderPayType` 必填
- `detailList` 必填，至少1条
- `payEffectiveTime` 必填

### 支付同步校验

- `requestId` 必填
- `globalOrderNumber` 必填
- `paymentNumber` 必填
- `payChannel` 必填
- `globalOrderPayType` 必填，且必须与订单同步时一致
- `payCurrency` 必填
- `payAmount` 必填，且必须大于0
- `payTime` 必填
- JPY币种时，金额必须与订单 `orderInpriceJp` 精确一致

## 错误处理

### 业务失败场景

调用方必须检查 `data.success` 字段，不能只检查外层 `code`。

常见业务失败场景：

1. **订单未同步:** `国际版订单尚未同步，请先调用订单同步接口`
2. **支付类型不匹配:** `globalOrderPayType与订单不一致`
3. **金额不匹配:** `支付金额与订单金额不一致`
4. **订单状态异常:** `订单状态不是待支付`
5. **支付流水号冲突:** `支付流水号与已有记录不一致`

## 数据库设计

### global_order_records 表

存储国际版订单与本地订单的映射关系。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGSERIAL | 主键 |
| request_id | VARCHAR(255) | 同步请求ID（唯一） |
| global_order_number | VARCHAR(255) | 国际版订单号（唯一） |
| global_account_id | VARCHAR(255) | 国际版用户ID |
| order_id | BIGINT | 本地订单ID |
| order_number | VARCHAR(255) | 本地订单号 |
| global_order_pay_type | INTEGER | 支付类型 |
| payment_sync_state | INTEGER | 支付同步状态（0=未支付，1=已支付，2=异常） |
| payment_number | VARCHAR(255) | 支付流水号 |
| payment_request_id | VARCHAR(255) | 支付请求ID |
| sync_time | TIMESTAMP | 同步时间 |
| payment_sync_time | TIMESTAMP | 支付同步时间 |

### global_order_payments 表

存储国际版订单的支付详情。

| 字段 | 类型 | 说明 |
|---|---|---|
| id | BIGSERIAL | 主键 |
| order_id | BIGINT | 订单ID |
| payment_number | VARCHAR(255) | 支付流水号（唯一） |
| pay_channel | VARCHAR(50) | 支付渠道 |
| pay_currency | VARCHAR(10) | 支付币种 |
| pay_amount | BIGINT | 支付金额（分） |
| pay_time | TIMESTAMP | 支付时间 |
| operator | VARCHAR(255) | 操作人 |

## 部署步骤

1. **执行数据库迁移:**
   ```bash
   psql $DATABASE_URL < scripts/migrations/007_create_global_order_tables.sql
   ```

2. **重启服务:**
   ```bash
   make build
   ./gateway
   ```

3. **测试接口:**
   ```bash
   # 订单同步
   curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \
     -H "Content-Type: application/json" \
     -d @test_sync_order.json

   # 支付同步
   curl -X POST http://localhost:8080/api/v1/internal/global/order/payment-success \
     -H "Content-Type: application/json" \
     -d @test_payment_success.json
   ```

## 注意事项

1. **必须先同步订单，再同步支付**
2. **支付接口调用方必须检查 `data.success`**
3. **重复调用必须保持 `requestId` 和业务内容一致**
4. **JPY支付金额必须与订单 `orderInpriceJp` 精确一致**
5. **这是内部接口，不需要JWT认证，但需要网络层面的安全防护**

## 测试用例

参见 `internal/service/global_order_service_test.go` 和 `internal/api/global_order_handler_test.go`
