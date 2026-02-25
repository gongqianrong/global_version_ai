# Rakutao International - 采集与搜索系统设计文档

> **Version**: 1.0
> **Date**: 2026-02-25
> **Status**: Approved
> **Related**: [Project Brief](../project-brief.md) | [PRD](../prd.md) | [Architecture](../architecture.md)

---

## 1. 概述

本文档定义 Rakutao 国际版的后端数据层架构，覆盖三个核心子系统：

| 子系统 | 职责 |
|--------|------|
| **采集系统** | 统一网关 + Adapter 模式处理多平台异构数据采集 |
| **搜索系统** | ES 缓存搜索 + 实时代理搜索 + 多语言翻译搜索 |
| **品牌识别系统** | 三级流水线（平台字段 → 规则匹配 → AI 提取）+ 品牌库 |

### 1.1 技术选型

| 组件 | 技术 | 理由 |
|------|------|------|
| 主引擎 | **Go** | 高并发采集调度、实时代理搜索、API 网关 |
| AI/爬虫 | **Python** | 品牌 AI 提取、LLM 翻译、新平台爬虫（Scrapy） |
| 搜索引擎 | **Elasticsearch** | 日语分词（kuromoji）、全文搜索、聚合查询 |
| 缓存 | **Redis** | 热数据缓存、搜索结果缓存 |
| 品牌库 | **PostgreSQL** | 结构化品牌数据、别名映射 |
| Go ↔ Python 通信 | **gRPC / HTTP** | Python 作为独立微服务被 Go 调用 |

### 1.2 设计原则

- **统一网关**：上层服务只与统一 Schema 交互，平台差异在 Adapter 层内部消化
- **能力声明**：每个平台通过 Capabilities 声明自身能力，上层逻辑据此决策
- **渐进增强**：缓存优先快速返回，实时搜索异步补充
- **WMS/运单预留**：Output Router 使用接口模式，预留未来 WMS 集成扩展点

---

## 2. 统一商品数据结构（Unified Product Schema）

这是整个系统的基石，所有平台的异构数据最终都转换成此统一结构。

```go
// UnifiedProduct - 统一商品数据结构
type UnifiedProduct struct {
    // === 标识 ===
    ID              string    // Rakutao 内部唯一 ID (生成规则: {platform}_{original_id})
    SourcePlatform  string    // 来源平台标识: "mercari", "rakuma", "tobu", "beams" ...
    SourceID        string    // 源平台原始商品 ID
    SourceURL       string    // 源平台原始链接 (内部使用，不暴露给用户)

    // === 核心商品信息 ===
    Title           string              // 原始标题 (日语)
    TitleTranslated map[string]string   // 翻译后标题 {"en": "...", "zh-TW": "..."}
    Description     string              // 原始描述 (日语)
    DescTranslated  map[string]string   // 翻译后描述
    Images          []string            // 图片 URL 列表 (第一张为主图)

    // === 价格 ===
    PriceJPY        int64     // 价格 (日元，整数)
    OriginalPrice   int64     // 原价 (有折扣时)
    ShippingType    string    // "free" | "buyer_pays" | "included"
    ShippingFeeJPY  int64     // 国内运费 (日元)

    // === 分类与品牌 ===
    Brand           *Brand    // 品牌信息 (可为空)
    Categories      []string  // 标准化分类路径: ["服饰", "女装", "包包"]
    SourceCategory  string    // 源平台原始分类 (保留用于映射调试)
    Tags            []string  // 标签: "new", "sale", "limited", "auction" ...

    // === 商品状态 ===
    Status          string    // "available" | "sold" | "reserved" | "delisted"
    Condition       string    // "new" | "like_new" | "good" | "fair" | "poor"
    Quantity        int       // 库存数量 (C2C 通常为 1)

    // === 卖家信息 ===
    Seller          SellerInfo

    // === 规格 (如有) ===
    Variants        []Variant // 规格选项: 颜色、尺寸等

    // === 时间 ===
    ListedAt        time.Time // 上架时间
    CollectedAt     time.Time // 采集时间
    UpdatedAt       time.Time // 最后更新时间
    CacheTTL        int       // 缓存有效期 (秒)

    // === 内容分级 (v2) ===
    ContentRating   string    // "general" | "r18"

    // === 物流预留 (WMS 需求明确后扩展) ===
    Logistics       *LogisticsInfo
}

type Brand struct {
    ID       string // 品牌库 ID
    Name     string // 标准品牌名: "Louis Vuitton"
    NameJA   string // 日语名: "ルイ・ヴィトン"
    Source   string // 品牌来源: "platform_field" | "rule_matched" | "ai_extracted"
}

type SellerInfo struct {
    SellerID   string  // 源平台卖家 ID
    SellerName string  // 卖家名称
    Rating     float64 // 卖家评分 (标准化为 0-5)
    ItemCount  int     // 在售商品数
}

type Variant struct {
    Name    string   // "颜色" | "サイズ"
    Options []string // ["赤", "青"] | ["S", "M", "L"]
}

// WMS/运单预留结构
type LogisticsInfo struct {
    EstimatedWeight  float64 // 预估重量 (g)
    EstimatedSize    string  // 预估尺寸 "S/M/L"
    WarehouseRegion  string  // 入库仓区域
    // WMS 需求明确后扩展更多字段
}
```

