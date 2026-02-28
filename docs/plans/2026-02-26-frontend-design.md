# Rakutao Frontend Design

## Overview

**Mobile-first**: Expo (React Native) App for iOS + Android, with Web (Next.js) planned as phase 2. Shared logic layer (`lib/`) will be extracted for future Web reuse.

## Tech Stack

- **Framework**: Expo SDK 52 + Expo Router v4 (file-based routing)
- **UI**: NativeWind v4 (Tailwind CSS for React Native)
- **i18n**: i18next + react-i18next + expo-localization (zh-TW / ja / en)
- **State**: React hooks (useState/useEffect), no external state lib
- **WebSocket**: React Native built-in WebSocket API
- **Build**: EAS Build (cloud) for iOS + Android

## Competitive Analysis Summary

Researched 8 platforms: Buyee, ZenMarket, FromJapan, Jauce, Neokyo, Sendico, JapanRabbit, Remambo.

### Adopted Best Practices

| Pattern | Source |
|---------|--------|
| Search bar with keyword + platform selector | Buyee, ZenMarket |
| Scrollable grid results (pull-to-refresh) | Mobile-native pattern |
| Platform source color badges | FromJapan |
| Auto-translate search keywords | Neokyo, Sendico |
| Transparent itemized fee breakdown | FromJapan, Sendico |
| Condition badge system | Buyee |
| Bottom tab navigation | Buyee App, standard mobile UX |
| Light background + bright accent colors | Industry consensus |

### Avoided Anti-Patterns

| Anti-Pattern | Source |
|--------------|--------|
| Separate search per platform | Buyee |
| Hidden shipping costs | Buyee |
| Push notification language bugs | Buyee App |
| No sign-up flow in app | Buyee App |
| Complex percentage-on-percentage fees | FromJapan |

### Our Differentiators

1. **Cross-platform unified search** — no competitor does this well
2. **Real-time streaming results via WebSocket** — no competitor has this
3. **AI-powered keyword translation** — only Neokyo/Sendico have basic versions

## Project Structure

```
mobile/                            # /Users/gongqianrong/Desktop/ai/mobile
├── app/                           # Expo Router (file-based routes)
│   ├── _layout.tsx                # Root layout (i18n provider, fonts)
│   ├── (tabs)/                    # Tab navigator
│   │   ├── _layout.tsx            # Tab bar config
│   │   └── index.tsx              # Search tab (main screen)
│   └── product/
│       └── [id].tsx               # Product detail screen
├── components/
│   ├── search/
│   │   ├── SearchBar.tsx          # Search input + platform picker
│   │   ├── FilterSheet.tsx        # Bottom sheet: platform/price/brand/condition
│   │   ├── ProductCard.tsx        # Product card in grid
│   │   └── StreamingBar.tsx       # Real-time progress bar
│   ├── product/
│   │   ├── ImageCarousel.tsx      # Swipeable image gallery
│   │   ├── PriceBreakdown.tsx     # Price + service fee card
│   │   └── DescriptionTabs.tsx    # Translated vs original tabs
│   └── common/
│       ├── PlatformBadge.tsx      # Colored platform label
│       └── ConditionBadge.tsx     # Colored condition label
├── hooks/
│   ├── useSearch.ts               # Search logic (REST + WS dual-channel)
│   └── useProduct.ts              # Product detail fetch
├── lib/
│   ├── types.ts                   # TypeScript types (backend-aligned)
│   ├── api.ts                     # REST API client
│   └── ws.ts                      # WebSocket client
├── i18n/
│   ├── index.ts                   # i18next config
│   └── messages/
│       ├── zh-TW.json             # Traditional Chinese (default)
│       ├── ja.json                # Japanese
│       └── en.json                # English
├── app.json                       # Expo config
├── babel.config.js
├── nativewind-env.d.ts
├── tailwind.config.js             # NativeWind config
├── tsconfig.json
├── package.json
└── .env                           # API_URL
```

## Search Screen Design

### Dual-Channel Flow (same as Web design)

```
User input (Chinese/English/Japanese)
    → Auto language detection + AI translation → Japanese keyword
    → GET /api/v1/search
    → HTTP response: ES cached results + streamID + aggregations
    → Immediately render results + filter chips
    → WebSocket /stream/{streamID}
    → Real-time append results with streaming progress bar
```

### Mobile Layout

