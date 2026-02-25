# Rakutao International - Architecture Design Document

> **Version**: 2.0
> **Date**: 2026-02-24
> **Status**: Draft
> **Related**: [Project Brief](./project-brief.md) | [PRD](./prd.md)
>
> **Changelog**:
> - v2.0 (2026-02-24): 新增 WMS 仓储、代购直播、直接支付、内容分级、源站缓存容灾、个性化推荐、智能客服、高价值商品等 v2 模块的前端架构设计；订单状态机更新为 10 态；新增 ADR-005~009
> - v1.0 (2026-02-11): MVP 初版架构设计

---

## 1. Architecture Overview

### 1.1 System Context

Rakutao 是一个日本商品聚合代购平台，v2.0 系统由四大层级组成：

```
┌─────────────────────────────────────────────────────────────────────┐
│                          End Users                                   │
│             (Vietnam / Taiwan / SEA / Global, iOS & Android)         │
└────────────────────────────┬────────────────────────────────────────┘
                             │ HTTPS / WebSocket (LiveStream)
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Rakutao Mobile App                               │
│                (React Native / Cross-Platform)                       │
│                                                                      │
│  ┌──────┐ ┌──────────┐ ┌──────┐ ┌────────┐ ┌────────┐ ┌────────┐  │
│  │ Home │ │Favorites │ │ Cart │ │Order   │ │ Wallet │ │Live    │  │
│  │      │ │          │ │      │ │Link    │ │        │ │Stream  │  │
│  └──────┘ └──────────┘ └──────┘ └────────┘ └────────┘ └────────┘  │
│  ┌─────────────┐ ┌─────────────┐ ┌──────────┐ ┌──────────────┐    │
│  │Content      │ │Payment      │ │WMS       │ │Smart         │    │
│  │Rating Gate  │ │Method Select│ │Packing   │ │Support Chat  │    │
│  └─────────────┘ └─────────────┘ └──────────┘ └──────────────┘    │
└────────────────────────────┬────────────────────────────────────────┘
                             │ REST API / WebSocket / HLS
                             ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     Rakutao Backend (API Gateway)                    │
│                                                                      │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │
│  │ Product  │ │  Order   │ │  Wallet  │ │  User    │ │  WMS     │ │
│  │ Service  │ │  Service │ │  Service │ │  Service │ │  Service │ │
│  └────┬─────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘ │
│       │                                                              │
│  ┌────▼─────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ │
│  │ Crawler  │ │Translat. │ │Notificat.│ │LiveStream│ │Recommend │ │
│  │ Engine + │ │ Service  │ │ Service  │ │ Service  │ │ Engine   │ │
│  │ Cache    │ │  (LLM)   │ │          │ │(WebRTC)  │ │          │ │
│  └────┬─────┘ └────┬─────┘ └──────────┘ └──────────┘ └──────────┘ │
│       │            │                                                 │
│  ┌────▼─────┐ ┌────▼─────┐ ┌──────────┐ ┌──────────┐              │
│  │Source    │ │Smart     │ │ Payment  │ │Content   │              │
│  │Health    │ │Support   │ │ Gateway  │ │Rating    │              │
│  │Monitor  │ │ (LLM)    │ │          │ │ Service  │              │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘              │
└────────┬──────────┬──────────────┬──────────────────────────────────┘
         │          │              │
         ▼          ▼              ▼
┌──────────────┐ ┌──────────┐ ┌──────────────┐
│ JP E-Commerce│ │ LLM API  │ │Payment       │
│  Platforms   │ │(Translat.│ │Providers     │
│              │ │+ Chat)   │ │              │
│ · Mercari    │ └──────────┘ │ · Apple Pay  │
│ · Rakuma     │              │ · Google Pay │
│ · TOBU       │              │ · PayPal     │
│ · SuperDel.  │              │ · MoMo/Zalo  │
│ · netsea     │              └──────────────┘
│ · BEAMS ...  │
└──────────────┘
```

### 1.2 Scope Boundary

| 层级 | 本项目范围 | 说明 |
|------|----------|------|
| **Mobile App** | **在范围内** | 跨平台移动端前端，全部页面和交互 |
| Backend API | 范围外 | 由独立后端团队提供，本文档定义 API 契约 |
| Crawler Engine + Cache | 范围外 | 数据抓取引擎 + 源站缓存层，后端团队负责 |
| LLM Translation + Chat | 范围外 | 翻译 + 智能客服 LLM 服务，后端封装后提供给前端 |
| WMS Service | 范围外 | 仓储管理微服务（入库/打包/发货），后端团队负责 |
| LiveStream Service | 范围外 | 直播推流/拉流服务（WebRTC/HLS），后端团队负责 |
| Payment Gateway | 范围外 | 支付网关（Apple Pay/Google Pay/PayPal 集成），后端团队负责 |
| Content Rating Service | 范围外 | 内容分级标记服务，后端团队负责 |
| Recommendation Engine | 范围外 | 个性化推荐引擎，后端团队负责 |

本文档以 **Mobile App 架构** 为核心，同时定义与后端的 API 契约边界。v2.0 新增 WMS 用户交互、直播观看、直接支付、内容分级、源站缓存降级等前端模块。

---

## 2. Technology Stack

### 2.1 Mobile App

| 层级 | 技术选型 | 选型理由 |
|------|---------|---------|
| **框架** | React Native 0.76+ | 跨平台（iOS/Android），社区生态成熟，适合中等复杂度电商应用；团队复用 Web 前端经验 |
| **语言** | TypeScript 5.x | 类型安全，提高代码质量和团队协作效率 |
| **导航** | React Navigation 7 | React Native 生态标准方案，支持 Tab / Stack / Modal 导航组合 |
| **状态管理** | Zustand + React Query | Zustand 管理客户端状态（购物车/语言/用户偏好），React Query 管理服务端状态（商品/订单/余额） |
| **网络层** | Axios + React Query | Axios 处理 HTTP 请求，React Query 提供缓存/重试/乐观更新 |
| **实时通信** | WebSocket (socket.io-client) | 直播弹幕、订单状态实时推送、源站健康状态推送 |
| **视频播放** | react-native-video + HLS | 直播观看（HLS 低延迟拉流）、直播回放播放 |
| **支付 SDK** | Apple Pay / Google Pay / PayPal SDK | Apple Pay: `@stripe/stripe-react-native`；Google Pay: `@stripe/stripe-react-native`；PayPal: `@paypal/react-native-checkout` |
| **本地存储** | MMKV + SQLite (expo-sqlite) | MMKV: 高性能 KV 存储（替代 AsyncStorage），用于源站缓存数据、离线商品浏览；SQLite: 结构化离线数据（搜索历史、收藏缓存） |
| **i18n** | react-i18next | 成熟的国际化方案，支持动态切换、命名空间、插值 |
| **安全存储** | react-native-keychain | 存储 auth token 等敏感信息到系统级安全区域 |
| **图片加载** | React Native Fast Image | 高性能图片加载，支持缓存和渐进式显示 |
| **表单** | React Hook Form + Zod | 轻量表单管理 + schema 校验（登录/注册/地址/工单） |
| **行为追踪** | 自定义 Analytics SDK | 用户行为事件采集（浏览/搜索/收藏/购买），用于推荐引擎（FR-REC-006），需用户同意（DPC-006） |

### 2.2 工程工具链

| 工具 | 用途 |
|------|------|
| **Expo** (managed workflow) | 构建和发布工具，简化 iOS/Android 构建流程 |
| **ESLint + Prettier** | 代码规范和格式化 |
| **Jest + React Native Testing Library** | 单元测试和组件测试 |
| **Detox** (可选) | E2E 端到端测试 |
| **Storybook React Native** | 组件开发和 UI 文档 |

---

## 3. App Architecture

### 3.1 分层架构

采用 **Feature-First Clean Architecture**，按业务功能模块组织，每个模块内部分层：

