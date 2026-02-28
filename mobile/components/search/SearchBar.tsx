import { useEffect, useState } from "react";
import { View, TextInput, TouchableOpacity, Text, Platform } from "react-native";
import { useTranslation } from "react-i18next";
import * as Haptics from "expo-haptics";
import Ionicons from "@expo/vector-icons/Ionicons";

interface SearchBarProps {
  onSearch: (keyword: string, platform: string) => void;
  loading?: boolean;
  initialKeyword?: string;
}

const PLATFORMS = [
  { value: "", labelKey: "search.allPlatforms" },
  { value: "yahoo_auction", label: "Yahoo" },
  { value: "surugaya", label: "駿河屋" },
] as const;

export default function SearchBar({ onSearch, loading, initialKeyword }: SearchBarProps) {
  const { t } = useTranslation();
  const [keyword, setKeyword] = useState("");
  const [platformIdx, setPlatformIdx] = useState(0);

  useEffect(() => {
    if (initialKeyword) {
      setKeyword(initialKeyword);
    }
  }, [initialKeyword]);

  function handleSubmit() {
    const trimmed = keyword.trim();
    if (!trimmed) return;
    if (Platform.OS !== "web") {
      Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Light);
    }
    onSearch(trimmed, PLATFORMS[platformIdx].value);
  }

  function cyclePlatform() {
    setPlatformIdx((prev) => (prev + 1) % PLATFORMS.length);
  }

  const currentPlatform = PLATFORMS[platformIdx];
  const platformLabel =
    currentPlatform.value === ""
      ? t(currentPlatform.labelKey)
      : currentPlatform.label;

  return (
    <View style={{ flexDirection: "row", alignItems: "center", gap: 8, paddingHorizontal: 16, paddingVertical: 12 }}>
      {/* Search input */}
      <View style={{ flex: 1, flexDirection: "row", alignItems: "center", borderRadius: 12, backgroundColor: "#FFFFFF", paddingHorizontal: 12, paddingVertical: 8, shadowColor: "#000", shadowOpacity: 0.05, shadowRadius: 4, shadowOffset: { width: 0, height: 1 }, elevation: 1 }}>
        <Ionicons name="search" size={20} color="#94A3B8" />
        <TextInput
          value={keyword}
          onChangeText={setKeyword}
          placeholder={t("search.placeholder")}
          placeholderTextColor="#94A3B8"
          returnKeyType="search"
          onSubmitEditing={handleSubmit}
          style={{ marginLeft: 8, flex: 1, fontSize: 16, color: "#0F172A", outlineStyle: "none" } as any}
        />
      </View>

      {/* Platform toggle */}
      <TouchableOpacity
        onPress={cyclePlatform}
        style={{ borderRadius: 12, backgroundColor: "#FFFFFF", paddingHorizontal: 12, paddingVertical: 12, shadowColor: "#000", shadowOpacity: 0.05, shadowRadius: 4, shadowOffset: { width: 0, height: 1 }, elevation: 1 }}
      >
        <Text style={{ fontSize: 14, fontWeight: "500", color: "#334155" }}>
          {platformLabel}
        </Text>
      </TouchableOpacity>

      {/* Search button */}
      <TouchableOpacity
        onPress={handleSubmit}
        disabled={loading}
        style={{ borderRadius: 12, backgroundColor: "#0EA5E9", paddingHorizontal: 16, paddingVertical: 12, opacity: loading ? 0.5 : 1 }}
      >
        <Ionicons name="arrow-forward" size={20} color="#FFFFFF" />
      </TouchableOpacity>
    </View>
  );
}
