// api-gateway is the public edge: JWT auth, Accept-Language negotiation with
// fallback chain, and the storefront API. v2.0: reads (products, search) and
// checkout travel over gRPC to the internal services (ADR-0007); multipart
// uploads and the AI proxy stay HTTP end-to-end. The negotiated locale is
// passed explicitly in the gRPC requests and stamped as X-Client-Locale on
// the remaining HTTP hops (ADR-0003).
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	stdhttputil "net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/gen/vitrine"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/httputil"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/jwtutil"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/locale"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/obs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const serviceName = "api-gateway"

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type Gateway struct {
	Secret      string
	Clients     *Clients
	CatalogHTTP *url.URL // multipart image upload stays HTTP (ADR-0007)
	AIURL       *url.URL
}

// bearerToken extracts "Bearer <token>" from the Authorization header.
func bearerToken(h string) string {
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// proxy returns a handler that forwards to target over HTTP, stamping the
// negotiated locale (and the authenticated subject when auth is required).
// Kept for the multipart image upload and the AI proxy only (ADR-0007).
func (g *Gateway) proxy(target *url.URL, auth bool, rewritePath string) http.Handler {
	rp := stdhttputil.NewSingleHostReverseProxy(target)
	origDirector := rp.Director
	rp.Director = func(req *http.Request) {
		origDirector(req)
		req.Host = target.Host
		if rewritePath != "" {
			req.URL.Path = rewritePath
		}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth {
			sub, err := jwtutil.Parse(g.Secret, bearerToken(r.Header.Get("Authorization")))
			if err != nil {
				httputil.Error(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			r.Header.Set("X-Auth-Subject", sub)
		}
		r.Header.Set("X-Client-Locale", locale.Negotiate(r.Header.Get("Accept-Language")))
		rp.ServeHTTP(w, r)
	})
}

// writeGRPCError maps a gRPC status to the edge's HTTP error JSON.
func writeGRPCError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		httputil.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	switch st.Code() {
	case codes.NotFound:
		httputil.Error(w, http.StatusNotFound, st.Message())
	case codes.InvalidArgument:
		httputil.Error(w, http.StatusBadRequest, st.Message())
	default:
		slog.Error("grpc call failed", "code", st.Code().String(), "message", st.Message())
		httputil.Error(w, http.StatusInternalServerError, "internal error")
	}
}

func productJSON(p *vitrine.Product) map[string]any {
	price := map[string]any{"minor_units": 0, "currency": "", "formatted": ""}
	if pp := p.GetPrice(); pp != nil {
		price = map[string]any{
			"minor_units": pp.GetMinorUnits(),
			"currency":    pp.GetCurrency(),
			"formatted":   pp.GetFormatted(),
		}
	}
	return map[string]any{
		"id":          p.GetId(),
		"name":        p.GetName(),
		"description": p.GetDescription(),
		"alt_text":    p.GetAltText(),
		"price":       price,
	}
}

// productsHandler serves GET /api/v1/products via the catalog gRPC contract.
func (g *Gateway) productsHandler(w http.ResponseWriter, r *http.Request) {
	loc := locale.Negotiate(r.Header.Get("Accept-Language"))
	res, err := g.Clients.Catalog.ListProducts(r.Context(), &vitrine.ListProductsRequest{Locale: loc})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(res.GetItems()))
	for _, p := range res.GetItems() {
		items = append(items, productJSON(p))
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"locale": res.GetLocale(), "items": items})
}

// productHandler serves GET /api/v1/products/{id}.
func (g *Gateway) productHandler(w http.ResponseWriter, r *http.Request) {
	loc := locale.Negotiate(r.Header.Get("Accept-Language"))
	res, err := g.Clients.Catalog.GetProduct(r.Context(), &vitrine.GetProductRequest{
		Locale: loc,
		Id:     r.PathValue("id"),
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, productJSON(res.GetProduct()))
}

// searchHandler serves GET /api/v1/search?q=&limit= (semantic or lexical —
// the response carries the strategy, ADR-0009).
func (g *Gateway) searchHandler(w http.ResponseWriter, r *http.Request) {
	loc := locale.Negotiate(r.Header.Get("Accept-Language"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		httputil.Error(w, http.StatusBadRequest, "query parameter q is required")
		return
	}
	req := &vitrine.SearchProductsRequest{Locale: loc, Query: q, Limit: 10}
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 50 {
			req.Limit = int32(n)
		}
	}
	res, err := g.Clients.Catalog.SearchProducts(r.Context(), req)
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(res.GetItems()))
	for _, p := range res.GetItems() {
		items = append(items, productJSON(p))
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"locale":   res.GetLocale(),
		"query":    q,
		"strategy": res.GetStrategy(),
		"items":    items,
	})
}

