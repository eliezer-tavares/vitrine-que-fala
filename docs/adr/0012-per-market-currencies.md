# ADR-0012: per-market currencies as data

- Status: accepted (v2.0)
- Date: 2026
- Deciders: Eliezer Tavares de Oliveira

## Context

v1.0 formatted BRL and USD only. Shopee's markets pay in Rp, ฿, ₫ and NT$ —
with real formatting differences: zero-decimal currencies (IDR, VND),
different thousands separators (Indonesia/Vietnam dot vs comma) and a suffix
symbol (₫ after the amount in Vietnam).

## Decision

- `products.currency` stays the single source of truth per product; the seed
  adds market products priced in IDR, THB, VND and TWD (`seed/002`).
- `internal/currency` grows with the market list: symbol table,
  zero-decimal set (IDR, VND), suffix set (VND) and per-locale
  thousands/decimal separators (pt/id/vi use `1.234,56`, others
  `1,234.56`).
- The **frontend formats with `Intl.NumberFormat`** (locale-aware, as since
  v1.0); the Go formatter exists for receipts and API convenience and must
  agree with Intl on the samples tested.
- Adding a market = currency rows + symbol entries + seed data. No schema
  migration — money is data, like locales (ADR-0003).

## Consequences

- Receipts read like the local market expects: `Rp 150.000`, `120.000 ₫`,
  `฿ 459.00`, `NT$ 32.00`.
- Known simplification: THB and TWD keep two decimals (satang/fen are real
  minor units); displaying them without decimals is a future product
  decision, not a data-model one.
