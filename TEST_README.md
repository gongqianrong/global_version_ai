# 国际版订单同步测试说明

## 📦 测试文件

我已经为您创建了以下测试工具和文档：

### 🧪 测试脚本

1. **test_global_order_sync.sh** - 完整自动化测试
   - 7个测试场景（订单同步、支付同步、幂等性、异常处理）
   - 自动验证结果
   - 彩色输出
   
2. **quick_test_global_order.sh** - 快速测试
   - 健康检查
   - 基本订单同步测试
   - 基本支付同步测试
   
3. **verify_global_order_data.sh** - 数据库验证
   - 数据一致性检查
   - 幂等性验证
   - 状态统计

### 📖 测试文档

1. **GLOBAL_ORDER_SYNC_TEST_GUIDE.md** - 详细测试指南
   - 测试场景说明
   - 手动测试步骤
   - 数据验证方法
   
2. **GLOBAL_ORDER_SYNC_CURL_TESTS.md** - Curl 命令集合
   - 复制即用的测试命令
   - 各种测试场景
   - 数据库查询命令
   
3. **GLOBAL_ORDER_SYNC_TEST_REPORT.md** - 测试报告模板
   - 测试结果记录
   - 故障排查指南
   - 已知问题跟踪

## 🚀 快速开始

### 在服务器上测试（推荐）

```bash
# 1. 进入项目目录
cd ~/global_version_ai

# 2. 拉取最新测试脚本
git pull origin master

# 3. 运行快速测试（1分钟）
bash quick_test_global_order.sh

# 4. 运行完整测试（2-3分钟）
bash test_global_order_sync.sh

# 5. 验证数据库
bash verify_global_order_data.sh
```

### 手动测试单个接口

#### 订单同步

```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "TEST-001",
    "globalOrderNumber": "GTEST001",
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
    "detailList": [{
      "globalOrderDetailNumber": "GTESTD001",
      "platform": 1,
      "goodsMid": "test_001",
      "goodsImg": "https://example.com/test.jpg",
      "goodsName": "测试商品",
      "goodsNum": 1,
      "goodsAmountJp": 10000,
      "goodsAmountCn": 500,
      "commissionFeeJp": 1000,
      "commissionFeeCn": 50,
      "handlingFeeJp": 0,
      "handlingFeeCn": 0,
      "goodsUrl": "https://example.com/test",
      "sellerId": "test_seller",
      "shippingFeeJp": 0,
      "shippingFeeCn": 0,
      "orderPurchaseType": 1,
      "purchaseDirect": 0,
      "discountType": 0
    }]
  }'
```

#### 支付同步

```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/payment-success \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "PAY-001",
    "globalOrderNumber": "GTEST001",
    "paymentNumber": "PAY001",
    "payChannel": "STRIPE",
    "payWay": 1,
    "payAmountJp": 11000,
    "payAmountCn": 550,
    "paySeccussTime": "2026-08-20T10:30:00+08:00"
  }'
```

## ✅ 期望结果

### 订单同步成功

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "success": true,
    "idempotent": false,
    "message": "国际版订单同步成功",
    "orderInfoId": "12345",
    "orderNumber": "RO20260820123456",
    "globalOrderNumber": "GTEST001",
    "orderState": 0
  }
}
```

### 支付同步成功

```json
{
  "code": 200,
  "message": "success",
  "data": {
    "success": true,
    "idempotent": false,
    "message": "支付同步成功",
    "orderNumber": "RO20260820123456",
    "orderState": 3,
    "paymentNumber": "PAY001"
  }
}
```

## 🎯 测试重点

- ✅ **功能正确性**：订单和支付能成功同步
- ✅ **幂等性**：相同 requestId 不会重复创建
- ✅ **数据一致性**：数据库记录正确
- ✅ **异常处理**：错误输入被正确拒绝
- ✅ **金额验证**：支付金额必须匹配订单金额

## 🐛 常见问题

### Q1: 测试脚本返回 404？
**A**: 服务可能未部署最新代码，运行 `git pull && sudo docker-compose restart gateway`

### Q2: 数据库表不存在？
**A**: 迁移未执行，运行 `docker-compose restart gateway`（会自动迁移）

### Q3: 幂等性测试失败？
**A**: 检查数据库是否有重复记录，可能是事务问题

### Q4: 本地无法测试？
**A**: 可以在服务器上测试，或者确保本地 Docker 环境正确运行

## 📞 需要帮助？

如果测试遇到问题：

1. 查看服务日志：`docker-compose logs gateway | tail -100`
2. 查看数据库状态：`docker-compose exec postgres psql -U postgres -d rakutao -c "\dt"`
3. 检查路由注册：`cat backend/internal/api/router.go | grep Global`

## 📚 更多信息

- 完整接口文档：`docs/GLOBAL_ORDER_SYNC_API.md`
- 详细测试指南：`docs/GLOBAL_ORDER_SYNC_TEST_GUIDE.md`
- Curl 命令集合：`docs/GLOBAL_ORDER_SYNC_CURL_TESTS.md`

---

**快速检查命令**：

```bash
# 检查服务健康
curl http://localhost:8080/health

# 查看最近订单
docker-compose exec postgres psql -U postgres -d rakutao -c \
  "SELECT * FROM global_order_records ORDER BY created_at DESC LIMIT 5;"

# 运行完整测试
bash test_global_order_sync.sh
```
