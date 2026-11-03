# RAM budget — engineering under an 8 GB constraint

This project is architected to **develop, run and demo on a modest 8 GB RAM notebook** (Dell Latitude, 500 GB HDD). That constraint shaped real design decisions, documented in the ADRs. This file is the account — v2.0 edition.

## Local development budget (8 GB total)

| Slice | ~RAM | Notes |
| --- | --- | --- |
| OS + screen reader (NVDA/Orca) + editor | 2.5–3.5 GB | unavoidable base |
| WSL2 / Docker (capped by `.wslconfig`) | up to 4 GB | `memory=4GB`, `processors=4` |
| PostgreSQL 16 + pgvector | ~0.35 GB | catalog data + HNSW index |
| Redis (cache + Redis Streams queue) | ~0.1 GB | one process, two jobs |
| ai-vision-service (stub pipeline + stub embedder) | ~0.15 GB | FastAPI, no models loaded |
| ai-vision-service (Florence pipeline, on demand) | +1.5–2.0 GB | Florence-2-base + one Opus-MT pair at a time (lazy) |
| Embeddings (multilingual-e5-small, on demand) | +0.5–0.6 GB | `EMBEDDER=e5`; lazy, batch-friendly |
| 4 Go services + gRPC | ~0.35 GB total | tiny static binaries |
| React dev server (`npm run dev`, outside Docker) | ~0.4 GB | |
| Browser with the storefront open | ~1–1.5 GB | the demo itself |
| Prometheus + Grafana (optional profile) | +0.7 GB | off unless `make obs-up` |

**Why this works:** the heavy models are **lazy-loaded and batched, one at a
time**. v2.0 adds five Opus-MT pairs and an embedding model — none of them
load unless a job actually runs, and the batch indexer processes locales
sequentially. The storefront demo reads pre-computed alt-text from
PostgreSQL; live uploads go through the queue with the `aria-live` progress
announcement. Peak RAM only happens during a batch — which can run
overnight.

## Disk (500 GB HDD)

| Item | ~Size |
| --- | --- |
| Florence-2-base weights | ~0.5 GB |
| Opus-MT pairs (en→pt, en→id, en→th, en→vi, en→zh) | ~1.5 GB total |
| multilingual-e5-small | ~0.5 GB |
| Tesseract + language packs (por, eng, ind, tha, vie, chi_tra) | ~0.2 GB |
| Docker images (compose stack) | ~6 GB |
| node_modules + toolchains | ~2 GB |

The HDD is slow, so models live in a single volume and batches are processed sequentially — throughput matters more than latency for the pre-compute path.

## Design decisions enabled by this budget

1. **Composed AI pipeline** (ADR-0005): Florence-2 caption (EN) + per-locale Opus-MT instead of a single multilingual VLM that would need 4 GB+ of weights.
2. **Stub everything by default** (`PIPELINE=stub`, `EMBEDDER=stub`): the full platform — CI, recruiter demos — works without downloading any weights.
3. **Redis Streams by default** (ADR-0004/0008): one less container locally; Pub/Sub selected only when deploying.
4. **gRPC without new processes** (ADR-0007): two listeners in the same binaries — no service-mesh sidecars.
5. **Observability without agents** (ADR-0010): pull-based Prometheus, optional profile.
6. **Heavy tests in CI** (Playwright + axe in six locales, E2E keyboard runs): GitHub Actions' RAM, not the notebook's.
7. **Pre-computed alt-text in seed data**: the "No-Eyes Experience" demo is instant in every locale.
