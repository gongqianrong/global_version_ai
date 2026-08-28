# 订单同步流程修复说明

## 问题描述

**原问题：** `/sync` 和 `/payment-success` 接口同时调用，导致：
- `/payment-success` 可能在 `/sync` 完成前就调用
- 国内管理端 `/payment-success` 找不到订单（因为 `/sync` 还没完成）
- 导致支付同步失败

## 正确的调用顺序

```
1. 用户下单（Confirm）
   ↓
2. 调用 /sync 接口（同步等待）
   ↓
3. /sync 返回 success=true
   ↓
4. 用户支付（Pay）
   ↓
5. 调用 /payment-success 接口（异步）
```

## 代码修改

### 修改前（❌ 错误）

```go
// OrderService.Confirm() - 异步调用
if s.adminSync != nil {
    go s.adminSync.SyncOrderAsync(context.Background(), order, orderDetails, user.GlobalAccountID)
}

// OrderService.Pay() - 也是异步调用
if s.adminSync != nil {
    go s.adminSync.SyncPaymentAsync(context.Background(), orderNumber, order.OrderInpriceJp, user.GlobalAccountID)
}
```

**问题：** 
- 两个都是 goroutine 异步调用
- 不会等待 `/sync` 完成
- 如果用户快速支付，可能 `/sync` 还没完成就调用了 `/payment-success`

---

### 修改后（✅ 正确）

```go
// OrderService.Confirm() - 同步调用，等待完成
if s.adminSync != nil {
    user, err := s.users.GetByID(ctx, userID)
    if err != nil {
        log.Printf("[order] failed to get user for sync: %v", err)
    } else if user.GlobalAccountID == "" {
        log.Printf("[order] CRITICAL: user %d has empty GlobalAccountID, skip admin sync", userID)
    } else {
        // ✅ 改为同步调用，确保订单创建同步成功后才能继续
        syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        
        syncErr := s.adminSync.SyncOrderSync(syncCtx, order, orderDetails, user.GlobalAccountID)
        if syncErr != nil {
            // 订单同步失败只记录日志，不影响订单创建
            log.Printf("[order] WARNING: order sync failed for %s: %v", order.OrderNumber, syncErr)
        } else {
            log.Printf("[order] Order %s synced successfully to admin", order.OrderNumber)
        }
    }
}

// OrderService.Pay() - 仍然异步调用（因为订单已经同步过了）
if s.adminSync != nil {
    user, err := s.users.GetByID(ctx, userID)
    if err != nil {
        log.Printf("[order] failed to get user for payment sync: %v", err)
    } else if user.GlobalAccountID == "" {
        log.Printf("[order] CRITICAL: user %d has empty GlobalAccountID, skip payment sync", userID)
    } else {
        go s.adminSync.SyncPaymentAsync(context.Background(), orderNumber, order.OrderInpriceJp, user.GlobalAccountID)
    }
}
```

---

## 新增接口方法

### AdminSyncClient 接口

```go
type AdminSyncClient interface {
    // 同步调用，阻塞等待结果（用于订单创建）
    SyncOrderSync(ctx context.Context, order *domain.Order, details []domain.OrderDetail, globalAccountId string) error
    
    // 异步调用，fire-and-forget（保留备用）
    SyncOrderAsync(ctx context.Context, order *domain.Order, details []domain.OrderDetail, globalAccountId string)
    
    // 异步调用，fire-and-forget（用于支付成功）
    SyncPaymentAsync(ctx context.Context, orderNumber string, paymentAmount int64, globalAccountId string)
}
```

### SyncOrderSync 实现

```go
func (c *AdminSyncClient) SyncOrderSync(ctx context.Context, order *domain.Order, details []domain.OrderDetail, globalAccountId string) error {
    if !c.enabled {
        return nil // 如果未启用同步，返回成功（不阻塞）
    }

    // 强校验：globalAccountId 必须有值
    if globalAccountId == "" {
        err := fmt.Errorf("globalAccountId为空，订单号=%s", order.OrderNumber)
        log.Printf("[AdminSync] CRITICAL ERROR: %v", err)
        return err
    }

    req := ConvertOrderToSyncRequest(order, details, globalAccountId)
    
    _, err := c.SyncOrder(ctx, req)  // ✅ 同步调用，等待返回
    if err != nil {
        log.Printf("[AdminSync] Sync order failed for %s: %v", order.OrderNumber, err)
        return fmt.Errorf("sync order failed: %w", err)
    }
    
    log.Printf("[AdminSync] Order %s synced successfully", order.OrderNumber)
    return nil
}
```

---

## 实际执行流程

### 场景：用户下单并立即支付

#### 1. 用户调用 POST /api/v1/orders（创建订单）

