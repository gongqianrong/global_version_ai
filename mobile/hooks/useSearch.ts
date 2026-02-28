import { useState, useCallback, useRef } from "react";
import { searchProducts } from "@/lib/api";
import { connectStream } from "@/lib/ws";
import { getMockSearchResponse } from "@/lib/mock";
import type { ProductSummary, SearchAggs, SearchQuery } from "@/lib/types";

const USE_MOCK = __DEV__;

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
          ? {
              results: [],
              total: 0,
              streaming: false,
              platforms: [],
              page: 1,
            }
          : {}),
      }));

      try {
        const resp = USE_MOCK
          ? getMockSearchResponse(query.keyword)
          : await searchProducts({ ...query, lang });

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
                    (p) => !existingIds.has(p.id),
                  );
                  return {
                    ...prev,
                    results: [...prev.results, ...newProducts],
                    total: prev.total + newProducts.length,
                    platforms: prev.platforms.map((p) =>
                      p.platform === event.platform
                        ? { ...p, done: true }
                        : p,
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
                      : p,
                  ),
                }));
              }
            },
            () => {
              setState((prev) => ({ ...prev, streaming: false }));
            },
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
    [lang],
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
