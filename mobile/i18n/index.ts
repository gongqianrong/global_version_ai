import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import * as Localization from "expo-localization";

import zhTW from "./messages/zh-TW.json";
import ja from "./messages/ja.json";
import en from "./messages/en.json";

const deviceLocale = Localization.getLocales()[0]?.languageTag || "ja";

function resolveLocale(tag: string): string {
  if (tag.startsWith("zh")) return "zh-TW";
  if (tag.startsWith("ja")) return "ja";
  return "en";
}

i18n.use(initReactI18next).init({
  resources: {
    "zh-TW": { translation: zhTW },
    ja: { translation: ja },
    en: { translation: en },
  },
  lng: resolveLocale(deviceLocale),
  fallbackLng: "ja",
  interpolation: {
    escapeValue: false,
  },
});

export default i18n;
