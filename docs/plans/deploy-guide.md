# 服务器部署执行计划

## 服务器信息

- 服务器地址: 52.195.4.10
- 网关端口: 8080
- GitHub 仓库: https://github.com/gongqianrong/global_version_ai.git

## 部署步骤

### 1. SSH 登录服务器

```bash
ssh ubuntu@52.195.4.10
```

### 2. 拉取最新代码

```bash
cd /path/to/global_version_ai
git pull origin master
```

### 3. 重新构建并启动 Docker 容器

```bash
docker-compose up --build -d
```

### 4. 检查服务状态

```bash
# 查看容器运行状态
docker-compose ps

# 查看网关日志
docker-compose logs -f gateway
```

### 5. 验证接口

```bash
# 健康检查
curl http://52.195.4.10:8080/health

# 商品详情（之前 404，现已修复）
curl http://52.195.4.10:8080/api/v1/products/surugaya_663043159

# 平台搜索
curl "http://52.195.4.10:8080/api/v1/platform/search?keyword=gundam&platform=surugaya"

# 骏河屋扩展接口
curl http://52.195.4.10:8080/api/v1/surugaya/discounts
curl http://52.195.4.10:8080/api/v1/surugaya/campaigns
curl "http://52.195.4.10:8080/api/v1/surugaya/products/663043159/reviews"
curl "http://52.195.4.10:8080/api/v1/surugaya/products/663043159/stores"
```

## 本次修复内容（3 个提交）

### 提交 1: 商品详情 adapter 回退
- ES 查不到商品时，自动回退到平台 adapter 的 `GetProduct()` 方法
- 修改文件:
  - `backend/internal/domain/product.go` — 新增 `ParseProductID()` 解析统一 ID
  - `backend/internal/api/platform_service.go` — 新增 `GetProduct()` 方法
  - `backend/internal/api/product.go` — `ProductHandler` 增加 fallback 逻辑
  - `backend/cmd/gateway/main.go` — 注入 platformService 作为 fallback

### 提交 2: 售罄商品价格回退
- 售罄商品 `types` 为空，无价格导致 normalize 失败
- 从 `detail["定価"]`（定价）取值作为回退
- 修改文件: `backend/internal/adapter/surugaya/adapter.go`

### 提交 3: Detail 字段类型修复
- `Detail` 字段从 `map[string]string` 改为 `map[string]interface{}`
- 因为 API 返回的 `定価: 4950` 是数字，无法解码为 string，导致整个 `GetProductDetail` 失败
- 修改文件: `backend/internal/adapter/surugaya/client.go`

## 如需本地重新编译

```bash
# 在本地 Mac 上交叉编译 Linux 二进制
export PATH=$HOME/go-sdk/go/bin:$PATH
cd /Users/gongqianrong/Desktop/ai/backend
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o gateway-linux ./cmd/gateway

# 运行测试
go test ./...
```

## 完整接口列表

| # | 方法 | 路径 | 说明 |
|---|------|------|------|
| 1 | GET | `/health` | 健康检查 |
| 2 | GET | `/api/v1/search?keyword=xxx` | ES 搜索 |
| 3 | GET | `/api/v1/search/stream/{streamID}` | 实时搜索 SSE 流 |
| 4 | GET | `/api/v1/products/{id}` | 商品详情（ES + adapter 回退） |
| 5 | GET | `/api/v1/platform/search?keyword=xxx&platform=surugaya` | 平台直接搜索 |
| 6 | GET | `/api/v1/surugaya/products/{id}/reviews` | 商品评论 + 相似商品 |
| 7 | GET | `/api/v1/surugaya/products/{id}/stores` | 其他店铺 |
| 8 | GET | `/api/v1/surugaya/discounts` | 折扣活动 |
| 9 | GET | `/api/v1/surugaya/comments?user_id=xxx` | 用户留言 |
| 10 | GET | `/api/v1/surugaya/campaigns` | 活动列表 |
| 11 | GET | `/api/v1/surugaya/campaigns/detail?url=xxx` | 活动详情 |
