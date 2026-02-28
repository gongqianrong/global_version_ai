import type {
  APIResponse,
  SearchQuery,
  SearchResponse,
  ProductResponse,
} from "./types";

const API_BASE = process.env.EXPO_PUBLIC_API_URL || "http://localhost:8080";

class ApiError extends Error {
  constructor(
    public code: number,
    message: string,
  ) {
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
  query: SearchQuery,
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

  return request<SearchResponse>(`/api/v1/search?${params.toString()}`);
}

export async function getProduct(
  id: string,
  lang?: string,
): Promise<ProductResponse> {
  const params = new URLSearchParams();
  if (lang) params.set("lang", lang);
  return request<ProductResponse>(
    `/api/v1/products/${encodeURIComponent(id)}?${params.toString()}`,
  );
}