```
src/
├── app/                          # App 入口和全局配置
│   ├── App.tsx                   # 根组件
│   ├── navigation/               # 导航配置
│   │   ├── RootNavigator.tsx     # 根导航（Auth vs Main）
│   │   ├── MainTabNavigator.tsx  # 底部 5 Tab 导航
│   │   ├── HomeStack.tsx         # Home Tab 内的 Stack 导航
│   │   ├── CartStack.tsx
│   │   ├── WalletStack.tsx
│   │   ├── FavoritesStack.tsx
│   │   └── OrderLinkStack.tsx
│   └── providers/                # 全局 Provider（QueryClient, i18n, Theme）
│
├── features/                     # 业务功能模块（核心）
│   ├── auth/                     # 认证模块
│   │   ├── screens/              # 页面组件
│   │   │   ├── SplashScreen.tsx
│   │   │   ├── LanguageSelectScreen.tsx
│   │   │   ├── LoginScreen.tsx
│   │   │   ├── RegisterScreen.tsx
│   │   │   └── PlatformLinkScreen.tsx
│   │   ├── api/                  # API 调用
│   │   ├── hooks/                # 自定义 hooks
│   │   ├── stores/               # Zustand stores
│   │   └── types/                # TypeScript 类型
│   │
│   ├── home/                     # 首页模块
│   │   ├── screens/
│   │   │   ├── HomeScreen.tsx
│   │   │   └── SearchResultScreen.tsx
│   │   ├── components/           # 模块私有组件
│   │   │   ├── ProductGrid.tsx
│   │   │   ├── CategoryTabs.tsx
│   │   │   └── PlatformSection.tsx
│   │   ├── api/
│   │   ├── hooks/
│   │   └── types/
│   │
│   ├── product/                  # 商品详情模块
│   │   ├── screens/
│   │   │   └── ProductDetailScreen.tsx
│   │   ├── components/
│   │   │   ├── ImageCarousel.tsx
│   │   │   ├── PriceDisplay.tsx
│   │   │   ├── VariantSelector.tsx
│   │   │   ├── ShippingInfo.tsx
│   │   │   ├── SellerInfo.tsx
│   │   │   └── RecommendedProducts.tsx
│   │   ├── api/
│   │   └── hooks/
│   │
│   ├── cart/                     # 购物车模块
│   │   ├── screens/
│   │   │   └── CartScreen.tsx
│   │   ├── components/
│   │   │   ├── CartItemCard.tsx
│   │   │   ├── SellerGroup.tsx
│   │   │   ├── PriceChangeIndicator.tsx
│   │   │   └── CartSummaryBar.tsx
│   │   ├── api/
│   │   ├── hooks/
│   │   └── stores/               # 购物车本地状态
│   │
│   ├── order-link/               # 指定购买模块
│   │   ├── screens/
│   │   │   └── OrderLinkScreen.tsx
│   │   ├── components/
│   │   │   ├── LinkInput.tsx
│   │   │   ├── ProcessGuide.tsx
│   │   │   └── PlatformLogos.tsx
│   │   └── api/
│   │
│   ├── wallet/                   # 钱包模块
│   │   ├── screens/
│   │   │   ├── WalletScreen.tsx
│   │   │   ├── OrderListScreen.tsx
│   │   │   ├── OrderDetailScreen.tsx
│   │   │   ├── ShipmentDetailScreen.tsx
│   │   │   └── TransactionHistoryScreen.tsx
│   │   ├── components/
│   │   │   ├── BalanceCard.tsx
│   │   │   ├── OrderCard.tsx
│   │   │   ├── OrderStatusTabs.tsx
│   │   │   └── OrderActions.tsx
│   │   ├── api/
│   │   └── hooks/
│   │
│   ├── favorites/                # 收藏夹模块
│   │   ├── screens/
│   │   │   └── FavoritesScreen.tsx
│   │   ├── components/
│   │   │   ├── FavoriteItemCard.tsx
│   │   │   ├── FollowSellerCard.tsx
│   │   │   └── BatchEditBar.tsx
│   │   ├── api/
│   │   └── hooks/
│   │
│   ├── profile/                  # 个人中心模块
│   │   ├── screens/
│   │   │   ├── ProfileScreen.tsx
│   │   │   ├── AddressListScreen.tsx
│   │   │   ├── AddressEditScreen.tsx
│   │   │   ├── ChangePasswordScreen.tsx
│   │   │   ├── LanguageSettingsScreen.tsx
│   │   │   ├── UserGuideScreen.tsx
│   │   │   ├── TicketScreen.tsx
│   │   │   ├── AdultContentSettingsScreen.tsx   # v2: 成人内容开关 + 年龄验证
│   │   │   ├── TrackingPreferencesScreen.tsx    # v2: 行为追踪授权管理
│   │   │   ├── DataExportScreen.tsx             # v2: 数据导出申请
│   │   │   └── AccountDeletionScreen.tsx        # v2: 账号删除
│   │   ├── api/
│   │   └── hooks/
│   │
│   ├── notifications/            # 通知模块
│   │   ├── screens/
│   │   │   └── NotificationListScreen.tsx
│   │   ├── api/
│   │   └── hooks/
│   │
│   ├── livestream/               # v2: 代购直播模块
│   │   ├── screens/
│   │   │   ├── LiveListScreen.tsx            # 直播列表（进行中/即将开始）
│   │   │   ├── LiveRoomScreen.tsx            # 直播间（视频流 + 弹幕 + 商品卡片）
│   │   │   └── LiveReplayScreen.tsx          # 直播回放
│   │   ├── components/
│   │   │   ├── LiveCard.tsx                  # 直播卡片（封面/状态/人数）
│   │   │   ├── DanmakuOverlay.tsx            # 弹幕浮层
│   │   │   ├── LiveProductCard.tsx           # 主播标记的商品浮卡
│   │   │   └── LiveStreamPlayer.tsx          # HLS/WebRTC 视频播放器
│   │   ├── api/
│   │   ├── hooks/
│   │   └── stores/                           # 直播间实时状态
│   │
│   ├── content-rating/           # v2: 内容分级模块
│   │   ├── screens/
│   │   │   └── AgeGateScreen.tsx             # 年龄验证门弹窗
│   │   ├── components/
│   │   │   ├── ContentRatingBadge.tsx        # G/M/A 标记组件
│   │   │   └── AgeVerificationForm.tsx       # 出生日期输入表单
│   │   ├── hooks/
│   │   │   └── useContentFilter.ts           # 按内容等级 + 用户验证状态过滤
│   │   └── stores/                           # 年龄验证状态、成人内容开关
│   │
│   ├── payment/                  # v2: 支付方式模块
│   │   ├── screens/
│   │   │   ├── CheckoutScreen.tsx            # 结算页（选择支付方式 + 确认）
│   │   │   └── PaymentResultScreen.tsx       # 支付结果页
│   │   ├── components/
│   │   │   ├── PaymentMethodList.tsx         # 支付方式列表（按地区/设备过滤）
│   │   │   ├── WalletPayOption.tsx           # 钱包余额支付选项
│   │   │   └── DirectPayOption.tsx           # Apple Pay / Google Pay / PayPal 选项
│   │   ├── api/
│   │   ├── hooks/
│   │   │   └── usePaymentMethods.ts          # 获取可用支付方式
│   │   └── stores/                           # 上次支付方式偏好
│   │
│   ├── wms/                      # v2: WMS 仓储用户交互模块
│   │   ├── screens/
│   │   │   ├── PackingRequestScreen.tsx      # 申请打包（勾选已入库商品）
│   │   │   └── ShippingFeeConfirmScreen.tsx  # 运费确认与支付
│   │   ├── components/
│   │   │   ├── WarehousedItemCard.tsx        # 已入库商品卡片
│   │   │   ├── PackingStatusBadge.tsx        # 打包状态标签
│   │   │   └── ShippingFeeBreakdown.tsx      # 运费明细（重量/体积/目的地）
│   │   ├── api/
│   │   └── hooks/
│   │
│   ├── smart-support/            # v2: 智能客服模块
│   │   ├── screens/
│   │   │   ├── SupportEntryScreen.tsx        # FAQ + LLM 客服入口
│   │   │   └── ChatScreen.tsx                # LLM 客服对话界面
│   │   ├── components/
│   │   │   ├── FAQSearchBar.tsx
│   │   │   ├── ChatBubble.tsx                # 对话气泡（用户/机器人/系统）
│   │   │   └── RichNotificationCard.tsx      # 富通知卡片（订单/物流嵌入）
│   │   ├── api/
│   │   └── hooks/
│   │
│   └── source-cache/             # v2: 源站缓存状态模块
│       ├── components/
│       │   ├── CacheModeIndicator.tsx         # 缓存模式提示（"数据更新于X分钟前"）
│       │   ├── StaleDataWarning.tsx           # 数据过期橙色警告标签
│       │   ├── PendingVerificationBanner.tsx  # 结算页源站确认提示
│       │   └── SourceHealthStatus.tsx         # 各源平台运行状态一览
│       ├── hooks/
│       │   ├── useSourceHealth.ts             # 监听源站健康状态
│       │   └── useCachedProduct.ts            # 判断商品数据是否来自缓存
│       └── stores/                            # 源站健康状态（各平台可用性）
│
├── shared/                       # 跨模块共享代码
│   ├── components/               # 通用 UI 组件（设计系统）
│   │   ├── buttons/
│   │   │   ├── LargeButton.tsx   # 93x48px, radius:12
│   │   │   ├── MediumButton.tsx  # 120x40px, radius:20
│   │   │   └── SmallButton.tsx   # 90x40px, radius:5
│   │   ├── cards/
│   │   │   ├── ProductCard.tsx   # 345x168px, radius:8
│   │   │   ├── OrderCard.tsx     # 345x235px, radius:8
│   │   │   └── ProductThumbnail.tsx # 114x114px, radius:10
│   │   ├── tags/
│   │   │   ├── ShippingTag.tsx   # H:21px, radius:5
│   │   │   ├── CapsuleTag.tsx    # H:25px, radius:20 (Translation)
│   │   │   └── BadgeDot.tsx      # 红点标识
│   │   ├── modals/
│   │   │   ├── CenterModal.tsx   # 320x245px, radius:32
│   │   │   └── BottomSheet.tsx   # W:375px, radius:8
│   │   ├── inputs/
│   │   ├── EmptyState.tsx
│   │   └── LoadingIndicator.tsx
│   │
│   ├── hooks/                    # 通用 hooks
│   │   ├── useAuth.ts
│   │   ├── useLanguage.ts
│   │   ├── useNetworkStatus.ts
│   │   ├── useRegionConfig.ts    # v2: 地区配置（支付方式/内容策略/风格）
│   │   └── useWebSocket.ts       # v2: WebSocket 连接管理
│   │
│   ├── services/                 # 通用服务
│   │   ├── api/
│   │   │   ├── client.ts         # Axios 实例配置
│   │   │   ├── interceptors.ts   # 请求/响应拦截器（auth token, error handling）
│   │   │   ├── wsClient.ts       # v2: WebSocket 客户端（直播弹幕/实时推送）
│   │   │   └── types.ts          # API 通用类型
│   │   ├── storage/
│   │   │   ├── secureStorage.ts  # Keychain 封装
│   │   │   ├── mmkvStorage.ts    # v2: MMKV 封装（高性能 KV 缓存）
│   │   │   └── asyncStorage.ts   # AsyncStorage 封装（向后兼容）
│   │   ├── payment/              # v2: 支付服务
│   │   │   ├── applePayService.ts
│   │   │   ├── googlePayService.ts
│   │   │   └── paypalService.ts
│   │   └── analytics/            # 埋点服务（含行为追踪同意管理）
│   │
│   ├── i18n/                     # 国际化配置
│   │   ├── index.ts              # i18next 初始化
│   │   ├── ja.json               # 日语（默认）
│   │   ├── zh-Hant.json          # 繁体中文
│   │   └── en.json               # 英语
│   │
│   ├── theme/                    # 设计系统 Token
│   │   ├── colors.ts             # #BF1E16, #2A82E4, #383838, #A6A6A6, #F5F5F5, #FFFFFF
│   │   ├── typography.ts         # 18/16/14/12px
│   │   ├── spacing.ts
│   │   └── index.ts
│   │
│   ├── constants/                # 常量
│   │   ├── orderStatus.ts        # 订单 10 态枚举（Pending→Fulfilled + Cancelled/Refunded）
│   │   ├── contentRating.ts      # v2: 内容分级常量（G/M/A）
│   │   ├── paymentMethods.ts     # v2: 支付方式常量
│   │   └── platforms.ts          # 支持平台列表
│   │
│   └── utils/                    # 工具函数
│       ├── currency.ts           # JPY 格式化
│       ├── date.ts               # 日期格式化
│       └── validation.ts         # 通用校验
│
└── types/                        # 全局类型定义
    └── global.d.ts
```

### 3.2 导航架构

```
RootNavigator
├── AuthStack (未登录)
│   ├── SplashScreen
│   ├── LanguageSelectScreen
│   ├── LoginScreen
│   ├── RegisterScreen
│   └── PlatformLinkScreen
│
├── AgeGateModal (全局 Modal)               ← v2: 年龄验证门，任意页面可触发
│
└── MainTabNavigator (已登录, 5 Tabs)
    ├── HomeStack (Tab: Home)
    │   ├── HomeScreen
    │   ├── SearchResultScreen
    │   ├── ProductDetailScreen ←──── 共享（含 ContentRatingBadge）
    │   ├── NotificationListScreen
    │   ├── LiveListScreen                   ← v2: 直播列表入口
    │   └── SourceHealthScreen               ← v2: 源站状态一览
    │
    ├── FavoritesStack (Tab: Favorites)
    │   ├── FavoritesScreen
    │   └── ProductDetailScreen ←──── 共享
    │
    ├── CartStack (Tab: Cart)
    │   ├── CartScreen（含 CacheModeIndicator）
    │   └── CheckoutScreen → PaymentMethodSelect → PaymentResult  ← v2
    │
    ├── OrderLinkStack (Tab: Order Link)
    │   ├── OrderLinkScreen          ← Tab 切换: Order Link / Cart
    │   └── CartScreen               ← 复用
    │
    └── WalletStack (Tab: Wallet)
        ├── WalletScreen
        ├── ChargeScreen
        ├── OrderListScreen（含 10 态 Tab + Cancelled&Refunded 合并 Tab）
        ├── OrderDetailScreen
        ├── PackingRequestScreen             ← v2: 申请打包
        ├── ShippingFeeConfirmScreen         ← v2: 运费确认与支付
        ├── ShipmentDetailScreen
        ├── TransactionHistoryScreen
        └── AuctionListScreen

    ┌── LiveStreamStack (从 HomeStack 推入)   ← v2
    │   ├── LiveRoomScreen (全屏直播间)
    │   └── LiveReplayScreen (直播回放)
    │
    ├── SmartSupportStack (从 Profile 推入)    ← v2
    │   ├── SupportEntryScreen (FAQ + LLM 入口)
    │   └── ChatScreen (LLM 客服对话)
    │
    └── ProfileStack (从 WalletStack 推入)
        ├── ProfileScreen
        ├── AddressListScreen / AddressEditScreen
        ├── ChangePasswordScreen
        ├── LanguageSettingsScreen
        ├── AdultContentSettingsScreen       ← v2
        ├── TrackingPreferencesScreen        ← v2
        ├── DataExportScreen                 ← v2
        └── AccountDeletionScreen            ← v2
```

