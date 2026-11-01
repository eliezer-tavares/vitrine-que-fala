import type { Page } from "@playwright/test";

export interface MockProduct {
  id: string;
  name: string;
  description: string;
  alt_text: string;
  price: { minor_units: number; currency: string; formatted: string };
}

export const MOCK_CATALOG: Record<string, MockProduct[]> = {
  "pt-BR": [
    {
      id: "11111111-1111-1111-1111-111111111111",
      name: "Poncho impermeável",
      description: "Poncho de poliéster com capuz",
      alt_text: "Poncho amarelo dobrado sobre uma mesa clara",
      price: { minor_units: 7900, currency: "BRL", formatted: "R$ 79,00" },
    },
    {
      id: "22222222-2222-2222-2222-222222222222",
      name: "Guarda-chuva automático",
      description: "Abre com um toque",
      alt_text: "Guarda-chuva preto fechado sobre fundo branco",
      price: { minor_units: 4500, currency: "BRL", formatted: "R$ 45,00" },
    },
  ],
  "en-US": [
    {
      id: "11111111-1111-1111-1111-111111111111",
      name: "Rain poncho",
      description: "Polyester poncho with hood",
      alt_text: "A folded yellow poncho on a light table",
      price: { minor_units: 7900, currency: "BRL", formatted: "R$ 79.00" },
    },
    {
      id: "22222222-2222-2222-2222-222222222222",
      name: "Automatic umbrella",
      description: "Opens with one touch",
      alt_text: "A closed black umbrella on a white background",
      price: { minor_units: 4500, currency: "BRL", formatted: "R$ 45.00" },
    },
  ],
};

/** Intercepts the backend API so E2E runs without any service running. */
export async function mockCatalog(page: Page): Promise<void> {
  await page.route("**/api/v1/products", async (route) => {
    const request = route.request();
    const header = request.headers()["accept-language"] ?? "pt-BR";
    const locale = header.toLowerCase().includes("en") ? "en-US" : "pt-BR";
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ locale, items: MOCK_CATALOG[locale] }),
    });
  });

  await page.route("**/api/v1/ai/alt-text", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        alt_texts: {
          "pt-BR": "Descrição gerada pela IA em português (E2E mock).",
          "en-US": "AI-generated description in English (E2E mock).",
        },
        ocr_text: ["BRAND X", "50% OFF"],
      }),
    });
  });
}
