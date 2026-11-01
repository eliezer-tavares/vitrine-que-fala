"""Response models for the AI service."""
from __future__ import annotations

from pydantic import BaseModel, Field


class AltTextResult(BaseModel):
    """Per-locale alt-text plus OCR fragments found inside the image."""

    alt_texts: dict[str, str] = Field(
        ...,
        description="Alt-text per locale. v1.0 locales: pt-BR, en-US.",
    )
    ocr_text: list[str] = Field(
        default_factory=list,
        description='Text fragments extracted from the image (brand, "50% OFF").',
    )
