import { forwardRef, useCallback, useMemo, useState } from "react";
import { View, Text, TouchableOpacity, TextInput } from "react-native";
import BottomSheet, {
  BottomSheetBackdrop,
  BottomSheetBackdropProps,
  BottomSheetScrollView,
} from "@gorhom/bottom-sheet";
import { useTranslation } from "react-i18next";
import type { SearchAggs } from "@/lib/types";

export interface FilterState {
  platforms: string[];
  brands: string[];
  conditions: string[];
  priceMin?: number;
  priceMax?: number;
}

interface FilterSheetProps {
  aggregations: SearchAggs | null;
  onApply: (filters: FilterState) => void;
}

const PLATFORM_NAMES: Record<string, string> = {
  yahoo_auction: "Yahoo Auctions",
  surugaya: "駿河屋",
};

const FilterSheet = forwardRef<BottomSheet, FilterSheetProps>(
  ({ aggregations, onApply }, ref) => {
    const { t } = useTranslation();
    const snapPoints = useMemo(() => ["70%"], []);
    const [filters, setFilters] = useState<FilterState>({
      platforms: [],
      brands: [],
      conditions: [],
    });
    const [priceMin, setPriceMin] = useState("");
    const [priceMax, setPriceMax] = useState("");

    function toggleItem(
      key: "platforms" | "brands" | "conditions",
      value: string,
    ) {
      setFilters((prev) => {
        const arr = prev[key];
        return {
          ...prev,
          [key]: arr.includes(value)
            ? arr.filter((v) => v !== value)
            : [...arr, value],
        };
      });
    }

    function handleApply() {
      onApply({
        ...filters,
        priceMin: priceMin ? Number(priceMin) : undefined,
        priceMax: priceMax ? Number(priceMax) : undefined,
      });
    }

    function handleClear() {
      setFilters({ platforms: [], brands: [], conditions: [] });
      setPriceMin("");
      setPriceMax("");
      onApply({ platforms: [], brands: [], conditions: [] });
    }

    const renderBackdrop = useCallback(
      (props: BottomSheetBackdropProps) => (
        <BottomSheetBackdrop
          {...props}
          disappearsOnIndex={-1}
          appearsOnIndex={0}
        />
      ),
      [],
    );

    return (
      <BottomSheet
        ref={ref}
        index={-1}
        snapPoints={snapPoints}
        enablePanDownToClose
        backdropComponent={renderBackdrop}
      >
        <BottomSheetScrollView className="px-4">
          {/* Header */}
          <View className="mb-4 flex-row items-center justify-between">
            <Text className="text-lg font-bold text-slate-900">
              {t("search.filters")}
            </Text>
            <TouchableOpacity onPress={handleClear}>
              <Text className="text-sm text-brand">{t("filters.clear")}</Text>
            </TouchableOpacity>
          </View>

          {/* Platform */}
          <Text className="mb-2 text-sm font-semibold text-slate-900">
            {t("filters.platform")}
          </Text>
          <View className="mb-4 flex-row flex-wrap gap-2">
            {(aggregations?.platforms || []).map((p) => (
              <TouchableOpacity
                key={p.key}
                onPress={() => toggleItem("platforms", p.key)}
                className={`rounded-full border px-3 py-1.5 ${
                  filters.platforms.includes(p.key)
                    ? "border-brand bg-brand-50"
                    : "border-slate-200 bg-white"
                }`}
              >
                <Text
                  className={`text-sm ${
                    filters.platforms.includes(p.key)
                      ? "text-brand-700"
                      : "text-slate-700"
                  }`}
                >
                  {PLATFORM_NAMES[p.key] || p.key} ({p.count})
                </Text>
              </TouchableOpacity>
            ))}
          </View>

          {/* Price */}
          <Text className="mb-2 text-sm font-semibold text-slate-900">
            {t("filters.priceRange")}
          </Text>
          <View className="mb-4 flex-row items-center gap-2">
            <TextInput
              value={priceMin}
              onChangeText={setPriceMin}
              placeholder={t("filters.priceMin")}
              keyboardType="numeric"
              className="flex-1 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
            />
            <Text className="text-slate-400">~</Text>
            <TextInput
              value={priceMax}
              onChangeText={setPriceMax}
              placeholder={t("filters.priceMax")}
              keyboardType="numeric"
              className="flex-1 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm"
            />
          </View>

          {/* Brand */}
          {aggregations?.brands && aggregations.brands.length > 0 && (
            <>
              <Text className="mb-2 text-sm font-semibold text-slate-900">
                {t("filters.brand")}
              </Text>
              <View className="mb-4 flex-row flex-wrap gap-2">
                {aggregations.brands.map((b) => (
                  <TouchableOpacity
                    key={b.key}
                    onPress={() => toggleItem("brands", b.key)}
                    className={`rounded-full border px-3 py-1.5 ${
                      filters.brands.includes(b.key)
                        ? "border-brand bg-brand-50"
                        : "border-slate-200 bg-white"
                    }`}
                  >
                    <Text
                      className={`text-sm ${
                        filters.brands.includes(b.key)
                          ? "text-brand-700"
                          : "text-slate-700"
                      }`}
                    >
                      {b.key} ({b.count})
                    </Text>
                  </TouchableOpacity>
                ))}
              </View>
            </>
          )}

          {/* Condition */}
          <Text className="mb-2 text-sm font-semibold text-slate-900">
            {t("filters.condition")}
          </Text>
          <View className="mb-4 flex-row flex-wrap gap-2">
            {["new", "like_new", "good", "fair", "poor"].map((c) => (
              <TouchableOpacity
                key={c}
                onPress={() => toggleItem("conditions", c)}
                className={`rounded-full border px-3 py-1.5 ${
                  filters.conditions.includes(c)
                    ? "border-brand bg-brand-50"
                    : "border-slate-200 bg-white"
                }`}
              >
                <Text
                  className={`text-sm ${
                    filters.conditions.includes(c)
                      ? "text-brand-700"
                      : "text-slate-700"
                  }`}
                >
                  {t(`conditions.${c}`)}
                </Text>
              </TouchableOpacity>
            ))}
          </View>

          {/* Apply button */}
          <TouchableOpacity
            onPress={handleApply}
            className="mb-8 rounded-xl bg-brand py-3"
          >
            <Text className="text-center text-base font-semibold text-white">
              {t("filters.apply")}
            </Text>
          </TouchableOpacity>
        </BottomSheetScrollView>
      </BottomSheet>
    );
  },
);

FilterSheet.displayName = "FilterSheet";
export default FilterSheet;
