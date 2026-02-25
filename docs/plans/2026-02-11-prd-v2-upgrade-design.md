# Rakutao International PRD v2 Upgrade Design

> **Date**: 2026-02-11
> **Status**: Approved
> **Scope**: 10 new requirement areas for PRD v2

---

## Summary of Changes

| # | Requirement | Design Decision |
|---|------------|-----------------|
| 1 | 新购买流程 | 10-state Order 状态机，增加 purchased/warehoused/packing_requested/packed/shipping_paid 阶段 |
| 2 | WMS 微服务 | 独立 Warehouse Service，通过消息队列与 App 后端解耦 |
| 3 | 支付方式 | Apple Pay/PayPal/Google Pay 最高优先级，支持直接支付+钱包并存模式 |
| 4 | 直播模式 | 代购直播，新增 LiveStream Service + LiveStream/LiveStreamItem 实体 |
| 5 | 擦边类商品 | 内容分级 G/M/A，年龄验证门，地区化内容屏蔽 |
| 6 | 个性化推荐+地区化 | UserPreference + 行为追踪 + RegionConfig 地区配置 |
| 7 | 指定购买升级 | 实时代理搜索 + UnlockedPlatform 平台解锁机制 |
| 8 | 站内信+工单自动化 | L0自助/L1机器人/L2人工三级策略，智能分类+自动处理 |
| 9 | 高价值商品 | 全流程优化：鉴定、保价物流、保险、专属客服、防欺诈风控 |
| 10 | 站点风格统一+稳定性 | 统一 UI 框架+地区差异化内容，微服务隔离+熔断降级+异步解耦 |

---

## 1. 新购买流程 — Order 状态机重构

### 10-State Order 状态机

```
pending → paid → purchasing → purchased → warehoused → packing_requested → packed → shipping_paid → shipped → fulfilled
  ↓         ↓        ↓
cancelled  refunded  refunded
```

| State | 说明 | 触发者 |
|-------|------|--------|
| `pending` | 用户已下单，待支付商品款 | 用户 |
| `paid` | 商品款已支付 | 系统 |
| `purchasing` | 平台采购者正在日本网站采购 | 运营 |
| `purchased` | 采购成功，等待入库 | 运营回写 |
| `warehoused` | 商品已入库（日本仓库） | WMS |
| `packing_requested` | 用户申请打包 | 用户 |
| `packed` | 仓库打包完成，待用户支付运费 | WMS |
| `shipping_paid` | 用户已支付国际运费 | 系统 |
| `shipped` | 仓库填写物流编号，已发货 | WMS |
| `fulfilled` | 用户确认收货 | 用户 |

---

## 2. WMS 仓储管理系统 — 新增微服务

### 职责

| 职责 | 说明 |
|------|------|
| **入库管理** | 采购成功后商品扫码入库，记录商品状态、重量、尺寸、实物照片 |
| **库位管理** | 商品存放位置追踪 |
| **打包管理** | 接收用户打包申请，合并同用户多件商品，记录打包重量/尺寸 |
| **运费计算** | 根据打包后实际重量/体积计算国际运费 |
| **物流填写** | 发货时录入物流单号、承运商信息 |
| **高价值商品处理** | 特殊包装、拍照存档、保价标记 |

### 数据所有权

`warehouse_item`, `package`, `shipment`

### 与 App 后端通信

| 调用方 | 被调用方 | 通信方式 | 说明 |
|-------|---------|---------|------|
| WMS | Order Service | 异步消息 | 入库成功 → Order 状态变更为 `warehoused` |
| WMS | Order Service | 异步消息 | 打包完成 → Order 状态变更为 `packed` + 推送实际运费 |
| WMS | Order Service | 异步消息 | 发货 → Order 状态变更为 `shipped` + 物流编号 |
| Order Service | WMS | 异步消息 | 用户申请打包 → WMS 创建打包任务 |
| Order Service | WMS | 异步消息 | 用户支付运费 → WMS 标记可发货 |

---

## 3. 支付方式 — Apple Pay / PayPal / Google Pay

### 支付模式：直接支付 + 钱包并存

- 商品款：可选钱包余额 OR 直接支付
- 运费（打包后）：同上
- 钱包充值：仍然支持

### 支付网关优先级

