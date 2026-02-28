# Rakutao Mobile App Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an Expo (React Native) mobile app with search screen (dual-channel HTTP+WebSocket) and product detail screen, supporting zh-TW/ja/en, for iOS and Android.

**Architecture:** Expo SDK 52 with Expo Router v4 for file-based routing. NativeWind v4 for Tailwind-style styling. i18next for i18n. Search uses dual-channel pattern: HTTP for cached ES results, WebSocket for real-time platform streaming. Shared `lib/` layer (types, api, ws) designed for future Web reuse.

**Tech Stack:** Expo SDK 52, React Native, TypeScript, NativeWind v4, Expo Router v4, i18next, react-i18next, expo-localization

**Backend API Base URL:** `http://localhost:8080` (gateway service)

**Verified environment:** Node v22.22.0, npm 10.9.4

---

### Task 1: Project Scaffolding (Expo + NativeWind)

**Files:**
- Create: `mobile/` (via create-expo-app)
- Modify: `mobile/package.json` (add deps)
- Create: `mobile/tailwind.config.js`
- Create: `mobile/global.css`
- Modify: `mobile/babel.config.js`

**Step 1: Create Expo project**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai && npx create-expo-app@latest mobile --template tabs
```
Expected: Project created at `mobile/` with tab-based Expo Router template.

**Step 2: Install NativeWind v4**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx expo install nativewind tailwindcss@^3 react-native-reanimated react-native-safe-area-context
```
Expected: Dependencies installed.

**Step 3: Install i18n dependencies**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx expo install expo-localization && npm install i18next react-i18next
```
Expected: i18n deps installed.

**Step 4: Install UI dependencies**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx expo install @gorhom/bottom-sheet expo-haptics
```
Expected: Bottom sheet and haptics installed.

**Step 5: Create tailwind.config.js**

Create `mobile/tailwind.config.js`:
```js
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./app/**/*.{js,jsx,ts,tsx}",
    "./components/**/*.{js,jsx,ts,tsx}",
  ],
  presets: [require("nativewind/preset")],
  theme: {
    extend: {
      colors: {
        brand: {
          DEFAULT: "#0EA5E9",
          50: "#F0F9FF",
          100: "#E0F2FE",
          500: "#0EA5E9",
          600: "#0284C7",
          700: "#0369A1",
        },
        auction: {
          DEFAULT: "#F97316",
          500: "#F97316",
          600: "#EA580C",
        },
      },
    },
  },
  plugins: [],
};
```

**Step 6: Create global.css**

Create `mobile/global.css`:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

**Step 7: Update babel.config.js**

Overwrite `mobile/babel.config.js`:
```js
module.exports = function (api) {
  api.cache(true);
  return {
    presets: [
      ["babel-preset-expo", { jsxImportSource: "nativewind" }],
      "nativewind/babel",
    ],
  };
};
```

**Step 8: Create nativewind-env.d.ts**

Create `mobile/nativewind-env.d.ts`:
```ts
/// <reference types="nativewind/types" />
```

**Step 9: Create metro.config.js for NativeWind**

Create `mobile/metro.config.js`:
```js
const { getDefaultConfig } = require("expo/metro-config");
const { withNativeWind } = require("nativewind/metro");

const config = getDefaultConfig(__dirname);

module.exports = withNativeWind(config, { input: "./global.css" });
```

**Step 10: Verify the app starts**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx expo export --platform web 2>&1 | tail -5
```
Expected: Export succeeds (we use web export as a quick compilation check; actual testing on device/simulator later).

---

### Task 2: TypeScript Types + API Client + WebSocket Client

**Files:**
- Create: `mobile/lib/types.ts`
- Create: `mobile/lib/api.ts`
- Create: `mobile/lib/ws.ts`

**Step 1: Create TypeScript types**

Create `mobile/lib/types.ts`:
```ts
// --- API Envelope ---
export interface APIResponse<T> {
  code: number;
  data: T;
  message?: string;
  request_id?: string;
}

// --- Search ---
export interface SearchQuery {
  keyword: string;
  platforms?: string[];
  brand_id?: string;
  categories?: string[];
  price_min?: number;
  price_max?: number;
  condition?: string[];
  sort?: string;
  page?: number;
  page_size?: number;
  lang?: string;
  content_rating?: string;
}

export interface SearchResponse {
  cached_results: ProductSummary[];
  cached_total: number;
  realtime_stream_id?: string;
  translated_keyword?: string;
  aggregations: SearchAggs;
}

export interface ProductSummary {
  id: string;
  title: string;
  title_original: string;
  image: string;
  price_jpy: number;
  platform: string;
  status: string;
  brand?: string;
  condition?: string;
  tags?: string[];
  is_translated: boolean;
}

export interface SearchAggs {
  platforms: AggBucket[];
  brands: AggBucket[];
  categories: AggBucket[];
  price_ranges: PriceRange[];
}

export interface AggBucket {
  key: string;
  count: number;
}

export interface PriceRange {
  min: number;
  max: number;
  count: number;
}

// --- Product Detail ---
export interface ProductResponse {
  id: string;
  platform: string;
  title: string;
  title_original: string;
  description: string;
  description_original: string;
  images: string[];
  price_jpy: number;
  service_fee_jpy: number;
  original_price: number;
  shipping_type: string;
  shipping_fee_jpy: number;
  brand?: Brand;
  categories: string[];
  condition: string;
  status: string;
  quantity: number;
  seller: SellerInfo;
  variants?: Variant[];
  content_rating: string;
  listed_at: string;
  is_translated: boolean;
}

export interface Brand {
  id: string;
  name: string;
  name_ja: string;
  source: string;
}

export interface SellerInfo {
  seller_id: string;
  seller_name: string;
  rating: number;
  item_count: number;
}

export interface Variant {
  name: string;
  options: string[];
}

// --- WebSocket Stream ---
export type StreamEventType = "results" | "done" | "error";

export interface StreamEvent {
  type: StreamEventType;
  platform?: string;
  products?: ProductSummary[];
  total?: number;
  platforms_searched?: string[];
  message?: string;
}

// --- UI Helpers ---
export type Platform = "yahoo_auction" | "surugaya";

export const PLATFORM_LABELS: Record<Platform, { name: string; color: string }> = {
  yahoo_auction: { name: "Yahoo Auctions", color: "#7B1FA2" },
  surugaya: { name: "駿河屋", color: "#2E7D32" },
};

export const CONDITION_LABELS: Record<
  string,
  { zh: string; ja: string; en: string; color: string; bg: string }
> = {
  new: { zh: "全新", ja: "新品", en: "New", color: "#166534", bg: "#DCFCE7" },
  like_new: { zh: "近全新", ja: "未使用に近い", en: "Like New", color: "#1E40AF", bg: "#DBEAFE" },
  good: { zh: "良好", ja: "目立った傷なし", en: "Good", color: "#854D0E", bg: "#FEF9C3" },
  fair: { zh: "尚可", ja: "やや傷あり", en: "Fair", color: "#9A3412", bg: "#FFEDD5" },
  poor: { zh: "較差", ja: "傷あり", en: "Poor", color: "#991B1B", bg: "#FEE2E2" },
};
```

**Step 2: Create REST API client**

Create `mobile/lib/api.ts`:
```ts
import type {
  APIResponse,
  SearchQuery,
  SearchResponse,
  ProductResponse,
} from "./types";

