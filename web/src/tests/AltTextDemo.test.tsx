import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import i18n from "../i18n";
import { AltTextDemo } from "../components/AltTextDemo";
import { AnnouncerProvider } from "../components/LiveRegion";

const altTextMock = vi.fn(async (_file: File, _locale: string) => ({
  alt_texts: {
    "pt-BR": "Alt-text em português gerado pela IA.",
    "en-US": "AI-generated English alt-text.",
  },
  ocr_text: ["BRAND X", "50% OFF"],
}));

vi.mock("../api", () => ({
  api: {
    listProducts: vi.fn(),
    altText: (file: File, locale: string) => altTextMock(file, locale),
  },
}));

function renderDemo() {
  return render(
    <AnnouncerProvider>
      <AltTextDemo />
    </AnnouncerProvider>,
  );
}

describe("AltTextDemo", () => {
  beforeEach(async () => {
    altTextMock.mockClear();
    await i18n.changeLanguage("pt-BR");
  });

  it("shows a helpful status when no file is picked", async () => {
    const user = userEvent.setup();
    renderDemo();

    await user.click(screen.getByRole("button", { name: /gerar descrição/i }));

    expect(
      screen.getByText(/escolha um arquivo de imagem/i),
    ).toBeInTheDocument();
    expect(altTextMock).not.toHaveBeenCalled();
  });

  it("announces progress and renders per-locale results", async () => {
    const user = userEvent.setup();
    renderDemo();

    const input = screen.getByLabelText(/escolha uma imagem de produto/i);
    await user.upload(input, new File(["image-bytes"], "product.png", { type: "image/png" }));
    await user.click(screen.getByRole("button", { name: /gerar descrição/i }));

    expect(await screen.findByText(/alt-text em português gerado pela ia/i)).toBeInTheDocument();
    expect(screen.getByText(/ai-generated english alt-text/i)).toBeInTheDocument();
    expect(screen.getByText(/descrição pronta/i)).toBeInTheDocument();
    expect(screen.getByText(/brand x/i)).toBeInTheDocument();
    expect(screen.getByText(/50% off/i)).toBeInTheDocument();
    expect(altTextMock).toHaveBeenCalledTimes(1);
  });

  it("falls back to visible text when speech is unavailable (ADR-0006)", async () => {
    // jsdom has no speechSynthesis -> the Listen button must surface the
    // fallback message instead of pretending to speak.
    const user = userEvent.setup();
    renderDemo();

    const input = screen.getByLabelText(/escolha uma imagem de produto/i);
    await user.upload(input, new File(["image-bytes"], "product.png", { type: "image/png" }));
    await user.click(screen.getByRole("button", { name: /gerar descrição/i }));
    await screen.findByText(/alt-text em português gerado pela ia/i);

    const listenButtons = screen.getAllByRole("button", { name: /ouvir em/i });
    expect(listenButtons).toHaveLength(2);
    await user.click(listenButtons[0]);

    const statuses = screen.getAllByRole("status");
    const texts = statuses.map((el) => el.textContent ?? "").join(" ");
    expect(texts).toMatch(/voz indisponível/i);
    expect(texts).toMatch(/o texto permanece visível/i);
  });
});
