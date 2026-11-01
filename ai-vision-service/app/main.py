"""ai-vision-service HTTP API.

Endpoints:
- GET  /healthz   — liveness.
- POST /describe  — multipart image upload -> per-locale alt-text + OCR.

The gateway proxies POST /api/v1/ai/alt-text here (path rewritten).
Architected by Eliezer Tavares de Oliveira.
"""
from __future__ import annotations

from fastapi import Depends, FastAPI, File, HTTPException, UploadFile

from app import __version__
from app.schemas import AltTextResult
from app.vision import VisionPipeline, build_pipeline

MAX_IMAGE_BYTES = 8 * 1024 * 1024  # 8 MB

app = FastAPI(
    title="Vitrine que Fala — ai-vision-service",
    version=__version__,
    description="100% open-source image captioning + OCR for the accessible "
    "storefront. Architected by Eliezer Tavares de Oliveira.",
    contact={"name": "Eliezer Tavares de Oliveira",
             "email": "contact@eliezertavaresdeoliveira.com"},
    license_info={"name": "MIT"},
)


def get_pipeline() -> VisionPipeline:
    """Dependency seam: tests override this with a fake pipeline."""
    return build_pipeline()


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.post(
    "/describe",
    response_model=AltTextResult,
    responses={413: {"description": "image too large"},
               503: {"description": "AI pipeline unavailable"}},
)
async def describe(
    image: UploadFile = File(..., description="Product image (max 8 MB)"),
    pipeline: VisionPipeline = Depends(get_pipeline),
) -> AltTextResult:
    data = await image.read()
    if not data:
        raise HTTPException(status_code=400, detail="empty image")
    if len(data) > MAX_IMAGE_BYTES:
        raise HTTPException(status_code=413, detail="image too large (max 8 MB)")

    try:
        alt_texts = pipeline.describe(data)
    except Exception as exc:
        raise HTTPException(status_code=503, detail="AI pipeline unavailable") from exc

    fragments = pipeline.ocr(data)
    return AltTextResult(alt_texts=alt_texts, ocr_text=fragments)
