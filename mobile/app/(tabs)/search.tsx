import { useCallback, useEffect, useRef, useState } from "react";
import {
  View,
  Text,
  FlatList,
  ActivityIndicator,
  TouchableOpacity,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useLocalSearchParams } from "expo-router";
import { useTranslation } from "react-i18next";
import BottomSheet from "@gorhom/bottom-sheet";
import Ionicons from "@expo/vector-icons/Ionicons";
import SearchBar from "@/components/search/SearchBar";
import StreamingBar from "@/components/search/StreamingBar";
import ProductCard from "@/components/search/ProductCard";
import FilterSheet, { type FilterState } from "@/components/search/FilterSheet";
import { useSearch } from "@/hooks/useSearch";
import type { ProductSummary } from "@/lib/types";

export default function SearchScreen() {
  const { t, i18n } = useTranslation();
  const lang = i18n.language;
  const { q } = useLocalSearchParams<{ q?: string }>();
  const {
    results,
    total,
    translatedKeyword,
    streaming,
    platforms,
    loading,
    error,
    hasMore,
    aggregations,
    search,
    loadMore,
    refresh,
  } = useSearch(lang);

  const [currentKeyword, setCurrentKeyword] = useState("");
  const [initialKeyword, setInitialKeyword] = useState("");
  const filterRef = useRef<BottomSheet>(null);

  useEffect(() => {
    if (q) {
      setInitialKeyword(q);
      setCurrentKeyword(q);
      search({ keyword: q, page: 1, page_size: 20, lang });
    }
  }, [q, search, lang]);

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
    [search, lang],
  );

  const hasSearched = currentKeyword !== "";

  const renderItem = useCallback(
    ({ item }: { item: ProductSummary }) => <ProductCard product={item} />,
    [],
  );

  const renderHeader = useCallback(
    () => (
      <View>
        {translatedKeyword && (
          <Text style={{ marginBottom: 4, paddingHorizontal: 16, fontSize: 14, color: "#64748B" }}>
            翻訳: {translatedKeyword}
          </Text>
        )}

        {hasSearched && !loading && (
          <View style={{ marginBottom: 8, flexDirection: "row", alignItems: "center", justifyContent: "space-between", paddingHorizontal: 16 }}>
            <Text style={{ fontSize: 14, color: "#64748B" }}>
              {t("search.results", { count: total })}
            </Text>
            <TouchableOpacity
              onPress={() => filterRef.current?.snapToIndex(0)}
              style={{ flexDirection: "row", alignItems: "center", gap: 4 }}
            >
              <Ionicons name="options-outline" size={18} color="#64748B" />
              <Text style={{ fontSize: 14, color: "#64748B" }}>
                {t("search.filters")}
              </Text>
            </TouchableOpacity>
          </View>
        )}

        <StreamingBar streaming={streaming} platforms={platforms} />

        {error && (
          <View style={{ marginHorizontal: 16, marginBottom: 8, borderRadius: 8, backgroundColor: "#FEF2F2", padding: 12 }}>
            <Text style={{ fontSize: 14, color: "#B91C1C" }}>{error}</Text>
          </View>
        )}
      </View>
    ),
    [translatedKeyword, hasSearched, loading, total, streaming, platforms, error, t, filterRef],
  );

  const renderEmpty = useCallback(() => {
    if (loading) {
      return (
        <View style={{ alignItems: "center", paddingVertical: 80 }}>
          <ActivityIndicator size="large" color="#0EA5E9" />
        </View>
      );
    }
    if (hasSearched) {
      return (
        <View style={{ alignItems: "center", paddingVertical: 80 }}>
          <Text style={{ fontSize: 18, color: "#94A3B8" }}>
            {t("common.noResults")}
          </Text>
        </View>
      );
    }
    return (
      <View style={{ alignItems: "center", paddingVertical: 80 }}>
        <Text style={{ fontSize: 24, fontWeight: "bold", color: "#1E293B" }}>
          {t("common.appName")}
        </Text>
        <Text style={{ marginTop: 12, fontSize: 16, color: "#64748B" }}>
          搜尋日本 Yahoo 拍賣、駿河屋等平台的商品
        </Text>
      </View>
    );
  }, [loading, hasSearched, t]);

  const renderFooter = useCallback(() => {
    if (!hasMore || !hasSearched) return null;
    return (
      <View style={{ alignItems: "center", paddingVertical: 16 }}>
        <ActivityIndicator size="small" color="#0EA5E9" />
      </View>
    );
  }, [hasMore, hasSearched]);

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: "#F8FAFC" }} edges={["top"]}>
      <SearchBar onSearch={handleSearch} loading={loading} initialKeyword={initialKeyword} />

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
      />

      <FilterSheet
        ref={filterRef}
        aggregations={aggregations}
        onApply={(filters: FilterState) => {
          if (!currentKeyword) return;
          filterRef.current?.close();
          search({
            keyword: currentKeyword,
            platforms: filters.platforms.length > 0 ? filters.platforms : undefined,
            brand_id: filters.brands[0],
            condition: filters.conditions.length > 0 ? filters.conditions : undefined,
            price_min: filters.priceMin,
            price_max: filters.priceMax,
            page: 1,
            page_size: 20,
            lang,
          });
        }}
      />
    </SafeAreaView>
  );
}
