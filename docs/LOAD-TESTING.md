# Load testing with k6 — methodology before numbers (ADR-0011)

The load tests live in [`k6/`](../k6/): `browse.js` (catalog in all six locales),
`search.js` (semantic + lexical paths) and `checkout.js` (token + gRPC-backed
order through the edge).

## The one rule

**k6 runs against a deployed environment (Cloud Run), never against the 8 GB
notebook.** The notebook is the development machine; measuring it would
measure the wrong thing. The ADR records this decision.

## How to run

```bash
# against Cloud Run (recommended):
BASE_URL=https://<gateway-url> k6 run k6/browse.js
BASE_URL=https://<gateway-url> k6 run k6/search.js
BASE_URL=https://<gateway-url> k6 run k6/checkout.js

# against a local docker compose stack (smoke only, tiny VUs):
BASE_URL=http://localhost:8080 k6 run k6/browse.js
```

Profiles are deliberately modest (3–5 VUs): the goal is a repeatable sanity
signal per release, not a stress campaign.

## Thresholds (the contract each release must meet)

| Script | Threshold |
| --- | --- |
| `browse.js` | p95 < 800 ms, error rate < 1% |
| `search.js` | p95 < 1200 ms, error rate < 1%, strategy reported as `semantic` or `lexical` |
| `checkout.js` | p95 < 1500 ms, error rate < 2%, receipt returns in the requested locale |

## Results per release (fill from real runs — never invent)

| Release | Environment | Script | VUs | p50 | p95 | Error rate | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| v2.0 | _to fill after the first Cloud Run deploy_ | | | | | | |

The table is intentionally empty: numbers that were never measured are
marketing, not engineering. Run the scripts, paste the summary, date it.

## Reading the search report

`search.js` exports two counters — `search_semantic` and `search_lexical` —
so the report shows how often pgvector answered versus the lexical fallback.
If every Thai query lands on `lexical`, the embeddings for that locale are
missing: run the batch indexer (`PIPELINE=florence EMBEDDER=e5` worker) and
re-run.
