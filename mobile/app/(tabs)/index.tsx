import { useCallback, useMemo } from "react";
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useRouter } from "expo-router";
import { useTranslation } from "react-i18next";
import Ionicons from "@expo/vector-icons/Ionicons";
import ProductCard from "@/components/search/ProductCard";
import { MOCK_CATEGORIES, getMockFeaturedProducts } from "@/lib/mock";
import type { ProductSummary } from "@/lib/types";

export default function HomeScreen() {
  const { t } = useTranslation();
  const router = useRouter();

  const featuredProducts = useMemo(() => getMockFeaturedProducts(), []);

  const handleCategoryPress = useCallback(
    (searchKeyword: string) => {
      if (searchKeyword) {
        router.navigate({ pathname: "/search", params: { q: searchKeyword } });
      } else {
        router.navigate("/search");
      }
    },
    [router],
  );

  const handleSearchBarPress = useCallback(() => {
    router.navigate("/search");
  }, [router]);

  const renderItem = useCallback(
    ({ item }: { item: ProductSummary }) => <ProductCard product={item} />,
    [],
  );

  const renderHeader = useCallback(
    () => (
      <View>
        {/* Search entry bar (non-editable, navigates to Search tab) */}
        <TouchableOpacity
          onPress={handleSearchBarPress}
          activeOpacity={0.7}
          style={{
            flexDirection: "row",
            alignItems: "center",
            marginHorizontal: 16,
            marginTop: 12,
            marginBottom: 16,
            borderRadius: 12,
            backgroundColor: "#FFFFFF",
            paddingHorizontal: 12,
            paddingVertical: 12,
            shadowColor: "#000",
            shadowOpacity: 0.05,
            shadowRadius: 4,
            shadowOffset: { width: 0, height: 1 },
            elevation: 1,
          }}
        >
          <Ionicons name="search" size={20} color="#94A3B8" />
          <Text style={{ marginLeft: 8, fontSize: 16, color: "#94A3B8" }}>
            {t("home.searchHint")}
          </Text>
        </TouchableOpacity>

        {/* Category grid (2 rows x 4 columns) */}
        <View style={{ paddingHorizontal: 16, marginBottom: 20 }}>
          <View
            style={{
              flexDirection: "row",
              flexWrap: "wrap",
              justifyContent: "space-between",
            }}
          >
            {MOCK_CATEGORIES.map((cat) => (
              <TouchableOpacity
                key={cat.labelKey}
                onPress={() => handleCategoryPress(cat.searchKeyword)}
                activeOpacity={0.7}
                style={{
                  width: "23%",
                  alignItems: "center",
                  marginBottom: 16,
                }}
              >
                <View
                  style={{
                    width: 52,
                    height: 52,
                    borderRadius: 16,
                    backgroundColor: "#EFF6FF",
                    alignItems: "center",
                    justifyContent: "center",
                    marginBottom: 6,
                  }}
                >
                  <Ionicons name={cat.icon} size={26} color="#0EA5E9" />
                </View>
                <Text
                  style={{
                    fontSize: 12,
                    color: "#334155",
                    textAlign: "center",
                  }}
                  numberOfLines={1}
                >
                  {t(cat.labelKey)}
                </Text>
              </TouchableOpacity>
            ))}
          </View>
        </View>

        {/* Featured section title */}
        <View
          style={{
            flexDirection: "row",
            alignItems: "center",
            paddingHorizontal: 16,
            marginBottom: 12,
          }}
        >
          <Text style={{ fontSize: 18, fontWeight: "bold", color: "#1E293B" }}>
            {t("home.featured")}
          </Text>
        </View>
      </View>
    ),
    [t, handleSearchBarPress, handleCategoryPress],
  );

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: "#F8FAFC" }} edges={["top"]}>
      <FlatList
        data={featuredProducts}
        renderItem={renderItem}
        keyExtractor={(item) => `home-${item.id}`}
        numColumns={2}
        columnWrapperStyle={{
          paddingHorizontal: 16,
          justifyContent: "space-between",
        }}
        ListHeaderComponent={renderHeader}
      />
    </SafeAreaView>
  );
}
