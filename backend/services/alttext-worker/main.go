package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// PGStore writes per-locale alt-text into product_translations (ADR-0001).
type PGStore struct {
	DB *pgxpool.Pool
}

func (s *PGStore) SetAltText(ctx context.Context, productID string, alt map[string]string) error {
	for loc, txt := range alt {
		ct, err := s.DB.Exec(ctx,
			`UPDATE product_translations SET alt_text = $3 WHERE product_id = $1 AND locale = $2`,
			productID, loc, txt)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			log.Printf("alttext-worker: no translation row for product %s locale %s", productID, loc)
		}
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := pgxpool.New(ctx, env("DATABASE_URL", "postgres://vitrine:vitrine-dev@db:5432/vitrine"))
	if err != nil {
		log.Fatalf("alttext-worker: postgres: %v", err)
	}
	defer pool.Close()

	stream := env("IMAGES_STREAM", "images.pending")
	group := env("GROUP", "alttext-workers")
	dead := env("DEAD_STREAM", "images.dead")
	workers := envInt("WORKERS", 4)

	host, _ := os.Hostname()
	rdb := redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "redis:6379")})

	for i := 0; i < workers; i++ {
		i := i
		q := &RedisQueue{R: rdb, Consumer: fmt.Sprintf("%s-%d", host, i)}
		if err := q.EnsureGroup(ctx, stream, group); err != nil {
			log.Fatalf("alttext-worker: ensure group: %v", err)
		}
		p := &Processor{
			AI:          NewHTTPAIClient(env("AI_SERVICE_URL", "http://ai:8083")),
			Store:       &PGStore{DB: pool},
			Queue:       q,
			Stream:      stream,
			Group:       group,
			DeadStream:  dead,
			MaxAttempts: envInt("MAX_ATTEMPTS", 3),
		}
		go func() {
			log.Printf("alttext-worker goroutine %d (%s) started", i, q.Consumer)
			runLoop(ctx, q, p, stream, group)
		}()
	}

	// Tiny health endpoint so orchestration can probe the worker.
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{Addr: ":" + env("PORT", "8084"), Handler: mux}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	log.Printf("alttext-worker running with %d goroutines, health on :%s", workers, env("PORT", "8084"))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
