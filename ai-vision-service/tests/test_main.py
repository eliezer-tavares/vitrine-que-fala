"""Contract tests for ai-vision-service.

The pipeline and embedder dependencies are overridden with fakes, so these
tests need no model weights, no Tesseract and no network — they validate the
API contract that the gateway, the Go worker and the storefront rely on.
"""
from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from app.main import app, get_pipeline
from app.vision import SUPPORTED_LOCALES, StubPipeline

EXPECTED_LOCALES = {"pt-BR", "en-US", "id-ID", "th-TH", "vi-VN", "zh-TW"}


class FakePipeline:
    """Deterministic pipeline used to pin the API contract."""

    def describe(self, image: bytes) -> dict[str, str]:
        return {loc: f"Descrição de teste ({loc})" for loc in SUPPORTED_LOCALES}

    def ocr(self, image: bytes) -> list[str]:
        return ["BRAND X", "50% OFF"] if image else []


class BrokenPipeline:
    def describe(self, image: bytes) -> dict[str, str]:
        raise RuntimeError("model exploded")

    def ocr(self, image: bytes) -> list[str]:
        return []


@pytest.fixture()
def client():
    app.dependency_overrides[get_pipeline] = lambda: FakePipeline()
    with TestClient(app) as c:
        yield c
    app.dependency_overrides.clear()


def test_healthz(client):
    resp = client.get("/healthz")
    assert resp.status_code == 200
    assert resp.json() == {"status": "ok"}


def test_metrics_exposes_prometheus_format(client):
    resp = client.get("/metrics")
    assert resp.status_code == 200
    assert "python_info" in resp.text


def test_describe_returns_all_six_locales_and_ocr(client):
    resp = client.post(
        "/describe",
        files={"image": ("product.jpg", b"fakejpegbytes", "image/jpeg")},
    )
    assert resp.status_code == 200
    body = resp.json()
    assert set(body["alt_texts"].keys()) == EXPECTED_LOCALES
    assert set(SUPPORTED_LOCALES) == EXPECTED_LOCALES
    for loc in EXPECTED_LOCALES:
        assert body["alt_texts"][loc].startswith("Descrição de teste")
    assert body["ocr_text"] == ["BRAND X", "50% OFF"]


def test_describe_rejects_empty_image(client):
    resp = client.post(
        "/describe",
        files={"image": ("product.jpg", b"", "image/jpeg")},
    )
    assert resp.status_code == 400


def test_describe_rejects_oversized_image(client):
    big = b"x" * (8 * 1024 * 1024 + 1)
    resp = client.post(
        "/describe",
        files={"image": ("product.jpg", big, "image/jpeg")},
    )
    assert resp.status_code == 413


def test_describe_degrades_to_503_on_pipeline_failure():
    app.dependency_overrides[get_pipeline] = lambda: BrokenPipeline()
    with TestClient(app) as c:
        resp = c.post(
            "/describe",
            files={"image": ("product.jpg", b"fakejpegbytes", "image/jpeg")},
        )
    assert resp.status_code == 503
    app.dependency_overrides.clear()


def test_stub_pipeline_is_deterministic_and_six_locale():
    stub = StubPipeline()
    alt = stub.describe(b"abc")
    assert set(alt.keys()) == EXPECTED_LOCALES
    assert stub.describe(b"abc") == alt
    assert stub.ocr(b"abc") == []
