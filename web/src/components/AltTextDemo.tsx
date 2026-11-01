import { useRef, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";

import { api, type AltTextResponse } from "../api";
import { useSpeech } from "../hooks/useSpeech";

/**
 * Live demo of the open-source AI pipeline: upload any image, get per-locale
 * alt-text. Progress is announced through a polite live region — CPU-only
 * inference takes seconds, and screen-reader users must not be left guessing
 * (ADR-0005). The Listen buttons exercise the voice fallback chain from
 * ADR-0006.
 */
export function AltTextDemo() {
  const { t, i18n } = useTranslation();
  const speak = useSpeech();
  const inputRef = useRef<HTMLInputElement>(null);
  const [status, setStatus] = useState<string>("");
  const [result, setResult] = useState<AltTextResponse | null>(null);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const file = inputRef.current?.files?.[0];
    if (!file) {
      setStatus(t("demo.noFile"));
      return;
    }
    setStatus(t("demo.generating"));
    try {
      const data = await api.altText(file, i18n.language);
      setResult(data);
      setStatus(t("demo.ready"));
    } catch {
      setStatus(t("demo.error"));
    }
  }

  return (
    <section aria-labelledby="demo-title" className="demo">
      <h2 id="demo-title">{t("demo.title")}</h2>
      <form
        onSubmit={(event) => {
          void handleSubmit(event);
        }}
      >
        <label htmlFor="demo-image">{t("demo.pickImage")}</label>
        <input
          id="demo-image"
          ref={inputRef}
          type="file"
          accept="image/*"
        />
        <button type="submit">{t("demo.submit")}</button>
      </form>
      <p role="status" aria-live="polite" className="status">
        {status}
      </p>

      {result && (
        <div className="demo-result">
          {Object.entries(result.alt_texts).map(([locale, text]) => (
            <div key={locale} className="alt-text-block">
              <h3>{t("demo.altFor", { locale })}</h3>
              <p>{text}</p>
              <button
                type="button"
                onClick={() => {
                  const outcome = speak(text, locale);
                  if (!outcome.spoken) {
                    window.setTimeout(() => {
                      const region = document.querySelector<HTMLElement>(".demo .status");
                      if (region) {
                        region.textContent = t("demo.speechFallback", { locale });
                      }
                    }, 0);
                  }
                }}
              >
                {t("demo.listen", { locale })}
              </button>
            </div>
          ))}
          {result.ocr_text.length > 0 && (
            <p className="ocr-note">
              {t("demo.ocr")}: {result.ocr_text.join(" · ")}
            </p>
          )}
        </div>
      )}
    </section>
  );
}
