import { render, screen, within } from "@testing-library/react";
import axe from "axe-core";
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
    })),
    altText: vi.fn(),
  },
}));

describe("App", () => {
  beforeEach(async () => {
    localStorage.clear();
    await i18n.changeLanguage("pt-BR");
  });

  it("renders the localized catalog", async () => {
    render(<App />);
    expect(
      await screen.findByRole("heading", { name: /vitrine que fala/i }),
    ).toBeInTheDocument();
    expect(await screen.findByText("Poncho impermeável")).toBeInTheDocument();
    expect(
      screen.getByRole("img", { name: /poncho amarelo dobrado/i }),
    ).toBeInTheDocument();
  });

  it("has no axe violations on the loaded home page (pt-BR)", async () => {
    const { container } = render(<App />);
    await screen.findByText("Poncho impermeável");
    const results = await axe.run(container, { resultTypes: ["violations"] });
    // Serializing violations into the assertion message makes CI failures
    // self-explanatory (id, impact, failing nodes).
    expect(
      results.violations.map((v) => ({
        id: v.id,
        impact: v.impact,
        nodes: v.nodes.map((n) => n.target),
      })),
    ).toEqual([]);
  });

  it("opens the product dialog and returns focus on close", async () => {
    const user = userEvent.setup();
    const { container } = render(<App />);
    const opener = await screen.findByRole("button", {
      name: /ver detalhes de poncho/i,
    });
    await user.click(opener);

    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(
      within(dialog).getByRole("heading", { name: /poncho impermeável/i }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: /fechar/i }));
    await waitForDialogGone(container);
    expect(opener).toHaveFocus();
  });

  it("has no axe violations with the product dialog open", async () => {
    const user = userEvent.setup();
    const { container } = render(<App />);
    await user.click(
      await screen.findByRole("button", { name: /ver detalhes de poncho/i }),
    );
    const results = await axe.run(container, { resultTypes: ["violations"] });
    expect(results.violations).toEqual([]);
  });
});

async function waitForDialogGone(container: HTMLElement) {
  const { waitFor } = await import("@testing-library/react");
  await waitFor(() => {
    expect(container.querySelector('[role="dialog"]')).toBeNull();
  });
}
