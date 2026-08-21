#!/bin/bash
# 服务器部署脚本 - 确保包含国际版订单同步功能

echo "=== 🚀 部署国际版订单同步功能到服务器 ==="
echo ""

SERVER_IP="${1:-52.195.4.10}"
echo "目标服务器: $SERVER_IP"
echo ""

echo "步骤 1: SSH 到服务器并拉取最新代码"
echo "----------------------------------------"
cat << 'EOF'
ssh ubuntu@52.195.4.10 << 'ENDSSH'
cd ~/global_version_ai
echo "当前分支:"
git branch
echo ""
echo "拉取最新代码..."
git pull origin master
echo ""
echo "检查关键提交是否存在:"
git log --oneline | grep -E "355c1bb|国际版订单同步" || echo "❌ 未找到国际版订单同步提交"
echo ""
echo "检查关键文件是否存在:"
ls -lh backend/internal/api/global_order_handler.go 2>/dev/null && echo "✅ global_order_handler.go 存在" || echo "❌ global_order_handler.go 不存在"
ls -lh backend/internal/service/global_order_service.go 2>/dev/null && echo "✅ global_order_service.go 存在" || echo "❌ global_order_service.go 不存在"
ENDSSH
EOF

echo ""
echo "步骤 2: 重新构建和部署"
echo "----------------------------------------"
cat << 'EOF'
ssh ubuntu@52.195.4.10 << 'ENDSSH'
cd ~/global_version_ai
echo "停止现有容器..."
sudo docker-compose down
echo ""
echo "清理旧镜像（可选）..."
sudo docker system prune -f
echo ""
echo "重新构建镜像..."
sudo docker-compose build --no-cache gateway
echo ""
echo "启动服务..."
sudo docker-compose up -d
echo ""
echo "等待服务启动..."
sleep 30
echo ""
echo "检查服务状态..."
sudo docker-compose ps
echo ""
echo "查看最新日志..."
sudo docker-compose logs gateway | tail -50
ENDSSH
EOF

echo ""
echo "步骤 3: 验证部署"
echo "----------------------------------------"
echo "执行以下命令测试接口:"
echo ""
echo "curl -s http://${SERVER_IP}:8080/health | grep -q 'status' && echo '✅ 服务运行正常'"
echo ""
echo "curl -s -I http://${SERVER_IP}:8080/api/v1/internal/global/order/sync | head -1"
echo ""

echo "=== 📝 手动执行步骤 ==="
echo ""
echo "1. SSH 到服务器:"
echo "   ssh ubuntu@${SERVER_IP}"
echo ""
echo "2. 进入项目目录:"
echo "   cd ~/global_version_ai"
echo ""
echo "3. 拉取最新代码:"
echo "   git pull origin master"
echo ""
echo "4. 检查关键提交:"
echo "   git log --oneline | head -20"
echo "   # 应该看到 355c1bb 提交"
echo ""
echo "5. 重新构建和部署:"
echo "   sudo docker-compose down"
echo "   sudo docker-compose build --no-cache gateway"
echo "   sudo docker-compose up -d"
echo ""
echo "6. 查看日志:"
echo "   sudo docker-compose logs gateway | tail -100"
echo ""
echo "7. 测试接口:"
echo "   bash test_global_order_sync.sh"
echo ""
