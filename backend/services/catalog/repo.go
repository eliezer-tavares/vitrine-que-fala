package main

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