**共享页面处理**：ProductDetailScreen 可从 Home、Favorites、Cart、LiveStream 等多个入口进入。使用 React Navigation 的 shared screen 或在每个 Stack 中注册同一组件，保证返回栈的正确行为。AgeGateModal 注册为全局 Modal，任何需要年龄验证的场景统一调用。

### 3.3 状态管理架构

```
┌──────────────────────────────────────────────────────────┐
│                       UI Layer                            │
│             (Screens & Components)                        │
└──────────┬────────────────┬────────────────┬─────────────┘
           │                │                │
     ┌─────▼─────┐   ┌─────▼──────┐   ┌─────▼──────┐
     │  Zustand   │   │React Query │   │ WebSocket  │
     │  Stores    │   │  Queries   │   │  Events    │
     │            │   │            │   │            │
     │ · auth     │   │ · products │   │ · live     │
     │ · language │   │ · orders   │   │   chat     │
     │ · cart UI  │   │ · wallet   │   │ · order    │
     │ · content  │   │ · favorites│   │   updates  │
     │   rating   │   │ · search   │   │ · source   │
     │ · source   │   │ · tickets  │   │   health   │
     │   health   │   │ · live-    │   │            │
     │ · payment  │   │   streams  │   └─────┬──────┘
     │   pref     │   │ · payment  │         │
     │ · tracking │   │   methods  │         │ WSS
     │   consent  │   │ · recommend│         │
     │ · ui state │   │ · source   │         │
     │            │   │   health   │         │
     └────────────┘   └─────┬──────┘         │
                            │                │
                      ┌─────▼──────┐   ┌─────▼──────┐
                      │  API Layer │   │  WS Layer  │
                      │  (Axios)   │   │ (socket.io)│
                      └─────┬──────┘   └─────┬──────┘
                            │ HTTPS          │ WSS
                            ▼                ▼
                         Backend API + WebSocket Server
```

**职责划分**：

| 状态类型 | 管理方案 | 示例 |
|---------|---------|------|
| **服务端数据** | React Query | 商品列表、订单列表、余额、收藏列表、通知列表、直播列表、推荐商品、支付方式、源站健康 |
| **客户端状态** | Zustand | 当前语言、auth token、购物车选中状态、年龄验证状态、成人内容开关、行为追踪授权、支付偏好、源站健康缓存、UI 开关 |
| **实时数据** | WebSocket Events | 直播弹幕流、订单状态推送、源站健康变更推送 |
| **表单状态** | React Hook Form | 登录/注册表单、地址编辑、工单创建、年龄验证表单 |
| **导航状态** | React Navigation | 当前路由、路由参数 |
| **持久化状态** | MMKV / Keychain | 语言偏好、离线商品缓存、搜索历史（MMKV）；auth token（Keychain） |

### 3.4 Key Zustand Stores

```typescript
// stores/authStore.ts
interface AuthState {
  token: string | null;
  user: User | null;
  isAuthenticated: boolean;
  login: (token: string, user: User) => void;
  logout: () => void;
}

// stores/languageStore.ts
interface LanguageState {
  locale: 'ja' | 'zh-Hant' | 'en';
  isFirstLaunch: boolean;
  setLocale: (locale: Locale) => void;
  setFirstLaunchDone: () => void;
}

// stores/cartUIStore.ts
interface CartUIState {
  selectedItems: Set<string>;     // 选中的商品 ID
  isEditMode: boolean;
  toggleItem: (id: string) => void;
  selectAll: () => void;
  clearSelection: () => void;
}

// v2: stores/contentRatingStore.ts
interface ContentRatingState {
  isAgeVerified: boolean;             // 是否已通过年龄验证
  adultContentEnabled: boolean;       // 成人内容开关（默认 false）
  verificationFailedUntil: string | null; // 验证失败后 24h 冷却期
  setAgeVerified: (verified: boolean) => void;
  setAdultContentEnabled: (enabled: boolean) => void;
}

// v2: stores/sourceHealthStore.ts
interface SourceHealthState {
  platformStatus: Record<string, PlatformHealthStatus>; // 各平台健康状态
  isCacheMode: boolean;               // 是否处于缓存模式
  updatePlatformStatus: (platform: string, status: PlatformHealthStatus) => void;
}

type PlatformHealthStatus = 'healthy' | 'degraded' | 'unavailable';

// v2: stores/paymentPreferenceStore.ts
interface PaymentPreferenceState {
  lastUsedMethod: PaymentMethodType | null;  // 上次使用的支付方式
  setLastUsedMethod: (method: PaymentMethodType) => void;
}

// v2: stores/trackingConsentStore.ts
interface TrackingConsentState {
  behaviorTrackingEnabled: boolean;   // 行为追踪授权（默认 false，需用户同意）
  setBehaviorTracking: (enabled: boolean) => void;
}
```

---

## 4. API Contract Design

### 4.1 API Base

```
Base URL:  https://api.rakutao.com/v1
Auth:      Bearer token in Authorization header
Format:    JSON
Encoding:  UTF-8
```

### 4.2 Common Response Envelope

```typescript
interface ApiResponse<T> {
  code: number;          // 业务状态码，0 = 成功
  message: string;       // 状态描述
  data: T;               // 业务数据
}

interface PaginatedResponse<T> {
  code: number;
  message: string;
  data: {
    items: T[];
    total: number;
    page: number;
    pageSize: number;
    hasMore: boolean;
  };
}
```

### 4.3 Core API Endpoints

#### Authentication

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| POST | `/auth/register` | 邮箱注册 | FR-AUTH-002 |
| POST | `/auth/login` | 邮箱密码登录 | FR-AUTH-001 |
| POST | `/auth/social-login` | 社交账号登录 | FR-AUTH-003 |
| POST | `/auth/forgot-password` | 发送重置密码邮件 | FR-AUTH-005 |
| POST | `/auth/reset-password` | 重置密码 | FR-AUTH-005 |
| POST | `/auth/logout` | 登出 | FR-AUTH-007 |
| POST | `/auth/link-platform` | 关联第三方平台账号 | FR-AUTH-006 |
| GET | `/auth/me` | 获取当前用户信息（Token 验证） | FR-AUTH-001 |
| POST | `/auth/refresh` | 刷新 Access Token | FR-AUTH-001 |

#### Products

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/products` | 商品列表（分页、分类筛选） | FR-HOME-001 |
| GET | `/products/search` | 搜索商品（含翻译） | FR-HOME-002 |
| GET | `/products/:id` | 商品详情 | FR-PD-* |
| GET | `/products/:id/variants` | 商品规格列表 | FR-PD-006 |
| GET | `/products/recommended` | 推荐商品 | FR-HOME-007, FR-PD-011 |
| GET | `/platforms` | 可用平台列表 | FR-HOME-004 |
| GET | `/categories` | 分类标签列表 | FR-HOME-003 |
| GET | `/exchange-rate` | 当前汇率 | FR-PD-003 |

#### Cart

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/cart` | 获取购物车列表 | FR-CART-001 |
| POST | `/cart/items` | 添加商品到购物车 | FR-PD-012 |
| DELETE | `/cart/items/:id` | 移除购物车商品 | FR-CART-009 |
| POST | `/cart/checkout` | 结算下单 | FR-CART-008 |
| POST | `/cart/archive-sold` | 归档已售罄商品 | FR-CART-004 |

#### Order Link (指定购买)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| POST | `/order-links` | 提交指定购买请求 | FR-OL-001, FR-OL-003 |
| GET | `/order-links` | 获取指定购买列表 | FR-OL-007 |
| GET | `/order-links/platforms` | 支持的平台列表 | FR-OL-005 |

#### Wallet

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/wallet/balance` | 获取余额 | FR-WAL-001 |
| POST | `/wallet/charge` | 发起充值 | FR-WAL-002 |
| GET | `/wallet/transactions` | 流水记录列表 | FR-ORD-013 |

#### Orders

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/orders` | 订单列表（分页、状态筛选、时间筛选） | FR-ORD-001~003 |
| GET | `/orders/:id` | 订单详情 | FR-ORD-004 |
| POST | `/orders/:id/refund` | 申请退款 | FR-ORD-006 |
| POST | `/orders/:id/urge` | 催促发货 | FR-ORD-007 |
| POST | `/orders/:id/confirm-receipt` | 确认收货 | FR-ORD-008 |
| POST | `/orders/:id/additional-payment` | 补充付款 | FR-ORD-009 |
| GET | `/orders/:id/shipment` | 运单详情 | FR-ORD-010 |
| GET | `/orders/auctions` | 拍卖订单列表 | FR-ORD-012 |

#### Favorites

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/favorites/products` | 收藏商品列表 | FR-FAV-002 |
| POST | `/favorites/products/:id` | 收藏商品 | FR-FAV-001 |
| DELETE | `/favorites/products/:id` | 取消收藏 | FR-FAV-001 |
| DELETE | `/favorites/products` | 批量取消收藏 | FR-FAV-006 |
| GET | `/favorites/sellers` | 关注卖家列表 | FR-FAV-003 |
| POST | `/favorites/sellers/:id` | 关注卖家 | FR-FAV-003 |
| DELETE | `/favorites/sellers/:id` | 取消关注 | FR-FAV-003 |

#### Profile

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/profile` | 获取个人信息 | FR-PROF-001 |
| PUT | `/profile` | 更新个人信息 | FR-PROF-001 |
| PUT | `/profile/avatar` | 上传头像 | FR-PROF-002 |
| PUT | `/profile/email` | 修改绑定邮箱 | FR-PROF-003 |
| POST | `/profile/email/verify` | 邮箱验证码发送 | FR-PROF-003 |
| PUT | `/profile/password` | 修改密码 | FR-PROF-005 |
| GET | `/profile/addresses` | 收货地址列表 | FR-PROF-004 |
| POST | `/profile/addresses` | 新增收货地址 | FR-PROF-004 |
| PUT | `/profile/addresses/:id` | 更新收货地址 | FR-PROF-004 |
| DELETE | `/profile/addresses/:id` | 删除收货地址 | FR-PROF-004 |

#### Tickets

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/tickets` | 工单列表 | FR-PROF-010 |
| POST | `/tickets` | 创建工单 | FR-PROF-010 |
| GET | `/tickets/:id` | 工单详情 | FR-PROF-010 |

#### Notifications

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/notifications` | 通知列表 | FR-NOTI-001 |
| PUT | `/notifications/:id/read` | 标记已读 | FR-NOTI-001 |
| PUT | `/notifications/read-all` | 全部已读 | FR-SS-008 |
| PUT | `/notifications/read-by-type` | 按类型批量已读 | FR-SS-008 |
| GET | `/notifications/unread-count` | 未读数量 | FR-NOTI-002 |

