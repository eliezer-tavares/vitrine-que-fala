import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import App from "../App";
import i18n, { STORAGE_KEY } from "../i18n";

vi.mock("../api", () => ({
  api: {
    listProducts: vi.fn(async () => ({ locale: "pt-BR", items: [] })),
    altText: vi.fn(),
  },
}));

describe("LanguageSwitcher (WCAG 3.1.1/3.1.2)", () => {
  beforeEach(async () => {
    localStorage.clear();
    await i18n.changeLanguage("pt-BR");
  });

  it("updates the html lang attribute and persists the choice", async () => {
    const user = userEvent.setup();
    render(<App />);

    const select = screen.getByRole("combobox", { name: /idioma/i });
    expect(document.documentElement.lang).toBe("pt-BR");

    await user.selectOptions(select, "en-US");

    expect(document.documentElement.lang).toBe("en-US");
    expect(localStorage.getItem(STORAGE_KEY)).toBe("en-US");
    expect(
      await screen.findByRole("heading", { name: /speaking storefront/i }),
    ).toBeInTheDocument();
  });

  it("announces the change through the polite live region", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.selectOptions(
      screen.getByRole("combobox", { name: /idioma/i }),
      "en-US",
    );

    expect(
      await screen.findByText(/language switched to english/i),
    ).toBeInTheDocument();
  });
});
