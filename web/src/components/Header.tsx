import { useTranslation } from "react-i18next";

import { useBlindMode } from "../context/BlindModeContext";
import { useAnnouncer } from "./LiveRegion";
import { LanguageSwitcher } from "./LanguageSwitcher";

export function Header() {
  const { t } = useTranslation();
  const { enabled, toggle } = useBlindMode();
  const announce = useAnnouncer();

  return (
    <header>
      <h1>{t("siteTitle")}</h1>
      <LanguageSwitcher />
      <button
        type="button"
        aria-pressed={enabled}
        onClick={() => {
          toggle();
          announce(
            t(enabled ? "blindMode.offAnnouncement" : "blindMode.onAnnouncement"),
          );
        }}
      >
        {t("blindMode.toggle")}
      </button>
      <p>{t("siteTagline")}</p>
    </header>
  );
}
