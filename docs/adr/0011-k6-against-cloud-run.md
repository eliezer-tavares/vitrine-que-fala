# ADR-0011: k6 against deployed environments only

- Status: accepted (v2.0)
- Date: 2026
- Deciders: Eliezer Tavares de Oliveira

## Context

The v2.0 roadmap promises "k6 load-test numbers". Numbers measured against
an 8 GB notebook with an HD would describe the notebook, not the platform —
and publishing them as platform numbers would be dishonest engineering.

## Decision

- Load-test scripts live in `k6/` (`browse.js`, `search.js`, `checkout.js`)
  and target `BASE_URL`, which is **always a deployed environment (Cloud
  Run)** in any result anyone quotes.
- Profiles are small and repeatable (3–5 VUs): the goal is a per-release
  sanity signal with explicit thresholds, not a stress campaign.
- Thresholds are the release contract: browse p95 < 800 ms, search p95 <
  1200 ms, checkout p95 < 1500 ms, error budgets 1–2%.
- `docs/LOAD-TESTING.md` ships an **empty results table** by design: numbers
  get filled from real runs and dated. Never invented.
- The search script separates `semantic` vs `lexical` outcomes using the
  API's `strategy` field, doubling as an embedding-coverage smoke test.

## Consequences

- The repo stays honest: infrastructure to measure exists before the first
  measurement is published.
- CI does not run k6 (no deployed target there); the Makefile exposes
  `k6-browse`/`k6-search` for release-time runs.
