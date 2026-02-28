import { View, Text } from "react-native";
import { PLATFORM_LABELS, type Platform } from "@/lib/types";

interface PlatformBadgeProps {
  platform: string;
}

export default function PlatformBadge({ platform }: PlatformBadgeProps) {
  const info = PLATFORM_LABELS[platform as Platform];
  if (!info) return null;

  return (
    <View
      className="rounded-full px-2 py-0.5"
      style={{ backgroundColor: info.color }}
    >
      <Text className="text-xs font-medium text-white">{info.name}</Text>
    </View>
  );
}