| 支付方式 | 优先级 | 覆盖地区 | 说明 |
|---------|--------|---------|------|
| **Apple Pay** | 🔴 最高 | 全球 iOS | iOS 用户首选 |
| **Google Pay** | 🔴 最高 | 全球 Android | Android 用户首选 |
| **PayPal** | 🔴 最高 | 全球 | 跨地区通用支付 |
| MoMo | 中 | 越南 | 本地支付补充 |
| ZaloPay | 中 | 越南 | 本地支付补充 |
| Stripe | 低 | 全球 | 信用卡兜底方案 |

### 支付场景矩阵

| 场景 | 可用支付方式 |
|------|------------|
| 钱包充值 | Apple Pay / Google Pay / PayPal / MoMo / ZaloPay |
| 商品下单支付 | 钱包余额 / Apple Pay / Google Pay / PayPal |
| 运费支付（打包后） | 钱包余额 / Apple Pay / Google Pay / PayPal |
| 补差价 | 钱包余额 / Apple Pay / Google Pay / PayPal |

---

## 4. 直播模式 — 代购直播

### 核心功能

| 功能 | 说明 |
|------|------|
| **直播推流** | 代购者通过 App 推流（RTMP），支持前后摄像头切换 |
| **实时观看** | 用户拉流观看（HLS/WebRTC 低延迟），弹幕互动 |
| **实时下单** | 用户在直播间点击"帮我买这个"，创建指定购买请求 |
| **商品标记** | 代购者实时标记商品价格、名称，生成临时商品卡片 |
| **直播回放** | 结束后自动生成回放，关联已标记商品 |
| **预告/订阅** | 提前发布直播预告，用户订阅提醒 |

### 新增实体

#### LiveStream（直播）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | string (UUID) | PK | — |
| host_id | string | FK → User (agent) | 主播（代购者） |
| title | string | NOT NULL | 直播标题 |
| description | text | NULLABLE | 直播描述 |
| status | enum | NOT NULL | scheduled / live / ended |
| scheduled_at | datetime | NULLABLE | 预定开播时间 |
| started_at | datetime | NULLABLE | 实际开播时间 |
| ended_at | datetime | NULLABLE | 结束时间 |
| viewer_count | integer | NOT NULL, DEFAULT 0 | 当前观看人数 |
| replay_url | string | NULLABLE | 回放地址 |
| thumbnail_url | string | NULLABLE | 封面图 |

#### LiveStreamItem（直播商品标记）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | string (UUID) | PK | — |
| stream_id | string | FK → LiveStream | — |
| title | string | NOT NULL | 商品名称 |
| estimated_price_jpy | integer | NOT NULL | 预估价格 |
| images | string[] | NULLABLE | 现场拍照 |
| marked_at | datetime | NOT NULL | 标记时间 |

### 直播下单流程

```
代购者开播 → 展示商品 → 标记商品信息
    │
    ▼
用户点击"帮我买" → 创建 OrderLink
    │
    ├── 关联 live_stream_id + live_stream_item_id
    ├── 状态: pending_confirmation
    │
    ▼
直播结束后，代购者统一确认价格 → 走正常 OrderLink 流程
```

### 新增微服务

**LiveStream Service** — 独立服务，负责推拉流管理、弹幕、商品标记。依赖第三方流媒体服务（Agora/声网、AWS IVS 等）。

---

## 5. 擦边类商品支持 — 内容分级与年龄验证

### 内容分级体系

| 分级 | 标识 | 说明 | 访问条件 |
|------|------|------|---------|
| **General** | G | 普通商品 | 无限制 |
| **Mature** | M | 轻度擦边（内衣、泳装、写真集等） | 年龄验证 ≥ 18 |
| **Adult** | A | 成人商品（成人漫画、周边等） | 年龄验证 ≥ 18 + 用户主动开启 |

### 年龄验证流程

```
用户首次访问 M/A 类商品
    │
    ▼
弹出年龄验证门（Age Gate）
    │
    ├── 输入出生日期 → 校验 ≥ 18
    ├── 验证通过 → 记录 age_verified_at
    ├── 验证失败 → 拒绝访问，24h 内不可重试
    │
    ▼
A 级商品额外需要: 用户在设置中手动开启"成人内容"开关
```

### 实体变更

**Product**: `content_rating enum NOT NULL DEFAULT 'G'` (G / M / A)

