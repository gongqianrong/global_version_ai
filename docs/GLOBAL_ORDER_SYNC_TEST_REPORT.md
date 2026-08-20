# 国际版订单同步接口测试报告

## 📋 测试概要

**测试对象**：国际版订单同步接口  
**测试版本**：V1.1  
**接口路径**：
- POST /api/v1/internal/global/order/sync
- POST /api/v1/internal/global/order/payment-success

## 🎯 测试目标

1. 验证订单同步接口功能正确性
2. 验证支付同步接口功能正确性
3. 验证幂等性机制
4. 验证数据一致性
5. 验证异常处理

## 📝 测试用例

| ID | 测试场景 | 输入 | 期望输出 | 优先级 |
|----|---------|------|---------|--------|
| TC01 | 订单首次同步 | 有效订单数据 | success=true, idempotent=false | P0 |
| TC02 | 订单幂等性 | 相同 requestId | success=true, idempotent=true | P0 |
| TC03 | 订单缺少必填字段 | 空 globalOrderNumber | code≠200, 错误信息 | P1 |
| TC04 | 支付首次同步 | 有效支付数据 | success=true, 订单状态=3 | P0 |
| TC05 | 支付幂等性 | 相同 requestId | success=true, idempotent=true | P0 |
| TC06 | 支付金额不匹配 | 错误的金额 | success=false, 状态=异常 | P0 |
| TC07 | 不存在订单支付 | 无效 globalOrderNumber | success=false | P1 |

## 🛠️ 测试工具

### 自动化测试脚本

1. **完整测试套件**
   ```bash
   bash test_global_order_sync.sh
   ```
   - 7个测试场景
   - 自动验证结果
   - 生成测试报告

2. **快速测试**
   ```bash
   bash quick_test_global_order.sh
   ```
   - 基本功能验证
   - 快速冒烟测试

3. **数据库验证**
   ```bash
   bash verify_global_order_data.sh
   ```
   - 数据一致性检查
   - 幂等性验证
   - 状态验证

### 手动测试

参考 [GLOBAL_ORDER_SYNC_CURL_TESTS.md](./GLOBAL_ORDER_SYNC_CURL_TESTS.md)

## 🚀 执行步骤

### 前置条件

1. ✅ 服务器部署最新代码
2. ✅ 数据库迁移完成
3. ✅ 服务正常运行
4. ✅ 网络连接正常

### 测试步骤

**步骤 1：部署最新代码**

```bash
# 在服务器上
cd ~/global_version_ai
git pull origin master
sudo docker-compose down
sudo docker-compose up -d
sleep 60
```

**步骤 2：健康检查**

```bash
curl http://localhost:8080/health
```

**步骤 3：运行自动化测试**

```bash
cd ~/global_version_ai
bash test_global_order_sync.sh
```

**步骤 4：验证数据库**

```bash
bash verify_global_order_data.sh
```

**步骤 5：记录测试结果**

将测试输出保存到文件：

```bash
bash test_global_order_sync.sh > test_results_$(date +%Y%m%d_%H%M%S).log 2>&1
```

## 📊 测试结果模板

```
=== 测试结果 ===

测试时间: 2026-08-20 11:00:00
测试环境: Production / Dev
服务器: 52.195.4.10

测试用例执行结果:
------------------------
TC01: ✅ PASS - 订单首次同步
TC02: ✅ PASS - 订单幂等性
TC03: ✅ PASS - 订单缺少必填字段
TC04: ✅ PASS - 支付首次同步
TC05: ✅ PASS - 支付幂等性
TC06: ✅ PASS - 支付金额不匹配
TC07: ✅ PASS - 不存在订单支付

总计: 7 / 通过: 7 / 失败: 0
成功率: 100%

数据一致性验证:
------------------------
✅ 订单数据一致
✅ 无重复记录
✅ 状态正确更新
✅ 金额计算正确

性能指标:
------------------------
订单同步平均响应时间: < 200ms
支付同步平均响应时间: < 150ms
数据库事务成功率: 100%

结论:
------------------------
✅ 所有测试通过
✅ 接口功能正常
✅ 数据一致性良好
✅ 可以上线使用
```

## ⚠️ 已知问题

目前无已知问题。

## 🔧 故障排查指南

### 问题 1：测试脚本返回 404

**症状**：curl 请求返回 404 Not Found

**可能原因**：
- 路由未正确注册
- 服务未部署最新代码
- URL 路径错误

**解决方法**：
```bash
# 1. 检查代码版本
git log -1 --oneline

# 2. 查看路由日志
docker-compose logs gateway | grep "global"

# 3. 重新部署
sudo docker-compose down
sudo docker-compose up -d
```

### 问题 2：数据库表不存在

**症状**：relation "global_order_records" does not exist

**解决方法**：
```bash
# 检查表
docker-compose exec postgres psql -U postgres -d rakutao -c "\dt global*"

# 如果不存在，手动运行迁移
docker-compose exec gateway sh
psql $DATABASE_URL -f /app/migrations/007_create_global_order_tables.sql
```

### 问题 3：幂等性测试失败

**症状**：相同 requestId 返回 idempotent=false

**检查步骤**：
```bash
# 查询记录
docker-compose exec postgres psql -U postgres -d rakutao -c "
SELECT * FROM global_order_records 
ORDER BY created_at DESC 
LIMIT 5;
"
```

**可能原因**：
- 数据库查询逻辑错误
- requestId 未正确存储
- 事务回滚导致记录丢失

### 问题 4：金额验证未生效

**症状**：错误金额仍然同步成功

**检查代码**：
```bash
# 查看服务层逻辑
cat backend/internal/service/global_order_service.go | grep -A 20 "ValidatePaymentAmount"
```

## 📚 相关文档

- [接口文档](./GLOBAL_ORDER_SYNC_API.md)
- [测试指南](./GLOBAL_ORDER_SYNC_TEST_GUIDE.md)
- [Curl 测试命令](./GLOBAL_ORDER_SYNC_CURL_TESTS.md)
- [实现总结](./GLOBAL_ORDER_SYNC_IMPLEMENTATION_SUMMARY.md)

## 📈 后续计划

### 短期

- [x] 完成基本功能测试
- [ ] 性能压力测试
- [ ] 并发测试
- [ ] 边界值测试

### 长期

- [ ] 集成到 CI/CD 流程
- [ ] 监控和告警
- [ ] 日志分析
- [ ] 性能优化

## ✍️ 测试签名

**测试人员**：_____________  
**审核人员**：_____________  
**日期**：_____________

---

**版本历史**：
- V1.0 - 2026-08-20 - 初始版本
