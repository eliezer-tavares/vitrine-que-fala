import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import App from "../App";
import i18n from "../i18n";

vi.mock("../api", () => ({
  api: {
    listProducts: vi.fn(async () => ({
      locale: "pt-BR",
      items: [
        {
          id: "1",
          name: "Poncho impermeável",
          description: "Poncho com capuz",
          alt_text: "Poncho amarelo dobrado sobre uma mesa clara",
          price: { minor_units: 7900, currency: "BRL", formatted: "R$ 79,00" },
        },
      ],
    })),
    altText: vi.fn(),
  },
}));

describe("BlindMode (Experiência Sem Olhos)", () => {
  beforeEach(async () => {
    localStorage.clear();
    await i18n.changeLanguage("pt-BR");
  });

  it("toggles aria-pressed and the html dataset flag", async () => {
    const user = userEvent.setup();
    render(<App />);

    const button = await screen.findByRole("button", {
      name: /experiência sem olhos/i,
    });
    expect(button).toHaveAttribute("aria-pressed", "false");
    expect(document.documentElement.dataset.blindMode).toBe("off");

    await user.click(button);

    expect(button).toHaveAttribute("aria-pressed", "true");
    expect(document.documentElement.dataset.blindMode).toBe("on");
    expect(localStorage.getItem("vqf.blind-mode")).toBe("on");
  });

  it("announces activation through the polite live region", async () => {
    const user = userEvent.setup();
    render(<App />);

    await user.click(
      await screen.findByRole("button", { name: /experiência sem olhos/i }),
    );

    expect(
      await screen.findByText(/experiência sem olhos ativada/i),
    ).toBeInTheDocument();
  });

  it("keeps the image in the DOM (CSS hides it in blind mode)", async () => {
    render(<App />);
    const image = await screen.findByRole("img", {
      name: /poncho amarelo dobrado/i,
    });
    expect(image).toHaveClass("product-image");
    // The hiding itself is CSS: html[data-blind-mode="on"] .product-image.
    expect(document.documentElement.dataset.blindMode).toBe("off");
  });
});
