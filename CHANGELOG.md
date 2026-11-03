# Changelog

All notable changes to this project are documented here.
Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [v2.0] — 2026

The complete release: six locales as data, gRPC mesh, Pub/Sub behind the
broker interface, multilingual semantic search, load-test harness and
observability — still 100% open source, still sized for an 8 GB notebook.

### Added
- **Six locales** (pt-BR, en-US, id-ID, th-TH, vi-VN, zh-TW) everywhere — catalog translations, receipts, storefront UI, Web Speech, OCR — added as **data, not code** (seed 002 + i18n JSONs; ADR-0003).
- **Per-market currencies**: IDR (Rp), THB (฿), VND (₫), TWD (NT$) with zero-decimal and suffix conventions; storefront formats via `Intl` (ADR-0012).
- **gRPC internal mesh** (ADR-0007): `backend/api/proto/vitrine.proto` with committed generated stubs; catalog/order serve dual HTTP+gRPC; the gateway reads via gRPC; the worker writes alt-text/embeddings via gRPC and no longer touches PostgreSQL.
- **Google Pub/Sub behind the broker interface** (ADR-0008): `BROKER=redis|pubsub`, shared topic names, emulator support; Redis Streams remains the local default.
- **Multilingual semantic search** (ADR-0009): pgvector + HNSW on `product_translations`, `POST /embed` (multilingual-e5-small, lazy; deterministic stub for CI), worker indexes embeddings best-effort, `GET /api/v1/search` reports `strategy: semantic|lexical`.
- **Load-test harness** (ADR-0011): k6 scripts for browse/search/checkout with thresholds and an honest, empty results table (`docs/LOAD-TESTING.md`).
- **Observability** (ADR-0010): slog JSON + `vitrine_http_*` Prometheus metrics on every service, optional compose profile with Prometheus + Grafana, `docs/OBSERVABILITY.md`.
- **Search in the storefront**: one search box works in all six locales, announces result counts via `aria-live`.
- ADR-0007..0012; `docs/LOAD-TESTING.md`; `docs/OBSERVABILITY.md`.

### Changed
- ai-vision-service: composed pipeline translates en→{pt,id,th,vi,zh} via per-pair lazy Opus-MT; zh-TW gets optional OpenCC s2t conversion; Tesseract reads all six scripts; `/metrics` endpoint; version 2.0.0.
- Playwright E2E and vitest axe scans now cover all six locales (including blind mode); keyboard E2E exercises a Thai switch.
- Seed ships per-market products (batik tote in IDR, Thai silk scarf in THB, nón lá in VND, bubble-tea cup in TWD) translated in all six locales.
- docker-compose uses the pgvector Postgres image and wires the gRPC ports, `BROKER` and `EMBEDDER` env vars.

### Accessibility
- axe-core runs in CI against the page in **all six locales**, home and blind mode; violations fail the build.
- Search results and failures announce through polite `aria-live` regions — silence is treated as a bug.
- Receipts and order errors localized in six languages (lookup tables with en-US fallback).

### Compatibility
- The REST edge stays compatible with v1.0 clients; gRPC is internal-only.
- `X-Client-Locale` header behavior unchanged; `Accept-Language` now resolves six locales (th-TH no longer falls back to en-US).

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
- Cloud Run first deployment with published k6 numbers (fill `docs/LOAD-TESTING.md`).
- OpenTelemetry tracing across the gRPC mesh; alerting rules for the metric inventory.