const API_BASE = process.env.EXPO_PUBLIC_API_URL || "http://localhost:8080";

class ApiError extends Error {
  constructor(public code: number, message: string) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });

  if (!res.ok) {
    throw new ApiError(res.status, `HTTP ${res.status}`);
  }

  const body: APIResponse<T> = await res.json();
  if (body.code !== 0) {
    throw new ApiError(body.code, body.message || "Unknown error");
  }

  return body.data;
}

export async function searchProducts(
  query: SearchQuery
): Promise<SearchResponse> {
  const params = new URLSearchParams();
  params.set("keyword", query.keyword);
  if (query.page) params.set("page", String(query.page));
  if (query.page_size) params.set("page_size", String(query.page_size));
  if (query.platforms?.length)
    params.set("platforms", query.platforms.join(","));
  if (query.brand_id) params.set("brand_id", query.brand_id);
  if (query.categories?.length)
    params.set("categories", query.categories.join(","));
  if (query.price_min) params.set("price_min", String(query.price_min));
  if (query.price_max) params.set("price_max", String(query.price_max));
  if (query.condition?.length)
    params.set("condition", query.condition.join(","));
  if (query.sort) params.set("sort", query.sort);
  if (query.lang) params.set("lang", query.lang);
  if (query.content_rating)
    params.set("content_rating", query.content_rating);

  return request<SearchResponse>(
    `/api/v1/search?${params.toString()}`
  );
}

export async function getProduct(
  id: string,
  lang?: string
): Promise<ProductResponse> {
  const params = new URLSearchParams();
  if (lang) params.set("lang", lang);
  return request<ProductResponse>(
    `/api/v1/products/${encodeURIComponent(id)}?${params.toString()}`
  );
}
```

**Step 3: Create WebSocket client**

Create `mobile/lib/ws.ts`:
```ts
import type { StreamEvent } from "./types";

const WS_BASE = process.env.EXPO_PUBLIC_WS_URL || "ws://localhost:8080";

export type StreamCallback = (event: StreamEvent) => void;

export function connectStream(
  streamID: string,
  onEvent: StreamCallback,
  onClose?: () => void
): WebSocket {
  const ws = new WebSocket(
    `${WS_BASE}/api/v1/search/stream/${streamID}`
  );

  ws.onmessage = (msg: MessageEvent) => {
    try {
      const event: StreamEvent = JSON.parse(
        typeof msg.data === "string" ? msg.data : ""
      );
      onEvent(event);
      if (event.type === "done" || event.type === "error") {
        ws.close();
      }
    } catch {
      // Ignore malformed messages
    }
  };

  ws.onerror = () => {
    ws.close();
  };

  ws.onclose = () => {
    onClose?.();
  };

  return ws;
}
```

**Step 4: Create .env for API URL**

Create `mobile/.env`:
```
EXPO_PUBLIC_API_URL=http://localhost:8080
EXPO_PUBLIC_WS_URL=ws://localhost:8080
```

**Step 5: Verify types compile**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx tsc --noEmit 2>&1 | head -20
```
Expected: No errors or only unrelated template warnings.

---

### Task 3: i18n Setup

**Files:**
- Create: `mobile/i18n/index.ts`
- Create: `mobile/i18n/messages/zh-TW.json`
- Create: `mobile/i18n/messages/ja.json`
- Create: `mobile/i18n/messages/en.json`

**Step 1: Create i18n config**

Create `mobile/i18n/index.ts`:
```ts
import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import * as Localization from "expo-localization";

import zhTW from "./messages/zh-TW.json";
import ja from "./messages/ja.json";
import en from "./messages/en.json";

const deviceLocale = Localization.getLocales()[0]?.languageTag || "zh-TW";

function resolveLocale(tag: string): string {
  if (tag.startsWith("zh")) return "zh-TW";
  if (tag.startsWith("ja")) return "ja";
  return "en";
}

i18n.use(initReactI18next).init({
  resources: {
    "zh-TW": { translation: zhTW },
    ja: { translation: ja },
    en: { translation: en },
  },
  lng: resolveLocale(deviceLocale),
  fallbackLng: "zh-TW",
  interpolation: {
    escapeValue: false,
  },
});

export default i18n;
```

**Step 2: Create zh-TW messages**

Create `mobile/i18n/messages/zh-TW.json`:
```json
{
  "common": {
    "appName": "Rakutao",
    "search": "搜尋",
    "home": "首頁",
    "loading": "載入中...",
    "error": "發生錯誤",
    "noResults": "沒有找到結果",
    "retry": "重試",
    "back": "返回"
  },
  "search": {
    "placeholder": "搜尋商品名稱、品牌...",
    "allPlatforms": "全部平台",
    "filters": "篩選",
    "sort": "排序",
    "relevance": "相關度",
    "priceLow": "價格低到高",
    "priceHigh": "價格高到低",
    "newest": "最新",
    "results": "{{count}} 個結果",
    "streaming": "正在取得即時結果...",
    "streamingDone": "所有平台搜尋完成",
    "platformsDone": "{{done}}/{{total}} 個平台完成",
    "loadMore": "載入更多",
    "fee": "含手續費 ¥{{total}}"
  },
  "filters": {
    "platform": "平台來源",
    "priceRange": "價格區間",
    "brand": "品牌",
    "condition": "商品狀態",
    "priceMin": "最低價",
    "priceMax": "最高價",
    "apply": "套用",
    "clear": "清除全部",
    "reset": "重置"
  },
  "product": {
    "itemPrice": "商品價格",
    "serviceFee": "手續費(10%)",
    "total": "合計",
    "shippingNote": "※國際運費另計",
    "buyNow": "立即購買",
    "placeBid": "出價競拍",
    "currentBid": "目前出價",
    "bidders": "{{count}} 人出價",
    "timeLeft": "剩餘時間",
    "seller": "賣家資訊",
    "rating": "評價",
    "description": "商品描述",
    "translated": "翻譯版",
    "original": "日文原文",
    "condition": "狀態",
    "platform": "來源平台",
    "brand": "品牌"
  },
  "conditions": {
    "new": "全新",
    "like_new": "近全新",
    "good": "良好",
    "fair": "尚可",
    "poor": "較差"
  }
}
```

**Step 3: Create ja messages**

