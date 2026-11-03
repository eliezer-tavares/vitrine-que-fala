import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";

import { api, type Product } from "../api";

interface Props {
  onResults: (products: Product[]) => void;
}

/**
 * Multilingual semantic search (ADR-0009): one input works in all six
 * locales — queries are embedded in the same multilingual-e5 space as the
 * catalog. Announcements go through a polite live region so screen-reader
 * users hear the result count, never silence.
 */
export function SearchBox({ onResults }: Props) {
  const { t, i18n } = useTranslation();
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState("");
  const [searching, setSearching] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const q = query.trim();
    if (!q) return;
    setSearching(true);
    try {
      const res = await api.searchProducts(q, i18n.language);
      onResults(res.items);
      setStatus(
        res.items.length > 0
          ? t("search.status", { total: res.items.length, query: q })
          : t("search.noResults", { query: q }),
      );
    } catch {
      setStatus(t("search.error"));
    } finally {
      setSearching(false);
    }
  }

  return (
    <section aria-labelledby="search-title" className="search">
      <h2 id="search-title" className="visually-hidden">
        {t("search.label")}
      </h2>
      <form
        role="search"
        onSubmit={(event) => {
          void handleSubmit(event);
        }}
      >
        <label htmlFor="search-input">{t("search.label")}</label>
        <input
          id="search-input"
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          aria-describedby="search-status"
        />
        <button type="submit" disabled={searching}>
          {t("search.submit")}
        </button>
      </form>
      <p id="search-status" role="status" aria-live="polite" className="status">
        {status}
      </p>
    </section>
  );
}
