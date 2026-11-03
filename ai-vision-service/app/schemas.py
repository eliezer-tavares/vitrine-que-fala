"""Request/response models for the AI service."""
from __future__ import annotations

from typing import Literal

from pydantic import BaseModel, Field


class AltTextResult(BaseModel):
    """Per-locale alt-text plus OCR fragments found inside the image."""

    alt_texts: dict[str, str] = Field(
        ...,
        description=(
            "Alt-text per locale. v2.0 locales: pt-BR, en-US, id-ID, "
            "th-TH, vi-VN, zh-TW."
        ),
    )
    ocr_text: list[str] = Field(
        default_factory=list,
        description='Text fragments extracted from the image (brand, "50% OFF").',
    )


class EmbedRequest(BaseModel):
    """Texts to embed in the shared multilingual space (ADR-0009)."""

    texts: list[str] = Field(..., min_length=1, max_length=32)
    input_type: Literal["query", "passage"] = "passage"


class EmbedResponse(BaseModel):
    """Vectors aligned with the request's texts order."""

    model: str = Field(..., description="Embedding model identifier.")
    dimensions: int = Field(..., description="Vector dimensionality (384).")
    embeddings: list[list[float]]
