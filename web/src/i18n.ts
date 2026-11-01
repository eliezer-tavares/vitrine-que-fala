import i18n from "i18next";
import { initReactI18next } from "react-i18next";

import enUS from "./i18n/locales/en-US.json";
import ptBR from "./i18n/locales/pt-BR.json";

export interface SupportedLocale {
  code: string;
  nativeName: string;
}

export const SUPPORTED: readonly SupportedLocale[] = [
  { code: "pt-BR", nativeName: "Português (Brasil)" },
  { code: "en-US", nativeName: "English" },
];

const STORAGE_KEY = "vqf.locale";
const LOCALE_CODES = SUPPORTED.map((l) => l.code);

function initialLocale(): string {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved && LOCALE_CODES.includes(saved)) return saved;
    const nav = (navigator.language || "").toLowerCase();
    if (nav.startsWith("pt")) return "pt-BR";
    if (nav.startsWith("en")) return "en-US";
  } catch {
    // non-browser environment (tests): fall through to default
  }
  return "pt-BR";
}

void i18n.use(initReactI18next).init({
  resources: {
    "pt-BR": { translation: ptBR },
    "en-US": { translation: enUS },
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
