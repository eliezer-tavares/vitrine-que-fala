package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/gen/vitrine"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/locale"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CatalogGRPC serves the internal catalog contract over gRPC (ADR-0007). It
// shares Repo/Cache with the HTTP handlers — one source of truth, two
// transports. Browser traffic never touches this server directly; the
// api-gateway is its only intended client (plus the alttext-worker for
// writes).
type CatalogGRPC struct {
	vitrine.UnimplementedCatalogServiceServer
	Repo     Repo
	Writer   Writer // alt-text/embedding writes (ADR-0007: catalog owns them)
	Cache    CacheStore
	TTL      time.Duration
	Embedder Embedder
}

func grpcPrice(p Product) *vitrine.Price {
	return &vitrine.Price{MinorUnits: p.PriceMinor, Currency: p.Currency, Formatted: p.Price.Formatted}
}

func grpcProduct(p Product) *vitrine.Product {
	return &vitrine.Product{
		Id: p.ID, Name: p.Name, Description: p.Description,
		AltText: p.AltText, Price: grpcPrice(p),
	}
}

// list is the shared list path: cache lookup, repo fallback, cache fill.
func (s *CatalogGRPC) list(ctx context.Context, loc string) ([]Product, error) {
	key := CacheKey(loc)
	if s.Cache != nil {
		if cached, ok := s.Cache.Get(ctx, key); ok {
			return decodeProducts(cached), nil
		}
	}
	items, err := s.Repo.List(ctx, loc)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].Price = newPrice(items[i], loc)
	}
	if s.Cache != nil {
		if body, err := marshalProducts(items); err == nil {
			s.Cache.Set(ctx, key, body, s.TTL)
		}
	}
	return items, nil
}

// ListProducts serves the localized listing.
func (s *CatalogGRPC) ListProducts(ctx context.Context, req *vitrine.ListProductsRequest) (*vitrine.ListProductsResponse, error) {
	loc := req.GetLocale()
	if loc == "" {
		loc = locale.Default
	}
	items, err := s.list(ctx, loc)
	if err != nil {
		slog.Error("grpc ListProducts failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	out := &vitrine.ListProductsResponse{Locale: loc}
	for i := range items {
		out.Items = append(out.Items, grpcProduct(items[i]))
	}
	return out, nil
}

// GetProduct serves one localized product.
func (s *CatalogGRPC) GetProduct(ctx context.Context, req *vitrine.GetProductRequest) (*vitrine.GetProductResponse, error) {
	loc := req.GetLocale()
	if loc == "" {
		loc = locale.Default
	}
	p, err := s.Repo.Get(ctx, req.GetId(), loc)
	if err == ErrNotFound {
		return nil, status.Error(codes.NotFound, "product not found")
	}
	if err != nil {
		slog.Error("grpc GetProduct failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	p.Price = newPrice(*p, loc)
	return &vitrine.GetProductResponse{Product: grpcProduct(*p)}, nil
}

// SearchProducts runs the semantic (pgvector) or lexical (ILIKE) query.
func (s *CatalogGRPC) SearchProducts(ctx context.Context, req *vitrine.SearchProductsRequest) (*vitrine.SearchProductsResponse, error) {
	loc := req.GetLocale()
	if loc == "" {
		loc = locale.Default
	}
	q := req.GetQuery()
	if q == "" {
		return nil, status.Error(codes.InvalidArgument, "query is required")
	}
	strategy := "lexical"
	var vec []float64
	if s.Embedder != nil {
		if embeddings, err := s.Embedder.Embed(ctx, []string{q}); err == nil && len(embeddings) == 1 {
			vec = embeddings[0]
			strategy = "semantic"
		} else if err != nil {
			slog.Warn("grpc SearchProducts: embedder unavailable, using lexical", "error", err)
		}
	}
	items, err := s.Repo.Search(ctx, loc, q, int(req.GetLimit()), vec)
	if err != nil {
		slog.Error("grpc SearchProducts failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	out := &vitrine.SearchProductsResponse{Locale: loc, Strategy: strategy}
	for i := range items {
		items[i].Price = newPrice(items[i], loc)
		out.Items = append(out.Items, grpcProduct(items[i]))
	}
	return out, nil
}

// UpdateAltText persists per-locale alt-text produced by alttext-worker
// (moved here from the worker's direct SQL — writes now go through the
// service that owns the table, ADR-0007).
func (s *CatalogGRPC) UpdateAltText(ctx context.Context, req *vitrine.UpdateAltTextRequest) (*vitrine.UpdateAltTextResponse, error) {
	if req.GetProductId() == "" {
		return nil, status.Error(codes.InvalidArgument, "product_id is required")
	}
	if err := s.Writer.SetAltText(ctx, req.GetProductId(), req.GetAltTexts()); err != nil {
		slog.Error("grpc UpdateAltText failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &vitrine.UpdateAltTextResponse{}, nil
}

// UpdateEmbedding persists the multilingual-e5 vector for one translation
// row (ADR-0009). Rows without embeddings simply never surface in semantic
// search — the lexical fallback covers them.
func (s *CatalogGRPC) UpdateEmbedding(ctx context.Context, req *vitrine.UpdateEmbeddingRequest) (*vitrine.UpdateEmbeddingResponse, error) {
	if req.GetProductId() == "" || req.GetLocale() == "" || len(req.GetVector()) == 0 {
		return nil, status.Error(codes.InvalidArgument, "product_id, locale and vector are required")
	}
	vec := make([]float64, len(req.GetVector()))
	for i, v := range req.GetVector() {
		vec[i] = float64(v)
	}
	if err := s.Writer.SetEmbedding(ctx, req.GetProductId(), req.GetLocale(), vec); err != nil {
		slog.Error("grpc UpdateEmbedding failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &vitrine.UpdateEmbeddingResponse{}, nil
}
