# ADR-0005 — Composed AI pipeline within an 8 GB RAM budget

Status: Accepted (v1.0)
Date: 2026
Deciders: Eliezer Tavares de Oliveira

## Context

The AI service must caption product images and extract embedded text, running on a modest 8 GB notebook (Dell Latitude) and on Cloud Run. A single multilingual vision-language model (e.g. Qwen2.5-VL 7B) that could caption natively in all six roadmap languages needs 4 GB+ of weights — impossible here alongside the OS, screen reader and browser.

## Decision

Compose small, lazy-loaded models instead of one large VLM:

```
image bytes ──> Florence-2-base (caption, en-US) ──> Opus-MT en→pt (translation)
        └──────> Tesseract por+eng (OCR fragments)
```

- Every model is loaded **on first use** (lazy) and the pipeline is selected by env (`PIPELINE=stub|florence`).
- The **stub pipeline** is deterministic and dependency-free: CI, local demo and recruiter runs never wait on downloads.
- The storefront consumes **pre-computed** alt-text from PostgreSQL (batch job via alttext-worker, can run overnight); live uploads are async jobs announced via `aria-live`.

## Consequences

- Peak AI RAM ≈ 1.2–1.5 GB (caption + one translation pair), inside budget.
- Latency on CPU is seconds per image — hidden by design (queue + pre-compute), and the async UI turns it into an accessibility feature, not a defect.
- v2.0 risk: caption-then-translate quality for Thai/Vietnamese/Chinese must be measured per language; if quality is poor, the fallback is M2M100-418M (~1.2 GB) or accepting a two-tier pipeline (native for supported languages). Decision deferred to the v2.0 ADR with numbers.
- Rejected alternative: hosted closed VLM API — violates ADR-0002.

## Links

- ADR-0002 (open-source-only), ADR-0004 (queue), docs/RAM-BUDGET.md.
