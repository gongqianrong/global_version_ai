import { View, Text } from "react-native";
import { useTranslation } from "react-i18next";
import { CONDITION_LABELS } from "@/lib/types";

interface ConditionBadgeProps {
  condition: string;
}

export default function ConditionBadge({ condition }: ConditionBadgeProps) {
  const { t } = useTranslation();
  const info = CONDITION_LABELS[condition];
  if (!info) return null;

  return (
    <View
      className="rounded-full px-2 py-0.5"
      style={{ backgroundColor: info.bg }}
    >
      <Text className="text-xs font-medium" style={{ color: info.color }}>
        {t(`conditions.${condition}`)}
      </Text>
    </View>
  );
}
