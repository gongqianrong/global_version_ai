#!/bin/bash
# 服务器快速部署脚本
# 在服务器上执行此脚本

set -e

echo "🚀 开始部署国际版订单同步功能..."
echo ""

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 1. 检查环境变量
echo "📋 步骤 1/5: 检查环境变量..."
if [ -z "$DATABASE_URL" ]; then
    echo -e "${RED}❌ 错误: DATABASE_URL 环境变量未设置${NC}"
    exit 1
fi
echo -e "${GREEN}✅ 环境变量检查通过${NC}"
echo ""

# 2. 拉取最新代码
echo "📦 步骤 2/5: 拉取最新代码..."
git pull origin master
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Git拉取失败${NC}"
    exit 1
fi
echo -e "${GREEN}✅ 代码更新成功 (Commit: 355c1bb)${NC}"
echo ""

# 3. 执行数据库迁移
echo "🗄️  步骤 3/5: 执行数据库迁移..."
cd backend
psql "$DATABASE_URL" < scripts/migrations/007_create_global_order_tables.sql
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ 数据库迁移失败${NC}"
    exit 1
fi
echo -e "${GREEN}✅ 数据库迁移完成${NC}"
echo ""

# 4. 验证表创建
echo "🔍 步骤 4/5: 验证数据库表..."
psql "$DATABASE_URL" -c "\d global_order_records" > /dev/null 2>&1
psql "$DATABASE_URL" -c "\d global_order_payments" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 表验证通过${NC}"
else
    echo -e "${RED}❌ 表验证失败${NC}"
    exit 1
fi
echo ""

# 5. 编译服务
echo "🔨 步骤 5/5: 编译服务..."
go build -o gateway ./cmd/gateway
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ 编译失败${NC}"
    exit 1
fi
echo -e "${GREEN}✅ 编译成功${NC}"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${GREEN}✅ 部署完成！${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${BLUE}📝 新增接口:${NC}"
echo "  • POST /api/v1/internal/global/order/sync"
echo "  • POST /api/v1/internal/global/order/payment-success"
echo ""
echo -e "${BLUE}🗄️  新增数据库表:${NC}"
echo "  • global_order_records (国际版订单映射)"
echo "  • global_order_payments (支付详情)"
echo ""
echo -e "${BLUE}🔧 下一步操作:${NC}"
echo "  1. 重启服务:"
echo "     sudo systemctl restart rakutao-gateway"
echo "     # 或者: ./gateway &"
echo ""
echo "  2. 测试接口:"
echo "     curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \\"
echo "       -H 'Content-Type: application/json' \\"
echo "       -d @docs/test_sync_order.json"
echo ""
echo "  3. 查看日志:"
echo "     tail -f /var/log/rakutao-gateway.log"
echo "     # 或者: journalctl -u rakutao-gateway -f"
echo ""
echo -e "${BLUE}📖 文档位置:${NC}"
echo "  • API文档: backend/docs/GLOBAL_ORDER_SYNC_API.md"
echo "  • 实现总结: backend/docs/GLOBAL_ORDER_SYNC_IMPLEMENTATION_SUMMARY.md"
echo ""
