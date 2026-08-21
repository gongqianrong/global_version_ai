#!/bin/bash
# 快速诊断脚本 - 检查服务器状态

echo "=== 🔍 服务器状态诊断 ==="
echo ""

echo "1. 检查 Git 版本"
cd ~/global_version_ai
echo "当前 commit:"
git log -1 --oneline
echo ""
echo "最新 commit 应该是:"
echo "d6d4a87 fix: 修复Swagger UI使用本地资源而非CDN"
echo ""

echo "2. 检查容器状态"
sudo docker-compose ps
echo ""

echo "3. 检查 Gateway 日志（最后30行）"
sudo docker-compose logs gateway | tail -30
echo ""

echo "4. 测试健康检查"
curl -s http://localhost:8080/health | jq
echo ""

echo "5. 测试公开接口（搜索）"
curl -s "http://localhost:8080/api/v1/platform/search?keyword=test&platform=surugaya" | head -100
echo ""

echo "6. 测试 Swagger"
echo "Swagger OpenAPI 访问测试:"
curl -s -I http://localhost:8080/swagger/openapi.yaml | head -5
echo ""

echo "7. 检查路由注册（查看日志中的路由）"
sudo docker-compose logs gateway 2>&1 | grep -i "route\|handler\|router" | tail -20
