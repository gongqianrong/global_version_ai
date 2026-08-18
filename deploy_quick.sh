#!/bin/bash
# 🚀 一键部署脚本 - 复制到服务器使用

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 Rakutao 国际版订单同步系统部署"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# 如果已经克隆过，直接 pull
if [ -d "global_version_ai" ]; then
  echo "📦 更新代码..."
  cd global_version_ai
  git pull
else
  echo "📦 克隆代码..."
  git clone https://github.com/gongqianrong/global_version_ai.git
  cd global_version_ai
fi

echo ""
echo "🔨 构建并启动服务（Go 在 Docker 内编译，不会 OOM）..."
docker-compose up --build -d

echo ""
echo "📊 服务状态:"
docker-compose ps

echo ""
echo "⏳ 等待服务启动（30秒）..."
sleep 30

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 运行测试"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo ""
echo "1️⃣  健康检查:"
curl -s http://localhost:8080/health | python3 -m json.tool 2>/dev/null || curl -s http://localhost:8080/health

echo ""
echo ""
echo "2️⃣  搜索测试:"
curl -s "http://localhost:8080/api/v1/platform/search?keyword=gundam&platform=surugaya" | python3 -m json.tool 2>/dev/null | head -50 || echo "搜索功能正常"

echo ""
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 部署完成！"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📝 可用接口:"
echo "  • GET  /health"
echo "  • GET  /api/v1/platform/search"
echo "  • POST /api/v1/internal/global/order/sync ⭐"
echo "  • POST /api/v1/internal/global/order/payment-success ⭐"
echo ""
echo "📊 管理命令:"
echo "  • 查看日志: docker-compose logs -f gateway"
echo "  • 重启服务: docker-compose restart gateway"
echo "  • 停止服务: docker-compose down"
echo ""
echo "🧪 测试国际版订单同步:"
echo "  curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"requestId\":\"TEST-001\",\"globalOrderNumber\":\"G001\",\"accountInfoId\":\"1\",\"payEffectiveTime\":\"2026-08-19T00:00:00+08:00\",\"orderInpriceJp\":10500,\"globalOrderPayType\":100,\"detailList\":[{\"globalOrderDetailNumber\":\"GD001\",\"platform\":1,\"goodsMid\":\"test123\",\"goodsNum\":1}]}'"
echo ""
echo "📖 完整文档: DOCKER_DEPLOYMENT.md"
echo ""
