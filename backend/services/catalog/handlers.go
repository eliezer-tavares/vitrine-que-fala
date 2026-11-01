package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

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
}

// Handler serves the localized catalog endpoints.
type Handler struct {
	Repo  Repo
	Cache CacheStore
	TTL   time.Duration
}

func newPrice(p Product, loc string) Price {
	return Price{
		MinorUnits: p.PriceMinor,
		Currency:   p.Currency,
		Formatted:  currency.Format(p.PriceMinor, p.Currency, loc),
	}
}

// List serves GET /api/v1/products: locale negotiation, cache lookup keyed
// by locale, repo fallback, then cache fill.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	loc := httputil.Negotiated(r)
	key := CacheKey(loc)
	if h.Cache != nil {
		if s, ok := h.Cache.Get(r.Context(), key); ok {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(s))
			return
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
	body, err := json.Marshal(map[string]any{"locale": loc, "items": items})
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	if h.Cache != nil {
		h.Cache.Set(r.Context(), key, string(body), h.TTL)
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(body)
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

// QueuePublisher abstracts the images.pending stream producer.
type QueuePublisher interface {
	XAdd(ctx context.Context, stream string, values map[string]any) error
}

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
	if err := h.Queue.XAdd(r.Context(), h.Stream, map[string]any{
		"product_id": id,
		"image_url":  in.ImageURL,
		"attempts":   int64(0),
	}); err != nil {
		httputil.Error(w, http.StatusInternalServerError, "could not enqueue image")
		return
	}
	httputil.JSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}