### Schema 设计决策

| 决策 | 理由 |
|------|------|
| `ID` 使用 `{platform}_{original_id}` | 全局唯一，且可反查来源 |
| 价格用 `int64` 而非 `float` | 日元无小数，避免浮点精度问题 |
| 翻译字段用 `map[string]string` | 灵活支持多语言，按需翻译 |
| `Brand` 独立结构体 + 可为空 | 不是所有商品都有品牌，品牌需要指向统一品牌库 |
| 保留 `SourceCategory` | 方便调试分类映射的准确性 |
| `Brand.Source` 标记来源 | 区分来源方式，便于评估 AI 提取准确率 |
| `Logistics` 指针类型可为空 | WMS 需求未明确，预留但不强制 |

---

## 3. 采集系统架构

### 3.1 整体分层

```
┌─────────────────────────────────────────────────────────────────┐
│                  Collection Gateway (Go)                        │
│                  统一采集网关 - 对上层暴露唯一入口                  │
│                                                                 │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Platform Registry (平台注册中心)               │  │
│  │  记录所有已接入平台的元信息、状态、能力声明                    │  │
│  └───────────────────────────────────────────────────────────┘  │
│                              │                                  │
│         ┌────────────────────┼────────────────────┐             │
│         ▼                    ▼                    ▼             │
│  ┌──────────────┐  ┌──────────────────┐  ┌──────────────────┐  │
│  │ Adapter 类型A │  │  Adapter 类型B    │  │  Adapter 类型C    │  │
│  │ 国内版代理    │  │  自有爬虫 (Python) │  │  自有API对接 (Go) │  │
│  │              │  │                  │  │                  │  │
│  │ · Mercari    │  │ · 新平台X        │  │ · 新平台Y         │  │
│  │ · Rakuma     │  │ · 新平台Z        │  │ · (有开放API的)    │  │
│  │ · TOBU       │  │                  │  │                  │  │
│  │ · netsea     │  │                  │  │                  │  │
│  │ · BEAMS ...  │  │                  │  │                  │  │
│  └──────────────┘  └──────────────────┘  └──────────────────┘  │
│         │                    │                    │             │
│         ▼                    ▼                    ▼             │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Normalizer (数据标准化层)                      │  │
│  │  异构数据 → UnifiedProduct Schema                          │  │
│  │  + 品牌提取 (调 Python AI 服务)                             │  │
│  └──────────────────────────┬────────────────────────────────┘  │
│                              │                                  │
│                              ▼                                  │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Output Router (输出路由)                       │  │
│  │                                                           │  │
│  │  ├─→ Elasticsearch (商品索引)                              │  │
│  │  ├─→ Cache Layer (Redis 热数据缓存)                        │  │
│  │  ├─→ ★ Extension Point: WMS/运单系统                      │  │
│  │  └─→ ★ Extension Point: 未来其他下游消费者                  │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 核心接口定义

```go
// === Adapter 统一接口 ===
// 每个平台实现这个接口，无论底层是调国内版API、爬虫、还是直接API对接

