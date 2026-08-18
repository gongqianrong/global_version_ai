#!/bin/bash
# 国际版订单同步功能部署脚本

set -e

echo "🚀 开始部署国际版订单同步功能..."

# 1. 检查数据库连接
echo "📊 检查数据库连接..."
if [ -z "$DATABASE_URL" ]; then
  echo "❌ 错误: DATABASE_URL 环境变量未设置"
  exit 1
fi

# 2. 执行数据库迁移
echo "📦 执行数据库迁移..."
psql "$DATABASE_URL" < scripts/migrations/007_create_global_order_tables.sql
echo "✅ 数据库迁移完成"

# 3. 验证表创建
echo "🔍 验证表创建..."
psql "$DATABASE_URL" -c "\d global_order_records" > /dev/null
psql "$DATABASE_URL" -c "\d global_order_payments" > /dev/null
echo "✅ 表验证通过"

# 4. 编译服务
echo "🔨 编译服务..."
go build -o gateway ./cmd/gateway
echo "✅ 编译完成"

# 5. 运行测试（可选）
if [ "$RUN_TESTS" = "true" ]; then
  echo "🧪 运行测试..."
  go test ./internal/service/... -v
  go test ./internal/api/... -v
  echo "✅ 测试通过"
fi

echo ""
echo "✅ 部署完成！"
echo ""
echo "📝 接口端点:"
echo "  - POST /api/v1/internal/global/order/sync"
echo "  - POST /api/v1/internal/global/order/payment-success"
echo ""
echo "📖 文档:"
echo "  - API 文档: docs/GLOBAL_ORDER_SYNC_API.md"
echo "  - 实现总结: docs/GLOBAL_ORDER_SYNC_IMPLEMENTATION_SUMMARY.md"
echo ""
echo "🧪 测试数据:"
echo "  - 订单同步: docs/test_sync_order.json"
echo "  - 支付同步: docs/test_payment_success.json"
echo ""
echo "🎯 快速测试:"
echo "  curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d @docs/test_sync_order.json"
echo ""