#### Orders (v2 补充)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| POST | `/orders/:id/request-packing` | 申请打包（含合并打包商品 ID 列表） | FR-ORD-014 |
| POST | `/orders/:id/confirm-shipping-fee` | 确认并支付运费 | FR-ORD-015 |

#### WMS (v2 新增 — 用户侧)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/wms/warehoused-items` | 获取用户已入库商品列表 | FR-WMS-001 |
| POST | `/wms/packing-requests` | 提交打包请求（含商品 ID 列表） | FR-WMS-002 |
| GET | `/wms/packing-requests/:id` | 打包请求详情（状态/运费） | FR-WMS-003 |
| POST | `/wms/packing-requests/:id/pay-shipping` | 支付国际运费 | FR-WMS-004 |

#### Payment (v2 新增)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/payment/methods` | 获取当前用户可用支付方式（按地区/设备过滤） | FR-PAY-006 |
| POST | `/payment/create-intent` | 创建支付意图（商品款/运费/补差价） | FR-PAY-001~003 |
| POST | `/payment/confirm` | 确认支付（附 provider token） | FR-PAY-001~003 |
| GET | `/payment/result/:id` | 查询支付结果 | FR-PAY-001~003 |

#### LiveStream (v2 新增)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/livestreams` | 直播列表（进行中/即将开始/回放） | FR-LIVE-001 |
| GET | `/livestreams/:id` | 直播详情（含流地址） | FR-LIVE-002 |
| GET | `/livestreams/:id/products` | 直播关联商品列表 | FR-LIVE-004 |
| POST | `/livestreams/:id/order` | 直播间下单（创建 OrderLink） | FR-LIVE-005 |
| POST | `/livestreams/:id/subscribe` | 订阅直播预告通知 | FR-LIVE-006 |
| GET | `/livestreams/:id/replay` | 直播回放信息 | FR-LIVE-007 |

> **直播弹幕**：通过 WebSocket 通道 `wss://api.rakutao.com/ws/live/:id` 收发实时消息（FR-LIVE-003）

#### Content Rating (v2 新增)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| POST | `/content-rating/verify-age` | 提交年龄验证（出生日期） | FR-CR-002 |
| GET | `/content-rating/status` | 获取当前用户年龄验证状态和内容可见等级 | FR-CR-003 |
| PUT | `/content-rating/adult-content` | 切换成人内容开关 | FR-CR-003, FR-PROF-011 |

#### Source Site Health (v2 新增)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/source-health/status` | 各源平台运行状态一览 | FR-SRC-010 |
| GET | `/source-health/cache-info/:productId` | 商品缓存信息（更新时间/是否过期） | FR-SRC-003, FR-SRC-008 |

> **实时推送**：源站健康状态变更通过 WebSocket 推送至客户端，触发缓存模式切换（FR-SRC-002）

#### Recommendations (v2 新增)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/recommendations/home` | 首页个性化推荐商品 | FR-REC-001 |
| GET | `/recommendations/product/:id` | 商品详情页推荐 | FR-REC-002 |
| POST | `/recommendations/events` | 上报行为事件（需用户授权） | FR-REC-006 |
| GET | `/recommendations/preferences` | 获取用户推荐偏好 | FR-PROF-012 |
| PUT | `/recommendations/preferences` | 手动调整推荐偏好 | FR-PROF-012 |

#### Smart Support (v2 新增)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/support/faq` | FAQ 列表（含搜索） | FR-SS-001 |
| POST | `/support/chat` | LLM 客服对话（发送消息 + 获取回复） | FR-SS-002 |
| POST | `/support/chat/transfer-human` | 转人工客服 | FR-SS-004 |

#### Profile (v2 补充)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| POST | `/profile/data-export` | 申请数据导出 | FR-PROF-013, DPC-004 |
| GET | `/profile/data-export/:id` | 数据导出状态/下载链接 | FR-PROF-013 |
| POST | `/profile/delete-account` | 申请删除账号 | FR-PROF-014, DPC-005 |
| POST | `/profile/delete-account/cancel` | 撤回删除申请 | FR-PROF-014 |
| PUT | `/profile/tracking-consent` | 更新行为追踪授权 | FR-PROF-015, DPC-006 |

#### Region Config (v2 新增)

| Method | Path | Description | FR |
|--------|------|-------------|-----|
| GET | `/region-config` | 获取当前地区配置（品类/风格/支付/合规） | FR-REG-001~004 |

---

## 5. Data Models

### 5.1 Core Entities

```typescript
// === User ===
interface User {
  id: string;
  email: string;
  nickname: string;
  avatarUrl: string | null;
  birthday: string | null;        // ISO date
  gender: 'male' | 'female' | 'other' | null;
  locale: 'ja' | 'zh-Hant' | 'en';
  linkedPlatforms: PlatformLink[];
  isAgeVerified: boolean;         // v2: 年龄验证状态
  adultContentEnabled: boolean;   // v2: 成人内容开关
  behaviorTrackingConsent: boolean; // v2: 行为追踪授权
  region: string;                 // v2: 用户所在地区代码（TW/VN/SG 等）
  createdAt: string;              // ISO datetime
}

interface PlatformLink {
  platform: 'mercari' | 'rakuma' | 'rakutao';
  linkedAt: string;
  accountName: string;
}

// === Product ===
interface Product {
  id: string;
  title: string;                  // 原文（日语）
  titleTranslated: string | null; // LLM 翻译后文本
  description: string;
  descriptionTranslated: string | null;
  images: string[];               // 图片 URL 列表
  thumbnailUrl: string;
  priceJpy: number;               // JPY 价格
  originalPriceJpy: number | null;// 原价（如有折扣）
  exchangeRate: number;           // 参考汇率
  currency: 'JPY';
  tags: ProductTag[];             // Free Shipping / New / Freight Collect
  shippingDomestic: number;       // 日本国内运费（0 = 包邮）
  shippingInternational: number | null; // 国际运费（null = 按重量计算）
  serviceCharge: number;
  category: string[];             // 分类层级
  seller: Seller;
  status: 'available' | 'sold' | 'reserved';
  shipFrom: string;               // 发货地
  shippingTime: string;           // 预估发货时间
  shippingMethod: string;
  dimensions: ProductDimensions | null;
  variants: ProductVariant[];
  updatedAt: string;              // 最后更新时间
  isTranslated: boolean;          // 是否已翻译
  sourcePlatform: string;         // 后端知道，前端不展示
  contentRating: ContentRating;   // v2: 内容分级（G/M/A）
  isHighValue: boolean;           // v2: 是否高价值商品（≥¥50,000 或奢侈品分类）
  cacheInfo: ProductCacheInfo | null; // v2: 缓存信息（仅缓存模式下返回）
}

// v2: 内容分级
type ContentRating = 'G' | 'M' | 'A';

// v2: 商品缓存信息（源站不可用时返回）
interface ProductCacheInfo {
  cachedAt: string;               // 缓存时间
  isStale: boolean;               // 是否过期（>60min）
  sourcePlatformStatus: PlatformHealthStatus; // 源平台当前状态
}

type ProductTag = 'free_shipping' | 'freight_collect' | 'new' | 'sale' | 'high_value';

interface Seller {
  id: string;
  name: string;
  productCount: number;
  rating: number;
}

interface ProductVariant {
  id: string;
  name: string;                   // 如 "颜色"
  options: VariantOption[];
}

interface VariantOption {
  id: string;
  label: string;
  available: boolean;
  priceAdjustment: number;        // 价格差异
}

interface ProductDimensions {
  length: number;
  width: number;
  height: number;
  weight: number;
  unit: 'cm' | 'mm';
  weightUnit: 'g' | 'kg';
}

// === Cart ===
interface CartItem {
  id: string;
  product: Product;
  selectedVariant: VariantOption | null;
  quantity: number;
  addedPriceJpy: number;          // 加入时的价格（用于变动提示）
  currentPriceJpy: number;        // 当前价格
  priceChanged: boolean;          // 是否有价格变动
}

interface CartGroup {
  seller: Seller;
  items: CartItem[];
}

// === Order (v2: 10 态生命周期) ===
interface Order {
  id: string;                     // 如 "RO202509M1012542I"
  status: OrderStatus;
  subStatus: OrderSubStatus | null; // v2: 子状态（如 pending_verification）
  items: OrderItem[];
  totalJpy: number;
  shippingJpy: number | null;     // 国际运费（Packed 后填充）
  additionalPaymentJpy: number | null;
  paymentMethod: PaymentMethodType | null; // v2: 支付方式
  waybill: Waybill | null;
  packingRequest: PackingRequest | null; // v2: 打包请求信息
  cancellationReason: string | null;
  isHighValue: boolean;           // v2: 高价值商品标记
  isCachedOrder: boolean;         // v2: 是否缓存模式下单
  createdAt: string;
  paidAt: string | null;
  purchasedAt: string | null;     // v2: 采购完成时间
  warehousedAt: string | null;    // v2: 入库时间
  packedAt: string | null;        // v2: 打包完成时间
  shippedAt: string | null;
  fulfilledAt: string | null;
}

// v2: 订单 10 态（对应 PRD §11 术语表）
type OrderStatus =
  | 'pending'        // 待支付
  | 'paid'           // 已支付
  | 'purchasing'     // 采购中
  | 'warehoused'     // 已入库
  | 'packing'        // 打包中
  | 'packed'         // 已打包（待支付运费）
  | 'shipped'        // 已发货
  | 'fulfilled'      // 已送达
  | 'cancelled'      // 已取消
  | 'refunded';      // 已退款

// v2: 订单子状态
type OrderSubStatus =
  | 'pending_verification'; // 缓存模式下单，待源站恢复校验

interface OrderItem {
  productId: string;
  title: string;
  thumbnailUrl: string;
  quantity: number;
  weight: number;
  priceJpy: number;
}

interface Waybill {
  id: string;                     // 如 "LO20250814171543739P"
  carrier: string;
  status: string;
  trackingUrl: string | null;
  estimatedDelivery: string | null;
}

// === Wallet ===
interface WalletBalance {
  balanceJpy: number;
  lastUpdated: string;
}

interface Transaction {
  id: string;
  type: 'charge' | 'payment' | 'refund' | 'additional_payment' | 'shipping_fee'; // v2: 新增运费支付类型
  amountJpy: number;
  orderId: string | null;
  paymentMethod: PaymentMethodType | null; // v2: 支付方式
  description: string;
  createdAt: string;
}

// === Notification (v2: 扩展类型 + Rich 通知) ===
interface Notification {
  id: string;
  type: NotificationType;
  title: string;
  body: string;
  isRead: boolean;
  relatedId: string | null;       // 关联的订单/商品 ID
  richContent: RichNotificationContent | null; // v2: 富内容（订单卡片/物流卡片）
  createdAt: string;
}

type NotificationType =
  | 'order_status'     // 订单状态变更
  | 'price_change'     // 收藏商品降价
  | 'system'           // 系统公告
  | 'ticket_reply'     // 工单回复
  | 'shipping_fee'     // v2: 运费确认通知
  | 'source_recovery'  // v2: 源站恢复通知（缓存订单校验结果）
  | 'live_start'       // v2: 直播开始通知
  | 'new_product';     // v2: 偏好品类新品上架

interface RichNotificationContent {
  type: 'order_card' | 'shipping_card' | 'product_card';
  data: Record<string, unknown>;  // 嵌入的卡片数据
}

// === Address ===
interface ShippingAddress {
  id: string;
  recipientName: string;
  phone: string;
  country: string;
  state: string;
  city: string;
  addressLine1: string;
  addressLine2: string | null;
  postalCode: string;
  isDefault: boolean;
}

// === Ticket ===
interface Ticket {
  id: string;
  category: 'order' | 'refund' | 'shipping' | 'account' | 'product';
  subject: string;
  description: string;
  status: 'open' | 'in_progress' | 'resolved' | 'closed';
  priority: 'normal' | 'urgent';  // v2: 高价值商品工单自动标记 urgent
  createdAt: string;
  updatedAt: string;
}

// === v2 新增数据模型 ===

// v2: 直播
interface LiveStream {
  id: string;
  title: string;
  hostName: string;               // 主播名称
  coverImageUrl: string;
  status: 'upcoming' | 'live' | 'ended';
  viewerCount: number;
  scheduledAt: string | null;     // 预告时间
  startedAt: string | null;
  endedAt: string | null;
  streamUrl: string | null;       // HLS 拉流地址
  replayUrl: string | null;       // 回放地址
  products: LiveProduct[];        // 关联商品
  contentRating: ContentRating;   // 直播间内容分级
}

interface LiveProduct {
  id: string;
  productId: string | null;       // 关联已有商品（可能为临时标记）
  name: string;
  estimatedPriceJpy: number;
  photoUrl: string;               // 现场照片
  markedAt: string;               // 主播标记时间
}

// v2: 打包请求
interface PackingRequest {
  id: string;
  orderIds: string[];             // 合并打包的订单 ID 列表
  status: 'pending' | 'packing' | 'packed';
  actualWeight: number | null;    // 实际重量 (g)
  actualVolume: string | null;    // 实际体积
  shippingFeeJpy: number | null;  // 计算出的国际运费
  createdAt: string;
  packedAt: string | null;
}

// v2: 支付方式
type PaymentMethodType =
  | 'wallet'       // 钱包余额
  | 'apple_pay'    // Apple Pay
  | 'google_pay'   // Google Pay
  | 'paypal'       // PayPal
  | 'momo'         // 越南 MoMo
  | 'zalopay'      // 越南 ZaloPay
  | 'kakaopay';    // 韩国 KakaoPay

interface PaymentMethod {
  type: PaymentMethodType;
  label: string;                  // 展示名称
  iconUrl: string;
  available: boolean;             // 当前设备/地区是否可用
  walletBalance: number | null;   // 仅 wallet 类型时有值
}

// v2: 源站健康状态
interface SourcePlatformHealth {
  platform: string;               // 平台标识
  displayName: string;
  status: PlatformHealthStatus;   // healthy / degraded / unavailable
  lastCheckedAt: string;
  lastAvailableAt: string;        // 最后可用时间
  cachedProductCount: number;     // 当前缓存商品数量
}

type PlatformHealthStatus = 'healthy' | 'degraded' | 'unavailable';

// v2: 地区配置
interface RegionConfig {
  regionCode: string;             // TW / VN / SG / MY / US 等
  defaultLocale: 'ja' | 'zh-Hant' | 'en';
  availablePaymentMethods: PaymentMethodType[];
  bannerStyle: 'minimal' | 'vibrant' | 'festive';  // 推广视觉风格
  contentRestrictions: ContentRating[]; // 该地区屏蔽的内容等级
  featuredCategories: string[];   // 主推品类
}
```