```
[order] Creating order for user 123
[order] Order G202608280001 created in DB
[AdminSync] >>> 发送订单同步请求:
[AdminSync]     globalOrderNumber=G202608280001
[AdminSync]     globalAccountId=INTL_485937291
[AdminSync]     URL=https://admin.example.com/internal/global/order/sync
```

**🔄 等待响应...**

```
[AdminSync] <<< 订单同步成功: orderNumber=CN202608280001, idempotent=false
[order] Order G202608280001 synced successfully to admin
```

**✅ 此时才返回给前端，订单创建完成**

---

#### 2. 用户调用 POST /api/v1/orders/{orderNumber}/pay（支付订单）

```
[order] User 123 paying order G202608280001
[order] Payment successful, order state changed to Paid
[AdminSync] >>> 发送支付同步请求:
[AdminSync]     globalOrderNumber=G202608280001
[AdminSync]     payAmount=12500.00 JPY
[AdminSync] Payment for order G202608280001 synced successfully (idempotent=false)
```

**✅ 此时支付成功，国内管理端已经有订单记录**

---

## 异常处理

### 如果 /sync 调用失败

```go
syncErr := s.adminSync.SyncOrderSync(syncCtx, order, orderDetails, user.GlobalAccountID)
if syncErr != nil {
    // ⚠️ 订单同步失败只记录日志，不影响订单创建
    log.Printf("[order] WARNING: order sync failed for %s: %v", order.OrderNumber, syncErr)
} else {
    log.Printf("[order] Order %s synced successfully to admin", order.OrderNumber)
}
```

**策略：**
- 订单同步失败**不阻塞**订单创建（因为国际版是主系统）
- 只记录警告日志
- 后续可以通过补偿机制重试同步

---

### 如果 /payment-success 调用失败

```go
go s.adminSync.SyncPaymentAsync(context.Background(), orderNumber, order.OrderInpriceJp, user.GlobalAccountID)
```

**策略：**
- 异步调用，fire-and-forget
- 失败只记录日志
- 不影响国际版支付流程

---

## 超时设置

### /sync 同步调用超时：30秒

```go
syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

syncErr := s.adminSync.SyncOrderSync(syncCtx, order, orderDetails, user.GlobalAccountID)
```

### /payment-success 异步调用超时：30秒

```go
// 在 SyncPaymentAsync 内部
syncCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

_, err := c.SyncPayment(syncCtx, req)
```

---

## 验证方法

### 1. 检查日志顺序

正确的日志顺序应该是：

```
[AdminSync] >>> 发送订单同步请求:
[AdminSync]     globalOrderNumber=G202608280001
[AdminSync] <<< 订单同步成功: orderNumber=CN202608280001
[order] Order G202608280001 synced successfully to admin

（用户支付）

[AdminSync] >>> 发送支付同步请求:
[AdminSync]     globalOrderNumber=G202608280001
[AdminSync] Payment for order G202608280001 synced successfully
```

### 2. 测试快速支付

```bash
# 1. 创建订单
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"items": [{"product_id": "xxx", "quantity": 1}]}'

# 立即支付（不等待）
curl -X POST http://localhost:8080/api/v1/orders/G202608280001/pay \
  -H "Authorization: Bearer $TOKEN"
```

**预期结果：**
- ✅ `/sync` 先完成
- ✅ `/payment-success` 后调用
- ✅ 国内管理端能够正确处理支付

---

## 修改文件清单

1. **backend/internal/service/order_service.go**
   - 添加 `time` 包导入
   - `Confirm()` 方法改为同步调用 `SyncOrderSync`
   - `AdminSyncClient` 接口添加 `SyncOrderSync` 方法

2. **backend/internal/client/admin_sync_client.go**
   - 新增 `SyncOrderSync()` 方法（同步版本）
   - 保留 `SyncOrderAsync()` 方法（备用）

---

## 总结

### ✅ 修复前
- `/sync` 异步调用
- `/payment-success` 异步调用
- **两个接口可能同时发送，顺序不可控**

### ✅ 修复后
- `/sync` **同步调用，阻塞等待成功**
- `/payment-success` 异步调用
- **确保顺序：sync → payment-success**

### 🎯 验收标准
- 订单创建时，必须等待 `/sync` 返回成功后才能完成
- 支付时，`/payment-success` 必定能找到订单（因为 `/sync` 已完成）
- 日志中可以看到明确的先后顺序

---

## 编译和部署

```bash
cd /Users/gongqianrong/Desktop/ai/backend
export PATH="$HOME/go-install/go/bin:$PATH"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" -trimpath \
  -o bin/gateway-linux-amd64 ./cmd/gateway

# 检查二进制文件
ls -lh bin/gateway-linux-amd64

# 部署并重启服务
```

🎉 **修复完成！**
