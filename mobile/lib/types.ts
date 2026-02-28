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

export const PLATFORM_LABELS: Record<
  Platform,
  { name: string; color: string }
> = {
  yahoo_auction: { name: "Yahoo Auctions", color: "#7B1FA2" },
  surugaya: { name: "駿河屋", color: "#2E7D32" },
};

export const CONDITION_LABELS: Record<
  string,
  { zh: string; ja: string; en: string; color: string; bg: string }
> = {
  new: {
    zh: "全新",
    ja: "新品",
    en: "New",
    color: "#166534",
    bg: "#DCFCE7",
  },
  like_new: {
    zh: "近全新",
    ja: "未使用に近い",
    en: "Like New",
    color: "#1E40AF",
    bg: "#DBEAFE",
  },
  good: {
    zh: "良好",
    ja: "目立った傷なし",
    en: "Good",
    color: "#854D0E",
    bg: "#FEF9C3",
  },
  fair: {
    zh: "尚可",
    ja: "やや傷あり",
    en: "Fair",
    color: "#9A3412",
    bg: "#FFEDD5",
  },
  poor: {
    zh: "較差",
    ja: "傷あり",
    en: "Poor",
    color: "#991B1B",
    bg: "#FEE2E2",
  },
};
