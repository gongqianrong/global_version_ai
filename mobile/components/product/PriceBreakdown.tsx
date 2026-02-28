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
        <Text className="text-sm text-slate-600">
          {t("product.itemPrice")}
        </Text>
        <Text className="text-sm font-medium">
          {formatPrice(originalPrice)}
        </Text>
      </View>
      <View className="mt-1 flex-row justify-between">
        <Text className="text-sm text-slate-600">
          {t("product.serviceFee")}
        </Text>
        <Text className="text-sm font-medium">
          {formatPrice(serviceFeeJPY)}
        </Text>
      </View>
      <View className="my-2 h-px bg-slate-200" />
      <View className="flex-row justify-between">
        <Text className="text-base font-semibold">{t("product.total")}</Text>
        <Text className="text-xl font-bold text-brand">
          {formatPrice(total)}
        </Text>
      </View>
      <Text className="mt-2 text-xs text-slate-500">
        {t("product.shippingNote")}
      </Text>
    </View>
  );
}