---

## 6. Core Flows: Technical Design

### 6.1 Authentication Flow

```
App Launch
    │
    ├── Check Keychain for stored token
    │   ├── Token exists → Validate token (GET /auth/me)
    │   │   ├── Valid → MainTabNavigator
    │   │   └── Invalid → Clear token → AuthStack
    │   └── No token
    │       ├── Check AsyncStorage: isFirstLaunch?
    │       │   ├── true → SplashScreen → LanguageSelectScreen
    │       │   └── false → LoginScreen
    │       └── After login success:
    │           ├── Store token in Keychain
    │           ├── Store user in Zustand
    │           ├── Sync locale to i18next
    │           └── Navigate to MainTabNavigator
```

**Social Auth Flow**：
1. 用户点击社交登录图标（Apple/Google/X/Facebook）
2. 调用 `react-native-apple-authentication` / `@react-native-google-signin/google-signin` 等获取 ID Token
3. 将 ID Token 发送至 `POST /auth/social-login`
4. 后端验证 token 并返回 Rakutao auth token

### 6.2 Search & Translation Flow

```
User Input (EN/ZH/JA)
    │
    ├── Input language detection (client-side heuristic)
    │   ├── If Japanese → skip translation
    │   └── If non-Japanese → send to backend with locale param
    │
    ▼
GET /products/search?q=handbag&locale=en
    │
    │  Backend:
    │  1. LLM translates "handbag" → "ハンドバッグ"
    │  2. Fan-out search to multiple platforms
    │  3. Aggregate results, deduplicate
    │  4. LLM translates results to user locale
    │  5. Cache translations
    │
    ▼
Response: PaginatedResponse<Product>
    │  (titleTranslated, descriptionTranslated populated)
    │  (isTranslated = true)
    │
    ▼
Client renders with Translation tag
```

**前端缓存策略**：
- React Query staleTime: 30 seconds（搜索结果短暂缓存）
- 商品详情 staleTime: 60 seconds
- 已翻译内容缓存 key 包含 locale，切换语言时自动 invalidate

### 6.3 Cart Price Monitoring

```
CartScreen onFocus / Pull-to-refresh
    │
    ▼
GET /cart (返回最新价格)
    │
    ▼
Client 比对:
  item.currentPriceJpy vs item.addedPriceJpy
    │
    ├── 相同 → 正常展示
    └── 不同 → 设 priceChanged = true
              → 显示 PriceChangeIndicator 组件
              → 红色/绿色箭头 + 差额

Sold items:
  item.product.status === 'sold'
    → 叠加 "Sold" 覆盖层
    → 显示 "Archive Sold Items" 按钮
```

### 6.4 Order State Machine (v2: 10 态)

```
┌───────────────┐     ┌──────────┐     ┌────────────┐     ┌────────────┐
│   pending     │────►│   paid   │────►│ purchasing │────►│ warehoused │
│ (待支付)      │     │ (已支付) │     │ (采购中)   │     │ (已入库)   │
│               │     └────┬─────┘     └────────────┘     └─────┬──────┘
│ 子状态:       │          │                                     │
│ pending_      │          │                                     ▼
│ verification  │     ┌────┴──────────────────────────┐   ┌────────────┐
│ (缓存模式)    │     │  可从 pending~purchasing      │   │  packing   │
└───────┬───────┘     │  任一状态进入                  │   │ (打包中)   │
        │             └───────┬──────────────┬────────┘   └─────┬──────┘
        │                     │              │                   │
        │                     ▼              ▼                   ▼
        │              ┌──────────┐   ┌──────────┐        ┌────────────┐
        │              │cancelled │   │ refunded │        │  packed    │
        │              │(已取消)  │   │ (已退款) │        │(已打包,待  │
        │              └──────────┘   └──────────┘        │ 支付运费)  │
        │                                                  └─────┬──────┘
        │                                                        │
        │                                                        ▼
        │                                                  ┌────────────┐
        │                                                  │  shipped   │
        │                                                  │ (已发货)   │
        │                                                  └─────┬──────┘
        │                                                        │
        │                                                        ▼
        │                                                  ┌────────────┐
        └─ 缓存校验失败 → cancelled ─────────────────────►│ fulfilled  │
                                                           │ (已送达)   │
                                                           └────────────┘

Actions per status:
  pending         → [Pay Now]
  pending_verify  → [Cancel]（等待源站校验）
  paid            → [Request Refund, Urge Shipment]
  purchasing      → [—]（等待采购）
  warehoused      → [Request Packing]（选择商品合并打包）
  packing         → [—]（等待仓库打包）
  packed          → [Pay Shipping Fee]（确认运费 + 选择支付方式）
  shipped         → [View Waybill, Track Shipment]
  fulfilled       → [Confirm Receipt]
  cancelled       → [View Reason]
  refunded        → [View Details]
```

### 6.5 Source Site Cache Flow (v2 新增)

```
Product Request (Search / Browse / Detail)
    │
    ├── Check sourceHealthStore: is platform healthy?
    │   ├── healthy → normal API request
    │   │
    │   └── unavailable / degraded →
    │       │
    │       ├── API returns cached data (server-side cache)
    │       │   response includes cacheInfo: { cachedAt, isStale }
    │       │
    │       ├── Client renders:
    │       │   ├── CacheModeIndicator: "数据更新于 X 分钟前"
    │       │   ├── if isStale (>60min): StaleDataWarning (橙色警告)
    │       │   └── Search degrades to cached-data-only search
    │       │
    │       └── User places order in cache mode:
    │           ├── CheckoutScreen shows PendingVerificationBanner
    │           ├── Order created with subStatus: 'pending_verification'
    │           └── After source recovers:
    │               ├── Price same → status: 'paid' (正常流程)
    │               ├── Price down → execute at lower price, refund diff
    │               ├── Price up → notify user: pay diff or cancel
    │               └── Sold out → auto cancel + full refund

Source Health WebSocket:
  wss://api.rakutao.com/ws/source-health
    │
    └── On status change event:
        ├── Update sourceHealthStore
        ├── If unavailable → show cache mode indicators
        └── If recovered → invalidate React Query caches
                          → trigger priority sync for cart + favorites
```

### 6.6 Payment Flow (v2 新增)

```
Checkout initiated (商品款 / 运费 / 补差价)
    │
    ▼
GET /payment/methods (按地区 + 设备过滤)
    │
    ▼
PaymentMethodList renders:
    ├── Wallet (show balance, if insufficient mark as disabled)
    ├── Apple Pay (iOS only)
    ├── Google Pay (Android only)
    ├── PayPal (all platforms)
    └── Local methods (MoMo/ZaloPay etc., per RegionConfig)
    │
    ▼ User selects method
    │
POST /payment/create-intent { orderId, method, amount }
    │
    ├── wallet → backend deducts directly → success/fail
    ├── apple_pay → returns clientSecret → Stripe SDK presentApplePay
    ├── google_pay → returns clientSecret → Stripe SDK presentGooglePay
    └── paypal → returns orderID → PayPal SDK approve flow
    │
    ▼ SDK callback
    │
POST /payment/confirm { intentId, providerToken }
    │
    ▼
Payment result → update order status → navigate to PaymentResultScreen
```

### 6.7 LiveStream Flow (v2 新增)

