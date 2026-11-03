"""Contract tests for POST /embed (ADR-0009).

The stub embedder answers offline; these tests pin the contract the Go
services rely on: model/dimensions metadata, vector alignment and
deterministic, normalized output.
"""
from __future__ import annotations

import math

import pytest
from fastapi.testclient import TestClient

from app.embeddings import (
    EMBEDDING_DIMENSIONS,
    EMBEDDING_MODEL,
    StubEmbedder,
    build_embedder,
)
from app.main import app, get_embedder


@pytest.fixture()
def client():
    app.dependency_overrides[get_embedder] = lambda: StubEmbedder()
    with TestClient(app) as c:
        yield c
    app.dependency_overrides.clear()


def test_embed_returns_metadata_and_aligned_vectors(client):
    resp = client.post(
        "/embed",
        json={"texts": ["Poncho amarelo", "yellow poncho"], "input_type": "passage"},
    )
    assert resp.status_code == 200
    body = resp.json()
    assert body["model"] == EMBEDDING_MODEL
    assert body["dimensions"] == EMBEDDING_DIMENSIONS
    assert len(body["embeddings"]) == 2
    for vec in body["embeddings"]:
        assert len(vec) == EMBEDDING_DIMENSIONS


def test_embed_is_deterministic_and_normalized(client):
    resp1 = client.post("/embed", json={"texts": ["ร่มพับอัตโนมัติ"]})
    resp2 = client.post("/embed", json={"texts": ["ร่มพับอัตโนมัติ"]})
    assert resp1.json()["embeddings"] == resp2.json()["embeddings"]

    vec = resp1.json()["embeddings"][0]
    norm = math.sqrt(sum(x * x for x in vec))
    assert norm == pytest.approx(1.0, abs=1e-6)


def test_embed_rejects_empty_text_list(client):
    resp = client.post("/embed", json={"texts": []})
    assert resp.status_code == 422


def test_embed_rejects_too_many_texts(client):
    resp = client.post("/embed", json={"texts": [f"text {i}" for i in range(33)]})
    assert resp.status_code == 422


def test_stub_embedder_differentiates_texts():
    stub = StubEmbedder()
    a, b = stub.embed(["หูฟ้าบลูทูธ", "蓝牙耳机"])
    assert a != b  # different texts land in different directions


def test_build_embedder_factory_defaults_to_stub():
    assert isinstance(build_embedder(), StubEmbedder)