type PlatformAdapter interface {
    // 平台标识
    PlatformID() string        // "mercari", "rakuma", "tobu" ...

    // 能力声明 — 不同平台能力不同
    Capabilities() AdapterCaps

    // 搜索商品 (用于实时代理搜索)
    Search(ctx context.Context, query SearchQuery) (*SearchResult, error)

    // 获取商品详情
    GetProduct(ctx context.Context, productID string) (*RawProduct, error)

    // 批量采集 (定时任务用)
    BatchCollect(ctx context.Context, params CollectParams) (<-chan RawProduct, error)

    // 健康检查
    HealthCheck(ctx context.Context) HealthStatus
}

// 能力声明 — 不是所有平台都支持所有操作
type AdapterCaps struct {
    SupportsSearch       bool  // 是否支持搜索
    SupportsRealtime     bool  // 是否支持实时查询 (vs 只能定时采集)
    SupportsBatchCollect bool  // 是否支持批量采集
    HasBrandField        bool  // 平台数据是否自带品牌字段
    HasCategoryField     bool  // 平台数据是否自带分类字段
    MaxQPS               int   // 平台允许的最大请求频率
}

// RawProduct — Adapter 返回的原始数据，保留平台原始字段
type RawProduct struct {
    Platform    string                 // 来源平台
    RawID       string                 // 源平台商品ID
    RawData     map[string]interface{} // 原始数据 (保留完整，用于调试和回溯)
    Normalized  *UnifiedProduct        // 标准化后的数据 (由 Normalizer 填充)
}

// === 输出路由接口 ===
type OutputSink interface {
    Name() string
    Write(ctx context.Context, products []UnifiedProduct) error
}
```

### 3.3 三类 Adapter 实现模式

**类型 A — 国内版代理 Adapter：**
- 调用国内版已有的采集接口
- 拿到各平台的原始数据
- 包装为 RawProduct 返回，标准化交给 Normalizer

**类型 B — 自有爬虫 Adapter（Python）：**
- Go 通过 gRPC/HTTP 调用 Python Scrapy 爬虫服务
- 用于国内版未覆盖的新平台
- Python 端负责反爬对抗、页面解析

**类型 C — 自有 API 对接 Adapter（Go）：**
- 直接用 Go 对接有开放 API 的新平台
- 按平台 API 文档实现

### 3.4 Platform Registry（平台注册中心）

```go
type PlatformRegistry struct {
    adapters map[string]PlatformAdapter // platformID → adapter
}

type PlatformMeta struct {
    ID          string       // "mercari"
    Name        string       // "メルカリ"
    NameEN      string       // "Mercari"
    Icon        string       // 平台 icon URL (用于前端 UI 标识)
    Type        string       // "domestic_proxy" | "self_crawler" | "self_api"
    Status      string       // "active" | "degraded" | "offline"
    Caps        AdapterCaps  // 能力声明
}

func (r *PlatformRegistry) Register(meta PlatformMeta, adapter PlatformAdapter)
func (r *PlatformRegistry) ActivePlatforms() []PlatformMeta
func (r *PlatformRegistry) GetAdapter(platformID string) (PlatformAdapter, error)
```

### 3.5 WMS / 运单预留设计

Output Router 使用 `OutputSink` 接口，WMS 需求明确后只需实现一个新的 Sink：

```go
// 场景1: 独立 WMS 系统 → 通过 API/MQ 推送数据
// 场景2: 国际版专属 WMS → 直接写同一数据库
// 具体实现取决于 WMS 架构决策
type WMSSink struct {
    endpoint string
}

