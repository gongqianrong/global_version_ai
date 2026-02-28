import type { Ionicons } from "@expo/vector-icons";
import type { ProductSummary, SearchResponse, ProductResponse, SearchAggs } from "./types";

type IoniconsName = React.ComponentProps<typeof Ionicons>["name"];

export interface MockCategory {
  icon: IoniconsName;
  labelKey: string;
  searchKeyword: string;
}

export const MOCK_CATEGORIES: MockCategory[] = [
  { icon: "body-outline", labelKey: "home.cat_figures", searchKeyword: "フィギュア" },
  { icon: "game-controller-outline", labelKey: "home.cat_games", searchKeyword: "ゲーム機" },
  { icon: "phone-portrait-outline", labelKey: "home.cat_electronics", searchKeyword: "電子機器" },
  { icon: "sparkles-outline", labelKey: "home.cat_anime", searchKeyword: "アニメグッズ" },
  { icon: "gift-outline", labelKey: "home.cat_toys", searchKeyword: "ブラインドボックス" },
  { icon: "camera-outline", labelKey: "home.cat_camera", searchKeyword: "カメラ" },
  { icon: "pricetag-outline", labelKey: "home.cat_brands", searchKeyword: "ブランド" },
  { icon: "grid-outline", labelKey: "home.cat_more", searchKeyword: "" },
];

export function getMockFeaturedProducts(): ProductSummary[] {
  return [...MOCK_PRODUCTS].sort(() => Math.random() - 0.5);
}

const MOCK_PRODUCTS: ProductSummary[] = [
  {
    id: "ya-001",
    title: "高達 RX-78-2 完美模型 MG 1/100",
    title_original: "ガンダム RX-78-2 パーフェクトグレード MG 1/100",
    image: "https://picsum.photos/seed/gundam1/400/300",
    price_jpy: 8500,
    platform: "yahoo_auction",
    status: "active",
    brand: "BANDAI",
    condition: "new",
    tags: ["模型", "高達"],
    is_translated: true,
  },
  {
    id: "sr-001",
    title: "寶可夢 皮卡丘 毛絨玩偶 30cm",
    title_original: "ポケモン ピカチュウ ぬいぐるみ 30cm",
    image: "https://picsum.photos/seed/pikachu/400/300",
    price_jpy: 3200,
    platform: "surugaya",
    status: "active",
    condition: "like_new",
    is_translated: true,
  },
  {
    id: "ya-002",
    title: "索尼 PlayStation 5 數位版 主機",
    title_original: "ソニー PlayStation 5 デジタルエディション 本体",
    image: "https://picsum.photos/seed/ps5/400/300",
    price_jpy: 49800,
    platform: "yahoo_auction",
    status: "active",
    brand: "SONY",
    condition: "good",
    is_translated: true,
  },
  {
    id: "sr-002",
    title: "航海王 路飛 景品公仔 BWFC",
    title_original: "ワンピース ルフィ プライズフィギュア BWFC",
    image: "https://picsum.photos/seed/luffy/400/300",
    price_jpy: 2800,
    platform: "surugaya",
    status: "active",
    brand: "BANPRESTO",
    condition: "new",
    is_translated: true,
  },
  {
    id: "ya-003",
    title: "任天堂 Switch OLED 白色",
    title_original: "Nintendo Switch 有機ELモデル ホワイト",
    image: "https://picsum.photos/seed/switch/400/300",
    price_jpy: 35800,
    platform: "yahoo_auction",
    status: "active",
    brand: "Nintendo",
    condition: "like_new",
    is_translated: true,
  },
  {
    id: "sr-003",
    title: "鬼滅之刃 炭治郎 模型 1/8",
    title_original: "鬼滅の刃 炭治郎 フィギュア 1/8スケール",
    image: "https://picsum.photos/seed/tanjiro/400/300",
    price_jpy: 12000,
    platform: "surugaya",
    status: "active",
    condition: "new",
    is_translated: true,
  },
  {
    id: "ya-004",
    title: "富士 instax mini 12 拍立得相機 粉色",
    title_original: "富士フイルム instax mini 12 チェキ ピンク",
    image: "https://picsum.photos/seed/instax/400/300",
    price_jpy: 9800,
    platform: "yahoo_auction",
    status: "active",
    brand: "FUJIFILM",
    condition: "new",
    is_translated: true,
  },
  {
    id: "sr-004",
    title: "七龍珠 孫悟空 超級賽亞人 景品",
    title_original: "ドラゴンボール 孫悟空 超サイヤ人 プライズ",
    image: "https://picsum.photos/seed/goku/400/300",
    price_jpy: 1980,
    platform: "surugaya",
    status: "active",
    brand: "BANPRESTO",
    condition: "good",
    is_translated: true,
  },
];

