package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/broker"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/currency"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/httputil"
)

// Product is the localized view of a catalog row. PriceMinor/Currency come
// from the product row; Price carries the display-ready formatting for the
// negotiated locale.
type Product struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AltText     string `json:"alt_text"`
	PriceMinor  int64  `json:"-"`
	Currency    string `json:"-"`
	Price       Price  `json:"price"`
}

// Price is the localized money view.
type Price struct {
	MinorUnits int64  `json:"minor_units"`
	Currency   string `json:"currency"`
	Formatted  string `json:"formatted"`
}

// ErrNotFound maps to HTTP 404.
var ErrNotFound = errors.New("product not found")

// Repo abstracts the storage of localized products.
type Repo interface {
	List(ctx context.Context, loc string) ([]Product, error)
	Get(ctx context.Context, id, loc string) (*Product, error)
	Search(ctx context.Context, loc, q string, limit int, vec []float64) ([]Product, error)
}

// Handler serves the localized catalog endpoints.
type Handler struct {
	Repo     Repo
	Cache    CacheStore
	TTL      time.Duration
	Embedder Embedder // optional: nil keeps search lexical (ADR-0009)
}

func newPrice(p Product, loc string) Price {
	return Price{
		MinorUnits: p.PriceMinor,
		Currency:   p.Currency,
		Formatted:  currency.Format(p.PriceMinor, p.Currency, loc),
	}
}

// marshalProducts/decodeProducts are the cache format: the ITEMS array only,
// shared by the HTTP and gRPC list paths so both hit the same entries.
func marshalProducts(items []Product) (string, error) {
	b, err := json.Marshal(items)
	return string(b), err
}

func decodeProducts(s string) []Product {
	var out []Product
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

// List serves GET /api/v1/products: locale negotiation, cache lookup keyed
// by locale, repo fallback, then cache fill.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	loc := httputil.Negotiated(r)
	key := CacheKey(loc)
	if h.Cache != nil {
		if s, ok := h.Cache.Get(r.Context(), key); ok {
			items := decodeProducts(s)
			if items != nil {
				httputil.JSON(w, http.StatusOK, map[string]any{"locale": loc, "items": items})
				return
			}
		}
	}
	items, err := h.Repo.List(r.Context(), loc)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	for i := range items {
		items[i].Price = newPrice(items[i], loc)
	}
	if h.Cache != nil {
		if body, err := marshalProducts(items); err == nil {
			h.Cache.Set(r.Context(), key, body, h.TTL)
		}
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"locale": loc, "items": items})
}

// Get serves GET /api/v1/products/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	loc := httputil.Negotiated(r)
	p, err := h.Repo.Get(r.Context(), r.PathValue("id"), loc)
	if errors.Is(err, ErrNotFound) {
		httputil.Error(w, http.StatusNotFound, "product not found")
		return
	}
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	p.Price = newPrice(*p, loc)
	httputil.JSON(w, http.StatusOK, p)
}

// QueuePublisher abstracts the images.pending producer. v2.0 routes it
// through the broker interface, so BROKER=pubsub moves this queue to Google
// Pub/Sub with zero changes here (ADR-0008).
type QueuePublisher = broker.Publisher

// ImageHandler enqueues alt-text jobs for a product image. The worker fills
// product_translations.alt_text per locale later (ADR-0005 pre-compute).
type ImageHandler struct {
	Queue  QueuePublisher
	Stream string
}

// Enqueue serves POST /api/v1/products/{id}/image {"image_url": "..."}.
func (h *ImageHandler) Enqueue(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ImageURL string `json:"image_url"`
	}
	if err := httputil.Decode(r, &in); err != nil || in.ImageURL == "" {
		httputil.Error(w, http.StatusBadRequest, "image_url is required")
		return
	}
	id := r.PathValue("id")
	if err := h.Queue.Publish(r.Context(), h.Stream, map[string]any{
		"product_id": id,
		"image_url":  in.ImageURL,
		"attempts":   int64(0),
	}); err != nil {
		httputil.Error(w, http.StatusInternalServerError, "could not enqueue image")
		return
	}
	httputil.JSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}
