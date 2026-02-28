import { useState } from "react";
import { View, Text, TouchableOpacity } from "react-native";
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