const MOCK_AGGS: SearchAggs = {
  platforms: [
    { key: "yahoo_auction", count: 4 },
    { key: "surugaya", count: 4 },
  ],
  brands: [
    { key: "BANDAI", count: 1 },
    { key: "SONY", count: 1 },
    { key: "BANPRESTO", count: 2 },
    { key: "Nintendo", count: 1 },
    { key: "FUJIFILM", count: 1 },
  ],
  categories: [],
  price_ranges: [
    { min: 0, max: 5000, count: 3 },
    { min: 5000, max: 15000, count: 3 },
    { min: 15000, max: 50000, count: 2 },
  ],
};

export function getMockSearchResponse(keyword: string): SearchResponse {
  const filtered = MOCK_PRODUCTS.filter(
    (p) =>
      p.title.includes(keyword) ||
      p.title_original.includes(keyword) ||
      (p.brand && p.brand.toLowerCase().includes(keyword.toLowerCase())) ||
      keyword === "test" ||
      keyword === "測試",
  );
  return {
    cached_results: filtered.length > 0 ? filtered : MOCK_PRODUCTS,
    cached_total: filtered.length > 0 ? filtered.length : MOCK_PRODUCTS.length,
    translated_keyword: "テスト検索",
    aggregations: MOCK_AGGS,
  };
}

export function getMockProduct(id: string): ProductResponse {
  const summary = MOCK_PRODUCTS.find((p) => p.id === id) || MOCK_PRODUCTS[0];
  return {
    id: summary.id,
    platform: summary.platform,
    title: summary.title,
    title_original: summary.title_original,
    description: `這是「${summary.title}」的詳細描述。\n\n商品狀態良好，附帶原包裝。日本國內購入，正品保證。\n\n尺寸：約 20 x 15 x 10 cm\n重量：約 500g`,
    description_original: `「${summary.title_original}」の詳細説明です。\n\n商品状態は良好で、元箱付きです。国内正規品です。\n\nサイズ：約 20 x 15 x 10 cm\n重量：約 500g`,
    images: [
      summary.image,
      `https://picsum.photos/seed/${summary.id}-2/400/300`,
      `https://picsum.photos/seed/${summary.id}-3/400/300`,
    ],
    price_jpy: summary.price_jpy,
    service_fee_jpy: Math.round(summary.price_jpy * 0.1),
    original_price: summary.price_jpy,
    shipping_type: "EMS",
    shipping_fee_jpy: 1500,
    brand: summary.brand
      ? { id: summary.brand.toLowerCase(), name: summary.brand, name_ja: summary.brand, source: "manual" }
      : undefined,
    categories: ["玩具", "收藏品"],
    condition: summary.condition || "good",
    status: "active",
    quantity: 1,
    seller: {
      seller_id: "seller-" + summary.platform,
      seller_name: summary.platform === "yahoo_auction" ? "tokyo_shop_2024" : "駿河屋公式",
      rating: 4.7,
      item_count: 1523,
    },
    content_rating: "general",
    listed_at: "2026-02-20T10:00:00Z",
    is_translated: true,
  };
}
