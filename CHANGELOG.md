# Changelog

All notable changes to this project are documented here.
Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [v1.0] — 2026

First public release. Bilingual (pt-BR / en-US) accessibility-first MVP.

### Added
- `api-gateway` (Go): JWT issuance/validation, `Accept-Language` negotiation with fallback chain, reverse proxy to catalog, order and AI services, locale propagation via `X-Client-Locale`.
- `catalog-service` (Go): PostgreSQL with normalized `product_translations` table, Redis cache keyed by locale, localized product listing/detail.
- `order-service` (Go): checkout publishing `OrderPlaced` events to Redis Streams, localized receipts.
- `alttext-worker` (Go): consumer-group worker pool over `images.pending` stream, calls the AI service, writes per-locale alt-text back to the catalog, retry with attempts counter and dead-letter stream.
- `ai-vision-service` (Python/FastAPI): image captioning + OCR, 100% open-source models (Florence-2-base, Opus-MT, Tesseract), deterministic stub pipeline for CI/demo, lazy model loading for 8 GB RAM machines.
- React storefront: react-i18next (pt-BR/en-US) with native language names, `<html lang>` synchronization (WCAG 3.1.1/3.1.2), `Intl` currency/date formatting, "No-Eyes Experience" mode, Web Speech API playback with graceful voice fallback, `aria-live` progress announcements, skip link, focus management, WCAG 2.2 AA target.
- Automated tests: Go unit tests, Python contract tests, vitest + axe-core component tests, Playwright E2E with keyboard-only flows and axe scans in both locales.
- CI/CD: GitHub Actions pipeline (lint, tests, a11y scans, Docker builds) and Cloud Run deploy workflow.
- Documentation: bilingual README, ADR-0001..0006, RAM budget for 8 GB machines, OpenAPI spec, accessibility statement, publication guide.

### Accessibility
- axe-core runs in CI against the page in both locales; violations fail the build.
- Keyboard-only E2E covers skip link, language switcher, product dialog and blind mode.
- Blind-mode toggle uses `aria-pressed`; async alt-text generation announces progress via polite `aria-live`.

## [Unreleased]
- v2.0: six locales as data (id-ID, th-TH, vi-VN, zh-TW), gRPC between services, Google Pub/Sub behind the broker interface, multilingual semantic search with multilingual-e5 + pgvector, k6 load-test numbers, per-market currencies.
