import { useTranslation } from "react-i18next";

import { SUPPORTED } from "../i18n";
import { useAnnouncer } from "./LiveRegion";

/** Language selector with native names (W3C practice) + WCAG 3.1.1 sync,
 * which happens inside i18n.ts via the languageChanged event. */
export function LanguageSwitcher() {
  const { t, i18n } = useTranslation();
  const announce = useAnnouncer();

  return (
    <select
      aria-label={t("language.label")}
      value={i18n.language}
      onChange={(event) => {
        const code = event.target.value;
        void i18n.changeLanguage(code);
        const native =
          SUPPORTED.find((l) => l.code === code)?.nativeName ?? code;
        announce(t("language.changed", { name: native }));
      }}
    >
      {SUPPORTED.map((locale) => (
        <option key={locale.code} value={locale.code}>
          {locale.nativeName}
        </option>
      ))}
    </select>
  );
}
