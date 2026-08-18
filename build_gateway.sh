#!/bin/bash
# 使用Docker编译Go代码，避免服务器OOM

set -e

echo "🔨 开始编译 Linux 版本的 gateway..."
echo ""

cd backend

# 使用Docker编译，避免本地需要Go环境
docker run --rm \
  -v "$(pwd)":/app \
  -w /app \
  golang:1.22-alpine \
  sh -c '
    echo "📦 安装依赖..."
    apk add --no-cache git ca-certificates tzdata
    
    echo "🔧 编译 gateway..."
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o gateway-linux ./cmd/gateway
    
    echo "✅ 编译完成！"
    ls -lh gateway-linux
  '

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ 编译成功！"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📦 二进制文件: backend/gateway-linux"
ls -lh gateway-linux
echo ""
echo "🚀 下一步: 提交到Git"
echo "   cd .."
echo "   git add backend/gateway-linux"
echo "   git commit -m 'build: 更新预编译的gateway-linux二进制'"
echo "   git push origin master"
echo ""
