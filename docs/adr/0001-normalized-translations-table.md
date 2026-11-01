# ADR-0001 — Normalized translations table (not per-locale columns)

Status: Accepted (v1.0)
Date: 2026
Deciders: Eliezer Tavares de Oliveira

## Context

Products carry localized fields: `name`, `description` and AI-generated `alt_text`. v1.0 ships two locales (pt-BR, en-US) and the v2.0 roadmap adds four more (id-ID, th-TH, vi-VN, zh-TW) — matching the markets of the platform this project mirrors (Shopee: Indonesia, Thailand, Vietnam, Taiwan, Brazil, Singapore/Philippines/Malaysia).

## Decision

Store translations in a dedicated table keyed by locale:

```sql
products(id, price_minor, currency, image_url, created_at)
product_translations(product_id, locale, name, description, alt_text,
                     PRIMARY KEY (product_id, locale))
```

Catalog queries join on the negotiated locale. Redis cache keys embed the locale (`catalog:v1:{locale}:products`).

## Consequences

- Adding a locale in v2.0 is **data**, not a schema migration — the "plug four locales" step touches no Go code.
- One indexed lookup pattern (`WHERE product_id = ? AND locale = ?`) serves every field; no sparse columns growing with each locale.
- A missing translation is explicit (no row) instead of an empty column — the API can then fall back to the default locale at the service layer.
- Rejected alternative: `name_pt`/`name_en`/… columns. Cheaper for exactly two locales, but it turns every new locale into a migration plus code churn, and bakes locale knowledge into the schema.

## Links

- ADR-0003 (i18n from day one) depends on this shape.
