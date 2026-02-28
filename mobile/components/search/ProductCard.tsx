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
      <View style={{ aspectRatio: 4 / 3 }} className="bg-slate-100">
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
