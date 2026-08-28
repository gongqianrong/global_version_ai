# 国际版订单同步接口实现文档（基于V1.2规范）

## 更新日期
2026-08-28

## 概述
本文档描述了基于《国际版订单同步接口文档_V1.2_参数校验修订.md》规范实现的订单同步功能。

## 主要变更（相对于之前版本）

### 1. 请求参数变更

#### 订单同步接口 (`/internal/global/order/sync`)

**变更项：**

| 变更 | 说明 |
|------|------|
| `globalAccountId` | ✅ 改为**必填**字段，类型从 `*string` 改为 `string` |
| ~~`accountInfoId`~~ | ❌ **移除**此字段，改由后端内部通过 `globalAccountId` 映射 |
| `goodsName` | ✅ 改为**必填**字段，类型从 `*string` 改为 `string` |
| `goodsUrl` | ✅ 改为**必填**字段，类型从 `*string` 改为 `string` |
| `discountType` | ✅ 改为**必填**字段，类型从 `*int` 改为 `int`，无折扣传 `0` |
| 日期字段 | ✅ 支持 `yyyy-MM-dd HH:mm:ss` 格式解析 |

#### 支付成功接口 (`/internal/global/order/payment-success`)

**变更项：**

| 变更 | 说明 |
|------|------|
| `globalOrderPayType` | ✅ **新增**必填字段，必须与订单同步时一致 |
| `payCurrency` | ✅ **新增**必填字段，实际支付币种（如JPY、USD） |
| `payAmount` | ✅ 统一字段名，替换原来的 `payAmountJp` |
| ~~`payAmountJp`~~ | ❌ **移除**，改用 `payAmount` + `payCurrency` |
| ~~`payAmountCn`~~ | ❌ **移除** |
| `payTime` | ✅ 字段名从 `paySeccussTime` 改为 `payTime`，支持 `yyyy-MM-dd HH:mm:ss` 格式 |

### 2. 参数校验增强

#### 订单同步校验规则

```go
✓ requestId                 不能为空
✓ globalOrderNumber         不能为空
✓ globalAccountId           不能为空（新增）
✓ globalOrderPayType        不能为空
✓ payEffectiveTime          不能为空
✓ detailList                不能为空

// 明细校验
✓ globalOrderDetailNumber   不能为空
✓ goodsMid                  不能为空
✓ goodsName                 不能为空（新增）
✓ goodsUrl                  不能为空（新增）
✓ platform                  不能为空
✓ discountType              自动校验（int类型）
```

#### 支付同步校验规则

```go
✓ requestId                 不能为空
✓ globalOrderNumber         不能为空
✓ paymentNumber             不能为空
✓ payChannel                不能为空
✓ globalOrderPayType        不能为空（新增）
✓ payCurrency               不能为空（新增）
✓ payAmount                 必须大于0
✓ payTime                   不能为空
```

### 3. 业务逻辑增强

#### 支付同步新增校验

1. **globalOrderPayType 一致性校验**
   ```
   订单同步时的 globalOrderPayType 必须与支付同步时一致
   不一致时标记为异常并返回失败
   ```

2. **JPY金额精确校验**
   ```
   当 payCurrency = "JPY" 时：
   - payAmount 必须与订单的 orderInpriceJp 精确匹配（误差为0）
   - 使用 BigDecimal 级别的精度比较
   不匹配时标记为异常并返回失败
   ```

3. **非JPY货币处理**
   ```
   当 payCurrency ≠ "JPY" 时：
   - 保存实际币种和金额用于对账
   - 不进行金额校验（由对账系统处理）
   ```

### 4. 数据库变更

#### global_order_records 表

```sql
-- global_account_id 改为 NOT NULL
global_account_id VARCHAR(255) NOT NULL

-- 新增索引
CREATE INDEX idx_global_order_records_global_account_id 
ON global_order_records(global_account_id);
```

#### global_order_payments 表

```sql
-- pay_currency 字段注释更新
COMMENT ON COLUMN global_order_payments.pay_currency 
IS '实际支付币种，例如JPY、USD';
```

### 5. 用户映射机制

由于移除了 `accountInfoId` 字段，新增了用户映射机制：

```go
// 通过 globalAccountId 映射本地 userID
// TODO: 应调用 mall-userms 服务进行映射/创建用户
// 当前使用简化的哈希映射（MVP阶段）
func hashGlobalAccountID(globalAccountID string) int64 {
    // 生成本地用户ID
    // 生产环境应替换为实际的用户服务调用
}
```

**后续优化方向：**
- 集成 mall-userms 服务进行真实用户映射
- 支持用户信息同步和更新
- 建立独立的用户映射表

## API端点

### 订单同步
```
POST /internal/global/order/sync
Content-Type: application/json
```

### 支付成功同步
```
POST /internal/global/order/payment-success
Content-Type: application/json
```

## 日期格式

所有日期时间字段**必须**使用以下格式：

```
yyyy-MM-dd HH:mm:ss
```

