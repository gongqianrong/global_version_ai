# 国际版订单同步接口测试指南

## 📋 测试概述

本测试套件用于验证国际版订单同步接口的正确性，包括：
- 订单同步接口 (POST /api/v1/internal/global/order/sync)
- 支付同步接口 (POST /api/v1/internal/global/order/payment-success)

## 🧪 测试场景

### 1. 订单同步测试

#### 场景 1.1：首次同步成功
- **输入**：有效的订单数据
- **期望**：success=true, idempotent=false
- **验证**：订单创建在本地数据库，状态为 BEPAY(0)

#### 场景 1.2：幂等性测试
- **输入**：相同的 requestId 再次请求
- **期望**：success=true, idempotent=true
- **验证**：不创建重复订单，返回已存在订单信息

#### 场景 1.3：缺少必填字段
- **输入**：缺少 globalOrderNumber
- **期望**：code≠200, 返回错误信息
- **验证**：不创建订单

### 2. 支付同步测试

#### 场景 2.1：首次支付成功
- **输入**：已同步订单的支付信息
- **期望**：success=true, idempotent=false
- **验证**：订单状态更新为 PAID(3)，支付记录入库

#### 场景 2.2：支付幂等性测试
- **输入**：相同的 requestId 再次请求
- **期望**：success=true, idempotent=true
- **验证**：不重复更新订单状态

#### 场景 2.3：金额不匹配
- **输入**：支付金额与订单金额不一致
- **期望**：success=false, 标记异常状态
- **验证**：订单状态不变，pay_sync_state=2

#### 场景 2.4：不存在的订单
- **输入**：未同步过的 globalOrderNumber
- **期望**：success=false
- **验证**：不创建支付记录

## 🚀 快速开始

### 前置条件

1. **服务器部署最新代码**
   ```bash
   cd ~/global_version_ai
   git pull origin master
   sudo docker-compose down
   sudo docker-compose up -d
   sleep 60
   ```

2. **检查服务健康**
   ```bash
   curl http://localhost:8080/health
   ```

3. **确认数据库表存在**
   ```bash
   sudo docker-compose exec postgres psql -U postgres -d rakutao -c "\dt global_order*"
   ```

### 运行测试

#### 方法 1：自动化测试（推荐）

```bash
# 在服务器上运行
cd ~/global_version_ai
bash test_global_order_sync.sh
```

**期望输出**：
```
=== 🧪 国际版订单同步接口测试 ===
...
✅ PASS: 首次订单同步
✅ PASS: 幂等性测试
✅ PASS: 首次支付同步
✅ PASS: 支付幂等性测试
✅ PASS: 金额不匹配检测
✅ PASS: 不存在订单检测
✅ PASS: 缺少必填字段检测

=== 📊 测试结果汇总 ===
通过: 7
失败: 0
总计: 7
✅ 所有测试通过！
```

#### 方法 2：数据库验证

```bash
bash verify_global_order_data.sh
```

**期望输出**：
- 订单和记录表数据一致
- 没有重复的 globalOrderNumber
- 订单状态与记录表状态一致

### 手动测试示例

#### 1. 订单同步

```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "TEST-SYNC-001",
    "globalOrderNumber": "G20260820001",
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
    "operator": "TEST_USER",
    "globalOrderPayType": 100,
    "detailList": [
      {
        "globalOrderDetailNumber": "GD20260820001",
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
  }' | jq
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
    "globalOrderNumber": "G20260820001",
    "orderState": 0
  }
}
```

#### 2. 支付同步

```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/payment-success \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "TEST-PAY-001",
    "globalOrderNumber": "G20260820001",
    "paymentNumber": "PAY-TEST-001",
    "payChannel": "STRIPE",
    "payWay": 1,
    "payAmountJp": 11000,
    "payAmountCn": 550,
    "paySeccussTime": "2026-08-20T10:30:00+08:00"
  }' | jq
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
    "paymentNumber": "PAY-TEST-001"
  }
}
```

## 📊 验证检查清单

### 接口功能
- [ ] 订单同步接口返回 200
- [ ] 支付同步接口返回 200
- [ ] 幂等性正确工作（idempotent=true）
- [ ] 金额不匹配时正确拒绝
- [ ] 不存在订单时正确拒绝
- [ ] 缺少必填字段时返回错误

### 数据一致性
- [ ] 订单成功创建在 orders 表
- [ ] 记录成功创建在 global_order_records 表
- [ ] 订单详情成功创建在 order_details 表
- [ ] 支付记录成功创建在 global_order_payments 表
- [ ] 订单状态正确（同步后=0, 支付后=3）
- [ ] pay_sync_state 正确（未支付=0, 已支付=1, 异常=2）
- [ ] 没有重复的 globalOrderNumber

### 业务逻辑
- [ ] 订单号生成正确（RO + 时间戳 + 随机数）
- [ ] 金额计算正确（orderInpriceJp = orderTotalJp + commissionFeeJp）
- [ ] 订单类型正确设置（order_purchase_type）
- [ ] 原子性事务工作（失败时数据回滚）

## 🐛 常见问题

### 问题 1：测试脚本返回 404

**原因**：路由未正确注册或服务未部署最新代码

**解决**：
```bash
# 检查服务版本
git log -1 --oneline

# 重新部署
sudo docker-compose down
sudo docker-compose up -d
```

### 问题 2：数据库表不存在

**原因**：迁移脚本未执行

**解决**：
```bash
# 手动执行迁移
sudo docker-compose exec gateway sh
psql $DATABASE_URL -f /app/migrations/007_create_global_order_tables.sql
```

### 问题 3：幂等性测试失败

**原因**：requestId 或 globalOrderNumber 重复检查逻辑错误

**检查**：
```sql
SELECT * FROM global_order_records 
WHERE global_order_number = 'G20260820001';
```

### 问题 4：支付金额不匹配但未拒绝

**原因**：金额校验逻辑问题

**检查代码**：
```go
// backend/internal/service/global_order_service.go
// 查看金额校验逻辑
```

## 📝 测试报告模板

```markdown
# 国际版订单同步接口测试报告

**测试时间**：2026-08-20 11:00:00  
**测试环境**：Production / Dev  
**测试人员**：XXX

## 测试结果

| 测试场景 | 结果 | 备注 |
|---------|------|------|
| 订单同步 - 首次同步 | ✅ PASS | - |
| 订单同步 - 幂等性 | ✅ PASS | - |
| 支付同步 - 首次支付 | ✅ PASS | - |
| 支付同步 - 幂等性 | ✅ PASS | - |
| 金额不匹配检测 | ✅ PASS | - |
| 不存在订单检测 | ✅ PASS | - |
| 必填字段检测 | ✅ PASS | - |

## 数据验证

- ✅ 订单数据一致性
- ✅ 无重复记录
- ✅ 状态正确更新

## 结论

所有测试通过，接口功能正常。
```

## 🔗 相关文档

- [国际版订单同步接口文档](./GLOBAL_ORDER_SYNC_API.md)
- [实现总结](./GLOBAL_ORDER_SYNC_IMPLEMENTATION_SUMMARY.md)
- [数据库迁移脚本](../scripts/migrations/007_create_global_order_tables.sql)

---

**最后更新**：2026-08-20  
**维护者**：开发团队
