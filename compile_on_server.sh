#!/bin/bash
# 在服务器上编译二进制并保存到本地仓库

echo "=== 🔨 在服务器上编译 Gateway 二进制 ==="
echo ""

cd ~/global_version_ai

echo "1. 使用 Docker 编译二进制..."
sudo docker run --rm \
  -v "$(pwd)/backend:/app" \
  -w /app \
  golang:1.23-alpine \
  sh -c "
    apk add --no-cache git ca-certificates tzdata && \
    mkdir -p bin && \
    export GOTOOLCHAIN=auto && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      go build -ldflags='-s -w' -trimpath \
      -o bin/gateway-linux-amd64 ./cmd/gateway && \
    ls -lh bin/gateway-linux-amd64 && \
    echo 'Binary size:' && du -h bin/gateway-linux-amd64 | cut -f1
  "

echo ""
echo "2. 检查编译结果..."
if [ -f "backend/bin/gateway-linux-amd64" ]; then
    echo "✅ 二进制文件编译成功"
    ls -lh backend/bin/gateway-linux-amd64
    file backend/bin/gateway-linux-amd64
else
    echo "❌ 编译失败"
    exit 1
fi

echo ""
echo "3. 提交到 Git..."
git add backend/bin/gateway-linux-amd64
BINARY_SIZE=$(du -h backend/bin/gateway-linux-amd64 | cut -f1)
git commit -m "build: 添加预编译的 gateway 二进制文件 (${BINARY_SIZE})"

echo ""
echo "4. 推送到远程仓库..."
git push origin master

echo ""
echo "5. 修改 docker-compose.yml 使用预编译模式..."
sed -i 's/dockerfile: Dockerfile$/dockerfile: Dockerfile.prebuilt/' docker-compose.yml

echo ""
echo "6. 重新部署..."
sudo docker-compose down
sudo docker-compose up -d

echo ""
echo "7. 等待服务启动（30秒）..."
sleep 30

echo ""
echo "8. 测试接口..."
curl -I http://localhost:8080/api/v1/internal/global/order/sync

echo ""
echo "=== ✅ 完成 ==="
