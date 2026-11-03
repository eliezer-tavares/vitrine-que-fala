# Accessibility statement — Vitrine que Fala

Target: **WCAG 2.2 AA** for the storefront, **in every supported locale**. This document says what is tested automatically, what is tested manually, and what is still limited — in plain language.

Architected by Eliezer Tavares de Oliveira (blind screen-reader user) — contact@eliezertavaresdeoliveira.com

## Tested automatically on every push (CI)

1. **axe-core scans** (`@axe-core/playwright`) against the home page in **all six locales** (pt-BR, en-US, id-ID, th-TH, vi-VN, zh-TW) — in blind mode too — with WCAG 2.0/2.1 A+AA rules. Violations fail the build. Non-Latin scripts (Thai, Vietnamese diacritics, Traditional Chinese) are exactly where lang/font/contrast bugs hide, so the v2.0 CI matrix covers them from the first commit.
2. **Component-level axe-core assertions** (vitest + React Testing Library) on the header, language switcher, search box, product dialog and demo panel — plus dedicated axe scans with the page rendered in th-TH and zh-TW.
3. **Keyboard-only E2E**: the Playwright suite drives the app without a mouse — skip link, language switcher (including a switch to Thai), search, product cards, dialog open/close, blind-mode toggle — asserting focus order and visible focus.
4. **ARIA state assertions**: `aria-pressed` on the blind-mode toggle, `aria-live="polite"` progress announcements (alt-text generation AND search result counts), `aria-modal` dialog labeling, `<html lang>` switching with the selected locale.
5. **HTML lang sync tests**: switching to any of the six locales updates `document.documentElement.lang` (WCAG 3.1.1/3.1.2) so screen readers switch pronunciation voice.

## Tested manually before every release (NVDA checklist)

Run on Windows + NVDA + Chrome, and spot-check with Orca + Firefox:

1. Page loads; NVDA announces the h1 and the skip link is the first Tab stop.
2. Language switcher: change to Thai, then Vietnamese — NVDA switches pronunciation voice (lang attribute follows); the selector lists each language in its own name.
3. Navigate products by arrow keys/browse mode: every image announces its alt-text in the selected language.
4. Search: type a query in the current language; the polite live region announces the result count (or the no-results message) — silence is a bug.
5. Open a product dialog: focus lands on the dialog title, Tab is trapped inside, Esc closes and focus returns to the opener.
6. Activate "Experiência Sem Olhos": NVDA confirms via announcement; images are hidden; keyboard flow still completes a purchase.
7. Alt-text demo: upload an image; the `aria-live` region announces "generating description…" then the result; the "listen" button reads it aloud in the selected language (six languages in v2.0).
8. Checkout: error and success messages are announced, not only shown — receipts are localized in all six languages.

## Known limitations (honest list, v2.0)

- The Web Speech API voice availability depends on the OS/browser. The v2.0 voice matrix (six locales) is broader but still OS-dependent: if no voice exists for the selected locale, the app falls back through the ADR-0006 chain and always shows the text on screen.
- The stub AI pipeline produces generic descriptions by design; the Florence-2 + Opus-MT pipeline quality varies by image type and by language pair (en→th and en→vi are the weakest pairs — documented honestly in ADR-0005's quality notes).
- Automated axe scans cannot catch everything (e.g., awkward screen-reader phrasing in Thai). That is what the manual checklist above is for — executed by a screen-reader user.
- The order flow is a simulation: no real payment provider is integrated.
- Deep pages beyond the product dialog are not implemented yet; the storefront is intentionally a focused demo surface.

Found a barrier? Please report to contact@eliezertavaresdeoliveira.com — barrier reports are treated as bugs, not feedback.