```
LiveListScreen
    │
    GET /livestreams → renders LiveCard list
    │
    ▼ User taps live stream
    │
LiveRoomScreen
    │
    ├── HLS player: react-native-video with streamUrl
    ├── WebSocket connect: wss://api.rakutao.com/ws/live/:id
    │   ├── Receive: danmaku messages → DanmakuOverlay
    │   ├── Receive: product_marked → LiveProductCard appears
    │   └── Send: user danmaku message
    │
    ▼ User taps "帮我买这个" on LiveProductCard
    │
    POST /livestreams/:id/order → creates OrderLink
    │
    ▼ After stream ends → price confirmation → normal order flow
```

---

## 7. Design System Implementation

### 7.1 Design Tokens

```typescript
// theme/colors.ts
export const colors = {
  // Primary
  primary:      '#BF1E16',    // 主按钮、选中状态、CTA
  secondary:    '#2A82E4',    // 文字按钮、新品标签

  // Neutral
  background:   '#FFFFFF',    // 主体背景
  surface:      '#F5F5F5',    // 页面底色、卡片背景
  textPrimary:  '#383838',    // 正文、标题
  textSecondary:'#A6A6A6',    // 副标题、辅助文字

  // Semantic
  success:      '#4CAF50',    // 验证通过（绿色勾号）
  error:        '#BF1E16',    // 错误状态（复用主色）
  warning:      '#FF9800',

  // Specific
  priceText:    '#BF1E16',    // 价格文字（红色）
  soldOverlay:  'rgba(0,0,0,0.5)',  // 售罄遮罩
} as const;

// theme/typography.ts
export const typography = {
  title:     { fontSize: 18, fontWeight: '700' },  // 标题
  subtitle:  { fontSize: 16, fontWeight: '600' },  // 正文标题、按钮
  body:      { fontSize: 14, fontWeight: '400' },  // 正文、小按钮
  caption:   { fontSize: 12, fontWeight: '400' },  // 提示词、标签
} as const;

// theme/components.ts
export const componentSizes = {
  button: {
    large:  { width: 93,  height: 48, borderRadius: 12 },
    medium: { width: 120, height: 40, borderRadius: 20 },
    small:  { width: 90,  height: 40, borderRadius: 5  },
  },
  card: {
    cartItem:   { width: 345, height: 168, borderRadius: 8  },
    orderItem:  { width: 345, height: 235, borderRadius: 8  },
  },
  thumbnail: { width: 114, height: 114, borderRadius: 10 },
  tag: {
    shipping: { height: 21, borderRadius: 5  },
    capsule:  { height: 25, borderRadius: 20 },
  },
  modal: {
    center: { width: 320, height: 245, borderRadius: 32 },
    bottom: { width: 375, borderRadius: 8  },
  },
  selected: { width: 16, height: 16 },    // 选中圆点
} as const;
```

### 7.2 Component Hierarchy

```
Design System (shared/components/)
│
├── Primitives (原子组件)
│   ├── Button (Large / Medium / Small, primary/outline variants)
│   ├── Text (title / subtitle / body / caption)
│   ├── Input (text / password / email, with validation icon)
│   ├── Checkbox (round / square, with label)
│   ├── Tag (filled / outline, configurable color)
│   ├── Badge (red dot, notification count)
│   └── Divider
│
├── Composites (分子组件)
│   ├── ProductCard (thumbnail + price + tags + ContentRatingBadge + CacheModeIndicator)
│   ├── OrderCard (order info + 10-state status tag + actions)
│   ├── CartItemCard (checkbox + product + price change)
│   ├── FavoriteItemCard (product + sale status + sold indicator)
│   ├── NotificationItem (icon + title + body + time + read state + RichContent)
│   ├── AddressCard (address info + default badge + actions)
│   ├── TransactionItem (type icon + description + amount + date + paymentMethod)
│   ├── SellerGroupHeader (seller info + group select)
│   ├── LiveCard (cover + status badge + viewer count)               ← v2
│   ├── LiveProductCard (photo + name + price + "帮我买" button)     ← v2
│   ├── PaymentMethodItem (icon + label + availability + balance)    ← v2
│   └── WarehousedItemCard (product + weight + warehouse photo)      ← v2
│
├── Patterns (组织组件)
│   ├── CenterModal (title + content + action buttons)
│   ├── BottomSheet (header + scrollable content + CTA)
│   ├── StickyBottomBar (total + action button, used in Cart/Detail)
│   ├── TabBar (5-tab bottom navigation)
│   ├── FilterBar (tabs + dropdowns, used in Orders)
│   ├── EmptyState (icon + message + action)
│   └── PullToRefresh
│
└── Screens use Composites + Patterns to compose pages
```

---

## 8. Caching & Offline Strategy

### 8.1 React Query Cache Configuration

```typescript
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,           // 30s 内不重新请求
      gcTime: 5 * 60_000,          // 5min 后清理缓存
      retry: 2,                     // 失败重试 2 次
      retryDelay: (attempt) => Math.min(1000 * 2 ** attempt, 10000),
    },
  },
});
```

| 数据类型 | staleTime | gcTime | 策略 |
|---------|-----------|--------|------|
| 商品列表 | 30s | 5min | 短暂缓存，频繁刷新（C2C 商品变化快） |
| 商品详情 | 60s | 10min | 允许略旧数据，减少请求 |
| 搜索结果 | 30s | 2min | 搜索结果时效性强 |
| 购物车 | 0 (always fresh) | 5min | 每次进入都刷新（价格变动检测） |
| 订单列表 | 30s | 10min | 状态变化不频繁 |
| 钱包余额 | 0 (always fresh) | 1min | 金额必须实时 |
| 收藏列表 | 60s | 10min | 可容忍略旧数据 |
| 个人信息 | 5min | 30min | 很少变化 |
| 汇率 | 60s | 5min | 准实时 |
| 通知列表 | 30s | 5min | 较频繁刷新 |
| 直播列表 | 10s | 2min | v2: 实时性要求高（LIVE 状态/人数） |
| 源站健康 | 0 (WebSocket push) | 5min | v2: 由 WebSocket 推送更新 |
| 支付方式 | 5min | 30min | v2: 很少变化 |
| 推荐商品 | 60s | 10min | v2: 可容忍略旧数据 |
| 地区配置 | 30min | 60min | v2: 极少变化，启动时加载 |

### 8.2 Offline & Source Site Resilience

```
MMKV 持久化层（高性能 KV 存储）:
├── @rakutao/language               # 语言偏好
├── @rakutao/first-launch           # 首次启动标记
├── @rakutao/recent-searches        # 最近搜索历史
├── @rakutao/age-verified           # v2: 年龄验证状态
├── @rakutao/adult-content          # v2: 成人内容开关
├── @rakutao/tracking-consent       # v2: 行为追踪授权
├── @rakutao/payment-pref           # v2: 上次支付方式
├── @rakutao/source-health-cache    # v2: 源站健康状态快照
└── @rakutao/cache-*                # React Query 持久化缓存

SQLite (expo-sqlite) — v2 离线结构化数据:
├── cached_products                 # 已浏览商品详情离线缓存
├── cached_favorites                # 收藏列表离线缓存
└── search_history                  # 搜索历史

Keychain / Keystore:
├── auth-token                      # JWT token
└── refresh-token                   # Refresh token

v2 源站不可用时（FR-SRC）:
├── 服务端缓存模式（后端控制）:
│   ├── 源站健康监控自动检测不可用 → 切换缓存模式
│   ├── API 返回缓存数据 + cacheInfo 字段
│   └── 缓存有效期：热门品类 15min / 普通 30min / 冷门 60min
│
├── 客户端降级展示:
│   ├── CacheModeIndicator: 显示"数据更新于 X 分钟前"
│   ├── StaleDataWarning: 超过 60min 显示橙色警告
│   ├── 搜索降级为缓存数据搜索
│   └── 下单时展示 PendingVerificationBanner
│
└── 客户端离线缓存（FR-SRC-007）:
    ├── MMKV + SQLite 存储已浏览商品
    ├── 弱网/无网时可查看，标注"离线数据"标签
    └── 恢复网络后优先同步购物车和收藏夹

Offline behavior（无网络时）:
├── 显示 SQLite 缓存的商品列表和收藏（只读）
├── 显示缓存的订单列表（只读）
├── 顶部显示离线横幅提示
└── 写操作（加购、下单）队列化，恢复网络后自动同步

弱网络时:
├── 图片加载：显示低分辨率占位图
├── 请求超时：10s 后显示重试按钮
└── 骨架屏：列表页使用 Skeleton 占位
```

---

## 9. i18n Architecture

### 9.1 Static UI Text (i18next)

```
src/shared/i18n/
├── index.ts          # 初始化配置
├── ja.json           # 日语（默认，完整）
├── zh-Hant.json      # 繁体中文（完整）
└── en.json           # 英语（完整）
```

**命名空间约定**：

```json
// en.json
{
  "common": {
    "signIn": "Sign In",
    "signUp": "Sign Up",
    "cancel": "Cancel",
    "continue": "Continue",
    "delete": "Delete",
    "edit": "Edit",
    "total": "Total"
  },
  "home": {
    "search": "Search...",
    "picks": "Picks",
    "brands": "Brands"
  },
  "cart": {
    "title": "Cart",
    "priceChanged": "Price changed",
    "soldOut": "Sold",
    "archiveSold": "Archive Sold Items",
    "buy": "Buy",
    "safeStatus": "Safe Status"
  },
  "wallet": {
    "balance": "Balance (JPY)",
    "charge": "Charge",
    "allOrders": "All Orders",
    "pending": "Pending",
    "paid": "Paid",
    "purchasing": "Purchasing",
    "warehoused": "Warehoused",
    "packing": "Packing",
    "packed": "Packed",
    "shipped": "Shipped",
    "fulfilled": "Fulfilled",
    "cancelled": "Cancelled",
    "refunded": "Refunded"
  },
  "product": {
    "addToCart": "Add to Cart",
    "buy": "Buy",
    "variantSelection": "Variant Selection",
    "translated": "Translation",
    "freeShipping": "Free Shipping",
    "freightCollect": "Freight Collect",
    "withinJapan": "Within Japan",
    "globalDelivery": "Global Delivery",
    "serviceCharge": "Service Charge"
  },
  "favorites": {
    "myFavorites": "My Favorites",
    "followSeller": "Follow Seller"
  },
  "profile": {
    "myHomepage": "My Homepage",
    "nickname": "Nickname Settings",
    "shippingAddress": "Shipping Address",
    "changePassword": "Change Password",
    "languageSettings": "Language Settings",
    "adultContent": "Adult Content",
    "trackingPreferences": "Personalization Data",
    "dataExport": "Export My Data",
    "deleteAccount": "Delete Account",
    "userGuide": "User Guide",
    "clearCache": "Clear Cache",
    "signOut": "Sign Out",
    "checkUpdates": "Check for Updates"
  },
  "orderLink": {
    "title": "Order Link",
    "enterLink": "Please enter the product's link or ID.",
    "additionalRequests": "Please let us know your additional requests.",
    "add": "Add",
    "submit": "Submit"
  },
  "livestream": {
    "live": "LIVE",
    "upcoming": "Coming Soon",
    "viewers": "viewers",
    "buyThis": "Buy This For Me",
    "replay": "Replay",
    "subscribe": "Notify Me"
  },
  "contentRating": {
    "ageGate": "Age Verification Required",
    "enterBirthday": "Please enter your date of birth",
    "verificationFailed": "You must be 18 or older",
    "retryIn": "Please try again in 24 hours",
    "adultContentWarning": "This content is for adults only"
  },
  "payment": {
    "selectMethod": "Select Payment Method",
    "walletBalance": "Wallet Balance",
    "insufficientBalance": "Insufficient balance",
    "payNow": "Pay Now",
    "paymentSuccess": "Payment Successful",
    "paymentFailed": "Payment Failed"
  },
  "wms": {
    "requestPacking": "Request Packing",
    "selectItemsToPack": "Select items to pack together",
    "shippingFee": "Shipping Fee",
    "confirmAndPay": "Confirm & Pay Shipping",
    "weight": "Weight",
    "volume": "Volume"
  },
  "sourceCache": {
    "updatedAgo": "Updated {{minutes}} min ago",
    "dataMayBeOutdated": "Data may be outdated",
    "pendingVerification": "Price and availability will be confirmed when source recovers",
    "offlineData": "Offline Data"
  },
  "support": {
    "faq": "Frequently Asked Questions",
    "chatBot": "Chat with AI Assistant",
    "transferHuman": "Talk to a Human",
    "createTicket": "Create Ticket"
  }
}
```

