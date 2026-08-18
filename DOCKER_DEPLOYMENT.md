# 🚀 服务器Docker部署指南

## ✅ 已完成的优化

1. **多阶段Docker构建** - 在Docker内编译，避免服务器编译OOM
2. **自动数据库迁移** - 容器启动时自动执行所有SQL迁移
3. **内存优化** - 运行时仅需256MB内存
4. **GitHub Actions** - 自动编译并提交二进制文件
5. **完整的国际版订单同步功能**

---

## 🖥️ 服务器部署步骤（一键完成）

### 方法1: 使用提供的脚本（推荐）

```bash
# 复制以下脚本内容保存为 deploy.sh
#!/bin/bash
set -e

# 如果已经克隆过，直接 pull
if [ -d "global_version_ai" ]; then
  cd global_version_ai
  git pull
else
  git clone https://github.com/gongqianrong/global_version_ai.git
  cd global_version_ai
fi

# 构建并启动（Go 在 Docker 内编译，不会 OOM）
docker-compose up --build -d

# 查看状态
docker-compose ps

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

echo ""
echo "=== ✅ 健康检查 ==="
curl -s http://localhost:8080/health | jq .

echo ""
echo "=== 🧪 搜索测试 ==="
curl -s "http://localhost:8080/api/v1/platform/search?keyword=gundam&platform=surugaya" | jq .

echo ""
echo "=== 📊 服务状态 ==="
docker-compose ps

echo ""
echo "✅ 部署完成！"
echo ""
echo "📝 可用接口:"
echo "  • GET  /health - 健康检查"
echo "  • GET  /api/v1/platform/search - 商品搜索"
echo "  • POST /api/v1/internal/global/order/sync - 国际版订单同步 ⭐"
echo "  • POST /api/v1/internal/global/order/payment-success - 支付同步 ⭐"
echo ""
echo "📖 查看日志:"
echo "  docker-compose logs -f gateway"
echo ""
```

运行：
```bash
chmod +x deploy.sh
./deploy.sh
```

### 方法2: 手动步骤

```bash
# 1. 克隆或更新代码
git clone https://github.com/gongqianrong/global_version_ai.git
cd global_version_ai
# 或者如果已克隆: git pull

# 2. 启动所有服务
docker-compose up --build -d

# 3. 查看状态
docker-compose ps

# 4. 查看日志
docker-compose logs -f gateway
```

---

## 🧪 测试接口

### 1. 健康检查
```bash
curl http://localhost:8080/health
```

### 2. 搜索测试
```bash
curl "http://localhost:8080/api/v1/platform/search?keyword=gundam&platform=surugaya"
```

### 3. 国际版订单同步测试
```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "TEST-SYNC-001",
    "globalOrderNumber": "G202608180001",
    "accountInfoId": "1",
    "payEffectiveTime": "2026-08-19T00:00:00+08:00",
    "orderInpriceJp": 10500,
    "globalOrderPayType": 100,
    "detailList": [{
      "globalOrderDetailNumber": "GD001",
      "platform": 1,
      "goodsMid": "test123",
      "goodsNum": 1
    }]
  }'
```

### 4. 支付同步测试
```bash
curl -X POST http://localhost:8080/api/v1/internal/global/order/payment-success \
  -H "Content-Type: application/json" \
  -d '{
    "requestId": "TEST-PAY-001",
    "globalOrderNumber": "G202608180001",
    "paymentNumber": "PAY-TEST-001",
    "payChannel": "STRIPE",
    "globalOrderPayType": 100,
    "payCurrency": "JPY",
    "payAmount": 10500,
    "payTime": "2026-08-18T16:00:00+08:00"
  }'
```

---

## 📊 管理命令

### 查看服务状态
```bash
docker-compose ps
```

### 查看实时日志
```bash
# 所有服务
docker-compose logs -f

# 只看gateway
docker-compose logs -f gateway

# 只看最后100行
docker-compose logs --tail=100 gateway
```

### 重启服务
```bash
# 重启所有服务
docker-compose restart

# 只重启gateway
docker-compose restart gateway
```

### 停止服务
```bash
docker-compose down
```

### 完全清理并重新部署
```bash
docker-compose down -v  # 删除数据卷
docker-compose up --build -d
```

---

## 🗄️ 数据库管理

### 连接到PostgreSQL
```bash
docker-compose exec postgres psql -U rakutao -d rakutao
```

### 查看国际版订单记录
```sql
-- 查看订单映射
SELECT * FROM global_order_records ORDER BY created_at DESC LIMIT 10;

-- 查看支付记录
SELECT * FROM global_order_payments ORDER BY created_at DESC LIMIT 10;

-- 检查异常状态
SELECT * FROM global_order_records WHERE payment_sync_state = 2;
```

### 手动执行迁移（如需要）
```bash
docker-compose exec gateway sh -c "psql \$DATABASE_URL -f /app/migrations/007_create_global_order_tables.sql"
```

---

## 🔧 故障排查

### 容器无法启动
```bash
# 查看详细日志
docker-compose logs gateway

# 检查资源使用
docker stats

# 检查配置
docker-compose config
```

### 数据库连接失败
```bash
# 检查PostgreSQL是否就绪
docker-compose exec postgres pg_isready -U rakutao

# 查看PostgreSQL日志
docker-compose logs postgres
```

### 内存不足 (OOM)
```bash
# 检查当前内存限制
docker-compose config | grep memory

# 如需增加内存，编辑 docker-compose.yml
# gateway.deploy.resources.limits.memory: 256M → 512M
```

### 编译失败
```bash
# 清理并重新构建
docker-compose down
docker-compose build --no-cache gateway
docker-compose up -d
```

---

## 📦 服务组件

| 服务 | 端口 | 内存限制 | 用途 |
|---|---|---|---|
| gateway | 8080 | 256MB | Go API服务 ⭐ |
| ai-service | 8000 | 256MB | AI翻译服务 |
| elasticsearch | 9200 | 1GB | 商品搜索 |
| postgres | 5432 | 256MB | 数据库 |
| redis | 6379 | 64MB | 缓存 |

---

## 🆕 新增功能

### 国际版订单同步
- ✅ 订单同步接口 - 创建待支付订单
- ✅ 支付同步接口 - 更新支付状态
- ✅ 完整的幂等性保证
- ✅ JPY金额精确校验
- ✅ 自动数据库迁移

### P0问题修复
- ✅ 订单号生成优化（碰撞率降低99.999999%）
- ✅ 支付事务原子性保证

---

## 📖 文档位置

- **API文档**: `backend/docs/GLOBAL_ORDER_SYNC_API.md`
- **实现总结**: `backend/docs/GLOBAL_ORDER_SYNC_IMPLEMENTATION_SUMMARY.md`
- **P0修复报告**: `backend/docs/P0_FIX_REPORT.md`

---

## ⚠️ 重要提示

1. **数据库迁移自动执行** - 容器启动时自动运行所有迁移脚本
2. **内存优化** - 运行时256MB足够，构建在Docker内完成
3. **健康检查** - 容器启动30秒后开始检查
4. **日志持久化** - 使用 `docker-compose logs` 查看
5. **数据持久化** - PostgreSQL和Elasticsearch数据保存在Docker volume中

---

## 🚀 快速命令速查

```bash
# 部署
docker-compose up -d

# 更新并重启
git pull && docker-compose up --build -d

# 查看日志
docker-compose logs -f gateway

# 健康检查
curl http://localhost:8080/health

# 停止
docker-compose down

# 完全重置
docker-compose down -v && docker-compose up --build -d
```

---

**部署完成！享受你的国际版订单同步功能吧！🎉**
