# AI Deployment Guide - 技术参考文档

**目标受众**：AI Agent / 自动化工具  
**用途**：项目部署、编译、Git 操作的完整技术规范

---

## 📋 项目架构概览

```
global_version_ai/
├── backend/              # Go 后端服务
│   ├── cmd/gateway/      # 主程序入口
│   ├── bin/              # 预编译二进制目录
│   │   └── gateway-linux-amd64  # Linux 预编译二进制（提交到Git）
│   ├── Dockerfile.prebuilt      # 使用预编译二进制的 Dockerfile
│   ├── docker-entrypoint.sh     # 容器启动脚本（自动运行迁移）
│   └── scripts/migrations/      # 数据库迁移SQL
├── ai-service/           # Python AI 服务
├── docker-compose.yml    # 服务编排配置
└── .github/workflows/    # CI/CD 自动化
    └── build-gateway.yml # 自动编译 Go 二进制
```

---

## 🔧 Go 编译注意事项

### 版本要求

```go
// backend/go.mod
go 1.24  // 必须！依赖 pgx/v5@v5.8.0 要求 >= 1.24
```

**关键依赖版本要求**：
- `github.com/jackc/pgx/v5@v5.8.0` - 要求 Go >= 1.24
- Go 1.24 目前未正式发布，使用 `GOTOOLCHAIN=auto` 自动下载

### 编译命令

```bash
# 标准 Linux 交叉编译（在任何平台）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" -trimpath \
  -o bin/gateway-linux-amd64 ./cmd/gateway

# 参数说明：
# CGO_ENABLED=0     - 禁用 CGO，生成静态二进制（不依赖 glibc）
# GOOS=linux        - 目标操作系统
# GOARCH=amd64      - 目标架构
# -ldflags="-s -w"  - 去除调试信息和符号表，减小体积
# -trimpath         - 去除文件系统路径信息（安全性）
```

### 依赖管理

```bash
# 下载依赖（会自动切换到 Go 1.24）
GOTOOLCHAIN=auto go mod download

# 整理依赖（更新 go.mod 和 go.sum）
GOTOOLCHAIN=auto go mod tidy

# 验证依赖
go mod verify
```

### 编译环境选择

**❌ 不要在服务器编译**
- 原因：服务器内存不足（256MB），Go 编译会 OOM
- 后果：Docker 构建卡死，进程被 kill

**✅ 使用 GitHub Actions 自动编译**
- 位置：`.github/workflows/build-gateway.yml`
- 触发条件：
  - `backend/**/*.go` 文件变更
  - `backend/go.mod` 或 `backend/go.sum` 变更
  - 手动触发 `workflow_dispatch`
- 输出：`backend/bin/gateway-linux-amd64`（自动提交到 Git）

**✅ 本地使用 Docker 编译（如果有 Docker）**
```bash
docker run --rm \
  -v "$(pwd)/backend":/workspace \
  -w /workspace \
  -e GOTOOLCHAIN=auto \
  golang:1.23-alpine \
  sh -c "
    apk add --no-cache git && \
    go mod download && \
    go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
      go build -ldflags='-s -w' -trimpath \
      -o bin/gateway-linux-amd64 ./cmd/gateway
  "
```

---

## 📦 Git 提交注意事项

### 提交前检查

```bash
# 1. 确保所有变更已测试
go test ./...

# 2. 检查 go.mod 和 go.sum 是否需要更新
go mod tidy

# 3. 查看待提交文件
git status

# 4. 排除不必要的文件
git reset <不需要的文件>
```

### 提交规范

**Commit Message 格式**（遵循 Conventional Commits）：
```
<type>(<scope>): <subject>

<body>

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>
```

**Type 类型**：
- `feat`: 新功能
- `fix`: 错误修复
- `build`: 构建系统或外部依赖变更
- `docs`: 文档变更
- `refactor`: 代码重构（不改变功能）
- `perf`: 性能优化
- `test`: 测试相关

