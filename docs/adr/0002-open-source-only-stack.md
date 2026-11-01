# ADR-0002 — 100% open-source AI stack (no paid/closed APIs)

Status: Accepted (v1.0)
Date: 2026
Deciders: Eliezer Tavares de Oliveira

## Context

The platform needs image captioning, translation and OCR. Closed hosted APIs (e.g. Gemini, OpenAI) are convenient but: (1) they cost money per call, (2) user images leave the machine, (3) the project stops working if a key/quota disappears, (4) the pipeline cannot be reproduced by anyone cloning the repository.

## Decision

Use only open-weight, self-hostable models:

- **Captioning:** Florence-2-base (MIT) — 0.23B parameters, runs on CPU, does detailed captioning and OCR-like tasks in one model.
- **Translation:** Opus-MT (Helsinki-NLP, Apache-2.0/CC-BY per pair) — one ~300 MB model per language pair, lazy-loaded.
- **OCR:** Tesseract (Apache-2.0) with `por`, `eng` (and later `ind`, `tha`, `vie`, `chi_tra`) language packs.
- **Embeddings (v2.0):** multilingual-e5 (MIT).

Licenses are verified per model and recorded here — that check is part of the ADR template.

## Consequences

- **Zero API cost** and **zero data egress**: product images never leave the deployment.
- **Reproducibility:** every clone runs the same pipeline; CI never depends on external quotas.
- **Hardware discipline:** models must fit an 8 GB RAM machine → composed pipeline (ADR-0005) instead of one large multilingual VLM.
- **Trade-off:** caption quality in Thai/Vietnamese/Chinese is reached via translation rather than natively; quality is measured per language when v2.0 lands and documented in the ADR that decides the v2.0 pipeline.
- **Stub pipeline** (`PIPELINE=stub`) keeps CI and recruiter demos independent of model downloads.

## Links

- ADR-0005 (composed pipeline under 8 GB) implements this decision.
