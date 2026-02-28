import { useState } from "react";
import { View, Image, FlatList, Dimensions, Text } from "react-native";

const { width: SCREEN_WIDTH } = Dimensions.get("window");

interface ImageCarouselProps {
  images: string[];
}

export default function ImageCarousel({ images }: ImageCarouselProps) {
  const [activeIndex, setActiveIndex] = useState(0);

  if (images.length === 0) {
    return (
      <View
        style={{ aspectRatio: 4 / 3 }}
        className="items-center justify-center bg-slate-100"
      >
        <Text className="text-slate-300">No Image</Text>
      </View>
    );
  }

  return (
    <View>
      <FlatList
        data={images}
        horizontal
        pagingEnabled
        showsHorizontalScrollIndicator={false}
        onMomentumScrollEnd={(e) => {
          const index = Math.round(
            e.nativeEvent.contentOffset.x / SCREEN_WIDTH,
          );
          setActiveIndex(index);
        }}
        renderItem={({ item }) => (
          <Image
            source={{ uri: item }}
            style={{ width: SCREEN_WIDTH, aspectRatio: 4 / 3 }}
            resizeMode="contain"
            className="bg-slate-100"
          />
        )}
        keyExtractor={(_, i) => String(i)}
      />
      {/* Page dots */}
      {images.length > 1 && (
        <View className="mt-2 flex-row justify-center gap-1.5">
          {images.map((_, i) => (
            <View
              key={i}
              className={`h-1.5 w-1.5 rounded-full ${
                i === activeIndex ? "bg-brand" : "bg-slate-300"
              }`}
            />
          ))}
        </View>
      )}
    </View>
  );
}
