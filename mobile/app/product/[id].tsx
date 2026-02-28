import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  ActivityIndicator,
} from "react-native";
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
        <Text className="text-lg text-red-500">
          {error || t("common.error")}
        </Text>
      </View>
    );
  }

  const title = product.is_translated
    ? product.title
    : product.title_original;

  return (
    <>
      <Stack.Screen
        options={{
          headerTitle: "",
          headerRight: () => <PlatformBadge platform={product.platform} />,
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
