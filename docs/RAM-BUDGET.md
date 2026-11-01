# RAM budget — engineering under an 8 GB constraint

This project is architected to **develop, run and demo on a modest 8 GB RAM notebook** (Dell Latitude, 500 GB HDD). That constraint shaped real design decisions, documented in the ADRs. This file is the account.

## Local development budget (8 GB total)

| Slice | ~RAM | Notes |
| --- | --- | --- |
| OS + screen reader (NVDA/Orca) + editor | 2.5–3.5 GB | unavoidable base |
| WSL2 / Docker (capped by `.wslconfig`) | up to 4 GB | `memory=4GB`, `processors=4` |
| PostgreSQL 16 | ~0.3 GB | catalog data |
| Redis (cache + Redis Streams queue) | ~0.1 GB | one process, two jobs |
| ai-vision-service (stub pipeline) | ~0.1 GB | FastAPI, no model loaded |
| ai-vision-service (Florence pipeline, on demand) | +1.2–1.5 GB | Florence-2-base + Opus-MT en→pt, lazy-loaded |
| 4 Go services | ~0.3 GB total | tiny static binaries |
| React dev server (`npm run dev`, outside Docker) | ~0.4 GB | |
| Browser with the storefront open | ~1–1.5 GB | the demo itself |

**Why this works:** the heavy models are **lazy-loaded and batched**. The storefront demo reads pre-computed alt-text from PostgreSQL; live uploads go through the queue and the `aria-live` progress announcement. Peak RAM only happens while the worker processes a batch — which can run overnight.

## Disk (500 GB HDD)

| Item | ~Size |
| --- | --- |
| Florence-2-base weights | ~0.5 GB |
| Opus-MT en→pt weights | ~0.3 GB |
| Tesseract + language packs (por, eng) | ~0.1 GB |
| Docker images (compose stack) | ~5–6 GB |
| node_modules + toolchains | ~1.5 GB |

The HDD is slow, so models live in a single volume and batches are processed sequentially — throughput matters more than latency for the pre-compute path.

## Design decisions enabled by this budget

1. **Composed AI pipeline** (ADR-0005): Florence-2 caption (EN) + Opus-MT translation instead of a single multilingual VLM that would need 4 GB+ of weights (e.g. Qwen2.5-VL). Quality per language is documented in the ADR.
2. **Stub pipeline by default** (`PIPELINE=stub`): the full platform — including CI and recruiters' local runs — works without downloading any weights.
3. **Redis Streams instead of RabbitMQ** (ADR-0004): one less container (~200 MB saved), and the broker sits behind an interface so v2.0 swaps it for Google Pub/Sub.
4. **Heavy tests in CI**: Playwright + axe and the full test matrix run on GitHub Actions, not on the notebook.
5. **Pre-computed alt-text in seed data**: the "No-Eyes Experience" demo is instant because the catalog ships with per-locale alt-text in PostgreSQL.

The next hardware step (16 GB) unlocks the v2.0 multilingual embedding service locally; until then multilingual-e5-small (~0.5 GB) fits the current budget.