### 9.2 Dynamic Content Translation (LLM)

动态商品内容的翻译由后端 LLM 服务处理，前端只需：

1. 请求时携带 `Accept-Language` header（`ja` / `zh-Hant` / `en`）
2. 后端返回 `titleTranslated` / `descriptionTranslated` 字段
3. 前端根据 `isTranslated` 字段决定是否显示 "Translation" 标签
4. 切换语言时 invalidate 所有商品相关的 React Query 缓存

```typescript
// API client interceptor: auto-attach locale
apiClient.interceptors.request.use((config) => {
  const locale = useLanguageStore.getState().locale;
  config.headers['Accept-Language'] = locale;
  return config;
});
```

---

## 10. Security Architecture

### 10.1 Authentication & Token Management

```
Login → Receive { accessToken, refreshToken }
         │
         ├── accessToken → Keychain (short-lived, e.g. 15min)
         ├── refreshToken → Keychain (long-lived, e.g. 30 days)
         │
         └── Axios interceptor:
             ├── Request: Attach Authorization: Bearer {accessToken}
             ├── Response 401:
             │   ├── Try refresh: POST /auth/refresh { refreshToken }
             │   │   ├── Success → update tokens, retry original request
             │   │   └── Failure → logout, navigate to LoginScreen
             │   └── Queue concurrent requests during refresh
             └── Response 403 (region block): show region restriction message
```

### 10.2 Data Security

| 数据 | 存储位置 | 保护方式 |
|------|---------|---------|
| Auth tokens | Keychain / Keystore | 系统级加密，App 卸载后自动清除 |
| User profile | React Query (内存) | 不持久化到磁盘 |
| 语言偏好 | MMKV | 无敏感性，明文存储 |
| 搜索历史 | SQLite | 无敏感性 |
| 支付信息 | **不在客户端存储** | 全部通过后端/第三方支付网关处理 |
| 年龄验证状态 | MMKV | v2: 本地标记，服务端同步校验 |
| 行为追踪数据 | **仅服务端** | v2: 不在客户端持久化，按同意状态采集 |
| 离线商品缓存 | SQLite | v2: 仅缓存公开商品数据，不含个人信息 |

### 10.3 Network Security

- 全部请求使用 HTTPS（TLS 1.2+）
- API 域名启用 Certificate Pinning（可选，增强安全）
- 敏感操作（修改密码、绑定邮箱、充值）需二次验证（验证码）
- 请求体中的敏感字段（密码）在传输前加密（后端提供公钥）

### 10.4 Region Restriction

```
App 启动时:
  1. 获取设备 locale 和 IP 地理信息
  2. 后端 API 检测请求 IP
  3. 如果 IP 属于中国大陆 → 返回 403 + region_blocked
  4. 客户端显示地区不可用提示页

应用商店层面:
  - App Store: 排除中国大陆区（上架时配置）
  - Google Play: 排除中国大陆区（国家/地区设置）
```

### 10.5 Content Rating & Age Verification (v2 新增)

```
内容过滤策略:
  1. API 返回商品数据包含 contentRating 字段 (G/M/A)
  2. 客户端 useContentFilter hook 根据以下条件过滤:
     ├── isAgeVerified === false → 隐藏 M/A 级商品
     ├── adultContentEnabled === false → 隐藏 A 级商品
     └── RegionConfig.contentRestrictions 包含该等级 → 完全隐藏
  3. 搜索/列表/推荐 API 均在服务端做初步过滤
  4. 客户端二次过滤（防止缓存数据泄露）

年龄验证安全:
  ├── 仅基于出生日期自申报（≥18 岁）
  ├── 验证失败 24h 内冷却期（存储在 MMKV + 服务端）
  ├── 验证状态服务端存储 + 客户端缓存
  └── 直播间内容分级：未验证用户不可进入 M/A 级直播间
```

### 10.6 Payment Security (v2 新增)

```
支付安全架构:
  ├── 不存储支付卡信息（全部通过第三方网关）
  ├── Apple Pay: Stripe SDK → tokenized payment → backend → Stripe API
  ├── Google Pay: Stripe SDK → tokenized payment → backend → Stripe API
  ├── PayPal: PayPal SDK → order approval → backend → PayPal API
  └── 钱包充值: 同上述支付通道 → backend → 余额入账

PCI-DSS 合规:
  ├── 客户端不接触 PAN (Primary Account Number)
  ├── 所有支付处理通过 PCI-DSS Level 1 认证的第三方
  └── 支付意图 (Payment Intent) 由后端创建，客户端仅传递 token
```

### 10.7 Data Privacy Compliance (v2 新增)

```
数据隐私实现 (对应 PRD §4A DPC-001~008):

客户端职责:
  ├── 首次注册展示隐私政策摘要 + 必须同意（FR-AUTH-004）
  ├── 行为追踪需单独获取同意（TrackingPreferencesScreen）
  ├── 用户可随时在设置中撤回追踪授权 → 立即停止采集
  ├── 提供数据导出入口（DataExportScreen）
  └── 提供账号删除入口（AccountDeletionScreen, 30 天可撤回）

技术实现:
  ├── trackingConsentStore 控制 analytics SDK 开关
  ├── 未授权时 analytics.track() 为空操作
  ├── Accept-Language header 携带地区信息用于合规过滤
  └── 个人信息不写入 SQLite/MMKV 离线缓存
```

---

## 11. Performance Strategy

### 11.1 Image Loading

```
商品缩略图 (114x114px):
  1. FastImage with priority: 'normal'
  2. 占位图: 灰色 skeleton 方块
  3. 缓存策略: disk + memory (FastImage 默认)
  4. CDN 端裁剪: 请求 thumbnail 尺寸 (?w=228&h=228 for @2x)

商品详情大图:
  1. FastImage with priority: 'high'
  2. 渐进式加载: 先低分辨率模糊图 → 高清图
  3. 预加载: 轮播时预加载下一张
```

### 11.2 List Rendering

```
商品列表 (HomeScreen):
  - FlashList (高性能列表替代 FlatList)
  - estimatedItemSize: 根据商品卡片高度预估
  - 瀑布流: @shopify/flash-list 的 MasonryFlashList
  - 分页: cursor-based pagination via React Query useInfiniteQuery

订单列表 (OrderListScreen):
  - FlashList
  - 按状态 Tab 分别缓存 query key
```

### 11.3 Bundle & Launch

```
Bundle optimization:
  - Hermes engine (enabled by default in RN 0.76+)
  - 代码分割: 按 Tab 动态 import (React.lazy + Suspense for non-initial tabs)
  - 图片资源: WebP 格式，按需加载

启动优化:
  - SplashScreen 原生实现 (react-native-splash-screen)
  - 延迟加载非首屏模块
  - 网络请求优先级: auth check > 商品列表 > 其他
```

---

## 12. Error Handling

### 12.1 Error Hierarchy

```typescript
// 全局错误类型
type AppError =
  | { type: 'network';    message: string }   // 网络不可达
  | { type: 'timeout';    message: string }   // 请求超时
  | { type: 'auth';       message: string }   // 认证失败（token 过期）
  | { type: 'region';     message: string }   // 地区限制
  | { type: 'validation'; fields: Record<string, string> }  // 字段校验
  | { type: 'business';   code: number; message: string }    // 业务错误
  | { type: 'server';     message: string }   // 服务器错误 (5xx)
  | { type: 'payment';    code: string; message: string }    // v2: 支付失败
  | { type: 'age_gate';   message: string }   // v2: 年龄验证失败
  | { type: 'source_unavailable'; platform: string };  // v2: 源站不可用（降级处理）
```

### 12.2 Error UI Mapping

| 错误类型 | UI 表现 |
|---------|---------|
| network | 顶部 Toast "No internet connection" + 重试按钮 |
| timeout | 页面内 "Request timed out" + 重试按钮 |
| auth (token expired) | 静默刷新 token；刷新失败弹窗提示重新登录 |
| region | 全屏提示页 "Service not available in your region" |
| validation | 表单字段内联错误提示（红色文字） |
| business | Toast 或弹窗展示后端返回的 message |
| server | 全屏错误页 "Something went wrong" + 重试 |
| payment | v2: 支付结果页展示失败原因 + 重试/切换支付方式 |
| age_gate | v2: AgeGateScreen 显示验证失败提示 + 24h 冷却倒计时 |
| source_unavailable | v2: **不显示错误**，无感降级至缓存模式 + CacheModeIndicator |

---

## 13. Testing Strategy

| 测试类型 | 工具 | 覆盖范围 | 目标 |
|---------|------|---------|------|
| **单元测试** | Jest | 工具函数、store 逻辑、hooks | 核心业务逻辑 80%+ 覆盖 |
| **组件测试** | React Native Testing Library | 共享 UI 组件、核心业务组件 | 设计系统组件 100% 覆盖 |
| **集成测试** | Jest + MSW (Mock Service Worker) | API 调用链路、状态管理流 | 核心流程（登录/搜索/下单/支付） |
| **快照测试** | Jest Snapshots | 关键页面 | 防止 UI 意外变更 |
| **E2E 测试** | Detox (可选) | 核心用户流程 | 注册→搜索→下单→订单查看 |

---

## 14. Monitoring & Analytics

### 14.1 Crash & Error Monitoring

| 场景 | 工具/方案 |
|------|---------|
| App 崩溃 | Sentry React Native SDK |
| JS 异常 | Sentry breadcrumbs + error boundaries |
| API 错误率 | 后端 APM 监控 |
| 性能指标 | React Native Performance API + 自定义 trace |

### 14.2 Business Analytics Events