**示例**：
```bash
git commit -m "feat: 实现国际版订单同步接口

- 新增 GlobalOrderSyncRequest/Response 领域模型
- 实现 GlobalOrderRepo 和 GlobalOrderService
- 新增 POST /api/v1/internal/global/order/sync 接口
- 新增 POST /api/v1/internal/global/order/payment-success 接口
- 数据库迁移：007_create_global_order_tables.sql

Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### 不应提交的文件

```gitignore
# 已在 .gitignore 中配置
.DS_Store
*.swp
__pycache__/
*.pyc
.env
backend/gateway          # 本地开发二进制
backend/bin/*            # 除了 gateway-linux-amd64

# 临时/备份文件
*.bak
*.tmp
*.log
```

### 敏感信息处理

**❌ 绝对不能提交**：
- 数据库密码（DATABASE_URL 中的密码）
- API 密钥
- JWT Secret
- SMTP 密码
- 个人访问令牌

**✅ 使用环境变量**：
```yaml
# docker-compose.yml
environment:
  - DATABASE_URL=postgres://user:${DB_PASSWORD}@postgres:5432/rakutao
  - JWT_SECRET=${JWT_SECRET}
```

### 推送前验证

```bash
# 1. 确保在正确分支
git branch --show-current  # 应该是 master

# 2. 拉取最新代码（避免冲突）
git pull origin master

# 3. 推送
git push origin master

# 4. 检查 GitHub Actions 状态
# https://github.com/gongqianrong/global_version_ai/actions
```

---

## 🚀 服务器部署流程

### 前置条件

```bash
# 1. SSH 登录服务器
ssh ubuntu@<server-ip>

# 2. 确保已安装 Docker 和 docker-compose
docker --version          # 应显示版本号
docker-compose --version  # 应显示版本号

# 3. 确保用户在 docker 组或使用 sudo
groups | grep docker  # 或使用 sudo docker
```

### 标准部署流程

```bash
# 进入项目目录
cd ~/global_version_ai

# 拉取最新代码
git pull origin master

# 停止旧服务
sudo docker-compose down

# 清理旧镜像（可选，节省空间）
sudo docker system prune -f

# 构建并启动服务（使用预编译二进制，无需编译）
sudo docker-compose up -d

# 等待服务启动（数据库迁移需要时间）
sleep 60

# 查看服务状态
sudo docker-compose ps

# 查看 gateway 日志（检查是否有错误）
sudo docker-compose logs gateway | tail -50

# 健康检查
curl http://localhost:8080/health
# 期望输出：{"status":"ok"}

# 功能测试
curl "http://localhost:8080/api/v1/platform/search?keyword=gundam&platform=surugaya"
```

### 服务启动顺序

```yaml
# docker-compose.yml 定义的依赖关系
gateway (8080)
  ↓ depends_on
  ├── elasticsearch (9200) - 商品搜索
  ├── ai-service (8000)    - AI 翻译
  ├── redis (6379)         - 缓存
  └── postgres (5432)      - 数据库
```

**启动检查点**：
1. PostgreSQL 健康检查通过
2. 数据库迁移自动执行（docker-entrypoint.sh）
3. Gateway 启动并暴露 8080 端口
4. 健康检查端点可访问

### 数据库迁移

**自动执行**（docker-entrypoint.sh）：
```bash
# 1. 等待 PostgreSQL 就绪
while ! pg_isready -h postgres -p 5432; do sleep 1; done

# 2. 按顺序执行迁移脚本
for sql in /app/migrations/*.sql; do
  psql $DATABASE_URL -f "$sql"
done
```

**手动执行**（如需）：
```bash
# 进入容器
sudo docker-compose exec gateway sh

# 执行特定迁移
psql $DATABASE_URL -f /app/migrations/007_create_global_order_tables.sql

# 验证表创建
psql $DATABASE_URL -c "\dt global_order*"
```

### 查看日志

```bash
# 查看所有服务日志
sudo docker-compose logs

# 查看特定服务日志
sudo docker-compose logs gateway
sudo docker-compose logs postgres
sudo docker-compose logs elasticsearch

# 实时跟踪日志
sudo docker-compose logs -f gateway

# 查看最后 N 行
sudo docker-compose logs --tail=100 gateway
```

### 故障排查

**问题 1：端口已被占用**
```bash
# 检查端口占用
sudo netstat -tlnp | grep 8080

# 停止占用进程
sudo kill <PID>
```

**问题 2：容器启动失败**
```bash
# 查看容器状态
sudo docker-compose ps

# 查看失败原因
sudo docker-compose logs <service-name>

# 重启特定服务
sudo docker-compose restart gateway
```

**问题 3：数据库连接失败**
```bash
# 检查 PostgreSQL 是否就绪
sudo docker-compose exec postgres pg_isready

# 检查数据库是否存在
sudo docker-compose exec postgres psql -U postgres -c "\l"

# 手动连接测试
sudo docker-compose exec gateway sh
psql $DATABASE_URL -c "SELECT 1"
```

**问题 4：内存不足**
```bash
# 查看容器资源使用
sudo docker stats

# 查看系统内存
free -h

# 清理未使用的镜像/容器
sudo docker system prune -a -f
```

**问题 5：Gateway 二进制缺失**
```bash
# 检查二进制是否存在
ls -lh backend/bin/gateway-linux-amd64

# 如果缺失，检查 GitHub Actions 是否完成
# https://github.com/gongqianrong/global_version_ai/actions

# 重新拉取
git pull origin master
```

---

## 🔄 更新代码流程

### 场景 A：仅代码变更（无 Go 文件）

```bash
cd ~/global_version_ai
git pull origin master
sudo docker-compose restart gateway
```

### 场景 B：Go 代码变更

```bash
# 1. 等待 GitHub Actions 编译完成
# 查看：https://github.com/gongqianrong/global_version_ai/actions

# 2. 拉取包含新二进制的代码
cd ~/global_version_ai
git pull origin master

# 3. 重新构建并启动
sudo docker-compose down
sudo docker-compose up -d

# 4. 验证
curl http://localhost:8080/health
```

### 场景 C：数据库变更

```bash
# 1. 添加新迁移脚本到 backend/scripts/migrations/
# 例如：008_add_new_table.sql

# 2. 推送到 Git
git add backend/scripts/migrations/008_*.sql
git commit -m "feat: 新增数据库迁移"
git push origin master

# 3. 服务器部署（迁移自动执行）
cd ~/global_version_ai
git pull origin master
sudo docker-compose down
sudo docker-compose up -d

# 4. 验证迁移
sudo docker-compose exec gateway sh
psql $DATABASE_URL -c "\dt"  # 查看所有表
```

### 场景 D：依赖变更（go.mod）

```bash
# 1. 本地更新依赖
go get github.com/new/package@v1.0.0
go mod tidy

# 2. 提交变更（触发自动编译）
git add backend/go.mod backend/go.sum
git commit -m "build: 升级依赖包"
git push origin master

# 3. 等待 GitHub Actions 编译新二进制

# 4. 服务器部署
cd ~/global_version_ai
git pull origin master
sudo docker-compose down
sudo docker-compose up -d
```

---

## 🛡️ 安全检查清单

### 代码提交前
- [ ] 移除所有 console.log / fmt.Println 调试语句
- [ ] 移除硬编码的密码/密钥
- [ ] 检查敏感信息是否使用环境变量
- [ ] 移除测试数据/账号
- [ ] 确保 .gitignore 包含敏感文件

### 部署前
- [ ] 更新生产环境变量（JWT_SECRET, DB_PASSWORD）
- [ ] 确认 CORS 配置正确
- [ ] 检查防火墙规则
- [ ] 备份数据库

### 运行时
- [ ] 定期更新依赖包（安全补丁）
- [ ] 监控日志（异常请求、错误）
- [ ] 定期备份数据库

---

## 📊 性能优化建议

### Docker 镜像优化
- ✅ 使用多阶段构建（builder + runtime）
- ✅ 使用 Alpine Linux（减小镜像体积）
- ✅ 静态编译（CGO_ENABLED=0）
- ✅ 去除调试信息（-ldflags="-s -w"）

### 部署优化
- ✅ 使用预编译二进制（避免服务器编译）
- ✅ 利用 Docker layer 缓存
- ✅ 限制容器内存（防止 OOM）
- ⚠️ 考虑使用 Docker BuildKit（更快的构建）

### 运行时优化
- 配置合适的 Go GC 参数
- 使用连接池（数据库、Redis）
- 启用 HTTP/2
- 添加监控和告警（Prometheus + Grafana）

---

## 🔗 相关资源

- **GitHub Repository**: https://github.com/gongqianrong/global_version_ai
- **GitHub Actions**: https://github.com/gongqianrong/global_version_ai/actions
- **Go 文档**: https://go.dev/doc/
- **Docker 文档**: https://docs.docker.com/
- **PostgreSQL 文档**: https://www.postgresql.org/docs/

---

## 📝 常用命令速查

```bash
# === Git ===
git status                      # 查看状态
git add -A                      # 添加所有文件
git commit -m "message"         # 提交
git push origin master          # 推送
git pull origin master          # 拉取

# === Docker ===
sudo docker-compose up -d       # 启动服务
sudo docker-compose down        # 停止服务
sudo docker-compose ps          # 查看状态
sudo docker-compose logs -f gateway  # 查看日志
sudo docker-compose restart gateway  # 重启服务
sudo docker system prune -f     # 清理资源

# === 数据库 ===
sudo docker-compose exec postgres psql -U postgres
sudo docker-compose exec gateway psql $DATABASE_URL

# === 健康检查 ===
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/platform/search?keyword=test&platform=surugaya

# === 资源监控 ===
sudo docker stats               # 容器资源使用
free -h                        # 系统内存
df -h                          # 磁盘空间
```

---

**最后更新**：2026-08-18  
**维护者**：@gongqianrong
