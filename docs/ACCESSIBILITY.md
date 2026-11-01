# Accessibility statement — Vitrine que Fala

Target: **WCAG 2.2 AA** for the storefront. This document says what is tested automatically, what is tested manually, and what is still limited — in plain language.

Architected by Eliezer Tavares de Oliveira (blind screen-reader user) — contact@eliezertavaresdeoliveira.com

## Tested automatically on every push (CI)

1. **axe-core scans** (`@axe-core/playwright`) against the home page in **pt-BR and en-US**, with WCAG 2.0/2.1 A+AA rules. Violations fail the build.
2. **Component-level axe-core assertions** (vitest + React Testing Library) on the header, language switcher, product dialog and demo panel.
3. **Keyboard-only E2E**: the Playwright suite drives the app without a mouse — skip link, language switcher, product cards, dialog open/close, blind-mode toggle — asserting focus order and visible focus.
4. **ARIA state assertions**: `aria-pressed` on the blind-mode toggle, `aria-live="polite"` progress announcements, `aria-modal` dialog labeling, `<html lang>` switching with the selected locale.
5. **HTML lang sync test**: switching language updates `document.documentElement.lang` (WCAG 3.1.1/3.1.2) so screen readers switch pronunciation voice.

## Tested manually before every release (NVDA checklist)

Run on Windows + NVDA + Chrome, and spot-check with Orca + Firefox:

1. Page loads; NVDA announces the h1 and the skip link is the first Tab stop.
2. Language switcher: change to English — NVDA switches pronunciation voice (lang attribute follows).
3. Navigate products by arrow keys/browse mode: every image announces its alt-text.
4. Open a product dialog: focus lands on the dialog title, Tab is trapped inside, Esc closes and focus returns to the opener.
5. Activate "Experiência Sem Olhos": NVDA confirms via announcement; images are hidden; keyboard flow still completes a purchase.
6. Alt-text demo: upload an image; the `aria-live` region announces "generating description…" then the result; the "listen" button reads it aloud in the selected language.
7. Checkout: error and success messages are announced, not only shown.

## Known limitations (honest list, v1.0)

- The Web Speech API voice availability depends on the OS/browser. If no voice exists for the selected locale, the app falls back to `en-US` voice and always shows the text on screen. The voice matrix is documented in the README as testing expands.
- The stub AI pipeline produces generic descriptions by design; the Florence-2 pipeline quality varies by image type.
- Automated axe scans cannot catch everything (e.g., awkward screen-reader phrasing). That is what the manual checklist above is for.
- The order flow is a simulation: no real payment provider is integrated in v1.0.
- Deep pages beyond the product dialog are not implemented yet; the storefront is intentionally a focused demo surface.

Found a barrier? Please report to contact@eliezertavaresdeoliveira.com — barrier reports are treated as bugs, not feedback.
