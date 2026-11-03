"""ai-vision-service HTTP API.

Endpoints:
- GET  /healthz   — liveness.
- POST /describe  — multipart image upload -> per-locale alt-text + OCR.
- POST /embed     — texts -> vectors in the shared multilingual space
                    (multilingual-e5-small; ADR-0009).
- GET  /metrics   — Prometheus exposition (ADR-0010).

The gateway proxies POST /api/v1/ai/alt-text here (path rewritten).
Architected by Eliezer Tavares de Oliveira.
"""
from __future__ import annotations

from fastapi import Depends, FastAPI, File, HTTPException, UploadFile
from fastapi.responses import PlainTextResponse
from prometheus_client import CONTENT_TYPE_LATEST, REGISTRY, generate_latest

from app import __version__
from app.embeddings import (
    EMBEDDING_DIMENSIONS,
    EMBEDDING_MODEL,
    Embedder,
    build_embedder,
)
from app.schemas import AltTextResult, EmbedRequest, EmbedResponse
from app.vision import VisionPipeline, build_pipeline

MAX_IMAGE_BYTES = 8 * 1024 * 1024  # 8 MB
MAX_EMBED_TEXTS = 32

app = FastAPI(
    title="Vitrine que Fala — ai-vision-service",
    version=__version__,
    description="100% open-source image captioning, OCR and multilingual "
    "embeddings for the accessible storefront. Architected by "
    "Eliezer Tavares de Oliveira.",
    contact={"name": "Eliezer Tavares de Oliveira",
             "email": "contact@eliezertavaresdeoliveira.com"},
    license_info={"name": "MIT"},
)


def get_pipeline() -> VisionPipeline:
    """Dependency seam: tests override this with a fake pipeline."""
    return build_pipeline()


def get_embedder() -> Embedder:
    """Dependency seam: tests override this with a fake embedder."""
    return build_embedder()


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok"}


@app.get("/metrics", response_class=PlainTextResponse)
def metrics() -> PlainTextResponse:
    """Prometheus pull endpoint (ADR-0010)."""
    return PlainTextResponse(generate_latest(REGISTRY), media_type=CONTENT_TYPE_LATEST)


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


@app.post(
    "/embed",
    response_model=EmbedResponse,
    responses={422: {"description": "validation error"},
               503: {"description": "embedder unavailable"}},
)
def embed(
    req: EmbedRequest,
    embedder: Embedder = Depends(get_embedder),
) -> EmbedResponse:
    if len(req.texts) > MAX_EMBED_TEXTS:
        raise HTTPException(
            status_code=422,
            detail=f"at most {MAX_EMBED_TEXTS} texts per request",
        )
    try:
        vectors = embedder.embed(req.texts, input_type=req.input_type)
    except Exception as exc:
        raise HTTPException(status_code=503, detail="embedder unavailable") from exc
    return EmbedResponse(
        model=EMBEDDING_MODEL,
        dimensions=EMBEDDING_DIMENSIONS,
        embeddings=vectors,
    )
