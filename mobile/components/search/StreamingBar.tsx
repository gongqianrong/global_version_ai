import { View, Text } from "react-native";
import { useTranslation } from "react-i18next";

interface PlatformStatus {
  platform: string;
  done: boolean;
}

interface StreamingBarProps {
  streaming: boolean;
  platforms: PlatformStatus[];
}

export default function StreamingBar({
  streaming,
  platforms,
}: StreamingBarProps) {
  const { t } = useTranslation();

  if (!streaming && platforms.length === 0) return null;

  const done = platforms.filter((p) => p.done).length;
  const total = platforms.length;
  const progress = total > 0 ? done / total : 0;

  return (
    <View className="mx-4 mb-2 flex-row items-center gap-3 rounded-lg bg-brand-50 px-3 py-2">
      {streaming && (
        <View className="h-3 w-3 rounded-full bg-brand opacity-75" />
      )}
      <Text className="flex-1 text-xs text-brand-700">
        {streaming
          ? t("search.platformsDone", { done, total })
          : t("search.streamingDone")}
      </Text>
      <View className="h-1.5 w-24 overflow-hidden rounded-full bg-slate-200">
        <View
          className="h-full rounded-full bg-brand"
          style={{ width: `${progress * 100}%` }}
        />
      </View>
    </View>
  );
}
