# 支付功能修复验证

## 🐛 问题描述

在对接国际版订单同步接口时，修改了 `OrderService.Pay` 方法使用 `PayOrderAtomic` 进行原子性支付事务，但由于类型断言和接口定义问题，导致支付功能完全失效。

## 🔍 问题定位过程

### 1. 检查代码变更
```bash
git log --oneline --since="2 days ago"
git show 355c1bb -- backend/internal/service/order_service.go
```

### 2. 发现根本原因
- `order_service.go` 中使用复杂的接口类型断言
- 接口定义中 `tx interface{}` 与实现 `tx pgx.Tx` 不匹配
- `OrderService.wallet` 是接口类型，但 `PayOrderAtomic` 需要具体类型

### 3. 关键代码问题

**修改前** (导致失败):
```go
type AtomicPaymentRepo interface {
    PayOrderAtomic(..., walletRepo interface {
        AdjustWithTx(ctx context.Context, tx interface{}, ...) (...)
    }) (...)
}

atomicRepo, ok := s.orders.(AtomicPaymentRepo)
if !ok {
    return nil, fmt.Errorf("order repository does not support atomic payment")
}
```

## ✅ 修复方案

### 1. 简化类型系统
```go
// order_repo.go
func (r *OrderRepo) PayOrderAtomic(
    ctx context.Context,
    orderNumber string,
    userID int64,
    amount int64,
    walletRepo *WalletRepo,  // 使用具体类型
) (*domain.WalletTransaction, error)
```

### 2. 添加 walletRepo 字段
```go
// order_service.go
type OrderService struct {
    products   ProductFetcher
    wallet     WalletService    // 接口（用于常规操作）
    walletRepo interface{}      // 具体实现（用于原子操作）
    orders     OrderRepository
    cart       CartRemover
}
```

### 3. 直接调用
```go
// 移除类型断言，直接调用
wtx, err := s.orders.PayOrderAtomic(ctx, orderNumber, userID, order.OrderInpriceJp, s.walletRepo)
```

## 📋 服务器部署验证

### 1. 等待 GitHub Actions 编译完成
```bash
# 访问查看编译状态
https://github.com/gongqianrong/global_version_ai/actions
```

### 2. 服务器部署
```bash
ssh ubuntu@your-server

cd ~/global_version_ai
git pull origin master

# 停止旧服务
sudo docker-compose down

# 启动新服务（使用新编译的二进制）
sudo docker-compose up -d

# 等待服务启动
sleep 60

# 查看日志
sudo docker-compose logs gateway | tail -50
```

### 3. 功能测试

#### a. 健康检查
```bash
curl http://localhost:8080/health
# 期望: {"status":"ok"}
```

#### b. 创建测试订单
```bash
# 1. 登录获取 token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123"}'

# 保存 token
TOKEN="<your-jwt-token>"

# 2. 查看钱包余额
curl -X GET http://localhost:8080/api/v1/wallet \
  -H "Authorization: Bearer $TOKEN"

# 3. 充值钱包（如果余额不足）
curl -X POST http://localhost:8080/api/v1/wallet/recharge \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount":10000,"payment_method":"test"}'

# 4. 创建订单（结算）
curl -X POST http://localhost:8080/api/v1/order/settlement \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"product_id": "surugaya:123456", "quantity": 1}
    ]
  }'

# 5. 确认订单
curl -X POST http://localhost:8080/api/v1/order/confirm \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"product_id": "surugaya:123456", "quantity": 1}
    ]
  }'

# 保存返回的 order_number
ORDER_NUMBER="<returned-order-number>"

# 6. **关键测试：支付订单**
curl -X POST http://localhost:8080/api/v1/order/pay \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"order_number":"'$ORDER_NUMBER'"}'

# 期望结果：
# {
#   "orderNumber": "RO20260818...",
#   "orderState": 3,
#   "paidAmount": 1500,
#   "balanceAfter": 8500
# }
```

#### c. 验证数据一致性
```bash
# 进入数据库
sudo docker-compose exec postgres psql -U postgres -d rakutao

# 1. 检查订单状态
SELECT order_number, order_state, order_inprice_jp, created_at 
FROM orders 
WHERE order_number = 'RO20260818...';
-- order_state 应该是 3 (Paid)

# 2. 检查钱包余额
SELECT user_id, balance, updated_at 
FROM wallets 
WHERE user_id = 1;
-- balance 应该正确扣减

# 3. 检查钱包交易记录
SELECT type, amount, balance_after, description, created_at 
FROM wallet_transactions 
WHERE user_id = 1 
ORDER BY created_at DESC 
LIMIT 5;
-- 应该有对应的支付扣款记录
```

## 🎯 测试检查清单

- [ ] GitHub Actions 编译成功
- [ ] 服务器部署无错误
- [ ] 健康检查通过
- [ ] 用户可以登录
- [ ] 订单创建成功
- [ ] **支付功能正常**（关键）
- [ ] 钱包余额正确扣减
- [ ] 数据库事务一致性
- [ ] 日志无异常错误

## 🛡️ 回滚方案（如果仍有问题）

```bash
# 回退到修复前的版本
cd ~/global_version_ai
git reset --hard a708ee0  # 修复前的 commit

sudo docker-compose down
sudo docker-compose up -d
```

## 📊 预期结果

修复后：
- ✅ 支付功能恢复正常
- ✅ 保持原子性事务特性（P0 修复）
- ✅ 钱包扣款和订单状态更新同步
- ✅ 无类型断言错误
- ✅ 代码更简洁、类型更安全

## 🔗 相关文档

- [修复 Commit](https://github.com/gongqianrong/global_version_ai/commit/95ba9a8)
- [P0 修复报告](./P0_FIX_REPORT.md)
- [部署指南](./QUICK_DEPLOYMENT.md)
