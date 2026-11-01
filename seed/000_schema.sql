-- Vitrine que Fala — v1.0 schema (ADR-0001: normalized translations table).
-- Locale-aware by design: adding locales in v2.0 is data, not migration.

CREATE TABLE IF NOT EXISTS products (
    id          UUID PRIMARY KEY,
    price_minor BIGINT       NOT NULL CHECK (price_minor >= 0),
    currency    TEXT         NOT NULL,
    image_url   TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS product_translations (
    product_id  UUID   NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    locale      TEXT   NOT NULL,
    name        TEXT   NOT NULL,
    description TEXT   NOT NULL,
    alt_text    TEXT,
    PRIMARY KEY (product_id, locale)
);

CREATE INDEX IF NOT EXISTS idx_product_translations_locale
    ON product_translations (locale);
