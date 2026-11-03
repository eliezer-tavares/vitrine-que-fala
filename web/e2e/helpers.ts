import type { Page } from "@playwright/test";

export interface MockProduct {
  id: string;
  name: string;
  description: string;
  alt_text: string;
  price: { minor_units: number; currency: string; formatted: string };
}

function product(
  name: string,
  description: string,
  alt_text: string,
  price: MockProduct["price"],
): MockProduct {
  return { id: "11111111-1111-1111-1111-111111111111", name, description, alt_text, price };
}

export const MOCK_CATALOG: Record<string, MockProduct[]> = {
  "pt-BR": [
    product("Poncho impermeável", "Poncho de poliéster com capuz",
      "Poncho amarelo dobrado sobre uma mesa clara",
      { minor_units: 7900, currency: "BRL", formatted: "R$ 79,00" }),
    { id: "22222222-2222-2222-2222-222222222222", name: "Guarda-chuva automático",
      description: "Abre com um toque", alt_text: "Guarda-chuva preto fechado sobre fundo branco",
      price: { minor_units: 4500, currency: "BRL", formatted: "R$ 45,00" } },
  ],
  "en-US": [
    product("Rain poncho", "Polyester poncho with hood",
      "A folded yellow poncho on a light table",
      { minor_units: 7900, currency: "BRL", formatted: "R$ 79.00" }),
    { id: "22222222-2222-2222-2222-222222222222", name: "Automatic umbrella",
      description: "Opens with one touch", alt_text: "A closed black umbrella on a white background",
      price: { minor_units: 4500, currency: "BRL", formatted: "R$ 45.00" } },
  ],
  "id-ID": [
    product("Ponco hujan kedap air", "Ponco poliester dengan hud",
      "Ponco kuning terlipat di atas meja terang",
      { minor_units: 7900, currency: "BRL", formatted: "R$ 79,00" }),
  ],
  "th-TH": [
    product("เสื้อกันฝนกันน้ำ", "เสื้อกันฝนโพลีเอสเตอร์",
      "เสื้อกันฝนสีเหลืองพับวางบนโต๊ะสีอ่อน",
      { minor_units: 7900, currency: "BRL", formatted: "R$ 79,00" }),
  ],
  "vi-VN": [
    product("Áo mưa chống thấm", "Áo mưa polyester có mũ",
      "Áo mưa vàng gấp gọn trên bàn sáng màu",
      { minor_units: 7900, currency: "BRL", formatted: "R$ 79,00" }),
  ],
  "zh-TW": [
    product("防水雨衣", "聚酯纖維雨衣",
      "摺疊好的黃色雨衣放在淺色桌上",
      { minor_units: 7900, currency: "BRL", formatted: "R$ 79,00" }),
  ],
};

/** Intercepts the backend API so E2E runs without any service running. */
export async function mockCatalog(page: Page): Promise<void> {
  await page.route("**/api/v1/products", async (route) => {
    const request = route.request();
    const header = (request.headers()["accept-language"] ?? "pt-BR").toLowerCase();
    const loc =
      header.includes("en") ? "en-US"
      : header.includes("id") ? "id-ID"
      : header.includes("th") ? "th-TH"
      : header.includes("vi") ? "vi-VN"
      : header.includes("zh") ? "zh-TW"
      : "pt-BR";
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ locale: loc, items: MOCK_CATALOG[loc] }),
    });
  });

  await page.route("**/api/v1/search*", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        locale: "pt-BR",
        query: "poncho",
        strategy: "semantic",
        items: MOCK_CATALOG["pt-BR"].slice(0, 1),
      }),
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
          "id-ID": "Deskripsi yang dihasilkan AI dalam bahasa Indonesia (E2E mock).",
          "th-TH": "คำอธิบายที่สร้างด้วย AI ภาษาไทย (E2E mock).",
          "vi-VN": "Mô tả do AI tạo bằng tiếng Việt (E2E mock).",
          "zh-TW": "以繁體中文生成的 AI 描述（E2E mock）。",
        },
        ocr_text: ["BRAND X", "50% OFF"],
      }),
    });
  });
}
