# Redis 缓存 + 分类接口接入

## 执行日期: 2026-02-28

---

## 一、Redis 数据缓存

### 需求
为热点接口添加 Redis 缓存，减少重复请求上游，提升响应速度。上游崩溃时返回兜底缓存保证可用性。

### 缓存策略

**只缓存热点数据（避免 Redis 内存爆满）：**

| 接口 | 热缓存 TTL | 兜底缓存 TTL | 说明 |
|------|-----------|-------------|------|
| `/api/v1/products/{id}` | 10 分钟 | 24 小时 | 同一商品多人查看 |
| `/api/v1/surugaya/products/{id}/reviews` | 30 分钟 | 24 小时 | 跟随商品详情访问 |
| `/api/v1/surugaya/products/{id}/stores` | 30 分钟 | 24 小时 | 跟随商品详情访问 |
| `/api/v1/surugaya/campaigns/detail` | 30 分钟 | 24 小时 | 活动详情半静态 |
| `/api/v1/surugaya/discounts` | 1 小时 | 24 小时 | 全站共享，变化慢 |
| `/api/v1/surugaya/campaigns` | 1 小时 | 24 小时 | 全站共享，变化慢 |
| `/api/v1/surugaya/categories*` | 1 小时 | 24 小时 | 分类菜单极少变 |

**不缓存：**
- 搜索 `/search`、`/platform/search`（关键词组合无限，命中率低）
- 用户留言 `/comments`（按用户ID查，命中率低）
- 健康检查 `/health`、WebSocket `/search/stream/*`

### 两层缓存机制

```
请求 → 查热缓存(cache:) → 命中 → 返回 X-Cache: HIT
                         → 未命中 → 调上游
                                    → 成功 → 更新热缓存 + 兜底缓存(fallback:) → X-Cache: MISS
                                    → 失败 → 查兜底缓存 → 有 → 返回旧数据 X-Cache: STALE
                                                        → 无 → 返回原始错误
```

### 实现文件

| 文件 | 操作 | 说明 |
|------|------|------|
| `backend/internal/cache/cache.go` | 新建 | Redis 客户端封装 (New/Ping/Get/Set/Close) |
| `backend/internal/cache/middleware.go` | 新建 | 缓存中间件 + captureWriter + cacheKey + ttlForPath |
| `backend/internal/cache/middleware_test.go` | 新建 | 17 个测试用例 |
| `backend/internal/api/router.go` | 修改 | RouterConfig 增加 CacheMiddleware 字段 |
| `backend/cmd/gateway/main.go` | 修改 | Redis 初始化 + REDIS_URL 环境变量 + 优雅降级 |
| `backend/go.mod` | 修改 | 添加 github.com/redis/go-redis/v9 |
| `docker-compose.yml` | 修改 | 启用 Redis 服务 + gateway 环境变量 |

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `REDIS_URL` | `redis://localhost:6379` | Redis 连接地址 |

---

## 二、骏河屋分类接口接入

### 需求
接入骏河屋分类菜单 API（来自 YAPI 文档 #496、#504）。

### 新增接口

| 接口 | 上游路径 | 说明 |
|------|---------|------|
| `GET /api/v1/surugaya/categories` | `/suruga/category/search/adv` | 一级分类菜单 |
| `GET /api/v1/surugaya/categories/{id}` | `/suruga/category/search/adv/{category_id}` | 子级分类 |

### 实现文件

| 文件 | 修改内容 |
|------|---------|
| `backend/internal/adapter/surugaya/client.go` | 新增 CategoryItem/CategoryData 结构体 + GetCategories() + GetSubCategories() |
| `backend/internal/api/surugaya_handler.go` | 新增 HandleCategories() + HandleSubCategories() |
| `backend/internal/api/router.go` | 注册 /categories 和 /categories/{id} 路由 |
| `backend/internal/cache/middleware.go` | 分类接口加入 1 小时缓存 |

---

## 三、验证

```bash
export PATH=$HOME/go-sdk/go/bin:$PATH

# 运行全部测试
go test ./... -v -count=1

# 本地启动（需要 Redis）
docker run -d -p 6379:6379 redis:7-alpine
go run ./cmd/gateway

# 测试缓存（第二次请求应返回 X-Cache: HIT）
curl -v http://localhost:8080/api/v1/surugaya/discounts
curl -v http://localhost:8080/api/v1/surugaya/discounts

# 测试分类接口
curl http://localhost:8080/api/v1/surugaya/categories
curl http://localhost:8080/api/v1/surugaya/categories/1
```