Create `mobile/i18n/messages/ja.json`:
```json
{
  "common": {
    "appName": "Rakutao",
    "search": "検索",
    "home": "ホーム",
    "loading": "読み込み中...",
    "error": "エラーが発生しました",
    "noResults": "結果が見つかりません",
    "retry": "再試行",
    "back": "戻る"
  },
  "search": {
    "placeholder": "商品名、ブランドで検索...",
    "allPlatforms": "すべて",
    "filters": "フィルター",
    "sort": "並び替え",
    "relevance": "関連度",
    "priceLow": "価格の安い順",
    "priceHigh": "価格の高い順",
    "newest": "新着順",
    "results": "{{count}} 件の結果",
    "streaming": "リアルタイム結果を取得中...",
    "streamingDone": "すべてのプラットフォーム検索完了",
    "platformsDone": "{{done}}/{{total}} 完了",
    "loadMore": "もっと見る",
    "fee": "手数料込 ¥{{total}}"
  },
  "filters": {
    "platform": "プラットフォーム",
    "priceRange": "価格帯",
    "brand": "ブランド",
    "condition": "商品状態",
    "priceMin": "最低価格",
    "priceMax": "最高価格",
    "apply": "適用",
    "clear": "クリア",
    "reset": "リセット"
  },
  "product": {
    "itemPrice": "商品価格",
    "serviceFee": "手数料(10%)",
    "total": "合計",
    "shippingNote": "※国際送料は別途",
    "buyNow": "今すぐ購入",
    "placeBid": "入札する",
    "currentBid": "現在の入札額",
    "bidders": "{{count}} 人入札",
    "timeLeft": "残り時間",
    "seller": "出品者情報",
    "rating": "評価",
    "description": "商品説明",
    "translated": "翻訳版",
    "original": "日本語原文",
    "condition": "状態",
    "platform": "出品元",
    "brand": "ブランド"
  },
  "conditions": {
    "new": "新品",
    "like_new": "未使用に近い",
    "good": "目立った傷なし",
    "fair": "やや傷あり",
    "poor": "傷あり"
  }
}
```

**Step 4: Create en messages**

Create `mobile/i18n/messages/en.json`:
```json
{
  "common": {
    "appName": "Rakutao",
    "search": "Search",
    "home": "Home",
    "loading": "Loading...",
    "error": "An error occurred",
    "noResults": "No results found",
    "retry": "Retry",
    "back": "Back"
  },
  "search": {
    "placeholder": "Search products, brands...",
    "allPlatforms": "All Platforms",
    "filters": "Filters",
    "sort": "Sort",
    "relevance": "Relevance",
    "priceLow": "Price: Low to High",
    "priceHigh": "Price: High to Low",
    "newest": "Newest",
    "results": "{{count}} results",
    "streaming": "Fetching live results...",
    "streamingDone": "All platforms done",
    "platformsDone": "{{done}}/{{total}} done",
    "loadMore": "Load More",
    "fee": "w/ fee ¥{{total}}"
  },
  "filters": {
    "platform": "Platform",
    "priceRange": "Price Range",
    "brand": "Brand",
    "condition": "Condition",
    "priceMin": "Min Price",
    "priceMax": "Max Price",
    "apply": "Apply",
    "clear": "Clear All",
    "reset": "Reset"
  },
  "product": {
    "itemPrice": "Item Price",
    "serviceFee": "Service Fee (10%)",
    "total": "Total",
    "shippingNote": "International shipping calculated separately",
    "buyNow": "Buy Now",
    "placeBid": "Place Bid",
    "currentBid": "Current Bid",
    "bidders": "{{count}} bidders",
    "timeLeft": "Time Left",
    "seller": "Seller Info",
    "rating": "Rating",
    "description": "Description",
    "translated": "Translated",
    "original": "Original (JP)",
    "condition": "Condition",
    "platform": "Platform",
    "brand": "Brand"
  },
  "conditions": {
    "new": "New",
    "like_new": "Like New",
    "good": "Good",
    "fair": "Fair",
    "poor": "Poor"
  }
}
```

**Step 5: Verify no syntax errors**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && node -e "require('./i18n/messages/zh-TW.json'); require('./i18n/messages/ja.json'); require('./i18n/messages/en.json'); console.log('OK')"
```
Expected: `OK`

---

### Task 4: Root Layout + Tab Layout

**Files:**
- Modify: `mobile/app/_layout.tsx`
- Modify: `mobile/app/(tabs)/_layout.tsx`

**Step 1: Update root layout with i18n and NativeWind**

Overwrite `mobile/app/_layout.tsx`:
```tsx
import "../global.css";
import "../i18n";
import { Stack } from "expo-router";
import { GestureHandlerRootView } from "react-native-gesture-handler";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { StatusBar } from "expo-status-bar";

export default function RootLayout() {
  return (
    <GestureHandlerRootView style={{ flex: 1 }}>
      <SafeAreaProvider>
        <StatusBar style="dark" />
        <Stack screenOptions={{ headerShown: false }}>
          <Stack.Screen name="(tabs)" />
          <Stack.Screen
            name="product/[id]"
            options={{ headerShown: true, headerTitle: "" }}
          />
        </Stack>
      </SafeAreaProvider>
    </GestureHandlerRootView>
  );
}
```

**Step 2: Update tab layout**

Overwrite `mobile/app/(tabs)/_layout.tsx`:
```tsx
import { Tabs } from "expo-router";
import { useTranslation } from "react-i18next";
import Ionicons from "@expo/vector-icons/Ionicons";

export default function TabLayout() {
  const { t } = useTranslation();

  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: "#0EA5E9",
        tabBarInactiveTintColor: "#64748B",
        tabBarStyle: {
          borderTopColor: "#E2E8F0",
        },
      }}
    >
      <Tabs.Screen
        name="index"
        options={{
          title: t("common.search"),
          tabBarIcon: ({ color, size }) => (
            <Ionicons name="search" size={size} color={color} />
          ),
        }}
      />
    </Tabs>
  );
}
```

**Step 3: Clean up template files**

Remove any extra template tab files (explore.tsx, etc.) that came with the tabs template, keeping only `index.tsx`.

**Step 4: Create minimal search tab placeholder**

Overwrite `mobile/app/(tabs)/index.tsx`:
```tsx
import { View, Text } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useTranslation } from "react-i18next";

export default function SearchScreen() {
  const { t } = useTranslation();

  return (
    <SafeAreaView className="flex-1 bg-slate-50">
      <View className="p-4">
        <Text className="text-2xl font-bold text-brand">
          {t("common.appName")}
        </Text>
        <Text className="mt-2 text-slate-500">{t("common.loading")}</Text>
      </View>
    </SafeAreaView>
  );
}
```

**Step 5: Verify the app compiles**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx expo export --platform web 2>&1 | tail -5
```
Expected: Export succeeds.

---

### Task 5: Common Components (PlatformBadge + ConditionBadge)

**Files:**
- Create: `mobile/components/common/PlatformBadge.tsx`
- Create: `mobile/components/common/ConditionBadge.tsx`

**Step 1: Create PlatformBadge**

Create `mobile/components/common/PlatformBadge.tsx`:
```tsx
import { View, Text } from "react-native";
import { PLATFORM_LABELS, type Platform } from "@/lib/types";

interface PlatformBadgeProps {
  platform: string;
}

export default function PlatformBadge({ platform }: PlatformBadgeProps) {
  const info = PLATFORM_LABELS[platform as Platform];
  if (!info) return null;

  return (
    <View
      className="rounded-full px-2 py-0.5"
      style={{ backgroundColor: info.color }}
    >
      <Text className="text-xs font-medium text-white">{info.name}</Text>
    </View>
  );
}
```

