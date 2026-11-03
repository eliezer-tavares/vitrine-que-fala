import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import enUS from "./i18n/locales/en-US.json";
import idID from "./i18n/locales/id-ID.json";
import ptBR from "./i18n/locales/pt-BR.json";
import thTH from "./i18n/locales/th-TH.json";
import viVN from "./i18n/locales/vi-VN.json";
import zhTW from "./i18n/locales/zh-TW.json";

export interface SupportedLocale {
  code: string;
  nativeName: string;
}

/** v2.0: the Shopee-market footprint (ADR-0003). Native names in the
 * selector, W3C practice — users must recognize their own language. */
export const SUPPORTED: readonly SupportedLocale[] = [
  { code: "pt-BR", nativeName: "Português (Brasil)" },
  { code: "en-US", nativeName: "English" },
  { code: "id-ID", nativeName: "Bahasa Indonesia" },
  { code: "th-TH", nativeName: "ไทย" },
  { code: "vi-VN", nativeName: "Tiếng Việt" },
  { code: "zh-TW", nativeName: "繁體中文" },
];

const STORAGE_KEY = "vqf.locale";
const LOCALE_CODES = SUPPORTED.map((l) => l.code);

// Browser language prefix -> supported locale (v2.0 adds the four markets).
const PREFIX_MAP: Record<string, string> = {
  pt: "pt-BR",
  en: "en-US",
  id: "id-ID",
  th: "th-TH",
  vi: "vi-VN",
  zh: "zh-TW",
};

function initialLocale(): string {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved && LOCALE_CODES.includes(saved)) return saved;
    const nav = (navigator.language || "").toLowerCase();
    const prefix = nav.split("-")[0];
    if (prefix && PREFIX_MAP[prefix]) return PREFIX_MAP[prefix];
  } catch {
    // non-browser environment (tests): fall through to default
  }
  return "pt-BR";
}

void i18n.use(initReactI18next).init({
  resources: {
    "pt-BR": { translation: ptBR },
    "en-US": { translation: enUS },
    "id-ID": { translation: idID },
    "th-TH": { translation: thTH },
    "vi-VN": { translation: viVN },
    "zh-TW": { translation: zhTW },
  },
  lng: initialLocale(),
  fallbackLng: "en-US",
  interpolation: { escapeValue: false },
});

// WCAG 3.1.1/3.1.2: the lang attribute must always match the visible
// language, otherwise screen readers pronounce content with the wrong voice.
i18n.on("languageChanged", (lng: string) => {
  document.documentElement.lang = lng;
  try {
    localStorage.setItem(STORAGE_KEY, lng);
  } catch {
    // storage unavailable: language still changes for this session
  }
});

export default i18n;
export { STORAGE_KEY };
