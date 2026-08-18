# 服务器部署指南

## 代码已提交但需要推送

代码已经成功提交到本地仓库，commit ID: `355c1bb`

**提交内容:**
- 国际版订单同步功能完整实现
- P0问题修复（订单号生成和支付事务一致性）
- OrderLink代购链接功能
- 完整的文档和测试数据
- 数据库迁移脚本

## 推送到GitHub的方法

### 方法1: 使用SSH密钥（推荐）

```bash
cd /Users/gongqianrong/Desktop/ai

# 1. 检查是否有SSH密钥
ls -la ~/.ssh/

# 2. 如果没有，生成新的SSH密钥
ssh-keygen -t ed25519 -C "your_email@example.com"

# 3. 将SSH公钥添加到GitHub
cat ~/.ssh/id_ed25519.pub
# 复制输出的内容到 GitHub > Settings > SSH and GPG keys > New SSH key

# 4. 修改远程仓库URL为SSH格式
git remote set-url origin git@github.com:gongqianrong/global_version_ai.git

# 5. 推送代码
git push origin master
```

### 方法2: 使用Personal Access Token

```bash
cd /Users/gongqianrong/Desktop/ai

# 1. 在GitHub创建Personal Access Token
# GitHub > Settings > Developer settings > Personal access tokens > Tokens (classic) > Generate new token
# 选择 repo 权限

# 2. 推送时使用token
git push https://<YOUR_TOKEN>@github.com/gongqianrong/global_version_ai.git master

# 或者配置credential helper来存储token
git config --global credential.helper osxkeychain
git push origin master
# 在弹出的窗口中输入token作为密码
```

### 方法3: 使用GitHub Desktop或其他GUI工具

如果你安装了GitHub Desktop，可以直接在应用中进行推送操作。

## 服务器部署步骤

一旦代码推送成功，在服务器上执行以下操作：

### 1. 拉取最新代码

```bash
# SSH到服务器
ssh your_server

# 进入项目目录
cd /path/to/global_version_ai

# 拉取最新代码
git pull origin master
```

### 2. 执行数据库迁移

```bash
cd backend

# 方法1: 使用部署脚本（推荐）
./scripts/deploy_global_order_sync.sh

# 方法2: 手动执行迁移
psql $DATABASE_URL < scripts/migrations/007_create_global_order_tables.sql
```

### 3. 编译和部署

```bash
# 编译Go服务
make build

# 或者直接构建
go build -o gateway ./cmd/gateway

# 重启服务（根据你的部署方式）
# 如果使用systemd:
sudo systemctl restart rakutao-gateway

# 如果使用docker:
docker-compose up -d --build

# 如果直接运行:
./gateway &
```

### 4. 验证部署

```bash
# 测试健康检查
curl http://localhost:8080/health

# 测试订单同步接口
curl -X POST http://localhost:8080/api/v1/internal/global/order/sync \
  -H "Content-Type: application/json" \
  -d @docs/test_sync_order.json

# 检查日志
tail -f /var/log/rakutao-gateway.log
# 或者
journalctl -u rakutao-gateway -f
```

### 5. 验证数据库表

```bash
psql $DATABASE_URL -c "\d global_order_records"
psql $DATABASE_URL -c "\d global_order_payments"
psql $DATABASE_URL -c "SELECT COUNT(*) FROM global_order_records"
```

## 环境变量检查

确保服务器上配置了以下环境变量：

```bash
# 必需
DATABASE_URL=postgresql://user:password@host:port/dbname
JWT_SECRET=your_jwt_secret

# 可选
PORT=8080
AI_SERVICE_URL=http://localhost:8000
ELASTICSEARCH_URL=http://localhost:9200
REDIS_URL=redis://localhost:6379
```

## 接口端点

部署成功后，以下接口可用：

- `POST /api/v1/internal/global/order/sync` - 订单同步
- `POST /api/v1/internal/global/order/payment-success` - 支付同步

## 监控和日志

### 查看订单同步日志

```bash
# 查看最近的订单同步
psql $DATABASE_URL -c "SELECT * FROM global_order_records ORDER BY created_at DESC LIMIT 10"

# 查看支付记录
psql $DATABASE_URL -c "SELECT * FROM global_order_payments ORDER BY created_at DESC LIMIT 10"

# 查看异常状态
psql $DATABASE_URL -c "SELECT * FROM global_order_records WHERE payment_sync_state = 2"
```

## 回滚方案

如果部署出现问题：

```bash
# 1. 回滚代码
git revert 355c1bb
git push origin master

# 2. 回滚数据库（如果需要）
# 删除表（注意：会丢失数据）
psql $DATABASE_URL -c "DROP TABLE IF EXISTS global_order_payments CASCADE"
psql $DATABASE_URL -c "DROP TABLE IF EXISTS global_order_records CASCADE"

# 3. 重启服务
sudo systemctl restart rakutao-gateway
```

## 技术支持

如有问题，查看：
- API文档: `backend/docs/GLOBAL_ORDER_SYNC_API.md`
- 实现总结: `backend/docs/GLOBAL_ORDER_SYNC_IMPLEMENTATION_SUMMARY.md`
- P0修复报告: `backend/docs/P0_FIX_REPORT.md`

## Commit信息

```
Commit: 355c1bb
Message: feat: 实现国际版订单同步功能和P0问题修复

主要更新:
- 国际版订单同步接口
- 支付同步接口  
- P0问题修复（订单号生成和支付事务）
- OrderLink功能
- 完整的文档和测试数据

文件变更: 25 files, +2999 insertions, -24 deletions
```