**正确示例：**
```json
{
  "orderAddtime": "2026-08-28 11:20:00",
  "payTime": "2026-08-28 11:35:00"
}
```

**错误示例（禁止）：**
```json
{
  "orderAddtime": "2026-08-28T11:20:00+08:00",  // ❌ ISO 8601格式
  "payTime": "2026-08-28T11:35:00Z"             // ❌ 带时区标识
}
```

## 完整请求示例

### 订单同步请求

```json
{
  "requestId": "GLOBAL-SYNC-20260828-0001",
  "globalOrderNumber": "G202608280001",
  "globalAccountId": "GU100001",
  "accountAddressId": "",
  "orderAddtime": "2026-08-28 11:20:00",
  "payEffectiveTime": "2026-08-28 12:20:00",
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
  "orderRemark": "international order",
  "operator": "SYSTEM_GLOBAL",
  "globalOrderPayType": 100,
  "detailList": [
    {
      "globalOrderDetailNumber": "GD20260828000101",
      "platform": 1,
      "goodsMid": "m123456789",
      "goodsImg": "https://example.com/item.jpg",
      "goodsName": "sample item",
      "goodsNum": 1,
      "goodsAmountJp": 10000,
      "goodsAmountCn": 500,
      "commissionFeeJp": 1000,
      "commissionFeeCn": 50,
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

### 支付成功请求

```json
{
  "requestId": "GLOBAL-PAY-20260828-0001",
  "globalOrderNumber": "G202608280001",
  "paymentNumber": "PAY-STRIPE-20260828-0001",
  "payChannel": "STRIPE",
  "globalOrderPayType": 100,
  "payCurrency": "JPY",
  "payAmount": 11000,
  "payTime": "2026-08-28 11:35:00",
  "operator": "SYSTEM_GLOBAL"
}
```

## 错误响应示例

### 参数校验失败

```json
{
  "code": 40002,
  "data": {
    "success": false,
    "idempotent": false,
    "message": "globalAccountId不能为空",
    "globalOrderNumber": "G202608280001",
    "orderInfoId": null,
    "orderNumber": null,
    "orderState": null
  }
}
```

### 金额不一致

```json
{
  "code": 0,
  "data": {
    "success": false,
    "idempotent": false,
    "message": "支付金额与订单金额不一致: 订单金额 110.00 JPY, 支付金额 100.00 JPY",
    "globalOrderNumber": "G202608280001",
    "paymentNumber": "PAY-STRIPE-20260828-0001",
    "orderInfoId": null,
    "orderNumber": null,
    "orderState": null
  }
}
```

### globalOrderPayType不一致

```json
{
  "code": 0,
  "data": {
    "success": false,
    "idempotent": false,
    "message": "globalOrderPayType与订单不一致: 订单为 100, 支付为 101",
    "globalOrderNumber": "G202608280001",
    "paymentNumber": "PAY-STRIPE-20260828-0001",
    "orderInfoId": null,
    "orderNumber": null,
    "orderState": null
  }
}
```

## 测试脚本

已提供完整的测试脚本：`test_global_order_sync_v1.2.sh`

**测试覆盖：**
1. ✅ 订单同步正常场景
2. ✅ 订单幂等性（重复requestId）
3. ✅ globalAccountId校验
4. ✅ goodsName校验
5. ✅ 支付同步正常场景
6. ✅ 支付幂等性
7. ✅ JPY金额一致性校验

**运行方式：**
```bash
./test_global_order_sync_v1.2.sh
```

## 实现文件

### Domain层
- `internal/domain/global_order.go` - 数据模型和自定义时间类型

### Service层
- `internal/service/global_order_service.go` - 业务逻辑和校验规则

### Repository层
- `internal/repo/global_order_repo.go` - 数据访问和用户映射

### API层
- `internal/api/global_order_handler.go` - HTTP处理器

### 数据库
- `scripts/migrations/007_create_global_order_tables.sql` - 表结构定义

## 待办事项

### 高优先级
- [ ] 集成 mall-userms 服务进行真实用户映射
- [ ] 添加单元测试覆盖
- [ ] 添加集成测试

### 中优先级
- [ ] 支持非JPY货币的金额校验（对账系统）
- [ ] 优化错误日志记录
- [ ] 添加监控指标

### 低优先级
- [ ] API文档自动生成（Swagger）
- [ ] 性能优化和压测
- [ ] 支持批量订单同步

## 兼容性说明

**不兼容变更：**
- ❌ 移除了 `accountInfoId` 字段
- ❌ 修改了支付请求的字段结构

**迁移指南：**
1. 所有调用方必须提供 `globalAccountId` 而不是 `accountInfoId`
2. 支付请求必须提供 `globalOrderPayType`、`payCurrency` 和 `payAmount`
3. 所有日期格式必须调整为 `yyyy-MM-dd HH:mm:ss`
4. 明细中的 `goodsName`、`goodsUrl`、`discountType` 必须显式提供

## 联系人

如有问题请联系开发团队。
