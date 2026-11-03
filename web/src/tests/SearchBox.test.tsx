import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SearchBox } from "../components/SearchBox";
import i18n from "../i18n";

const searchMock = vi.fn();

vi.mock("../api", () => ({
  api: {
    searchProducts: (...args: unknown[]) => searchMock(...args),
  },
}));

describe("SearchBox (ADR-0009)", () => {
  beforeEach(async () => {
    searchMock.mockReset();
    await i18n.changeLanguage("pt-BR");
  });

  it("labels the input and submits the query", async () => {
    searchMock.mockResolvedValue({
      locale: "pt-BR",
      query: "poncho",
      strategy: "semantic",
      items: [
        {
          id: "p-1",
          name: "Poncho impermeável",
          description: "d",
          alt_text: "a",
          price: { minor_units: 7900, currency: "BRL", formatted: "R$ 79,00" },
        },
      ],
    });
    const onResults = vi.fn();
    const user = userEvent.setup();
    render(<SearchBox onResults={onResults} />);

    await user.type(
      screen.getByRole("searchbox", { name: /buscar produtos/i }),
      "poncho",
    );
    await user.click(screen.getByRole("button", { name: "Buscar" }));

    expect(searchMock).toHaveBeenCalledWith("poncho", "pt-BR");
    expect(
      await screen.findByText(/1 resultado\(s\) para "poncho"/i),
    ).toBeInTheDocument();
    expect(onResults).toHaveBeenCalledWith([
      expect.objectContaining({ id: "p-1" }),
    ]);
  });

  it("announces the no-results message politely", async () => {
    searchMock.mockResolvedValue({
      locale: "pt-BR",
      query: "zzz",
      strategy: "lexical",
      items: [],
    });
    const user = userEvent.setup();
    render(<SearchBox onResults={vi.fn()} />);

    await user.type(screen.getByRole("searchbox"), "zzz");
    await user.click(screen.getByRole("button", { name: "Buscar" }));

    expect(
      await screen.findByText(/nenhum resultado para "zzz"/i),
    ).toBeInTheDocument();
  });

  it("reports failures instead of going silent", async () => {
    searchMock.mockRejectedValue(new Error("network down"));
    const user = userEvent.setup();
    render(<SearchBox onResults={vi.fn()} />);

    await user.type(screen.getByRole("searchbox"), "poncho");
    await user.click(screen.getByRole("button", { name: "Buscar" }));

    expect(await screen.findByText(/a busca falhou/i)).toBeInTheDocument();
  });

  it("ignores empty queries", async () => {
    const user = userEvent.setup();
    render(<SearchBox onResults={vi.fn()} />);

    await user.click(screen.getByRole("button", { name: "Buscar" }));
    expect(searchMock).not.toHaveBeenCalled();
  });
});
