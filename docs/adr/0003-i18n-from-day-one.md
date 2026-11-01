# ADR-0003 — i18n from day one (no retrofit)

Status: Accepted (v1.0)
Date: 2026
Deciders: Eliezer Tavares de Oliveira

## Context

v1.0 is bilingual (pt-BR/en-US) and v2.0 targets six locales. Retrofitting internationalization into a working app is famously painful: strings extracted late, hardcoded layouts, missing plural rules, and the classic bug where `<html lang>` stops matching the visible language — which makes screen readers pronounce one language with another language's voice.

## Decision

Internationalization is a day-one constraint, not a feature:

1. **Backend:** locale is negotiated at the edge (`Accept-Language` → supported locales → fallback chain) and travels downstream as `X-Client-Locale`; cache keys embed the locale; the DB schema (ADR-0001) is locale-aware.
2. **Frontend:** react-i18next with all strings in locale files from the first commit; `document.documentElement.lang` is updated on `languageChanged`; currency/date formatting goes through `Intl`.
3. **Tests:** locale negotiation and `lang`-attribute sync are unit-tested; axe scans run **per locale**.

## Consequences

- v2.0's four extra locales are translation files + DB rows, zero refactor.
- The `<html lang>` bug class is designed out, not patched later — a WCAG 3.1.1/3.1.2 guarantee, and something a screen-reader user notices within seconds of browsing.
- Slightly slower initial development (every string routed through i18n), repaid when locale #3 costs one JSON file.

## Links

- ADR-0001 (schema) and ADR-0006 (speech voice fallback) implement pieces of this.