**Step 2: Create ConditionBadge**

Create `mobile/components/common/ConditionBadge.tsx`:
```tsx
import { View, Text } from "react-native";
import { useTranslation } from "react-i18next";
import { CONDITION_LABELS } from "@/lib/types";

interface ConditionBadgeProps {
  condition: string;
}

export default function ConditionBadge({ condition }: ConditionBadgeProps) {
  const { t } = useTranslation();
  const info = CONDITION_LABELS[condition];
  if (!info) return null;

  return (
    <View className="rounded-full px-2 py-0.5" style={{ backgroundColor: info.bg }}>
      <Text className="text-xs font-medium" style={{ color: info.color }}>
        {t(`conditions.${condition}`)}
      </Text>
    </View>
  );
}
```

**Step 3: Verify compile**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx tsc --noEmit 2>&1 | head -20
```
Expected: No errors.

---

### Task 6: SearchBar + StreamingBar + ProductCard Components

**Files:**
- Create: `mobile/components/search/SearchBar.tsx`
- Create: `mobile/components/search/StreamingBar.tsx`
- Create: `mobile/components/search/ProductCard.tsx`

**Step 1: Create SearchBar**

Create `mobile/components/search/SearchBar.tsx`:
```tsx
import { useState } from "react";
import { View, TextInput, TouchableOpacity, Text } from "react-native";
import { useTranslation } from "react-i18next";
import * as Haptics from "expo-haptics";
import Ionicons from "@expo/vector-icons/Ionicons";

interface SearchBarProps {
  onSearch: (keyword: string, platform: string) => void;
  loading?: boolean;
}

const PLATFORMS = [
  { value: "", labelKey: "search.allPlatforms" },
  { value: "yahoo_auction", label: "Yahoo" },
  { value: "surugaya", label: "駿河屋" },
] as const;

export default function SearchBar({ onSearch, loading }: SearchBarProps) {
  const { t } = useTranslation();
  const [keyword, setKeyword] = useState("");
  const [platformIdx, setPlatformIdx] = useState(0);

  function handleSubmit() {
    const trimmed = keyword.trim();
    if (!trimmed) return;
    Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    onSearch(trimmed, PLATFORMS[platformIdx].value);
  }

  function cyclePlatform() {
    setPlatformIdx((prev) => (prev + 1) % PLATFORMS.length);
  }

  const currentPlatform = PLATFORMS[platformIdx];
  const platformLabel =
    currentPlatform.value === ""
      ? t(currentPlatform.labelKey)
      : currentPlatform.label;

  return (
    <View className="flex-row items-center gap-2 px-4 py-3">
      {/* Search input */}
      <View className="flex-1 flex-row items-center rounded-xl bg-white px-3 py-2 shadow-sm">
        <Ionicons name="search" size={20} color="#94A3B8" />
        <TextInput
          value={keyword}
          onChangeText={setKeyword}
          placeholder={t("search.placeholder")}
          placeholderTextColor="#94A3B8"
          returnKeyType="search"
          onSubmitEditing={handleSubmit}
          className="ml-2 flex-1 text-base text-slate-900"
        />
      </View>

      {/* Platform toggle */}
      <TouchableOpacity
        onPress={cyclePlatform}
        className="rounded-xl bg-white px-3 py-3 shadow-sm"
      >
        <Text className="text-sm font-medium text-slate-700">
          {platformLabel}
        </Text>
      </TouchableOpacity>

      {/* Search button */}
      <TouchableOpacity
        onPress={handleSubmit}
        disabled={loading}
        className="rounded-xl bg-brand px-4 py-3"
        style={{ opacity: loading ? 0.5 : 1 }}
      >
        <Ionicons name="arrow-forward" size={20} color="#FFFFFF" />
      </TouchableOpacity>
    </View>
  );
}
```

**Step 2: Create StreamingBar**

Create `mobile/components/search/StreamingBar.tsx`:
```tsx
import { View, Text } from "react-native";
import { useTranslation } from "react-i18next";

interface PlatformStatus {
  platform: string;
  done: boolean;
}

interface StreamingBarProps {
  streaming: boolean;
  platforms: PlatformStatus[];
}

export default function StreamingBar({
  streaming,
  platforms,
}: StreamingBarProps) {
  const { t } = useTranslation();

  if (!streaming && platforms.length === 0) return null;

  const done = platforms.filter((p) => p.done).length;
  const total = platforms.length;
  const progress = total > 0 ? done / total : 0;

  return (
    <View className="mx-4 mb-2 flex-row items-center gap-3 rounded-lg bg-brand-50 px-3 py-2">
      {streaming && (
        <View className="h-3 w-3 rounded-full bg-brand opacity-75" />
      )}
      <Text className="flex-1 text-xs text-brand-700">
        {streaming
          ? t("search.platformsDone", { done, total })
          : t("search.streamingDone")}
      </Text>
      <View className="h-1.5 w-24 overflow-hidden rounded-full bg-slate-200">
        <View
          className="h-full rounded-full bg-brand"
          style={{ width: `${progress * 100}%` }}
        />
      </View>
    </View>
  );
}
```

**Step 3: Create ProductCard**

Create `mobile/components/search/ProductCard.tsx`:
```tsx
import { View, Text, Image, TouchableOpacity, Dimensions } from "react-native";
import { useRouter } from "expo-router";
import { useTranslation } from "react-i18next";
import PlatformBadge from "@/components/common/PlatformBadge";
import ConditionBadge from "@/components/common/ConditionBadge";
import type { ProductSummary } from "@/lib/types";

interface ProductCardProps {
  product: ProductSummary;
}

const CARD_GAP = 12;
const CARD_PADDING = 16;
const screenWidth = Dimensions.get("window").width;
const cardWidth = (screenWidth - CARD_PADDING * 2 - CARD_GAP) / 2;

