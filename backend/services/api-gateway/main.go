// api-gateway is the public edge: JWT auth, Accept-Language negotiation with
// fallback chain, and reverse proxying to the internal services. The
// negotiated locale travels downstream as X-Client-Locale (ADR-0003) and
// validated subjects as X-Auth-Subject.
package main

import (
	"log"
	"net/http"
	stdhttputil "net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/httputil"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/jwtutil"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/locale"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type Gateway struct {
	Secret string
}

// bearerToken extracts "Bearer <token>" from the Authorization header.
func bearerToken(h string) string {
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// proxy returns a handler that forwards to target, stamping the negotiated
// locale (and the authenticated subject when auth is required).
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
	g := &Gateway{Secret: env("JWT_SECRET", "dev-secret-change-me")}

	catalog := mustURL(env("CATALOG_URL", "http://localhost:8081"))
	order := mustURL(env("ORDER_URL", "http://localhost:8082"))
	ai := mustURL(env("AI_URL", "http://localhost:8083"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/v1/auth/token", g.tokenHandler)

	// Catalog is public (product browsing is the storefront's core).
	mux.Handle("GET /api/v1/products", g.proxy(catalog, false, ""))
	mux.Handle("GET /api/v1/products/{id}", g.proxy(catalog, false, ""))
	mux.Handle("POST /api/v1/products/{id}/image", g.proxy(catalog, true, ""))

	// Checkout requires auth; the AI demo stays public in v1.0 for the
	// recruiter flow (documented in the OpenAPI + v2.0 plan).
	mux.Handle("POST /api/v1/orders", g.proxy(order, true, ""))
	mux.Handle("POST /api/v1/ai/alt-text", g.proxy(ai, false, "/describe"))

	addr := ":" + env("PORT", "8080")
	log.Printf("api-gateway listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		log.Fatalf("gateway: invalid URL %q: %v", raw, err)
	}
	return u
}
