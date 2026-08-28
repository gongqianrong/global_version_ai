# Go 编译配置和命令备忘

## Go 环境配置

```bash
export PATH="$HOME/go-install/go/bin:$PATH"
export GOPATH="$HOME/go"
export GOTOOLCHAIN=auto
```

## 编译命令

### 编译 Linux AMD64 版本

```bash
cd /Users/gongqianrong/Desktop/ai/backend

# 设置环境变量
export PATH="$HOME/go-install/go/bin:$PATH"
export GOPATH="$HOME/go"
export GOTOOLCHAIN=auto

# 编译
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" -trimpath \
  -o bin/gateway-linux-amd64 ./cmd/gateway
```

### 一键编译脚本

```bash
#!/bin/bash
cd /Users/gongqianrong/Desktop/ai/backend && \
export PATH="$HOME/go-install/go/bin:$PATH" && \
export GOPATH="$HOME/go" && \
export GOTOOLCHAIN=auto && \
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" -trimpath \
  -o bin/gateway-linux-amd64 ./cmd/gateway && \
echo "✅ 编译成功！" && \
ls -lh bin/gateway-linux-amd64
```

## 编译参数说明

| 参数 | 说明 |
|------|------|
| `CGO_ENABLED=0` | 禁用 CGO，编译纯静态二进制文件 |
| `GOOS=linux` | 目标操作系统为 Linux |
| `GOARCH=amd64` | 目标架构为 AMD64 (x86_64) |
| `-ldflags="-s -w"` | 去除调试信息和符号表，减小文件体积 |
| `-trimpath` | 移除文件路径信息，提高安全性 |

## 常用命令

### 检查 Go 版本
```bash
export PATH="$HOME/go-install/go/bin:$PATH"
go version
```

### 查看编译产物
```bash
ls -lh /Users/gongqianrong/Desktop/ai/backend/bin/
```

### 清理编译缓存
```bash
cd /Users/gongqianrong/Desktop/ai/backend
go clean -cache
```

### 检查依赖
```bash
cd /Users/gongqianrong/Desktop/ai/backend
go mod tidy
go mod verify
```

## 编译后操作

### 1. 本地测试（可选）
```bash
# 如果本地是 macOS，需要编译 macOS 版本测试
cd /Users/gongqianrong/Desktop/ai/backend
export PATH="$HOME/go-install/go/bin:$PATH"
go build -o bin/gateway-darwin-amd64 ./cmd/gateway
./bin/gateway-darwin-amd64
```

### 2. 提交到 Git
```bash
cd /Users/gongqianrong/Desktop/ai
git add backend/bin/gateway-linux-amd64
git commit -m "build: 更新编译产物"
git push origin master
```

### 3. 部署到服务器
```bash
# 使用项目中的部署脚本
cd /Users/gongqianrong/Desktop/ai
./deploy_to_server.sh
```

## 故障排查

### 编译错误：未找到 go 命令
```bash
# 确保设置了正确的 PATH
export PATH="$HOME/go-install/go/bin:$PATH"
go version
```

### 导入错误：未使用的导入
```bash
# 自动清理未使用的导入
cd /Users/gongqianrong/Desktop/ai/backend
gofmt -w ./internal/...
```

### 依赖错误
```bash
cd /Users/gongqianrong/Desktop/ai/backend
go mod download
go mod tidy
```

## 最近编译记录

- **2026-08-28**: 优化国际版订单同步接口（V1.2规范）
  - 提交 ID: `03e41cb`
  - 二进制大小: 20MB
  - Go 版本: go1.24.0
  - 主要变更: 
    - 移除 accountInfoId，改用 globalAccountId
    - 新增 CustomTime 支持自定义日期格式
    - 新增支付类型和金额一致性校验
    - 修复导入错误
