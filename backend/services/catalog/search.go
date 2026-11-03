package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/httputil"
	"github.com/pgvector/pgvector-go"
)

// Embedder turns text into a vector in the shared multilingual space
// (ADR-0009). The reference implementation lives in ai-vision-service
// (multilingual-e5-small, 384 dims); tests use a fake.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

// HTTPEmbedder calls POST /embed on the AI service. Same timeout posture as
// the worker's caption client: lazy model loads make the first call slow.
type HTTPEmbedder struct {
	BaseURL string
	HTTP    *http.Client
}

func NewHTTPEmbedder(baseURL string) *HTTPEmbedder {
	return &HTTPEmbedder{BaseURL: baseURL, HTTP: &http.Client{Timeout: 90 * time.Second}}
}

func (e *HTTPEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	payload, err := json.Marshal(map[string]any{"texts": texts, "input_type": "query"})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.BaseURL+"/embed", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := e.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("embed failed with status %s", res.Status)
	}
	var out struct {
		Embeddings [][]float64 `json:"embeddings"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Embeddings, nil
}

const (
	searchSemanticSQL = `SELECT p.id, p.price_minor, p.currency, t.name, t.description, COALESCE(t.alt_text, '')
                FROM product_translations t
                JOIN products p ON p.id = t.product_id
                WHERE t.locale = $1 AND t.embedding IS NOT NULL
                ORDER BY t.embedding <=> $2
                LIMIT $3`
	searchLexicalSQL = `SELECT p.id, p.price_minor, p.currency, t.name, t.description, COALESCE(t.alt_text, '')
                FROM product_translations t
                JOIN products p ON p.id = t.product_id
                WHERE t.locale = $1
                  AND (t.name ILIKE $2 OR t.description ILIKE $2 OR COALESCE(t.alt_text, '') ILIKE $2)
                ORDER BY p.created_at, p.id
                LIMIT $3`
)

// Search answers a localized query: semantic nearest-neighbors when the
// query embedding is available, lexical ILIKE otherwise (ADR-0009). The
// caller reports which strategy answered via the API payload.
func (r *PGRepo) Search(ctx context.Context, loc, q string, limit int, vec []float64) ([]Product, error) {
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}
	var (
		rows pgxRows
		err  error
	)
	if vec != nil {
		rows, err = r.DB.Query(ctx, searchSemanticSQL, loc, pgvector.NewVector(toFloat32(vec)), limit)
	} else {
		pattern := "%" + strings.ReplaceAll(strings.ReplaceAll(q, "%", ""), "_", "") + "%"
		rows, err = r.DB.Query(ctx, searchLexicalSQL, loc, pattern, limit)
	}
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

// pgxRows is the minimal row-cursor interface used by Search (keeps the
// function testable against a fake pool if needed later).
type pgxRows interface {
	Next() bool
	Scan(dest ...any) error
	Close()
	Err() error
}

// Search serves GET /api/v1/search?q=&limit= — the direct HTTP face of the
// gRPC SearchProducts the gateway calls (used by smoke tests and curl).
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	loc := httputil.Negotiated(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		httputil.Error(w, http.StatusBadRequest, "query parameter q is required")
		return
	}
	limit := 10
	if n := r.URL.Query().Get("limit"); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v >= 1 && v <= 50 {
			limit = v
		}
	}

	strategy := "lexical"
	var vec []float64
	if h.Embedder != nil {
		if embeddings, err := h.Embedder.Embed(r.Context(), []string{q}); err == nil && len(embeddings) == 1 {
			vec = embeddings[0]
			strategy = "semantic"
		}
	}

	items, err := h.Repo.Search(r.Context(), loc, q, limit, vec)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	for i := range items {
		items[i].Price = newPrice(items[i], loc)
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"locale":   loc,
		"query":    q,
		"strategy": strategy,
		"items":    items,
	})
}