| 事件 | 触发点 | 关键参数 |
|------|-------|---------|
| `app_launch` | App 启动 | locale, platform, version, region |
| `language_selected` | 语言选择页确认 | locale |
| `sign_up` | 注册成功 | method (email/social) |
| `sign_in` | 登录成功 | method |
| `search` | 搜索提交 | query, locale, resultCount, isCacheMode |
| `product_view` | 进入商品详情 | productId, source (home/search/favorite/live), contentRating |
| `add_to_cart` | 加入购物车 | productId, priceJpy, isCachedData |
| `order_link_submit` | 指定购买提交 | linkUrl, platform |
| `checkout` | 发起结算 | itemCount, totalJpy, paymentMethod |
| `payment_success` | 支付成功 | orderId, totalJpy, paymentMethod |
| `payment_failed` | 支付失败 | orderId, paymentMethod, errorCode |
| `favorite_add` | 收藏商品 | productId |
| `seller_follow` | 关注卖家 | sellerId |
| `wallet_charge` | 发起充值 | amountJpy, paymentMethod |
| `ticket_create` | 创建工单 | category |
| `live_join` | v2: 进入直播间 | streamId, hostName |
| `live_order` | v2: 直播间下单 | streamId, productName, priceJpy |
| `packing_request` | v2: 申请打包 | orderIds, itemCount |
| `shipping_fee_paid` | v2: 支付运费 | packingRequestId, feeJpy, paymentMethod |
| `age_verified` | v2: 年龄验证 | result (pass/fail) |
| `adult_content_toggle` | v2: 成人内容开关 | enabled (true/false) |
| `tracking_consent` | v2: 行为追踪授权变更 | enabled (true/false) |
| `cache_mode_order` | v2: 缓存模式下单 | orderId, platform, cacheAge |
| `source_recovery_result` | v2: 源站恢复校验结果 | orderId, result (same/lower/higher/sold_out) |
| `data_export_request` | v2: 数据导出申请 | — |
| `account_deletion_request` | v2: 账号删除申请 | — |

> **注意**: 标记为行为追踪的事件（`product_view`, `search`, `favorite_add`, `add_to_cart`）仅在用户授权后采集（FR-REC-006, DPC-006）。trackingConsentStore 控制 analytics SDK 开关。

---

## 15. Deployment & Release

### 15.1 Build Pipeline

```
Feature Branch → PR → Code Review → Merge to develop
                                          │
                                    ┌─────▼──────┐
                                    │   CI/CD    │
                                    │  (EAS Build)│
                                    └─────┬──────┘
                                          │
                              ┌───────────┼───────────┐
                              ▼           ▼           ▼
                          iOS Build   Android Build   Tests
                              │           │           │
                              ▼           ▼           ▼
                         TestFlight   Internal Test  Pass/Fail
                              │           │
                              ▼           ▼
                         App Store    Google Play
                         (Review)     (Review)
                              │           │
                              ▼           ▼
                           Release     Release
```

### 15.2 Environment Configuration

| 环境 | API Base | 用途 |
|------|---------|------|
| `development` | `https://dev-api.rakutao.com/v1` | 本地开发 |
| `staging` | `https://staging-api.rakutao.com/v1` | 内部测试、QA |
| `production` | `https://api.rakutao.com/v1` | 生产环境 |

通过 `.env.development` / `.env.staging` / `.env.production` 配置，构建时注入。

### 15.3 Version Strategy

- 版本号格式: `MAJOR.MINOR.PATCH` (如 `2.0.0`)
- v2.0 发布: `2.0.0`
- 强制更新: 后端接口 `GET /app/version` 返回最低支持版本，客户端比对决定是否强制更新

---

## Appendix A: Requirement Traceability

| PRD Section | Architecture Section | 说明 |
|-------------|---------------------|------|
| FR-ONB-* | 3.2 Navigation (AuthStack) | 引导流程导航设计 |
| FR-AUTH-* | 4.3 Auth API, 6.1 Auth Flow, 10.1 Token Mgmt | 认证全链路 |
| FR-HOME-* | 3.1 features/home, 4.3 Products API, 6.2 Search Flow | 首页功能模块 |
| FR-PD-* | 3.1 features/product, 5.1 Product Model | 商品详情 |
| FR-CART-* | 3.1 features/cart, 4.3 Cart API, 6.3 Price Monitor | 购物车 |
| FR-OL-* | 3.1 features/order-link, 4.3 Order Link API | 指定购买 |
| FR-WAL-* | 3.1 features/wallet, 4.3 Wallet API | 钱包 |
| FR-ORD-* | 3.1 features/wallet (orders), 6.4 Order State Machine (10 态) | 订单管理 |
| FR-FAV-* | 3.1 features/favorites, 4.3 Favorites API | 收藏夹 |
| FR-PROF-* | 3.1 features/profile, 4.3 Profile API (+v2 数据隐私) | 个人中心 |
| FR-NOTI-* | 3.1 features/notifications, 4.3 Notifications API | 通知系统 |
| FR-TAB-001 | 3.2 MainTabNavigator | 底部导航 |
| **FR-LIVE-*** | **3.1 features/livestream, 4.3 LiveStream API, 6.7 LiveStream Flow** | **v2: 代购直播** |
| **FR-CR-*** | **3.1 features/content-rating, 4.3 Content Rating API, 10.5 Age Verify** | **v2: 内容分级** |
| **FR-REC-*** | **4.3 Recommendations API, 14.2 Analytics (tracking consent)** | **v2: 个性化推荐** |
| **FR-REG-*** | **4.3 Region Config API, shared/hooks/useRegionConfig** | **v2: 地区化运营** |
| **FR-WMS-*** | **3.1 features/wms, 4.3 WMS API** | **v2: WMS 仓储交互** |
| **FR-HV-*** | **5.1 Order.isHighValue, 4.3 WMS/Orders API** | **v2: 高价值商品** |
| **FR-PAY-*** | **3.1 features/payment, 4.3 Payment API, 6.6 Payment Flow, 10.6** | **v2: 直接支付** |
| **FR-SS-*** | **3.1 features/smart-support, 4.3 Smart Support API** | **v2: 智能客服** |
| **FR-SRC-*** | **3.1 features/source-cache, 4.3 Source Health API, 6.5 Cache Flow, 8.2** | **v2: 源站容灾** |
| NFR-001~003 | 11. Performance Strategy | 性能要求 |
| NFR-004~005 | 9. i18n Architecture | 多语言 |
| NFR-006 | 10.4 Region Restriction | 地区限制 |
| NFR-007~009 | 10. Security Architecture | 安全 |
| NFR-010~012 | 8. Caching & Offline, 12. Error Handling | 可用性 |
| NFR-013~014 | 7. Design System | UI 规范 |
| **NFR-017~020** | **12. Error Handling (degradation), 6.5 Cache Flow** | **v2: 稳定性/熔断降级** |
| **NFR-024~027** | **8. Caching & Offline (source resilience)** | **v2: 源站缓存性能** |
| **NFR-028~029** | **10.7 Data Privacy Compliance** | **v2: 数据隐私** |
| **DPC-001~008** | **10.7 Data Privacy, 3.1 features/profile (v2 screens)** | **v2: 合规要求** |

## Appendix B: ADR (Architecture Decision Records)

### ADR-001: React Native over Flutter

**决策**: 使用 React Native 而非 Flutter

**原因**:
- 团队已有 React/TypeScript 经验，学习成本低
- npm 生态系统中可复用大量现有库
- 社交登录（Apple/Google/X/Facebook）在 RN 生态中方案成熟
- Hot Reload 开发体验好
- 社区活跃度和长期维护信心高

**取舍**: Flutter 在 UI 渲染一致性和性能上略优，但 RN 在开发效率和生态覆盖度上的优势对本项目更重要。

### ADR-002: Zustand + React Query over Redux Toolkit

**决策**: 使用 Zustand 管理客户端状态，React Query 管理服务端状态

**原因**:
- Zustand 体积小、API 简洁，适合中等规模应用
- React Query 天然适合 "服务端状态" 模式（商品、订单等数据来自 API）
- 避免 Redux 的 boilerplate 和复杂度
- 两者职责分明，不会产生状态管理混乱

### ADR-003: Feature-First Directory Structure

**决策**: 按业务功能模块（feature）组织代码，而非按技术层级（components/screens/api）

**原因**:
- 功能模块之间耦合低（Cart 不需要知道 Profile 的实现）
- 新增功能模块时只需添加一个目录，不影响其他模块
- 每个模块内部完整（screen + component + api + hook），易于理解和维护
- 共享代码提取到 `shared/` 目录，避免重复

### ADR-004: JPY as Single Display Currency

**决策**: 前端统一以 JPY 展示价格，汇率仅作参考

**原因**: 见 project-brief "为什么价格只显示 JPY" 章节。技术层面，这简化了前端的价格计算和显示逻辑——不需要实时转换、不需要处理多币种舍入误差、不需要担心缓存中的汇率过期。

### ADR-005: Stripe as Payment Gateway (v2)

**决策**: 通过 Stripe 统一接入 Apple Pay 和 Google Pay，PayPal 通过 PayPal SDK 直接接入

**原因**:
- Stripe 提供 Apple Pay + Google Pay 的统一 SDK (`@stripe/stripe-react-native`)，减少集成工作量
- Stripe 已通过 PCI-DSS Level 1 认证，客户端无需接触卡号
- PayPal 有独立 SDK 且市场覆盖面广（尤其东南亚），独立接入更灵活
- 本地支付方式（MoMo/ZaloPay）后续通过各自 SDK 按需接入

**取舍**: 统一使用 Stripe 处理所有支付（Stripe 也支持 PayPal）更简单，但 PayPal 独立接入可避免 Stripe 中间费用，且 PayPal 在东南亚用户中信任度更高。

### ADR-006: MMKV over AsyncStorage (v2)

**决策**: 使用 MMKV 替代 AsyncStorage 作为主要 KV 存储

**原因**:
- MMKV 读写性能是 AsyncStorage 的 30~100 倍（mmap 内存映射）
- v2 源站缓存需要频繁读写大量商品数据，AsyncStorage 性能瓶颈明显
- MMKV 支持加密存储、多进程访问
- 向后兼容：保留 asyncStorage.ts 封装层，逐步迁移

### ADR-007: WebSocket for Real-time Features (v2)

**决策**: 使用 socket.io-client 管理 WebSocket 连接，用于直播弹幕、源站健康推送、订单状态推送

**原因**:
- socket.io 提供自动重连、心跳、房间/命名空间等特性，减少手动管理
- 直播弹幕需要低延迟双向通信，HTTP 轮询不适合
- 源站健康状态变更需要秒级推送，避免客户端轮询浪费带宽
- 多个实时功能共享一个 WebSocket 连接，通过命名空间隔离

**取舍**: 原生 WebSocket 更轻量但需要自行实现重连/心跳逻辑。socket.io 增加约 30KB 包体积但大幅降低开发复杂度。

### ADR-008: HLS over WebRTC for Live Viewing (v2)

**决策**: 直播观看端使用 HLS 拉流，WebRTC 仅用于主播推流端（未来）

**原因**:
- HLS 兼容性最好，所有设备/网络均支持
- react-native-video 对 HLS 支持成熟
- 代购直播对延迟要求适中（3~5s 可接受），HLS 足够
- WebRTC 在 React Native 中集成复杂度高，仅在需要超低延迟交互时考虑

### ADR-009: Client-side Content Filtering as Defense-in-depth (v2)

**决策**: 内容分级过滤在服务端和客户端双重执行

**原因**:
- 服务端 API 根据用户验证状态过滤 M/A 级商品（主过滤）
- 客户端 `useContentFilter` hook 二次过滤（防御层）
- React Query 缓存可能包含其他用户的数据（如共享设备场景）
- 客户端过滤确保即使缓存数据泄露也不会展示不当内容
- 性能影响极小（仅一个 filter 操作）
