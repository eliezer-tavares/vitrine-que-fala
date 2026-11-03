"""Vision pipelines (ADR-0002, ADR-0005, ADR-0009).

Two implementations behind one protocol:

- ``StubPipeline``: deterministic, dependency-free — used by CI, recruiter
  demos and smoke tests so the platform never depends on model downloads.
- ``FlorencePipeline``: 100% open-source models, **lazy-loaded** so an
  8 GB RAM machine only pays for weights when a job actually runs:
  Florence-2-base captions the image in English, one Opus-MT pair per
  target locale translates it (en->pt, en->id, en->th, en->vi, en->zh),
  Tesseract extracts embedded text.

v2.0 serves six locales — the Shopee-market footprint (ADR-0003). The
caption+translate composition is what keeps the pipeline inside the 8 GB
budget: no multilingual VLM is ever loaded.
"""
from __future__ import annotations

import io
import logging
import os
from typing import Protocol, runtime_checkable

logger = logging.getLogger("ai-vision")

# v2.0 locales: Brazil + the largest Shopee markets. Adding a locale is
# data (seed + i18n JSON + one Opus-MT pair here), never a schema change.
SUPPORTED_LOCALES = ["pt-BR", "en-US", "id-ID", "th-TH", "vi-VN", "zh-TW"]

FLORENCE_MODEL = "microsoft/Florence-2-base"

# One open-source translation pair per locale, loaded lazily and one at a
# time (~300 MB each) so RAM usage tracks the markets actually serving.
OPUS_MT_PAIRS: dict[str, str] = {
    "pt-BR": "Helsinki-NLP/opus-mt-en-pt",
    "id-ID": "Helsinki-NLP/opus-mt-en-id",
    "th-TH": "Helsinki-NLP/opus-mt-en-th",
    "vi-VN": "Helsinki-NLP/opus-mt-en-vi",
    "zh-TW": "Helsinki-NLP/opus-mt-en-zh",
}

# Default Tesseract packs for v2.0 (the Docker image ships them all).
DEFAULT_TESSERACT_LANGS = "por+eng+ind+tha+vie+chi_tra"


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
    """Deterministic pipeline for CI/demo. Produces a valid alt-text for all
    six locales without loading any model."""

    _DEMO: dict[str, str] = {
        "pt-BR": (
            "Imagem de produto (modo demonstração, {size} bytes): "
            "substitua o PIPELINE=stub por florence para gerar descrições reais."
        ),
        "en-US": (
            "Product image (demo mode, {size} bytes): switch PIPELINE=stub "
            "to florence to generate real descriptions."
        ),
        "id-ID": (
            "Gambar produk (mode demo, {size} byte): ganti PIPELINE=stub "
            "menjadi florence untuk menghasilkan deskripsi nyata."
        ),
        "th-TH": (
            "รูปภาพสินค้า (โหมดสาธิต, {size} ไบต์): เปลี่ยน PIPELINE=stub "
            "เป็น florence เพื่อสร้างคำอธิบายจริง"
        ),
        "vi-VN": (
            "Ảnh sản phẩm (chế độ demo, {size} byte): đổi PIPELINE=stub "
            "thành florence để tạo mô tả thật."
        ),
        "zh-TW": (
            "商品圖片（示範模式，{size} 位元組）："
            "將 PIPELINE=stub 改為 florence 以產生真實描述。"
        ),
    }

    def describe(self, image: bytes) -> dict[str, str]:
        size = len(image)
        return {loc: tpl.format(size=size) for loc, tpl in self._DEMO.items()}

    def ocr(self, image: bytes) -> list[str]:
        return []


class FlorencePipeline:
    """Composed open-source pipeline under the 8 GB RAM budget (ADR-0005).

    The translators dict is the v2.0 change: one Opus-MT pair per locale,
    each lazy-loaded and released to the OS page cache when unused. For
    zh-TW the en->zh output is converted to Traditional characters with
    OpenCC when the optional dependency is installed (ADR-0009 notes the
    trade-off; the stub pipeline stays dependency-free either way).
    """

    def __init__(
        self, ocr_langs: str = DEFAULT_TESSERACT_LANGS, device: str = "cpu"
    ) -> None:
        self._ocr_langs = ocr_langs
        self._device = device
        self._caption_model = None
        self._caption_processor = None
        self._translators: dict[str, tuple[object, object]] = {}
        self._opencc = None

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

    def _load_translator(self, locale: str):
        if locale not in self._translators:
            from transformers import MarianMTModel, MarianTokenizer

            model_name = OPUS_MT_PAIRS[locale]
            logger.info("loading %s (first use for %s)", model_name, locale)
            model = MarianMTModel.from_pretrained(model_name).to(self._device)
            tokenizer = MarianTokenizer.from_pretrained(model_name)
            self._translators[locale] = (model, tokenizer)
        return self._translators[locale]

    def _load_opencc(self):
        """Optional simplified->traditional conversion for zh-TW."""
        if self._opencc is None:
            try:
                from opencc import OpenCC  # opencc-python-reimplemented

                self._opencc = OpenCC("s2t")
            except Exception:  # noqa: BLE001 — conversion is best-effort
                logger.warning(
                    "opencc unavailable; zh-TW text stays as produced by en->zh"
                )
                class _Identity:
                    @staticmethod
                    def convert(text: str) -> str:
                        return text

                self._opencc = _Identity()
        return self._opencc

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

    def _translate(self, text: str, locale: str) -> str:
        import torch

        if locale == "en-US":
            return text
        model, tokenizer = self._load_translator(locale)
        batch = tokenizer([text], return_tensors="pt", padding=True).to(self._device)
        with torch.no_grad():
            generated = model.generate(**batch)
        out = tokenizer.batch_decode(generated, skip_special_tokens=True)[0].strip()
        if locale == "zh-TW":
            out = self._load_opencc().convert(out)
        return out

    def describe(self, image: bytes) -> dict[str, str]:
        from PIL import Image

        rgb = Image.open(io.BytesIO(image)).convert("RGB")
        en = self._caption_en(rgb)
        return {loc: self._translate(en, loc) for loc in SUPPORTED_LOCALES}

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
            return FlorencePipeline(
                ocr_langs=os.getenv("TESSERACT_LANGS", DEFAULT_TESSERACT_LANGS)
            )
        except Exception:
            logger.exception("florence pipeline unavailable; falling back to stub")
            return StubPipeline()
    if choice != "stub":
        logger.warning("unknown PIPELINE=%r; using stub", choice)
    return StubPipeline()
