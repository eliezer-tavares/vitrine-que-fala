"""Text embeddings for multilingual semantic search (ADR-0009).

Two implementations behind one protocol:

- ``StubEmbedder``: deterministic, dependency-free 384-dim vectors (hashed
  bag-of-tokens, L2-normalized). Good enough for CI, demos and the
  semantic-vs-lexical contract tests — and honest about being a placeholder.
- ``E5Embedder``: ``intfloat/multilingual-e5-small`` (MIT, ~120M params,
  384 dims), lazy-loaded so the 8 GB RAM budget only pays when indexing or
  querying actually happens. One shared vector space covers all six locales.

E5 requires the ``query: ``/``passage: `` prefixes to work as trained; the
protocol therefore carries an ``input_type`` parameter end to end.
"""
from __future__ import annotations

import hashlib
import logging
import math
import os
import re
from typing import Protocol, runtime_checkable

logger = logging.getLogger("ai-vision")

EMBEDDING_MODEL = "intfloat/multilingual-e5-small"
EMBEDDING_DIMENSIONS = 384

_TOKEN_RE = re.compile(r"\w+", re.UNICODE)


@runtime_checkable
class Embedder(Protocol):
    """Anything that can turn texts into vectors in the shared space."""

    def embed(self, texts: list[str], input_type: str = "passage") -> list[list[float]]:
        """Return one vector per text, L2-normalized."""
        ...


def _pseudo_random_unit(seed_material: bytes) -> float:
    """Deterministic uniform in [-1, 1] from a digest (no deps, no seeds)."""
    value = int.from_bytes(seed_material[:8], "big") / float(1 << 64)
    return value * 2.0 - 1.0


class StubEmbedder:
    """Hashed bag-of-tokens embedder: deterministic, language-agnostic and
    normalized, so cosine similarity behaves for contract tests. It is NOT a
    semantic model — it exists so CI and recruiter demos never download
    weights (same philosophy as StubPipeline, ADR-0005)."""

    def embed(self, texts: list[str], input_type: str = "passage") -> list[list[float]]:
        out: list[list[float]] = []
        for text in texts:
            vec = [0.0] * EMBEDDING_DIMENSIONS
            tokens = _TOKEN_RE.findall(text.lower()) or ["empty"]
            for tok in tokens:
                digest = hashlib.md5(tok.encode("utf-8")).digest()
                for i in range(EMBEDDING_DIMENSIONS):
                    # Re-hash with an index suffix so dimensions decorrelate.
                    d = hashlib.md5(digest + str(i).encode()).digest()
                    vec[i] += _pseudo_random_unit(d)
            norm = math.sqrt(sum(x * x for x in vec)) or 1.0
            out.append([x / norm for x in vec])
        return out


class E5Embedder:
    """multilingual-e5-small with lazy loading and mean pooling."""

    def __init__(self, device: str = "cpu", batch_size: int = 8) -> None:
        self._device = device
        self._batch_size = batch_size
        self._model = None
        self._tokenizer = None

    def _load(self):
        if self._model is None:
            import torch
            from transformers import AutoModel, AutoTokenizer

            logger.info("loading %s (first use)", EMBEDDING_MODEL)
            self._tokenizer = AutoTokenizer.from_pretrained(EMBEDDING_MODEL)
            self._model = AutoModel.from_pretrained(EMBEDDING_MODEL).to(self._device)
            self._model.eval()
            del torch  # torch is imported inside the hot path below anyway
        return self._tokenizer, self._model

    def embed(self, texts: list[str], input_type: str = "passage") -> list[list[float]]:
        import torch

        tokenizer, model = self._load()
        prefixed = [f"{input_type}: {t}" for t in texts]
        out: list[list[float]] = []
        for start in range(0, len(prefixed), self._batch_size):
            batch = prefixed[start : start + self._batch_size]
            encoded = tokenizer(
                batch,
                padding=True,
                truncation=True,
                max_length=256,
                return_tensors="pt",
            ).to(self._device)
            with torch.no_grad():
                hidden = model(**encoded).last_hidden_state
            mask = encoded["attention_mask"].unsqueeze(-1).float()
            summed = (hidden * mask).sum(dim=1)
            counts = mask.sum(dim=1).clamp(min=1e-9)
            mean = summed / counts
            mean = torch.nn.functional.normalize(mean, p=2, dim=1)
            out.extend(mean.cpu().tolist())
        return out


def build_embedder() -> Embedder:
    """Factory selected by the EMBEDDER env var (stub|e5). Unknown values
    fall back to the stub with a loud warning — same contract as the
    pipeline factory."""
    choice = os.getenv("EMBEDDER", "stub").strip().lower()
    if choice == "e5":
        try:
            return E5Embedder()
        except Exception:
            logger.exception("e5 embedder unavailable; falling back to stub")
            return StubEmbedder()
    if choice != "stub":
        logger.warning("unknown EMBEDDER=%r; using stub", choice)
    return StubEmbedder()