**User**: `age_verified_at datetime NULLABLE`, `adult_content_enabled boolean DEFAULT false`

### 业务规则

- 搜索/列表默认过滤 M/A 级内容，仅已验证用户可见
- A 级需双重校验：`age_verified_at IS NOT NULL AND adult_content_enabled = true`
- 直播间展示 M/A 类商品时，直播间标记内容等级
- 推送/推荐不主动推送 A 级商品
- 部分地区按法规完全屏蔽 A 级内容

### 地区合规

| 地区 | 限制 |
|------|------|
| 台湾 | 成人商品可售，需年龄验证 |
| 越南 | 可能需屏蔽 A 级 |
| 其他东南亚 | 按国家配置开关 |

---

## 6. 个性化推荐与地区化推广

### 用户偏好模型

#### UserPreference（用户偏好）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| user_id | string | PK, FK → User | 一对一 |
| preferred_categories | string[] | NULLABLE | 偏好分类 |
| preferred_price_range | jsonb | NULLABLE | `{ min_jpy, max_jpy }` |
| preferred_platforms | string[] | NULLABLE | 偏好来源平台 |
| updated_at | datetime | NOT NULL | — |

### 行为追踪事件

| 事件 | 数据 | 用途 |
|------|------|------|
| `product_view` | product_id, category, price, duration | 兴趣信号 |
| `product_search` | query, locale, result_count | 搜索意图 |
| `product_favorite` | product_id, category | 强兴趣信号 |
| `product_purchase` | product_id, category, price | 最强信号 |
| `product_share` | product_id | 社交兴趣 |

### 推荐策略

| 场景 | 算法 | 说明 |
|------|------|------|
| 首页推荐 | 协同过滤 + 热度 | "猜你喜欢" + 地区热门 |
| 商品详情页 | 相似商品 | 同分类/同价位/同卖家 |
| 搜索结果排序 | 相关性 + 偏好加权 | 偏好分类排名靠前 |
| Push 推送 | 价格变动 + 新品匹配 | 收藏降价、偏好新品 |

### 地区配置

#### RegionConfig（地区配置）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | string (UUID) | PK | — |
| region_code | string | UNIQUE, NOT NULL | TW / VN / SG / MY 等 |
| featured_categories | string[] | NOT NULL | 该地区主推分类 |
| banner_style | enum | NOT NULL | minimal / colorful / festive |
| promotion_tags | jsonb | NULLABLE | 地区专属标签 |
| blocked_content_ratings | enum[] | DEFAULT [] | 该地区屏蔽的内容等级 |
| supported_payment_methods | string[] | NOT NULL | 该地区可用支付方式 |
| locale | string | NOT NULL | 默认语言 |

### 地区推广风格

| 地区 | 推广风格 | 热门品类 |
|------|---------|---------|
| 台湾 | 简约日系、文青风 | 动漫周边、美妆、文具、潮牌 |
| 越南 | 活泼多彩、促销导向 | 电子配件、美妆、母婴、日用品 |
| 新加坡/马来 | 高端精致 | 奢侈品、限定品、高端护肤 |

---

## 7. 指定购买升级 — 实时代理搜索

### 实时代理搜索架构

```
用户搜索（已解锁平台）
    │
    ├── 已支持平台 → 搜索本地索引（现有逻辑）
    ├── 用户解锁平台 → 实时代理搜索
    │       ├── Crawler Engine 实时请求目标平台
    │       ├── 解析结果 → 标准化为 Product 格式
    │       ├── 不入库，标记 source_type = 'proxy_realtime'
    │       └── 翻译（缓存优先）
    │
    ▼
合并结果 → 返回用户
```

### 平台解锁机制

#### UnlockedPlatform（用户解锁平台）

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | string (UUID) | PK | — |
| user_id | string | FK → User | — |
| platform_domain | string | NOT NULL | 如 `yahoo-auction.jp` |
| platform_name | string | NOT NULL | 显示名称 |
| unlocked_at | datetime | NOT NULL | 首次指定购买该平台的时间 |
| order_count | integer | NOT NULL, DEFAULT 1 | 该平台历史订单数 |

**约束**: `(user_id, platform_domain) UNIQUE`

**触发**: OrderLink 进入 `paid` 状态后自动创建记录。

### 实体变更

**Product**: `source_type enum NOT NULL DEFAULT 'indexed'` (`indexed` / `proxy_realtime`)

