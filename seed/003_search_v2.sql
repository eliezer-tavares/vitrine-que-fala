-- Vitrine que Fala — v2.0: multilingual semantic search (ADR-0009).
-- Requires the pgvector extension; docker-compose uses the pgvector/pgvector
-- image. Locales stay data: the same 384-dim space serves all six languages
-- via multilingual-e5-small.

CREATE EXTENSION IF NOT EXISTS vector;

ALTER TABLE product_translations
    ADD COLUMN IF NOT EXISTS embedding vector(384);

-- HNSW over cosine distance (<=>): approximate nearest-neighbor search that
-- stays fast on modest hardware for catalog-sized data.
CREATE INDEX IF NOT EXISTS idx_product_translations_embedding
    ON product_translations USING hnsw (embedding vector_cosine_ops);
