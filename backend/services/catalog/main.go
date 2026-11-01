// catalog-service serves the localized product catalog.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/httputil"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// RedisQueuePublisher publishes alt-text jobs to the images.pending stream.
type RedisQueuePublisher struct {
	R *redis.Client
}

func (p *RedisQueuePublisher) XAdd(ctx context.Context, stream string, values map[string]any) error {
	return p.R.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: values}).Err()
}

func main() {
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, env("DATABASE_URL", "postgres://vitrine:vitrine-dev@db:5432/vitrine"))
	if err != nil {
		log.Fatalf("catalog: postgres: %v", err)
	}
	defer pool.Close()

	rdb := redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "redis:6379")})

	h := &Handler{
		Repo:  &PGRepo{DB: pool},
		Cache: &RedisCache{R: rdb, TTL: time.Minute},
		TTL:   time.Minute,
	}
	images := &ImageHandler{Queue: &RedisQueuePublisher{R: rdb}, Stream: env("IMAGES_STREAM", "images.pending")}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/v1/products", h.List)
	mux.HandleFunc("GET /api/v1/products/{id}", h.Get)
	mux.HandleFunc("POST /api/v1/products/{id}/image", images.Enqueue)

	addr := ":" + env("PORT", "8081")
	log.Printf("catalog-service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