```
┌─────────────────────────────┐
│ ┌─────────────────────────┐ │
│ │ 🔍 搜尋商品...    [全部▾]│ │  ← SearchBar + platform picker
│ └─────────────────────────┘ │
│                             │
│ [平台] [價格] [品牌] [狀態]  │  ← Filter chips (tap → bottom sheet)
│                             │
│ 翻譯: グッチ バッグ          │  ← Translated keyword hint
│ 1,523 個結果                │
│                             │
│ ━━━━━━━━░░░░ 2/3 平台完成   │  ← StreamingBar
│                             │
│ ┌───────────┐┌───────────┐ │
│ │ [商品圖]   ││ [商品圖]   │ │
│ │           ││           │ │
│ │ GUCCI..   ││ Nike..    │ │  ← 2-column FlatList grid
│ │ ¥5,000    ││ ¥3,200    │ │
│ │ +¥500手續費││ +¥320手續費│ │
│ │ Yahoo🟣   ││ 駿河屋🟢  │ │
│ │ [全新]    ││ [良好]    │ │
│ └───────────┘└───────────┘ │
│ ┌───────────┐┌───────────┐ │
│ │  ...      ││  ...      │ │
│ └───────────┘└───────────┘ │
│                             │
├─────────────────────────────┤
│  [🔍 搜尋]                  │  ← Bottom tab bar
└─────────────────────────────┘
```

### Key Mobile UX

- **FlatList** with pull-to-refresh and infinite scroll (load more on scroll bottom)
- **Bottom sheet** for filters (not sidebar — mobile-friendly)
- **Filter chips** at top for quick filter toggle
- **Haptic feedback** on search submit
- **Skeleton loading** while fetching

## Product Detail Screen Design

```
┌─────────────────────────────┐
│ ← 返回          Yahoo🟣     │  ← Navigation header + platform badge
│                             │
│ ┌─────────────────────────┐ │
│ │                         │ │
│ │    可滑動的商品圖片       │ │  ← Swipeable image carousel
│ │    (全螢幕手勢)          │ │
│ │                         │ │
│ │         ● ○ ○ ○ ○       │ │  ← Page indicator dots
│ └─────────────────────────┘ │
│                             │
│ GUCCI グッチ レザー ショルダー │  ← Title
│ バッグ 新品未使用            │
│                             │
│ [GUCCI] [全新]              │  ← Brand + condition badges
│                             │
│ ┌─────────────────────────┐ │
│ │ 商品價格      ¥45,000   │ │
│ │ 手續費(10%)   ¥4,500    │ │  ← PriceBreakdown card
│ │ ───────────────────── │ │
│ │ 合計          ¥49,500   │ │
│ │ ※國際運費另計            │ │
│ └─────────────────────────┘ │
│                             │
│ ┌───────────┐┌────────────┐│
│ │  立即購買   ││  出價競拍   ││ ← Action buttons
│ └───────────┘└────────────┘│
│                             │
│ ┌─────────┐┌──────────┐   │
│ │ 翻譯版   ││ 日文原文   │   │  ← Description tabs
│ └─────────┘└──────────┘   │
│ 全新未使用的 GUCCI 皮革...  │
│                             │
│ 賣家: seller_name ★★★★☆   │  ← Seller info
└─────────────────────────────┘
```

### Key Mobile UX

- **ScrollView** with sticky header
- **Image carousel** with pinch-to-zoom (react-native-reanimated)
- **Sticky action buttons** at bottom (always visible)
- **Tab-based description** for translated vs original text

## Color Scheme (shared with future Web)

```
Primary:    #0EA5E9 (sky-500)    — brand, buttons, links
Accent:     #F97316 (orange-500) — auction/bidding emphasis
Success:    #22C55E (green-500)  — buy/confirm
Danger:     #EF4444 (red-500)    — ending soon/errors
Background: #F8FAFC (slate-50)   — screen background
Card:       #FFFFFF              — white cards
Text:       #0F172A (slate-900)  — primary text
Muted:      #64748B (slate-500)  — secondary text
```

## i18n

- i18next + react-i18next (React Native standard)
- expo-localization for device locale detection
- 3 locales: zh-TW (default), ja, en
- Language switcher in settings or profile tab
- Fallback: zh-TW → en

## Future: Web Phase 2

When Web is needed, shared `lib/` (types, api, ws) and `i18n/messages/` will be extracted to a shared package or copied. Web will use Next.js + Tailwind with the same color scheme and API contracts.