// ordersHandler serves POST /api/v1/orders over gRPC after JWT validation.
func (g *Gateway) ordersHandler(w http.ResponseWriter, r *http.Request) {
	loc := locale.Negotiate(r.Header.Get("Accept-Language"))

	sub, err := jwtutil.Parse(g.Secret, bearerToken(r.Header.Get("Authorization")))
	if err != nil {
		httputil.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var in struct {
		Items []struct {
			ProductID string `json:"product_id"`
			Quantity  int    `json:"quantity"`
		} `json:"items"`
	}
	if err := httputil.Decode(r, &in); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid order payload")
		return
	}
	items := make([]*vitrine.OrderItem, 0, len(in.Items))
	for _, it := range in.Items {
		items = append(items, &vitrine.OrderItem{ProductId: it.ProductID, Quantity: int32(it.Quantity)})
	}

	res, err := g.Clients.Order.CreateOrder(context.Background(), &vitrine.CreateOrderRequest{
		Items:   items,
		Subject: sub,
		Locale:  loc,
	})
	if err != nil {
		writeGRPCError(w, err)
		return
	}
	httputil.JSON(w, http.StatusCreated, map[string]any{
		"order_id": res.GetOrderId(),
		"locale":   res.GetLocale(),
		"message":  res.GetMessage(),
		"items":    in.Items,
	})
}

// tokenHandler issues demo JWTs (POST /api/v1/auth/token {"sub": "..."}).
func (g *Gateway) tokenHandler(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Sub string `json:"sub"`
	}
	if err := httputil.Decode(r, &in); err != nil || strings.TrimSpace(in.Sub) == "" {
		httputil.Error(w, http.StatusBadRequest, "sub is required")
		return
	}
	tok, err := jwtutil.Generate(g.Secret, in.Sub, time.Hour)
	if err != nil {
		httputil.Error(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{
		"access_token": tok,
		"token_type":   "Bearer",
		"expires_in":   3600,
	})
}

func main() {
	obs.Init(serviceName)

	clients, cleanup, err := NewClients(
		env("GRPC_CATALOG_ADDR", "catalog:9081"),
		env("GRPC_ORDER_ADDR", "order:9082"),
	)
	if err != nil {
		log.Fatalf("gateway: grpc clients: %v", err)
	}
	defer cleanup()

	g := &Gateway{
		Secret:      env("JWT_SECRET", "dev-secret-change-me"),
		Clients:     clients,
		CatalogHTTP: mustURL(env("CATALOG_URL", "http://catalog:8081")),
		AIURL:       mustURL(env("AI_URL", "http://ai:8083")),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /metrics", obs.MetricsHandler())
	mux.Handle("POST /api/v1/auth/token", obs.Instrument(serviceName, "POST /api/v1/auth/token", http.HandlerFunc(g.tokenHandler)))

	// Catalog reads over gRPC (ADR-0007).
	mux.Handle("GET /api/v1/products", obs.Instrument(serviceName, "GET /api/v1/products", http.HandlerFunc(g.productsHandler)))
	mux.Handle("GET /api/v1/products/{id}", obs.Instrument(serviceName, "GET /api/v1/products/{id}", http.HandlerFunc(g.productHandler)))
	mux.Handle("GET /api/v1/search", obs.Instrument(serviceName, "GET /api/v1/search", http.HandlerFunc(g.searchHandler)))

	// Multipart image upload stays HTTP (binary proxying, ADR-0007).
	mux.Handle("POST /api/v1/products/{id}/image", obs.Instrument(serviceName, "POST /api/v1/products/{id}/image", g.proxy(g.CatalogHTTP, true, "")))

	// Checkout over gRPC; the AI demo proxy stays HTTP for the recruiter
	// flow (documented in the OpenAPI + ADR-0007).
	mux.Handle("POST /api/v1/orders", obs.Instrument(serviceName, "POST /api/v1/orders", http.HandlerFunc(g.ordersHandler)))
	mux.Handle("POST /api/v1/ai/alt-text", obs.Instrument(serviceName, "POST /api/v1/ai/alt-text", g.proxy(g.AIURL, false, "/describe")))

	addr := ":" + env("PORT", "8080")
	slog.Info("api-gateway listening", "addr", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		log.Fatalf("gateway: invalid URL %q: %v", raw, err)
	}
	return u
}