export default function ProductCard({ product }: ProductCardProps) {
  const router = useRouter();
  const { t } = useTranslation();

  const title = product.is_translated ? product.title : product.title_original;
  const serviceFee = Math.round(product.price_jpy * 0.1);
  const totalPrice = product.price_jpy + serviceFee;

  const formatPrice = (n: number) => `¥${n.toLocaleString()}`;

  return (
    <TouchableOpacity
      onPress={() => router.push(`/product/${product.id}`)}
      activeOpacity={0.7}
      style={{ width: cardWidth }}
      className="mb-3 overflow-hidden rounded-xl bg-white shadow-sm"
    >
      {/* Image */}
      <View className="aspect-[4/3] bg-slate-100">
        {product.image ? (
          <Image
            source={{ uri: product.image }}
            className="h-full w-full"
            resizeMode="cover"
          />
        ) : (
          <View className="flex-1 items-center justify-center">
            <Text className="text-slate-300">No Image</Text>
          </View>
        )}
        <View className="absolute left-1.5 top-1.5">
          <PlatformBadge platform={product.platform} />
        </View>
      </View>

      {/* Content */}
      <View className="p-2.5">
        <Text className="text-sm text-slate-900" numberOfLines={2}>
          {title}
        </Text>

        <Text className="mt-1.5 text-lg font-bold text-slate-900">
          {formatPrice(product.price_jpy)}
        </Text>
        <Text className="text-xs text-slate-500">
          {t("search.fee", { total: totalPrice.toLocaleString() })}
        </Text>

        <View className="mt-1.5 flex-row flex-wrap gap-1">
          {product.condition && (
            <ConditionBadge condition={product.condition} />
          )}
          {product.brand && (
            <View className="rounded-full border border-slate-200 px-2 py-0.5">
              <Text className="text-xs text-slate-600">{product.brand}</Text>
            </View>
          )}
        </View>
      </View>
    </TouchableOpacity>
  );
}
```

**Step 4: Verify compile**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx tsc --noEmit 2>&1 | head -20
```
Expected: No errors.

---

### Task 7: FilterSheet Component (Bottom Sheet)

**Files:**
- Create: `mobile/components/search/FilterSheet.tsx`

**Step 1: Create FilterSheet**

Create `mobile/components/search/FilterSheet.tsx`:
```tsx
import { forwardRef, useCallback, useMemo, useState } from "react";
import { View, Text, TouchableOpacity, TextInput, ScrollView } from "react-native";
import BottomSheet, { BottomSheetBackdrop, BottomSheetScrollView } from "@gorhom/bottom-sheet";
import { useTranslation } from "react-i18next";
import type { SearchAggs } from "@/lib/types";

export interface FilterState {
  platforms: string[];
  brands: string[];
  conditions: string[];
  priceMin?: number;
  priceMax?: number;
}

interface FilterSheetProps {
  aggregations: SearchAggs | null;
  onApply: (filters: FilterState) => void;
}

const PLATFORM_NAMES: Record<string, string> = {
  yahoo_auction: "Yahoo Auctions",
  surugaya: "駿河屋",
};

const FilterSheet = forwardRef<BottomSheet, FilterSheetProps>(
  ({ aggregations, onApply }, ref) => {
    const { t } = useTranslation();
    const snapPoints = useMemo(() => ["70%"], []);
    const [filters, setFilters] = useState<FilterState>({
      platforms: [],
      brands: [],
      conditions: [],
    });
    const [priceMin, setPriceMin] = useState("");
    const [priceMax, setPriceMax] = useState("");

    function toggleItem(
      key: "platforms" | "brands" | "conditions",
      value: string
    ) {
      setFilters((prev) => {
        const arr = prev[key];
        return {
          ...prev,
          [key]: arr.includes(value)
            ? arr.filter((v) => v !== value)
            : [...arr, value],
        };
      });
    }

    function handleApply() {
      onApply({
        ...filters,
        priceMin: priceMin ? Number(priceMin) : undefined,
        priceMax: priceMax ? Number(priceMax) : undefined,
      });
    }

    function handleClear() {
      setFilters({ platforms: [], brands: [], conditions: [] });
      setPriceMin("");
      setPriceMax("");
      onApply({ platforms: [], brands: [], conditions: [] });
    }

    const renderBackdrop = useCallback(
      (props: any) => (
        <BottomSheetBackdrop {...props} disappearsOnIndex={-1} appearsOnIndex={0} />
      ),
      []
    );

    return (
      <BottomSheet
        ref={ref}
        index={-1}
        snapPoints={snapPoints}
        enablePanDownToClose
        backdropComponent={renderBackdrop}
      >
        <BottomSheetScrollView className="px-4">
          {/* Header */}
          <View className="mb-4 flex-row items-center justify-between">
            <Text className="text-lg font-bold text-slate-900">
              {t("search.filters")}
            </Text>
            <TouchableOpacity onPress={handleClear}>
              <Text className="text-sm text-brand">{t("filters.clear")}</Text>
            </TouchableOpacity>
          </View>

          {/* Platform */}
          <Text className="mb-2 text-sm font-semibold text-slate-900">
            {t("filters.platform")}
          </Text>
          <View className="mb-4 flex-row flex-wrap gap-2">
            {(aggregations?.platforms || []).map((p) => (
              <TouchableOpacity
                key={p.key}
                onPress={() => toggleItem("platforms", p.key)}
                className={`rounded-full border px-3 py-1.5 ${
                  filters.platforms.includes(p.key)
                    ? "border-brand bg-brand-50"
                    : "border-slate-200 bg-white"
                }`}
              >
                <Text
                  className={`text-sm ${
                    filters.platforms.includes(p.key)
                      ? "text-brand-700"
                      : "text-slate-700"
                  }`}
                >
                  {PLATFORM_NAMES[p.key] || p.key} ({p.count})
                </Text>
              </TouchableOpacity>
            ))}
          </View>

          {/* Price */}
          <Text className="mb-2 text-sm font-semibold text-slate-900">
            {t("filters.priceRange")}
          </Text>
          <View className="mb-4 flex-row items-center gap-2">
            <TextInput
              value={priceMin}
              onChangeText={setPriceMin}
              placeholder={t("filters.priceMin")}
              keyboardType="numeric"
              className="flex-1 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
            />
            <Text className="text-slate-400">~</Text>
            <TextInput
              value={priceMax}
              onChangeText={setPriceMax}
              placeholder={t("filters.priceMax")}
              keyboardType="numeric"
              className="flex-1 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
            />
          </View>

          {/* Brand */}
          {aggregations?.brands && aggregations.brands.length > 0 && (
            <>
              <Text className="mb-2 text-sm font-semibold text-slate-900">
                {t("filters.brand")}
              </Text>
              <View className="mb-4 flex-row flex-wrap gap-2">
                {aggregations.brands.map((b) => (
                  <TouchableOpacity
                    key={b.key}
                    onPress={() => toggleItem("brands", b.key)}
                    className={`rounded-full border px-3 py-1.5 ${
                      filters.brands.includes(b.key)
                        ? "border-brand bg-brand-50"
                        : "border-slate-200 bg-white"
                    }`}
                  >
                    <Text
                      className={`text-sm ${
                        filters.brands.includes(b.key)
                          ? "text-brand-700"
                          : "text-slate-700"
                      }`}
                    >
                      {b.key} ({b.count})
                    </Text>
                  </TouchableOpacity>
                ))}
              </View>
            </>
          )}

          {/* Condition */}
          <Text className="mb-2 text-sm font-semibold text-slate-900">
            {t("filters.condition")}
          </Text>
          <View className="mb-4 flex-row flex-wrap gap-2">
            {["new", "like_new", "good", "fair", "poor"].map((c) => (
              <TouchableOpacity
                key={c}
                onPress={() => toggleItem("conditions", c)}
                className={`rounded-full border px-3 py-1.5 ${
                  filters.conditions.includes(c)
                    ? "border-brand bg-brand-50"
                    : "border-slate-200 bg-white"
                }`}
              >
                <Text
                  className={`text-sm ${
                    filters.conditions.includes(c)
                      ? "text-brand-700"
                      : "text-slate-700"
                  }`}
                >
                  {t(`conditions.${c}`)}
                </Text>
              </TouchableOpacity>
            ))}
          </View>

          {/* Apply button */}
          <TouchableOpacity
            onPress={handleApply}
            className="mb-8 rounded-xl bg-brand py-3"
          >
            <Text className="text-center text-base font-semibold text-white">
              {t("filters.apply")}
            </Text>
          </TouchableOpacity>
        </BottomSheetScrollView>
      </BottomSheet>
    );
  }
);

FilterSheet.displayName = "FilterSheet";
export default FilterSheet;
```

