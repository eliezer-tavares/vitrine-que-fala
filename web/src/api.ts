export interface Price {
  minor_units: number;
  currency: string;
  formatted: string;
}

export interface Product {
  id: string;
  name: string;
  description: string;
  alt_text: string;
  price: Price;
}

export interface ProductListResponse {
  locale: string;
  items: Product[];
}

export interface AltTextResponse {
  alt_texts: Record<string, string>;
  ocr_text: string[];
}

export interface SearchResponse {
  locale: string;
  query: string;
  /** "semantic" (pgvector) or "lexical" (ILIKE fallback) — ADR-0009. */
  strategy: string;
  items: Product[];
}

const BASE: string = import.meta.env.VITE_API_URL ?? "/api/v1";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, init);
  if (!res.ok) {
    throw new Error(`API error ${res.status}`);
  }
  return (await res.json()) as T;
}

export const api = {
  listProducts(locale: string): Promise<ProductListResponse> {
    return request<ProductListResponse>("/products", {
      headers: { "Accept-Language": locale },
    });
  },

  searchProducts(query: string, locale: string): Promise<SearchResponse> {
    return request<SearchResponse>(
      `/search?q=${encodeURIComponent(query)}`,
      { headers: { "Accept-Language": locale } },
    );
  },

  async altText(file: File, locale: string): Promise<AltTextResponse> {
    const form = new FormData();
    form.append("image", file);
    const res = await fetch(`${BASE}/ai/alt-text`, {
      method: "POST",
      body: form,
      headers: { "Accept-Language": locale },
    });
    if (!res.ok) {
      throw new Error(`API error ${res.status}`);
    }
    return (await res.json()) as AltTextResponse;
  },
};
