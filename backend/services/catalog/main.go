// catalog-service serves the localized product catalog over HTTP (edge/dev)
// and gRPC (internal mesh — ADR-0007), with pgvector-backed semantic search
// (ADR-0009) and Prometheus metrics (ADR-0010).
package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/gen/vitrine"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/broker"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/httputil"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/obs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

const serviceName = "catalog"

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	obs.Init(serviceName)
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, env("DATABASE_URL", "postgres://vitrine:vitrine-dev@db:5432/vitrine"))
	if err != nil {
		log.Fatalf("catalog: postgres: %v", err)
	}
	defer pool.Close()

	// The cache stays Redis even with the Pub/Sub broker selected: it is a
	// cache, not an event bus (ADR-0008 keeps the two concerns separate).
	rdb := redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "redis:6379")})

	var embedder Embedder
	if ai := env("AI_SERVICE_URL", ""); ai != "" {
		embedder = NewHTTPEmbedder(ai)
		slog.Info("semantic search enabled", "embedder", ai)
	}

	repo := &PGRepo{DB: pool}
	h := &Handler{
		Repo:     repo,
		Cache:    &RedisCache{R: rdb, TTL: time.Minute},
		TTL:      time.Minute,
		Embedder: embedder,
	}

	// images.pending publisher goes through the broker interface too:
	// BROKER=pubsub moves this queue to Google Pub/Sub unchanged (ADR-0008).
	pub, err := broker.NewPublisher(ctx)
	if err != nil {
		log.Fatalf("catalog: broker publisher: %v", err)
	}
	images := &ImageHandler{Queue: pub, Stream: env("IMAGES_STREAM", "images.pending")}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /metrics", obs.MetricsHandler())
	mux.Handle("GET /api/v1/products", obs.Instrument(serviceName, "GET /api/v1/products", http.HandlerFunc(h.List)))
	mux.Handle("GET /api/v1/products/{id}", obs.Instrument(serviceName, "GET /api/v1/products/{id}", http.HandlerFunc(h.Get)))
	mux.Handle("GET /api/v1/search", obs.Instrument(serviceName, "GET /api/v1/search", http.HandlerFunc(h.Search)))
	mux.Handle("POST /api/v1/products/{id}/image", obs.Instrument(serviceName, "POST /api/v1/products/{id}/image", http.HandlerFunc(images.Enqueue)))

	// gRPC side of the contract (gateway reads, worker writes).
	lis, err := net.Listen("tcp", ":"+env("GRPC_PORT", "9081"))
	if err != nil {
		log.Fatalf("catalog: grpc listen: %v", err)
	}
	gs := grpc.NewServer()
	vitrine.RegisterCatalogServiceServer(gs, &CatalogGRPC{
		Repo:     repo,
		Writer:   repo,
		Cache:    h.Cache,
		TTL:      h.TTL,
		Embedder: embedder,
	})
	go func() {
		slog.Info("catalog grpc serving", "addr", lis.Addr().String())
		if err := gs.Serve(lis); err != nil {
			slog.Error("catalog grpc stopped", "error", err)
		}
	}()

	addr := ":" + env("PORT", "8081")
	slog.Info("catalog http listening", "addr", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
