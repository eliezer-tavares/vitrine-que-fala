import AxeBuilder from "@axe-core/playwright";
import { expect, test } from "@playwright/test";

import { mockCatalog } from "./helpers";

test.describe.configure({ mode: "parallel" });

for (const [locale, heading] of [
  ["pt-BR", /vitrine que fala/i],
  ["en-US", /speaking storefront/i],
  ["id-ID", /etalase yang berbicara/i],
  ["th-TH", /หน้าร้านที่พูดได้/],
  ["vi-VN", /gian hàng biết nói/i],
  ["zh-TW", /會說話的店面/],
] as const) {
  test(`axe scan passes on the home page (${locale})`, async ({ page }) => {
    await mockCatalog(page);
    await page.addInitScript((lng) => {
      localStorage.setItem("vqf.locale", lng);
    }, locale);
    await page.goto("/");
    await expect(page.getByRole("heading", { level: 1, name: heading })).toBeVisible();
    await expect(page.getByRole("listitem").first()).toBeVisible();

    const results = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();

    expect(results.violations).toEqual([]);
  });

  test(`axe scan passes in blind mode (${locale})`, async ({ page }) => {
    await mockCatalog(page);
    await page.addInitScript((lng) => {
      localStorage.setItem("vqf.locale", lng);
    }, locale);
    await page.goto("/");
    await page.getByRole("button", { name: /sem olhos|no-eyes|tanpa mata|ไร้สายตา|không cần mắt|無視覺體驗/i }).click();
    await expect(
      page.locator("html[data-blind-mode='on']"),
    ).toHaveCount(1);

    const results = await new AxeBuilder({ page })
      .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
      .analyze();

    expect(results.violations).toEqual([]);
  });
}
