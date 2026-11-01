import { useTranslation } from "react-i18next";

/** First Tab stop of the page: jumps focus straight to <main id="main">. */
export function SkipLink() {
  const { t } = useTranslation();
  return (
    <a className="skip-link" href="#main">
      {t("skipLink")}
    </a>
  );
}
