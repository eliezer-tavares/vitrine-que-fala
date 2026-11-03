package main

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Writer is the write face of the catalog (ADR-0007): alttext-worker output
// flows through the gRPC contract into these methods — the service that owns
// product_translations owns its writes.
type Writer interface {
	SetAltText(ctx context.Context, productID string, alt map[string]string) error
	SetEmbedding(ctx context.Context, productID, locale string, vec []float64) error
}

// PGRepo reads localized products from the normalized translations table
// (ADR-0001). Locale is a query parameter — adding locales never changes
// this code.
type PGRepo struct {
	DB *pgxpool.Pool
}

const (
	listSQL = `SELECT p.id, p.price_minor, p.currency, t.name, t.description, COALESCE(t.alt_text, '')
                FROM products p
                JOIN product_translations t ON t.product_id = p.id AND t.locale = $1
                ORDER BY p.created_at, p.id`
	getSQL = `SELECT p.id, p.price_minor, p.currency, t.name, t.description, COALESCE(t.alt_text, '')
                FROM products p
                JOIN product_translations t ON t.product_id = p.id AND t.locale = $1
                WHERE p.id = $2`
)

// List returns every product with its translation for loc.
func (r *PGRepo) List(ctx context.Context, loc string) ([]Product, error) {
	rows, err := r.DB.Query(ctx, listSQL, loc)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.PriceMinor, &p.Currency, &p.Name, &p.Description, &p.AltText); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Get returns one product; ErrNotFound when the row (or its translation) is
// missing.
func (r *PGRepo) Get(ctx context.Context, id, loc string) (*Product, error) {
	var p Product
	err := r.DB.QueryRow(ctx, getSQL, loc, id).
		Scan(&p.ID, &p.PriceMinor, &p.Currency, &p.Name, &p.Description, &p.AltText)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SetAltText writes per-locale alt-text (v2.0: called through the gRPC
// contract by alttext-worker).
func (r *PGRepo) SetAltText(ctx context.Context, productID string, alt map[string]string) error {
	for loc, txt := range alt {
		ct, err := r.DB.Exec(ctx,
			`UPDATE product_translations SET alt_text = $3 WHERE product_id = $1 AND locale = $2`,
			productID, loc, txt)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return errMissingTranslation
		}
	}
	return nil
}

// SetEmbedding writes one multilingual-e5 vector for a translation row
// (ADR-0009).
func (r *PGRepo) SetEmbedding(ctx context.Context, productID, loc string, vec []float64) error {
	ct, err := r.DB.Exec(ctx,
		`UPDATE product_translations SET embedding = $3 WHERE product_id = $1 AND locale = $2`,
		productID, loc, toFloat32(vec))
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return errMissingTranslation
	}
	return nil
}

// errMissingTranslation reports a write against a row that does not exist.
var errMissingTranslation = errors.New("translation row not found")

// toFloat32 converts e5 embeddings (float64 in our pipeline) to the
// float32 storage pgvector uses.
func toFloat32(vec []float64) []float32 {
	f := make([]float32, len(vec))
	for i, v := range vec {
		f[i] = float32(v)
	}
	return f
}