**Step 2: Verify compile**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx tsc --noEmit 2>&1 | head -20
```
Expected: No errors.

---

### Task 8: useSearch Hook + Search Screen Assembly

**Files:**
- Create: `mobile/hooks/useSearch.ts`
- Modify: `mobile/app/(tabs)/index.tsx`

**Step 1: Create useSearch hook**

Create `mobile/hooks/useSearch.ts`:
```ts
import { useState, useCallback, useRef } from "react";
import { searchProducts } from "@/lib/api";
import { connectStream } from "@/lib/ws";
import type { ProductSummary, SearchAggs, SearchQuery } from "@/lib/types";

interface PlatformStatus {
  platform: string;
  done: boolean;
}

export interface SearchState {
  results: ProductSummary[];
  total: number;
  aggregations: SearchAggs | null;
  translatedKeyword: string | null;
  streaming: boolean;
  platforms: PlatformStatus[];
  loading: boolean;
  error: string | null;
  page: number;
  pageSize: number;
  hasMore: boolean;
}

export function useSearch(lang: string) {
  const [state, setState] = useState<SearchState>({
    results: [],
    total: 0,
    aggregations: null,
    translatedKeyword: null,
    streaming: false,
    platforms: [],
    loading: false,
    error: null,
    page: 1,
    pageSize: 20,
    hasMore: false,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const lastQueryRef = useRef<SearchQuery | null>(null);

  const search = useCallback(
    async (query: SearchQuery) => {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }

      lastQueryRef.current = query;
      const isNewSearch = (query.page || 1) === 1;

      setState((prev) => ({
        ...prev,
        loading: true,
        error: null,
        ...(isNewSearch
          ? { results: [], total: 0, streaming: false, platforms: [], page: 1 }
          : {}),
      }));

      try {
        const resp = await searchProducts({ ...query, lang });

        setState((prev) => {
          const newResults = isNewSearch
            ? resp.cached_results || []
            : [...prev.results, ...(resp.cached_results || [])];
          return {
            ...prev,
            results: newResults,
            total: resp.cached_total,
            aggregations: resp.aggregations,
            translatedKeyword: resp.translated_keyword || null,
            loading: false,
            page: query.page || 1,
            hasMore: newResults.length < resp.cached_total,
          };
        });

        // Connect WebSocket for real-time results (only on first page)
        if (resp.realtime_stream_id && isNewSearch) {
          setState((prev) => ({
            ...prev,
            streaming: true,
            platforms: [
              { platform: "yahoo_auction", done: false },
              { platform: "surugaya", done: false },
            ],
          }));

          const ws = connectStream(
            resp.realtime_stream_id,
            (event) => {
              if (event.type === "results" && event.products) {
                setState((prev) => {
                  const existingIds = new Set(prev.results.map((r) => r.id));
                  const newProducts = event.products!.filter(
                    (p) => !existingIds.has(p.id)
                  );
                  return {
                    ...prev,
                    results: [...prev.results, ...newProducts],
                    total: prev.total + newProducts.length,
                    platforms: prev.platforms.map((p) =>
                      p.platform === event.platform
                        ? { ...p, done: true }
                        : p
                    ),
                  };
                });
              } else if (event.type === "done") {
                setState((prev) => ({
                  ...prev,
                  streaming: false,
                  platforms: prev.platforms.map((p) => ({
                    ...p,
                    done: true,
                  })),
                }));
              } else if (event.type === "error") {
                setState((prev) => ({
                  ...prev,
                  platforms: prev.platforms.map((p) =>
                    p.platform === event.platform
                      ? { ...p, done: true }
                      : p
                  ),
                }));
              }
            },
            () => {
              setState((prev) => ({ ...prev, streaming: false }));
            }
          );

          wsRef.current = ws;
        }
      } catch (err) {
        setState((prev) => ({
          ...prev,
          loading: false,
          error: err instanceof Error ? err.message : "Search failed",
        }));
      }
    },
    [lang]
  );

  const loadMore = useCallback(() => {
    if (state.loading || !state.hasMore || !lastQueryRef.current) return;
    search({ ...lastQueryRef.current, page: state.page + 1 });
  }, [state.loading, state.hasMore, state.page, search]);

  const refresh = useCallback(() => {
    if (!lastQueryRef.current) return;
    search({ ...lastQueryRef.current, page: 1 });
  }, [search]);

  return { ...state, search, loadMore, refresh };
}
```

**Step 2: Assemble search screen**

Overwrite `mobile/app/(tabs)/index.tsx`:
```tsx
import { useCallback, useRef, useState } from "react";
import { View, Text, FlatList, ActivityIndicator, RefreshControl } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useTranslation } from "react-i18next";
import BottomSheet from "@gorhom/bottom-sheet";
import SearchBar from "@/components/search/SearchBar";
import StreamingBar from "@/components/search/StreamingBar";
import ProductCard from "@/components/search/ProductCard";
import FilterSheet, { type FilterState } from "@/components/search/FilterSheet";
import { useSearch } from "@/hooks/useSearch";
import type { ProductSummary } from "@/lib/types";
import { TouchableOpacity } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";

