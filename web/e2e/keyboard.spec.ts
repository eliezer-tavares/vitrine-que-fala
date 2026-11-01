import { expect, test } from "@playwright/test";

import { mockCatalog } from "./helpers";

test("keyboard-only flow: skip link, language switch, blind mode, dialog", async ({
  page,
}) => {
  await mockCatalog(page);
  await page.goto("/");

  // First Tab lands on the skip link; Enter jumps to <main>.
  await page.keyboard.press("Tab");
  await expect(
    page.getByRole("link", { name: /pular para o conteúdo|skip to main content/i }),
  ).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("main")).toBeFocused();

  // Language switcher is reachable by keyboard and updates the page language.
  await page.getByRole("combobox", { name: /idioma|language/i }).focus();
  await page
    .getByRole("combobox", { name: /idioma|language/i })
    .selectOption("en-US");
  await expect(
    page.getByRole("heading", { name: /speaking storefront/i }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "en-US");

  // Blind-mode toggle is a keyboard-operable toggle button.
  const blindToggle = page.getByRole("button", {
    name: /no-eyes experience|experiência sem olhos/i,
  });
  await blindToggle.focus();
  await expect(blindToggle).toHaveAttribute("aria-pressed", "false");
  await page.keyboard.press("Space");
  await expect(blindToggle).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator("html[data-blind-mode='on']")).toHaveCount(1);
});

test("product dialog traps focus and returns it on Escape", async ({ page }) => {
  await mockCatalog(page);
  await page.addInitScript(() => {
    localStorage.setItem("vqf.locale", "pt-BR");
  });
  await page.goto("/");

  const opener = page.getByRole("button", { name: /ver detalhes de poncho/i });
  await opener.click();

  const dialog = page.getByRole("dialog", { name: /poncho impermeável/i });
  await expect(dialog).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "pt-BR");

  // Focus starts inside the dialog and Tab cycles without escaping it.
  const insideDialog = dialog.getByRole("button", { name: /fechar/i });
  await expect(insideDialog).toBeFocused();
  for (let i = 0; i < 6; i += 1) {
    await page.keyboard.press("Tab");
  }
  const active = page.evaluate(() => document.activeElement?.tagName ?? "");
  expect(await active).not.toBe("BODY");

  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(opener).toBeFocused();
});

test("alt-text demo announces progress and shows results", async ({ page }) => {
  await mockCatalog(page);
  await page.addInitScript(() => {
    localStorage.setItem("vqf.locale", "pt-BR");
  });
  await page.goto("/");

  await page
    .getByLabel(/escolha uma imagem de produto/i)
    .setInputFiles({
      name: "product.png",
      mimeType: "image/png",
      buffer: Buffer.from("png-bytes"),
    });
  await page.getByRole("button", { name: /gerar descrição/i }).click();

  await expect(
    page.getByText(/descrição gerada pela ia em português/i),
  ).toBeVisible();
  await expect(page.getByText(/descrição pronta/i)).toBeVisible();
  await expect(page.getByText(/50% off/i)).toBeVisible();
});