func (w *WMSSink) Name() string { return "wms" }
func (w *WMSSink) Write(ctx context.Context, products []UnifiedProduct) error {
    // 将商品数据同步到 WMS 系统
}
```

---

## 4. 搜索系统

### 4.1 搜索全景流程

```
用户输入 "gucci bag" (英语)
         │
         ▼
┌─────────────────────────────────────────────────────────┐
│                Search Gateway (Go)                       │
│                                                         │
│  1. 语言检测 + 翻译                                      │
│     "gucci bag" → Python AI 服务 → "グッチ バッグ"        │
│                                                         │
│  2. 并发双通道搜索                                        │
│     ┌──────────────────────┬──────────────────────────┐  │
│     │  通道 A: ES 缓存搜索  │  通道 B: 实时代理搜索     │  │
│     │  关键词: グッチ バッグ  │  并发调各平台 Adapter     │  │
│     │  ← 毫秒级返回         │  ← 秒级返回              │  │
│     └──────────┬───────────┘└────────────┬─────────────┘  │
│                │                         │               │
│  3. 结果聚合 + 去重 + 排序                                │
│     ├─ ES 结果先返回 (快)                                 │
│     ├─ 实时结果增量补充 (流式)                             │
│     └─ 按平台打上 UI 标识                                 │
│                                                         │
│  4. 翻译商品标题 → 用户语言                               │
│                                                         │
│  5. 返回统一格式结果                                      │
└─────────────────────────────────────────────────────────┘
```

### 4.2 搜索入口

| 入口 | 说明 |
|------|------|
| **全站搜索**（首页搜索栏） | 搜所有平台，结果混排，每条结果带平台 UI 标识（icon + 颜色） |
| **平台内搜索**（从平台列表进入） | 只搜指定平台，UI 风格统一，顶部显示当前平台标识 |

两种入口都支持多语言翻译搜索。

### 4.3 ES 索引设计

```json
{
  "settings": {
    "analysis": {
      "analyzer": {
        "ja_analyzer": {
          "type": "custom",
          "tokenizer": "kuromoji_tokenizer",
          "filter": [
            "kuromoji_baseform",
            "kuromoji_part_of_speech",
            "cjk_width",
            "lowercase"
          ]
        },
        "multilang_analyzer": {
          "type": "custom",
          "tokenizer": "icu_tokenizer",
          "filter": ["icu_folding", "lowercase"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id":                { "type": "keyword" },
      "source_platform":   { "type": "keyword" },
      "status":            { "type": "keyword" },

      "title": {
        "type": "text",
        "analyzer": "ja_analyzer",
        "fields": {
          "keyword": { "type": "keyword" }
        }
      },
      "title_en":          { "type": "text", "analyzer": "multilang_analyzer" },
      "title_zh_tw":       { "type": "text", "analyzer": "multilang_analyzer" },
      "description":       { "type": "text", "analyzer": "ja_analyzer" },

      "brand_id":          { "type": "keyword" },
      "brand_name":        { "type": "keyword" },
      "brand_name_ja": {
        "type": "text",
        "analyzer": "ja_analyzer",
        "fields": { "keyword": { "type": "keyword" } }
      },

      "categories":        { "type": "keyword" },
      "tags":              { "type": "keyword" },
      "condition":         { "type": "keyword" },
      "content_rating":    { "type": "keyword" },

      "price_jpy":         { "type": "long" },
      "shipping_type":     { "type": "keyword" },

      "seller_id":         { "type": "keyword" },
      "seller_rating":     { "type": "float" },

      "listed_at":         { "type": "date" },
      "collected_at":      { "type": "date" },
      "updated_at":        { "type": "date" },

      "images":            { "type": "keyword", "index": false }
    }
  }
}
```

**索引设计决策：**

| 决策 | 理由 |
|------|------|
| `kuromoji_tokenizer` 日语分词 | 日语商品标题必须用专用分词器 |
| `title` + `title_en` + `title_zh_tw` 多字段 | 用户可能用任何语言搜索已翻译的标题 |
| `brand_name` 用 keyword | 品牌名精确匹配，不需要分词 |
| `brand_name_ja` 用 text + ja_analyzer | 日语品牌名需要分词（如 "ルイ・ヴィトン"） |
| `source_platform` 用 keyword | 支持按平台过滤（平台内搜索场景） |
| `price_jpy` 用 long | 支持范围查询和排序 |

### 4.4 搜索 API 设计

```go
// === 搜索请求 ===
type SearchQuery struct {
    Keyword       string   // 用户输入的原始关键词 (任意语言)
    KeywordJA     string   // 翻译后的日语关键词 (由翻译服务填充)
    Platforms     []string // 指定平台过滤 (空=全站搜索)
    BrandID       string   // 品牌过滤
    Categories    []string // 分类过滤
    PriceMin      int64    // 最低价 (JPY)
    PriceMax      int64    // 最高价 (JPY)
    Condition     []string // 商品成色过滤
    SortBy        string   // "relevance" | "price_asc" | "price_desc" | "newest"
    Page          int
    PageSize      int
    UserLang      string   // 用户语言 "en" | "zh-TW" | "ja"
    ContentRating string   // "general" | "all" (需年龄验证)
}

// === 搜索响应 ===
type SearchResponse struct {
    CachedResults    []ProductSummary // 缓存结果 (立即返回)
    CachedTotal      int64
    RealtimeStreamID string           // 前端用此 ID 通过 WebSocket 接收增量结果
    Aggregations     SearchAggs       // 聚合信息 (用于筛选面板)
    TranslatedKeyword string          // 翻译后的日语关键词，展示给用户确认
}

type ProductSummary struct {
    ID             string
    Title          string   // 已翻译为用户语言的标题
    TitleOriginal  string   // 原始日语标题
    Image          string   // 主图
    PriceJPY       int64
    Platform       string   // 平台标识 (用于 UI icon)
    Status         string
    Brand          string   // 品牌名 (如有)
    Tags           []string
    IsTranslated   bool     // 标题是否为翻译内容 (显示 Translation 标签)
}

type SearchAggs struct {
    Platforms  []AggBucket // 各平台结果数量
    Brands     []AggBucket // 品牌分布
    PriceRange PriceRange  // 价格区间
    Categories []AggBucket // 分类分布
}
```

### 4.5 多语言搜索翻译流程

```
用户输入 → 语言检测 → 分流处理 → 搜索
                │
     ┌──────────┼──────────┐
     ▼          ▼          ▼
   日语        英语/中文    其他语言
   直接搜      翻译→日语    翻译→日语
                │          │
                ▼          ▼
          Python AI 翻译服务
          ├─ 商品领域优化 (不是通用翻译)
          ├─ 品牌名不翻译 ("Gucci" → "グッチ" 映射而非翻译)
          ├─ 同义词扩展 ("bag" → "バッグ,カバン,鞄")
          └─ 返回日语关键词
```

**翻译关键细节：**

| 场景 | 处理方式 |
|------|---------|
| 品牌名 | 从品牌库查映射（"Gucci"→"グッチ"），不走通用翻译 |
| 品类词 | 同义词扩展（"包"→"バッグ,カバン,鞄,ポーチ"） |
| 混合输入 | "gucci 红色包" → "グッチ 赤 バッグ" 逐词处理 |
| 搜索确认 | 翻译后的日语关键词返回给前端，用户可确认或修改 |

### 4.6 实时代理搜索流程

```
前端发起搜索
    │
    ├─ HTTP: 立即返回 ES 缓存结果 (~200ms)
    │
    └─ WebSocket: 开启实时代理搜索流
         │
         ├─ 并发调用各活跃平台 Adapter.Search()
         │   ├─ Mercari adapter → ~2s 返回
         │   ├─ Rakuma adapter  → ~1.5s 返回
         │   ├─ TOBU adapter    → ~3s 返回 (超时跳过)
         │   └─ ...
         │
         ├─ 每个平台返回后立即：
         │   ├─ Normalize → UnifiedProduct
         │   ├─ 与 ES 缓存结果去重 (按 source_platform + source_id)
         │   └─ 通过 WebSocket 推送增量结果给前端
         │
         └─ 全部完成或超时 (5s) → 关闭流
              └─ 新采集到的数据异步写入 ES (下次搜索就在缓存里了)
```

---

## 5. 品牌识别系统

### 5.1 整体架构

```
┌─────────────────────────────────────────────────────────┐
│                Brand Service (Go + Python)               │
│                                                         │
│  ┌───────────────────────────────────────────────────┐  │
│  │           Brand Registry (品牌库 - PostgreSQL)      │  │
│  │                                                   │  │
│  │  brand_id │ name_std    │ name_ja     │ aliases   │  │
│  │  b_001    │ Gucci       │ グッチ       │ [...]     │  │
│  │  b_002    │ Louis V.    │ ルイ・ヴィトン │ [...]     │  │
│  │  b_003    │ Nike        │ ナイキ       │ [...]     │  │
│  └───────────────────────────────────────────────────┘  │
│                         │                               │
│           ┌─────────────┼──────────────┐                │
│           ▼             ▼              ▼                │
│     Level 1:       Level 2:       Level 3:              │
│     平台字段直取    规则匹配        AI 提取 (Python)      │
│     (成本:0)       (成本:低)       (成本:高)             │
│     (准确率:高)    (准确率:中高)    (准确率:中)           │
│           │             │              │                │
│           └─────────────┼──────────────┘                │
│                         ▼                               │
│              品牌标准化 + 置信度                          │
│              → 写入 UnifiedProduct.Brand                 │
└─────────────────────────────────────────────────────────┘
```

### 5.2 品牌库数据模型

```go
type BrandEntry struct {
    ID          string    // "b_001"
    NameStd     string    // 标准英文名: "Louis Vuitton"
    NameJA      string    // 日语名: "ルイ・ヴィトン"
    NameZhTW    string    // 繁中名: "路易威登"
    Aliases     []string  // 别名/缩写: ["LV", "ルイヴィトン", "LOUIS VUITTON"]
    Category    string    // 品牌品类: "luxury" | "sportswear" | "electronics" | ...
    LogoURL     string    // 品牌 logo (用于前端展示)
    IsVerified  bool      // 是否人工验证过
    CreatedAt   time.Time
}
```

### 5.3 三级识别流水线

品牌识别按优先级依次尝试，命中即停：

**Level 1: 平台字段直取**
- 如果 Adapter 返回的 raw data 中有 brand 字段，在品牌库中查找匹配
- 命中则直接使用，标记 Source = "platform_field"

**Level 2: 规则匹配**
- 用品牌库的 aliases 列表在商品标题 + 描述中做全匹配
- 例: "ルイ・ヴィトン モノグラム ショルダーバッグ" → 匹配到 Louis Vuitton
- 命中则使用，标记 Source = "rule_matched"

**Level 3: AI 提取**
- 调用 Python AI 服务 (LLM)
- 从标题 + 描述中识别品牌，返回品牌名 + 置信度
- 置信度 >= 0.7 才写入商品，< 0.7 标记为待人工确认
- 标记 Source = "ai_extracted"

### 5.4 品牌提取接口

```go
type BrandExtractor interface {
    Extract(ctx context.Context, input BrandExtractionInput) (*BrandExtractionResult, error)
    BatchExtract(ctx context.Context, inputs []BrandExtractionInput) ([]BrandExtractionResult, error)
}

type BrandExtractionInput struct {
    Title       string // 商品标题 (日语)
    Description string // 商品描述 (日语)
    Category    string // 源平台分类 (辅助判断)
}

type BrandExtractionResult struct {
    BrandName   string  // 提取到的品牌名
    Confidence  float64 // 置信度 0-1
    Source      string  // 识别方式
    MatchedID   string  // 品牌库中匹配到的 ID (如有)
}
```

### 5.5 品牌库维护策略

| 策略 | 说明 |
|------|------|
| **初始导入** | 从各平台的品牌列表批量导入，建立基础品牌库 |
| **AI 发现新品牌** | AI 提取到品牌库中不存在的品牌名 → 进入待审核队列 |
| **人工审核** | 运营定期审核 AI 新发现的品牌，确认后入库 |
| **别名自动扩展** | 同一品牌被不同写法命中时，自动将新写法加入 aliases |
| **置信度阈值** | AI 提取置信度 < 0.7 时不写入商品，标记为待人工确认 |

### 5.6 品牌在搜索中的应用

- ES 中 `brand_id` → 精确过滤（筛选面板）
- ES 中 `brand_name` → 精确匹配（搜 "Gucci" 直接匹配）
- ES 中 `brand_name_ja` → 日语分词匹配（搜 "グッチ" 匹配）
- 搜索翻译服务识别品牌名后走品牌库映射而非通用翻译

---

## 6. 系统交互总览

```
┌──────────┐     ┌──────────────────────────────────────────────┐
│  Mobile  │────→│          Search Gateway (Go)                 │
│   App    │     │  ├─ 翻译 (Python AI) ──→ 日语关键词           │
│          │←────│  ├─ ES 缓存搜索 ──→ 即时结果                  │
│          │  WS │  └─ 实时代理搜索 ──→ 增量结果                  │
│          │←────│                                              │
└──────────┘     └──────────────┬───────────────────────────────┘
                                │
                 ┌──────────────▼───────────────────────────────┐
                 │      Collection Gateway (Go)                 │
                 │  ├─ Platform Registry (平台注册中心)           │
                 │  ├─ Adapters (A:国内版 / B:爬虫 / C:API)      │
                 │  ├─ Normalizer (标准化 + 品牌提取)             │
                 │  └─ Output Router                            │
                 │      ├─→ Elasticsearch                       │
                 │      ├─→ Redis Cache                         │
                 │      └─→ ★ WMS (预留)                        │
                 └──────────────────────────────────────────────┘
                                │
              ┌─────────────────┼─────────────────┐
              ▼                 ▼                  ▼
     ┌──────────────┐  ┌──────────────┐   ┌──────────────┐
     │ 国内版采集接口 │  │ Python AI    │   │ 新平台       │
     │ (已有平台)    │  │ · 品牌提取    │   │ (爬虫/API)   │
     │              │  │ · 翻译服务    │   │              │
     │              │  │ · 新平台爬虫   │   │              │
     └──────────────┘  └──────────────┘   └──────────────┘
```

---

## 7. 关键设计决策汇总 (ADR)

| # | 决策 | 理由 |
|---|------|------|
| ADR-B01 | Go + Python 双引擎 | Go 负责高并发数据管道，Python 负责 AI/爬虫，各取所长 |
| ADR-B02 | 统一网关 + Adapter 模式 | 平台差异在 Adapter 层消化，上层只面对统一 Schema |
| ADR-B03 | 能力声明（Capabilities） | 不同平台能力不同，上层逻辑据此决策 |
| ADR-B04 | 缓存优先 + 实时补充的混合搜索 | 兼顾响应速度和数据新鲜度 |
| ADR-B05 | 品牌三级识别流水线 | 成本从低到高逐级尝试，命中即停 |
| ADR-B06 | Output Router 接口模式 | 预留 WMS/运单系统扩展点 |
| ADR-B07 | 品牌名走品牌库映射而非通用翻译 | 保证品牌搜索准确性 |
| ADR-B08 | 保留 RawData 原始数据 | 标准化失败时不丢数据，支持调试回溯 |