export default function SearchScreen() {
  const { t, i18n } = useTranslation();
  const lang = i18n.language;
  const {
    results,
    total,
    aggregations,
    translatedKeyword,
    streaming,
    platforms,
    loading,
    error,
    hasMore,
    search,
    loadMore,
    refresh,
  } = useSearch(lang);

  const [currentKeyword, setCurrentKeyword] = useState("");
  const filterRef = useRef<BottomSheet>(null);

  const handleSearch = useCallback(
    (keyword: string, platform: string) => {
      setCurrentKeyword(keyword);
      search({
        keyword,
        platforms: platform ? [platform] : undefined,
        page: 1,
        page_size: 20,
        lang,
      });
    },
    [search, lang]
  );

  const handleFilter = useCallback(
    (filters: FilterState) => {
      if (!currentKeyword) return;
      filterRef.current?.close();
      search({
        keyword: currentKeyword,
        platforms: filters.platforms.length > 0 ? filters.platforms : undefined,
        brand_id: filters.brands[0],
        condition:
          filters.conditions.length > 0 ? filters.conditions : undefined,
        price_min: filters.priceMin,
        price_max: filters.priceMax,
        page: 1,
        page_size: 20,
        lang,
      });
    },
    [search, currentKeyword, lang]
  );

  const hasSearched = currentKeyword !== "";

  const renderItem = useCallback(
    ({ item }: { item: ProductSummary }) => <ProductCard product={item} />,
    []
  );

  const renderHeader = useCallback(
    () => (
      <View>
        {/* Translated keyword hint */}
        {translatedKeyword && (
          <Text className="mb-1 px-4 text-sm text-slate-500">
            翻訳: {translatedKeyword}
          </Text>
        )}

        {/* Result count + filter button */}
        {hasSearched && !loading && (
          <View className="mb-2 flex-row items-center justify-between px-4">
            <Text className="text-sm text-slate-500">
              {t("search.results", { count: total })}
            </Text>
            <TouchableOpacity
              onPress={() => filterRef.current?.snapToIndex(0)}
              className="flex-row items-center gap-1"
            >
              <Ionicons name="options-outline" size={18} color="#64748B" />
              <Text className="text-sm text-slate-500">
                {t("search.filters")}
              </Text>
            </TouchableOpacity>
          </View>
        )}

        {/* Streaming indicator */}
        <StreamingBar streaming={streaming} platforms={platforms} />

        {/* Error */}
        {error && (
          <View className="mx-4 mb-2 rounded-lg bg-red-50 p-3">
            <Text className="text-sm text-red-700">{error}</Text>
          </View>
        )}
      </View>
    ),
    [translatedKeyword, hasSearched, loading, total, streaming, platforms, error, t]
  );

  const renderEmpty = useCallback(() => {
    if (loading) {
      return (
        <View className="items-center py-20">
          <ActivityIndicator size="large" color="#0EA5E9" />
        </View>
      );
    }
    if (hasSearched) {
      return (
        <View className="items-center py-20">
          <Text className="text-lg text-slate-400">
            {t("common.noResults")}
          </Text>
        </View>
      );
    }
    // Landing state
    return (
      <View className="items-center py-20">
        <Text className="text-3xl font-bold text-slate-800">
          {t("common.appName")}
        </Text>
        <Text className="mt-3 text-base text-slate-500">
          搜尋日本 Yahoo 拍賣、駿河屋等平台的商品
        </Text>
      </View>
    );
  }, [loading, hasSearched, t]);

  const renderFooter = useCallback(() => {
    if (!hasMore || !hasSearched) return null;
    return (
      <View className="items-center py-4">
        <ActivityIndicator size="small" color="#0EA5E9" />
      </View>
    );
  }, [hasMore, hasSearched]);

  return (
    <SafeAreaView className="flex-1 bg-slate-50" edges={["top"]}>
      <SearchBar onSearch={handleSearch} loading={loading} />

      <FlatList
        data={results}
        renderItem={renderItem}
        keyExtractor={(item) => item.id}
        numColumns={2}
        columnWrapperStyle={{
          paddingHorizontal: 16,
          justifyContent: "space-between",
        }}
        ListHeaderComponent={renderHeader}
        ListEmptyComponent={renderEmpty}
        ListFooterComponent={renderFooter}
        onEndReached={loadMore}
        onEndReachedThreshold={0.5}
        refreshControl={
          hasSearched ? (
            <RefreshControl refreshing={false} onRefresh={refresh} />
          ) : undefined
        }
      />

      <FilterSheet
        ref={filterRef}
        aggregations={aggregations}
        onApply={handleFilter}
      />
    </SafeAreaView>
  );
}
```

**Step 3: Verify compile**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx tsc --noEmit 2>&1 | head -20
```
Expected: No errors.

---

### Task 9: Product Detail Components (ImageCarousel + PriceBreakdown + DescriptionTabs)

**Files:**
- Create: `mobile/components/product/ImageCarousel.tsx`
- Create: `mobile/components/product/PriceBreakdown.tsx`
- Create: `mobile/components/product/DescriptionTabs.tsx`

**Step 1: Create ImageCarousel**

Create `mobile/components/product/ImageCarousel.tsx`:
```tsx
import { useState } from "react";
import {
  View,
  Image,
  FlatList,
  Dimensions,
  Text,
} from "react-native";

const { width: SCREEN_WIDTH } = Dimensions.get("window");

interface ImageCarouselProps {
  images: string[];
}

export default function ImageCarousel({ images }: ImageCarouselProps) {
  const [activeIndex, setActiveIndex] = useState(0);

  if (images.length === 0) {
    return (
      <View className="aspect-[4/3] items-center justify-center bg-slate-100">
        <Text className="text-slate-300">No Image</Text>
      </View>
    );
  }

  return (
    <View>
      <FlatList
        data={images}
        horizontal
        pagingEnabled
        showsHorizontalScrollIndicator={false}
        onMomentumScrollEnd={(e) => {
          const index = Math.round(
            e.nativeEvent.contentOffset.x / SCREEN_WIDTH
          );
          setActiveIndex(index);
        }}
        renderItem={({ item }) => (
          <Image
            source={{ uri: item }}
            style={{ width: SCREEN_WIDTH, aspectRatio: 4 / 3 }}
            resizeMode="contain"
            className="bg-slate-100"
          />
        )}
        keyExtractor={(_, i) => String(i)}
      />
      {/* Page dots */}
      {images.length > 1 && (
        <View className="mt-2 flex-row justify-center gap-1.5">
          {images.map((_, i) => (
            <View
              key={i}
              className={`h-1.5 w-1.5 rounded-full ${
                i === activeIndex ? "bg-brand" : "bg-slate-300"
              }`}
            />
          ))}
        </View>
      )}
    </View>
  );
}
```

**Step 2: Create PriceBreakdown**

Create `mobile/components/product/PriceBreakdown.tsx`:
```tsx
import { View, Text } from "react-native";
import { useTranslation } from "react-i18next";

interface PriceBreakdownProps {
  originalPrice: number;
  serviceFeeJPY: number;
}

export default function PriceBreakdown({
  originalPrice,
  serviceFeeJPY,
}: PriceBreakdownProps) {
  const { t } = useTranslation();
  const formatPrice = (n: number) => `¥${n.toLocaleString()}`;
  const total = originalPrice + serviceFeeJPY;

  return (
    <View className="rounded-xl bg-white p-4 shadow-sm">
      <View className="flex-row justify-between">
        <Text className="text-sm text-slate-600">{t("product.itemPrice")}</Text>
        <Text className="text-sm font-medium">{formatPrice(originalPrice)}</Text>
      </View>
      <View className="mt-1 flex-row justify-between">
        <Text className="text-sm text-slate-600">{t("product.serviceFee")}</Text>
        <Text className="text-sm font-medium">{formatPrice(serviceFeeJPY)}</Text>
      </View>
      <View className="my-2 h-px bg-slate-200" />
      <View className="flex-row justify-between">
        <Text className="text-base font-semibold">{t("product.total")}</Text>
        <Text className="text-xl font-bold text-brand">{formatPrice(total)}</Text>
      </View>
      <Text className="mt-2 text-xs text-slate-500">
        {t("product.shippingNote")}
      </Text>
    </View>
  );
}
```

