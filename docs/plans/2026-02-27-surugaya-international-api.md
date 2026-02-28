# 骏河屋国际版 Go Gateway API 接口文档

> Go 网关对外暴露的骏河屋商品相关 HTTP 接口
> 基础路径: `http://{GATEWAY_HOST}:{PORT}` (默认端口 8080)
> 整理时间: 2026-02-27

---

## 目录

1. [通用说明](#1-通用说明)
2. [搜索商品](#2-搜索商品)
3. [实时搜索流](#3-实时搜索流-websocket)
4. [商品详情](#4-商品详情)
5. [健康检查](#5-健康检查)
6. [骏河屋扩展接口](#6-骏河屋扩展接口待接入)
7. [数据模型](#7-数据模型)

---

## 1. 通用说明

### 响应格式

所有接口统一返回:

```json
{
  "code": 0,
  "data": { ... },
  "message": "",
  "request_id": "a1b2c3d4e5f67890"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | `0` 成功，非 0 为错误 |
| data | object | 业务数据（错误时省略） |
| message | string | 错误消息（成功时省略） |
| request_id | string | 16 位 hex 请求追踪 ID |

### 错误码

| code | HTTP Status | 说明 |
|------|-------------|------|
| 0 | 200 | 成功 |
| 40001 | 400 | 关键词被内容策略拦截 |
| 40002 | 400 | 缺少必填参数 |
| 40401 | 404 | 商品未找到 |
| 50001 | 500 | 内部服务错误 |
| 50003 | 503 | 搜索服务不可用 |

### 公共 Header

| Header | 说明 |
|--------|------|
| Content-Type | `application/json; charset=utf-8` |
| X-Request-ID | 请求追踪 ID |

### CORS

- 允许所有来源
- 允许方法: GET, POST, PUT, DELETE, OPTIONS
- 允许头: Accept, Content-Type, Authorization
- 预检缓存: 300s

---

## 2. 搜索商品

### `GET /api/v1/search`

跨平台搜索，返回 ES 缓存结果 + 实时流 ID。通过 `platforms=surugaya` 筛选骏河屋商品。

### 请求参数 (Query)

| 参数 | 必填 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| keyword | **是** | string | — | 搜索关键词（自动翻译为日语） |
| platforms | 否 | string | 全平台 | 平台筛选，逗号分隔。骏河屋: `surugaya` |
| page | 否 | int | 1 | 页码 |
| page_size | 否 | int | 20 | 每页条数，最大 100 |
| brand_id | 否 | string | — | 品牌 ID 筛选 |
| categories | 否 | string | — | 分类筛选，逗号分隔 |
| price_min | 否 | int64 | 0 | 最低价格 (JPY) |
| price_max | 否 | int64 | 0 | 最高价格 (JPY) |
| condition | 否 | string | — | 品相筛选，逗号分隔: `new`, `like_new`, `good`, `fair`, `poor` |
| sort | 否 | string | 相关性 | 排序: `price_asc`, `price_desc`, `newest`, `release_date_desc`, `release_date_asc` |
| lang | 否 | string | zh-TW | 用户语言: `zh-TW`, `en`, `ja` |
| content_rating | 否 | string | general | 内容分级: `general`, `r18` |

### 请求示例

```
GET /api/v1/search?keyword=ガンダム&platforms=surugaya&page=1&page_size=20&condition=new,good&sort=price_asc&lang=zh-TW
```

### 响应 data

```json
{
  "cached_results": [
    {
      "id": "surugaya_663043159",
      "title": "FW GUNDAM CONVERGE SB ニカーヤ",
      "title_original": "FW GUNDAM CONVERGE SB アーガマ級強襲用宇宙巡洋艦 ニカーヤ",
      "image": "https://cdn.suruga-ya.jp/pics_webp/boxart_m/663043159m.jpg.webp",
      "price_jpy": 5445,
      "platform": "surugaya",
      "status": "available",
      "brand": "[バンダイ]",
      "condition": "",
      "tags": ["食玩　トレーディングフィギュア"],
      "is_translated": false
    }
  ],
  "cached_total": 150,
  "realtime_stream_id": "stream_a1b2c3d4e5f6",
  "translated_keyword": "ガンダム",
  "aggregations": {
    "platforms": [
      { "key": "surugaya", "count": 150 }
    ],
    "brands": [
      { "key": "[バンダイ]", "count": 80 }
    ],
    "categories": [
      { "key": "食玩　トレーディングフィギュア", "count": 40 },
      { "key": "プラモデル", "count": 35 }
    ],
    "price_ranges": [
      { "min": 0, "max": 1000, "count": 20 },
      { "min": 1000, "max": 5000, "count": 80 },
      { "min": 5000, "max": 10000, "count": 50 }
    ]
  }
}
```

### 骏河屋特有逻辑

- `condition` 筛选映射: `new` → 骏河屋 `新品`, 其他 → `中古`
- `sort` 映射: `price_asc` → `price:ascending`, `newest` → `modificationTime:descending`
- `content_rating=r18` 时关闭骏河屋安全搜索
- 默认只搜索有库存商品 (`inStock=On`)
- 价格已含 10% 服务费 (`price_jpy = 原价 × 1.1`)
- `brand` 字段来自骏河屋原始数据（如 `"[バンダイ] "`）
- `status`: 骏河屋 `品切れ` → `sold`, `null` → `available`

---

## 3. 实时搜索流 (WebSocket)

### `WS /api/v1/search/stream/{streamID}`

搜索接口返回 `realtime_stream_id` 后，客户端建立 WebSocket 连接接收实时结果。

### 连接流程

1. 调用 `/api/v1/search` 获取 `realtime_stream_id`
2. 建立 WebSocket: `ws://{host}/api/v1/search/stream/{streamID}`
3. 服务端并发搜索所有实时平台（含骏河屋）
4. 每个平台结果以事件推送
5. 全部完成后发送 `done` 事件

### 约束

| 参数 | 值 |
|------|------|
| 流有效期 (TTL) | 30 秒 |
| 搜索超时 | 5 秒 |
| 事件缓冲区 | 50 条 |
| 并发消费 | 仅允许 1 个客户端 |

### 事件格式

#### results — 平台搜索结果

```json
{
  "type": "results",
  "platform": "surugaya",
  "products": [
    {
      "id": "surugaya_GU375453",
      "title": "1-27[SR]：機動戦士Gundam GQuuuuuuX キービジュアル",
      "title_original": "...",
      "image": "https://cdn.suruga-ya.jp/pics_webp/boxart_m/gu375453m.jpg.webp",
      "price_jpy": 638,
      "platform": "surugaya",
      "status": "available",
      "brand": "[バンダイ]",
      "condition": "",
      "tags": [],
      "is_translated": false
    }
  ],
  "total": 150
}
```

#### error — 平台搜索失败

```json
{
  "type": "error",
  "platform": "surugaya",
  "message": "connection timeout"
}
```

#### done — 搜索完成

```json
{
  "type": "done",
  "platforms_searched": ["surugaya", "yahoo_auction"]
}
```

### 错误响应

| HTTP Status | 场景 |
|-------------|------|
| 400 | streamID 缺失 |
| 404 | 流不存在或已过期 (>30s) |
| 409 | 流已被其他客户端占用 |

---

## 4. 商品详情

### `GET /api/v1/products/{id}`

获取单个商品完整信息。骏河屋商品 ID 格式: `surugaya_{goods_id}`。

### 请求参数

| 参数 | 位置 | 必填 | 说明 |
|------|------|------|------|
| id | URL Path | **是** | 商品 ID，如 `surugaya_663043159` |
| lang | Query | 否 | 用户语言，默认 `zh-TW` |

### 请求示例

```
GET /api/v1/products/surugaya_663043159?lang=zh-TW
```

### 响应 data

```json
{
  "id": "surugaya_663043159",
  "platform": "surugaya",
  "title": "FW GUNDAM CONVERGE SB アーガマ級強襲用宇宙巡洋艦 ニカーヤ",
  "title_original": "FW GUNDAM CONVERGE SB アーガマ級強襲用宇宙巡洋艦 ニカーヤ",
  "description": "プレミアムバンダイ限定",
  "description_original": "プレミアムバンダイ限定",
  "images": [
    "https://cdn.suruga-ya.jp/pics_webp/boxart_m/663043159m.jpg.webp"
  ],
  "price_jpy": 5445,
  "service_fee_jpy": 495,
  "original_price": 4950,
  "shipping_type": "",
  "shipping_fee_jpy": 0,
  "brand": {
    "id": "",
    "name": "[バンダイ]",
    "name_ja": "",
    "source": "platform_field"
  },
  "categories": [],
  "source_category": "食玩　トレーディングフィギュア",
  "condition": "",
  "status": "available",
  "quantity": 1,
  "seller": {
    "seller_id": "surugaya",
    "seller_name": "駿河屋",
    "rating": 0,
    "item_count": 0
  },
  "tags": ["食玩　トレーディングフィギュア"],
  "content_rating": "general",
  "is_translated": false
}
```

### 骏河屋特有字段说明

| 字段 | 说明 |
|------|------|
| `seller.seller_id` | 固定为 `"surugaya"`（骏河屋是商店，非 marketplace） |
| `seller.seller_name` | 固定为 `"駿河屋"` 或商品所属店铺名 |
| `brand.source` | 骏河屋搜索结果直接返回品牌，source 为 `"platform_field"` |
| `original_price` | 骏河屋原价 (JPY)，不含服务费 |
| `price_jpy` | 含 10% 服务费后价格 |
| `service_fee_jpy` | 10% 服务费 |

---

## 5. 健康检查

### `GET /health`

### 响应

```json
{
  "status": "ok",
  "platforms": {
    "surugaya": {
      "status": "healthy",
      "message": "surugaya API is reachable"
    },
    "yahoo_auction": {
      "status": "healthy",
      "message": "yahoo_auction domestic API is reachable"
    }
  }
}
```

| 全局 status | 说明 |
|-------------|------|
| `ok` | 所有平台健康 |
| `degraded` | 部分平台异常 |

| 平台 status | 说明 |
|-------------|------|
| `healthy` | 正常 |
| `unhealthy` | 异常（message 含错误信息） |

---

## 6. 骏河屋扩展接口（待接入）

以下接口已在 Surugaya Adapter `Client()` 中实现，尚未暴露为 HTTP 端点。后续需在 `api/` 层新增 handler 接入。

| Go Client 方法 | 上游路径 | 说明 | 建议端点 |
|----------------|----------|------|----------|
| `GetProductExtend(goodsID)` | `/suruga/product/extend/{goods_id}` | 评论与相似商品 | `GET /api/v1/products/{id}/extend` |
| `GetProductStores(goodsID)` | `/suruga/product/extend/store/{goods_id}` | 其他可购商店 | `GET /api/v1/products/{id}/stores` |
| `GetDiscount()` | `/suruga/product/discount` | 折扣活动信息 | `GET /api/v1/surugaya/discount` |
| `GetUserComments(params)` | `/suruga/product/user/comments` | 用户评论历史 | `GET /api/v1/surugaya/user/comments` |
| `GetCampaigns()` | `/suruga/product/campaign` | 活动列表 | `GET /api/v1/surugaya/campaigns` |
| `GetCampaignDetail(url)` | `/suruga/product/campaign/detail` | 活动详情 | `GET /api/v1/surugaya/campaigns/detail` |

### 扩展接口上游数据结构参考

#### GetProductExtend — 评论与相似商品

```json
{
  "comments": [
    {
      "star": "5.0",
      "title": "評論タイトル",
      "userId": "user123",
      "userName": "ユーザー名",
      "comment": "評論内容",
      "other": "1人中、1人の方が参考になったと言っています。"
    }
  ],
  "otherList": [
    {
      "img": "https://...",
      "mid": "product_param",
      "name": "類似商品名",
      "price": 3000,
      "url": "https://www.suruga-ya.jp/..."
    }
  ]
}
```

#### GetProductStores — 其他可购商店

```json
{
  "stores": [
    {
      "storeName": "店舗名",
      "price": 4950,
      "stock": 3,
      "buyState": true,
      "state": "中古",
      "mailOrder": 500,
      "pstime": "1-3日",
      "score": "4.8",
      "goodsLink": "https://...",
      "storeLink": "https://...",
      "cartId": "rem_xxx",
      "tenpoCD": "400490",
      "branchNumber": "001",
      "kubun": "中古",
      "scoreDesc": 3380
    }
  ]
}
```

#### GetDiscount — 折扣活动

```json
{
  "timeSale": [
    { "title": "タイムセール", "content": "期間限定", "picLink": "https://..." }
  ],
  "unifiedSale": [
    {
      "title": "まとめ買い割引",
      "content": "活動内容",
      "activeTime": "2026-03-01 ~ 2026-03-31",
      "conditions": [
        { "count": "3", "discount": "10%" },
        { "count": "5", "discount": "15%" }
      ]
    }
  ]
}
```

#### GetCampaigns — 活动列表

```json
{
  "campaigns": [
    {
      "title": "春のキャンペーン",
      "detail": "詳細テキスト",
      "image": "https://...",
      "conditions": [
        { "num": "3", "discount": "10%OFF" }
      ],
      "urlList": [
        { "alt": "詳細", "url": "https://..." }
      ]
    }
  ]
}
```

#### GetCampaignDetail — 活动详情

```json
{
  "campaigns": [
    { "title": "キャンペーン名", "desc": "説明", "image": "https://...", "condition": "条件" }
  ],
  "groups": [
    {
      "title": "グループ名",
      "items": [
        {
          "id": "663043159",
          "name": "商品名",
          "image": "https://...",
          "url": "https://...",
          "canBuy": true,
          "priceList": [
            { "isRange": false, "price": 4950, "status": "中古", "minPrice": "", "maxPrice": "" }
          ],
          "detailParams": {}
        }
      ],
      "otherSearchParams": {}
    }
  ]
}
```

---

## 7. 数据模型

### ProductSummary（搜索结果列表项）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 统一 ID: `surugaya_{goods_id}` |
| title | string | 标题（翻译后，如有） |
| title_original | string | 原始日语标题 |
| image | string | 主图 URL |
| price_jpy | int64 | 含服务费价格 (JPY) |
| platform | string | 固定 `"surugaya"` |
| status | string | `available` / `sold` |
| brand | string | 品牌名 |
| condition | string | `new` / `like_new` / `good` / `fair` / `poor` / `""` |
| tags | string[] | 标签 |
| is_translated | bool | 标题是否已翻译 |

### UnifiedProduct（商品详情完整模型）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | `surugaya_{goods_id}` |
| source_platform | string | `"surugaya"` |
| source_id | string | 骏河屋原始商品 ID |
| source_url | string | `https://www.suruga-ya.jp/product/detail/{id}` |
| title | string | 商品名 |
| title_translated | map[string]string | 翻译: `{"en": "...", "zh-TW": "..."}` |
| description | string | 商品描述 |
| images | string[] | 图片列表 |
| price_jpy | int64 | 含 10% 服务费价格 |
| service_fee_jpy | int64 | 服务费 |
| original_price | int64 | 原价 |
| brand | Brand? | 品牌信息 |
| categories | string[] | 标准化分类 |
| source_category | string | 原始分类 |
| tags | string[] | 标签 (子分类、厂商、JAN 码) |
| status | string | `available` / `sold` / `reserved` / `delisted` |
| condition | string | 品相 |
| quantity | int | 库存，默认 1 |
| seller | SellerInfo | 卖家信息（骏河屋固定） |
| content_rating | string | `general` / `r18` |

### Brand

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 品牌库 ID |
| name | string | 品牌名，如 `"[バンダイ]"` |
| name_ja | string | 日语名 |
| source | string | `platform_field` / `rule_matched` / `ai_extracted` |

### SellerInfo

| 字段 | 类型 | 说明 |
|------|------|------|
| seller_id | string | 骏河屋固定: `"surugaya"` |
| seller_name | string | 固定: `"駿河屋"` |
| rating | float64 | 评分（骏河屋暂为 0） |
| item_count | int | 商品数（骏河屋暂为 0） |

### SearchAggs（聚合/筛选面板）

| 字段 | 类型 | 说明 |
|------|------|------|
| platforms | AggBucket[] | 平台统计 |
| brands | AggBucket[] | 品牌统计 |
| categories | AggBucket[] | 分类统计 |
| price_ranges | PriceRange[] | 价格区间分布 |

```json
// AggBucket
{ "key": "surugaya", "count": 150 }

// PriceRange
{ "min": 1000, "max": 5000, "count": 80 }
```

---

## 附录: 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| PORT | 8080 | 网关监听端口 |
| SURUGAYA_API_URL | http://153.231.197.185 | 骏河屋国内版 API |
| YAHOO_AUCTION_API_URL | http://localhost:3001 | Yahoo 拍卖 API |
| ELASTICSEARCH_URL | http://localhost:9200 | ES 地址 |
| ES_INDEX_NAME | rakutao_products | ES 索引名 |
| AI_SERVICE_URL | http://localhost:8000 | AI 翻译/品牌服务 |

## 附录: 骏河屋适配层文件

| 文件 | 说明 |
|------|------|
| `backend/internal/adapter/surugaya/client.go` | 8 个上游 API 方法 + 数据结构 |
| `backend/internal/adapter/surugaya/adapter.go` | PlatformAdapter 实现 + 字段映射 |
| `backend/internal/adapter/surugaya/adapter_test.go` | 单元测试 |
| `backend/internal/normalizer/normalizer.go` | RawProduct → UnifiedProduct 标准化 |
| `backend/internal/api/search.go` | 搜索 handler |
| `backend/internal/api/product.go` | 商品详情 handler |
| `backend/internal/api/realtime.go` | WebSocket 实时流 handler |
| `backend/internal/api/router.go` | 路由注册 |
