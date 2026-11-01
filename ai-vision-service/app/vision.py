"""Vision pipelines (ADR-0002 and ADR-0005).

Two implementations behind one protocol:

- ``StubPipeline``: deterministic, dependency-free — used by CI, recruiter
  demos and smoke tests so the platform never depends on model downloads.
- ``FlorencePipeline``: 100% open-source models, **lazy-loaded** so an
  8 GB RAM machine only pays for weights when a job actually runs:
  Florence-2-base captions the image in English, Opus-MT translates to
  pt-BR, Tesseract extracts embedded text (por+eng).
"""
from __future__ import annotations

import io
import logging
import os
from typing import Protocol, runtime_checkable

logger = logging.getLogger("ai-vision")

# v1.0 locales. v2.0 adds id-ID, th-TH, vi-VN, zh-TW (data, not code — ADR-0001).
SUPPORTED_LOCALES = ["pt-BR", "en-US"]

FLORENCE_MODEL = "microsoft/Florence-2-base"
OPUS_MT_EN_PT = "Helsinki-NLP/opus-mt-en-pt"


@runtime_checkable
class VisionPipeline(Protocol):
    """Anything that can caption and OCR a product image."""

    def describe(self, image: bytes) -> dict[str, str]:
        """Return alt-text keyed by locale."""
        ...

    def ocr(self, image: bytes) -> list[str]:
        """Return text fragments found inside the image."""
        ...


class StubPipeline:
    """Deterministic pipeline for CI/demo. Produces a valid alt-text for both
    locales without loading any model."""

    def describe(self, image: bytes) -> dict[str, str]:
        size = len(image)
        return {
            "pt-BR": (
                f"Imagem de produto (modo demonstração, {size} bytes): "
                "substitua o PIPELINE=stub por florence para gerar descrições reais."
            ),
            "en-US": (
                f"Product image (demo mode, {size} bytes): switch PIPELINE=stub "
                "to florence to generate real descriptions."
            ),
        }

    def ocr(self, image: bytes) -> list[str]:
        return []


class FlorencePipeline:
    """Composed open-source pipeline under the 8 GB RAM budget (ADR-0005)."""

    def __init__(self, ocr_langs: str = "por+eng", device: str = "cpu") -> None:
        self._ocr_langs = ocr_langs
        self._device = device
        self._caption_model = None
        self._caption_processor = None
        self._translator = None

    # -- lazy loading -----------------------------------------------------
    def _load_caption(self):
        if self._caption_model is None:
            from transformers import AutoModelForCausalLM, AutoProcessor

            logger.info("loading %s (first use)", FLORENCE_MODEL)
            self._caption_processor = AutoProcessor.from_pretrained(
                FLORENCE_MODEL, trust_remote_code=True
            )
            self._caption_model = AutoModelForCausalLM.from_pretrained(
                FLORENCE_MODEL, trust_remote_code=True
            ).to(self._device)
        return self._caption_processor, self._caption_model

    def _load_translator(self):
        if self._translator is None:
            from transformers import MarianMTModel, MarianTokenizer

            logger.info("loading %s (first use)", OPUS_MT_EN_PT)
            self._translator = MarianMTModel.from_pretrained(OPUS_MT_EN_PT).to(
                self._device
            )
            self._translator_tokenizer = MarianTokenizer.from_pretrained(OPUS_MT_EN_PT)
        return self._translator, self._translator_tokenizer

    # -- pipeline ---------------------------------------------------------
    def _caption_en(self, rgb_image) -> str:
        import torch  # local import keeps torch out of the stub path

        processor, model = self._load_caption()
        inputs = processor(
            text="<MORE_DETAILED_CAPTION>", images=rgb_image, return_tensors="pt"
        ).to(self._device)
        with torch.no_grad():
            generated = model.generate(
                input_ids=inputs["input_ids"],
                pixel_values=inputs["pixel_values"],
                max_new_tokens=256,
                num_beams=3,
            )
        return processor.batch_decode(generated, skip_special_tokens=True)[0].strip()

    def _translate_pt(self, text: str) -> str:
        import torch

        model, tokenizer = self._load_translator()
        batch = tokenizer([text], return_tensors="pt", padding=True).to(self._device)
        with torch.no_grad():
            generated = model.generate(**batch)
        return tokenizer.batch_decode(generated, skip_special_tokens=True)[0].strip()

    def describe(self, image: bytes) -> dict[str, str]:
        from PIL import Image

        rgb = Image.open(io.BytesIO(image)).convert("RGB")
        en = self._caption_en(rgb)
        return {"en-US": en, "pt-BR": self._translate_pt(en)}

    def ocr(self, image: bytes) -> list[str]:
        try:
            import pytesseract
            from PIL import Image

            text = pytesseract.image_to_string(
                Image.open(io.BytesIO(image)), lang=self._ocr_langs
            )
        except Exception:  # noqa: BLE001 — OCR is best-effort, never fatal
            logger.warning("OCR unavailable or failed; returning no fragments")
            return []
        return [line.strip() for line in text.splitlines() if line.strip()]


def build_pipeline() -> VisionPipeline:
    """Factory selected by the PIPELINE env var (stub|florence). Unknown
    values fall back to the stub with a loud warning — never crash the demo."""
    choice = os.getenv("PIPELINE", "stub").strip().lower()
    if choice == "florence":
        try:
            return FlorencePipeline(ocr_langs=os.getenv("TESSERACT_LANGS", "por+eng"))
        except Exception:
            logger.exception("florence pipeline unavailable; falling back to stub")
            return StubPipeline()
    if choice != "stub":
        logger.warning("unknown PIPELINE=%r; using stub", choice)
    return StubPipeline()