### 实时代理商品限制

| 功能 | 已入库商品 | 实时代理商品 |
|------|----------|------------|
| 搜索 | ✅ | ✅ |
| 商品详情 | ✅ | ✅（实时拉取） |
| 加入购物车 | ✅ | ❌（仅指定购买） |
| 收藏 | ✅ | ✅（缓存基本信息） |
| 下单方式 | 普通购买 | 指定购买（OrderLink） |

---

## 8. 站内信+工单自动化

### 三级策略

| 层级 | 策略 | 预估减少人工 |
|------|------|------------|
| **L0 自助** | FAQ + 智能搜索 | 30-40% |
| **L1 机器人** | LLM 客服机器人 | 20-30% |
| **L2 人工** | 机器人无法解决时转人工 | 剩余 |

### 智能客服流程

```
用户发起咨询 → 自动分类（LLM）
    ├── 订单相关 → 自动查询+回复
    ├── 物流相关 → 自动查询+回复
    ├── 退款相关 → 判断是否符合自动退款条件
    ├── 商品咨询 → LLM 回答
    └── 无法识别 → 转人工
```

### 自动化规则

| 规则 | 触发条件 | 自动动作 |
|------|---------|---------|
| 自动关闭 | resolved 7 天无回复 | status → closed |
| 自动催促 | in_progress 48h 无更新 | 通知处理人 |
| 自动升级 | open 24h 未认领 | 通知管理员 |
| 自动退款 | refund 类 + paid 状态 + 金额<阈值 | 自动退款+关单 |
| 重复合并 | 同用户 24h 同 order_id | 合并工单 |

### Ticket 实体变更

新增: `auto_resolved boolean`, `bot_conversation jsonb`, `intent string`, `priority enum (low/normal/high/urgent)`

---

## 9. 高价值商品全流程优化

### 判定条件

| 条件 | 阈值 |
|------|------|
| 商品单价 | ≥ ¥50,000 |
| 用户手动标记 | 下单时勾选"贵重物品" |
| 分类规则 | 奢侈品/手表/珠宝/限定品 |

### 实体变更

**Product**: `is_high_value boolean DEFAULT false`

**Order**: `is_high_value boolean DEFAULT false`, `insurance_jpy integer NULLABLE`, `authentication_status enum NULLABLE (pending/passed/failed)`

### 全流程

```
采购 → 拍照/视频存档
入库 → 强制质检 + 可选鉴定 + 安全库位
打包 → 加固包装 + 全程拍照 + 保价标记
物流 → 强制保价 + 独立单号 + 优先发货
售后 → 专属客服(urgent) + 实物证据
```

### 防欺诈风控

| 规则 | 触发条件 | 动作 |
|------|---------|------|
| 新用户大额 | 注册 <7d + ≥ ¥100,000 | 人工审核 |
| 频繁高额 | 24h ≥ 3 笔高价值 | 人工审核 |
| 异常地址 | 收货地址与注册地区不一致 | 风险提示 |
| 支付异常 | 多次失败后换方式成功 | 标记观察 |

---

## 10. 站点风格统一 + 系统稳定性

### 统一项

| 项目 | 说明 |
|------|------|
| 导航结构 | 所有站点相同 Tab 布局、页面层级 |
| 交互模式 | 统一搜索、筛选、结算流程 |
| 组件库 | 共享 Design System |
| 功能布局 | 页面结构固定 |

### 差异化项（由 RegionConfig 控制）

色彩/主题、首页内容、语言、支付方式、内容分级

### 系统稳定性

| 策略 | 说明 |
|------|------|
| 微服务隔离 | WMS/LiveStream 独立部署 |
| 熔断降级 | 翻译→原文、推荐→热门、直播→不影响购物 |
| 异步解耦 | 通知/翻译/推荐/WMS 通过消息队列 |
| 数据库保护 | 读写分离、连接池、慢查询告警 |
| 自动扩容 | 关键服务水平扩展 |
| 健康检查 | `/health` + 异常自动重启 |
| 限流保护 | API Gateway 按用户/IP 限流 |

### 新增 NFR

| NFR | 指标 |
|-----|------|
| 系统可用性 | ≥ 99.9% |
| 故障隔离 | 单服务宕机不影响下单/支付 |
| 故障恢复 | < 5min |
