# 🚀 快速部署指南

简洁的项目部署和代码提交操作手册。

---

## 📦 代码提交（3 步）

### 1. 提交代码

```bash
cd ~/Desktop/ai

# 查看改动
git status

# 添加文件
git add <文件路径>
# 或添加所有
git add -A

# 提交（清晰描述改动）
git commit -m "feat: 实现新功能"

# 推送到 GitHub
git push origin master
```

### 2. 等待自动编译

- 访问 https://github.com/gongqianrong/global_version_ai/actions
- 等待绿色 ✓（约 2-3 分钟）
- 如果改了 Go 代码，GitHub 会自动编译并提交二进制文件

---

## 🖥️ 服务器部署（4 步）

### 1. 登录服务器

```bash
ssh ubuntu@<服务器IP>
```

### 2. 更新代码

```bash
cd ~/global_version_ai
git pull origin master
```

### 3. 重启服务

```bash
# 停止旧服务
sudo docker-compose down

# 启动新服务
sudo docker-compose up -d
```

### 4. 验证部署

```bash
# 等待 1 分钟（数据库初始化）
sleep 60

# 检查服务状态
sudo docker-compose ps

# 健康检查（应返回 {"status":"ok"}）
curl http://localhost:8080/health
```

---

## 🔧 Go 编译要点

### ⚠️ 重要

- **不要在服务器编译**（内存不足会卡死）
- **使用 GitHub Actions 自动编译**
- **本地没安装 Go 也能部署**（用预编译二进制）

### 如果需要本地编译

```bash
cd backend

# Linux 版本（服务器用）
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -ldflags="-s -w" -trimpath \
  -o bin/gateway-linux-amd64 ./cmd/gateway

# 提交二进制
git add bin/gateway-linux-amd64
git commit -m "build: 更新预编译二进制"
git push origin master
```

---

## 📝 Commit 规范

遵循格式：`<类型>: <简短描述>`

### 常用类型

- `feat`: 新功能
- `fix`: 修 Bug
- `build`: 编译相关
- `docs`: 文档修改
- `refactor`: 代码重构

### 示例

```bash
git commit -m "feat: 添加用户登录功能"
git commit -m "fix: 修复订单金额计算错误"
git commit -m "docs: 更新 API 文档"
```

---

## 🔍 常见问题

### 问题 1：推送失败

```bash
# 先拉取最新代码
git pull origin master

# 如果有冲突，手动解决后再提交
git add <冲突文件>
git commit -m "merge: 解决冲突"
git push origin master
```

### 问题 2：服务启动失败

```bash
# 查看日志找原因
sudo docker-compose logs gateway

# 重启服务
sudo docker-compose restart gateway

# 完全重建
sudo docker-compose down
sudo docker-compose up -d --force-recreate
```

### 问题 3：端口被占用

```bash
# 查找占用端口的进程
sudo netstat -tlnp | grep 8080

# 停止该进程
sudo kill <进程ID>
```

### 问题 4：GitHub Actions 编译失败

- 检查 go.mod 语法是否正确
- 确认 Go 版本是 1.24
- 查看 Actions 页面的错误日志

---

## 📊 常用命令

### Git 操作

```bash
git status              # 查看状态
git pull               # 拉取最新代码
git add -A             # 添加所有改动
git commit -m "xxx"    # 提交
git push origin master # 推送
git log --oneline      # 查看提交历史
```

### Docker 操作

```bash
sudo docker-compose up -d          # 启动所有服务
sudo docker-compose down           # 停止所有服务
sudo docker-compose ps             # 查看服务状态
sudo docker-compose logs gateway   # 查看日志
sudo docker-compose restart gateway # 重启服务
sudo docker system prune -f        # 清理未使用资源
```

### 健康检查

```bash
# 基础健康检查
curl http://localhost:8080/health

# 搜索功能测试
curl "http://localhost:8080/api/v1/platform/search?keyword=gundam&platform=surugaya"

# 查看数据库
sudo docker-compose exec postgres psql -U postgres -d rakutao
```

---

## 🎯 完整部署流程示例

```bash
# 1. 本地修改代码
vim backend/internal/service/order_service.go

# 2. 提交到 GitHub
git add backend/internal/service/order_service.go
git commit -m "fix: 修复订单创建逻辑"
git push origin master

# 3. 等待 GitHub Actions 编译（查看 Actions 页面）

# 4. SSH 到服务器
ssh ubuntu@your-server

# 5. 部署
cd ~/global_version_ai
git pull origin master
sudo docker-compose down
sudo docker-compose up -d

# 6. 验证
sleep 60
curl http://localhost:8080/health
sudo docker-compose logs gateway | tail -50

# 7. 完成！
```

---

## 🛡️ 安全提醒

- ❌ **不要提交密码/密钥到 Git**
- ✅ 使用 `.env` 文件或环境变量
- ✅ 定期更新依赖包
- ✅ 定期备份数据库

---

## 📞 获取帮助

- **GitHub Issues**: https://github.com/gongqianrong/global_version_ai/issues
- **查看日志**: `sudo docker-compose logs gateway`
- **详细文档**: 见 `docs/AI_DEPLOYMENT_GUIDE.md`

---

**快捷链接**
- [GitHub 仓库](https://github.com/gongqianrong/global_version_ai)
- [GitHub Actions](https://github.com/gongqianrong/global_version_ai/actions)
- [API 文档](./GLOBAL_ORDER_SYNC_API.md)

最后更新：2026-08-18