**Step 3: Create DescriptionTabs**

Create `mobile/components/product/DescriptionTabs.tsx`:
```tsx
import { useState } from "react";
import { View, Text, TouchableOpacity, ScrollView } from "react-native";
import { useTranslation } from "react-i18next";

interface DescriptionTabsProps {
  description: string;
  descriptionOriginal: string;
}

export default function DescriptionTabs({
  description,
  descriptionOriginal,
}: DescriptionTabsProps) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<"translated" | "original">("translated");

  return (
    <View>
      {/* Tab headers */}
      <View className="flex-row border-b border-slate-200">
        <TouchableOpacity
          onPress={() => setTab("translated")}
          className={`flex-1 py-3 ${
            tab === "translated" ? "border-b-2 border-brand" : ""
          }`}
        >
          <Text
            className={`text-center text-sm font-medium ${
              tab === "translated" ? "text-brand" : "text-slate-500"
            }`}
          >
            {t("product.translated")}
          </Text>
        </TouchableOpacity>
        <TouchableOpacity
          onPress={() => setTab("original")}
          className={`flex-1 py-3 ${
            tab === "original" ? "border-b-2 border-brand" : ""
          }`}
        >
          <Text
            className={`text-center text-sm font-medium ${
              tab === "original" ? "text-brand" : "text-slate-500"
            }`}
          >
            {t("product.original")}
          </Text>
        </TouchableOpacity>
      </View>

      {/* Content */}
      <Text className="mt-3 text-sm leading-6 text-slate-700">
        {tab === "translated"
          ? description || descriptionOriginal
          : descriptionOriginal}
      </Text>
    </View>
  );
}
```

**Step 4: Verify compile**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx tsc --noEmit 2>&1 | head -20
```
Expected: No errors.

---

### Task 10: Product Detail Screen + Final Verification

**Files:**
- Create: `mobile/app/product/[id].tsx`
- Create: `mobile/hooks/useProduct.ts`

**Step 1: Create useProduct hook**

Create `mobile/hooks/useProduct.ts`:
```ts
import { useState, useEffect } from "react";
import { getProduct } from "@/lib/api";
import type { ProductResponse } from "@/lib/types";

export function useProduct(id: string, lang: string) {
  const [product, setProduct] = useState<ProductResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function fetchProduct() {
      setLoading(true);
      setError(null);
      try {
        const data = await getProduct(id, lang);
        if (!cancelled) setProduct(data);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to load product");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchProduct();
    return () => {
      cancelled = true;
    };
  }, [id, lang]);

  return { product, loading, error };
}
```

**Step 2: Create product detail screen**

Create `mobile/app/product/[id].tsx`:
```tsx
import { View, Text, ScrollView, TouchableOpacity, ActivityIndicator } from "react-native";
import { useLocalSearchParams, Stack } from "expo-router";
import { useTranslation } from "react-i18next";
import { useProduct } from "@/hooks/useProduct";
import ImageCarousel from "@/components/product/ImageCarousel";
import PriceBreakdown from "@/components/product/PriceBreakdown";
import DescriptionTabs from "@/components/product/DescriptionTabs";
import PlatformBadge from "@/components/common/PlatformBadge";
import ConditionBadge from "@/components/common/ConditionBadge";

export default function ProductScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const { t, i18n } = useTranslation();
  const { product, loading, error } = useProduct(id!, i18n.language);

  if (loading) {
    return (
      <View className="flex-1 items-center justify-center bg-slate-50">
        <ActivityIndicator size="large" color="#0EA5E9" />
      </View>
    );
  }

  if (error || !product) {
    return (
      <View className="flex-1 items-center justify-center bg-slate-50">
        <Text className="text-lg text-red-500">{error || t("common.error")}</Text>
      </View>
    );
  }

  const title = product.is_translated ? product.title : product.title_original;

  return (
    <>
      <Stack.Screen
        options={{
          headerTitle: "",
          headerRight: () => (
            <PlatformBadge platform={product.platform} />
          ),
        }}
      />

      <ScrollView className="flex-1 bg-slate-50">
        {/* Image carousel */}
        <ImageCarousel images={product.images} />

        <View className="px-4 py-4">
          {/* Title */}
          <Text className="text-xl font-bold text-slate-900">{title}</Text>

          {/* Badges */}
          <View className="mt-2 flex-row flex-wrap gap-2">
            {product.condition && (
              <ConditionBadge condition={product.condition} />
            )}
            {product.brand && (
              <View className="rounded-full border border-slate-200 px-2.5 py-0.5">
                <Text className="text-sm text-slate-700">
                  {product.brand.name}
                </Text>
              </View>
            )}
          </View>

          {/* Price breakdown */}
          <View className="mt-4">
            <PriceBreakdown
              originalPrice={product.original_price}
              serviceFeeJPY={product.service_fee_jpy}
            />
          </View>

          {/* Seller info */}
          <View className="mt-4 rounded-xl bg-white p-4 shadow-sm">
            <Text className="text-sm font-semibold text-slate-900">
              {t("product.seller")}
            </Text>
            <View className="mt-2 flex-row items-center gap-3">
              <Text className="text-sm text-slate-600">
                {product.seller.seller_name}
              </Text>
              <Text className="text-sm text-slate-500">
                {"★".repeat(Math.round(product.seller.rating))}{" "}
                {product.seller.rating.toFixed(1)}
              </Text>
            </View>
          </View>

          {/* Description */}
          <View className="mt-4">
            <Text className="mb-2 text-base font-semibold text-slate-900">
              {t("product.description")}
            </Text>
            <DescriptionTabs
              description={product.description}
              descriptionOriginal={product.description_original}
            />
          </View>

          {/* Bottom spacing for action buttons */}
          <View className="h-24" />
        </View>
      </ScrollView>

      {/* Sticky action buttons at bottom */}
      <View className="absolute bottom-0 left-0 right-0 flex-row gap-3 border-t border-slate-200 bg-white px-4 pb-8 pt-3">
        <TouchableOpacity className="flex-1 rounded-xl bg-green-600 py-3.5">
          <Text className="text-center text-base font-semibold text-white">
            {t("product.buyNow")}
          </Text>
        </TouchableOpacity>
        <TouchableOpacity className="flex-1 rounded-xl border-2 border-auction py-3.5">
          <Text className="text-center text-base font-semibold text-auction">
            {t("product.placeBid")}
          </Text>
        </TouchableOpacity>
      </View>
    </>
  );
}
```

**Step 3: Full build verification**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx tsc --noEmit && echo "TypeScript OK"
```
Expected: `TypeScript OK`

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx expo export --platform web 2>&1 | tail -5
```
Expected: Export succeeds.

**Step 4: Start dev server**

Run:
```bash
cd /Users/gongqianrong/Desktop/ai/mobile && npx expo start
```
Expected: Expo dev server starts. Scan QR code with Expo Go app on phone, or press `i` for iOS simulator / `a` for Android emulator.
