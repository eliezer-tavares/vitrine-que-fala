# ADR-0009: pgvector + multilingual-e5-small for multilingual semantic search

- Status: accepted (v2.0)
- Date: 2026
- Deciders: Eliezer Tavares de Oliveira

## Context

Six locales now share one catalog. Lexical `ILIKE` search breaks the moment
a customer types a Thai query against a catalog that also contains
Portuguese — and maintaining per-language synonym tables does not scale to a
small team of one.

## Decision

- **pgvector** extends PostgreSQL (image `pgvector/pgvector:pg16`): a
  `vector(384)` column on `product_translations` plus an HNSW cosine index
  (`seed/003_search_v2.sql`). No new datastore to operate.
- The embedding model is **multilingual-e5-small** (MIT, ~120M params,
  ~0.5 GB lazy-loaded): one shared vector space for all six locales, with
  the E5 `query:`/`passage:` prefixes honored end to end.
- The AI service exposes `POST /embed` behind the `EMBEDDER` env
  (`stub|e5`); the **stub embedder** is deterministic and dependency-free so
  CI and demos never download weights (same philosophy as ADR-0005).
- The worker **indexes embeddings best-effort**: after writing alt-text it
  embeds each locale's text and writes it through the catalog gRPC; an
  embedding failure logs, skips, and the job still succeeds — search falls
  back to lexical for that row until the next batch re-index.
- Search is **two-strategy**: semantic nearest-neighbors when a query
  embedding is available, lexical `ILIKE` otherwise. The API response
  reports `strategy: "semantic" | "lexical"` so load tests and dashboards
  can tell which path answered.
- The query path uses the same `/embed` endpoint with `input_type: "query"`.

## Consequences

- One search box works in every market; the Thai query against a mixed
  catalog becomes a cosine similarity, not a string match.
- RAM cost stays honest: ~0.5 GB for e5, only when `EMBEDDER=e5` — the stub
  keeps the default stack light.
- Embeddings are eventually consistent with translations (best-effort
  indexing + batch repair), documented in `docs/LOAD-TESTING.md`.
- zh-TW note: the caption pipeline translates en→zh and converts to
  Traditional with OpenCC when the optional package is installed; the
  embedder itself is natively multilingual, so search quality does not
  depend on that conversion.
