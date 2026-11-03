// alttext-worker consumes the images.pending queue and fills per-locale
// alt-text + embeddings. v2.0: talks to the catalog over gRPC (ADR-0007),
// consumes through the broker interface (Redis Streams locally, Google
// Pub/Sub when BROKER=pubsub — ADR-0008) and indexes embeddings for
// semantic search (ADR-0009).
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/broker"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/obs"
)

const serviceName = "alttext-worker"

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return def
}

func main() {
	obs.Init(serviceName)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	stream := env("IMAGES_STREAM", "images.pending")
	group := env("GROUP", "alttext-workers")
	dead := env("DEAD_STREAM", "images.dead")
	workers := envInt("WORKERS", 4)

	ai := NewHTTPAIClient(env("AI_SERVICE_URL", "http://ai:8083"))
	catalog, err := NewCatalogClient(env("GRPC_CATALOG_ADDR", "catalog:9081"))
	if err != nil {
		slog.Error("catalog grpc client failed", "error", err)
		os.Exit(1)
	}

	// One subscriber (consumer identity) per goroutine — the same model the
	// Redis consumer groups used, now broker-agnostic.
	host, _ := os.Hostname()
	for i := 0; i < workers; i++ {
		i := i
		consumer := fmt.Sprintf("%s-%d", host, i)

		sub, err := broker.NewSubscriber(ctx, consumer)
		if err != nil {
			slog.Error("broker subscriber failed", "error", err)
			os.Exit(1)
		}
		if err := sub.EnsureGroup(ctx, stream, group); err != nil {
			slog.Error("ensure consumer group failed", "error", err)
			os.Exit(1)
		}

		p := &Processor{
			AI:          ai,
			Embedder:    ai, // the same service serves /describe and /embed
			Store:       catalog,
			Queue:       sub,
			Stream:      stream,
			Group:       group,
			DeadStream:  dead,
			MaxAttempts: envInt("MAX_ATTEMPTS", 3),
		}
		go func() {
			slog.Info("worker goroutine started", "index", i, "consumer", consumer)
			runLoop(ctx, sub, p, stream, group)
		}()
	}

	// Health + metrics endpoint so orchestration can probe the worker.
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/metrics", obs.MetricsHandler())
	srv := &http.Server{Addr: ":" + env("PORT", "8084"), Handler: mux}
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()
	port := env("PORT", "8084")
	slog.Info("worker running", "goroutines", workers, "health_port", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("health server failed", "error", err)
		os.Exit(1)
	}
}

// runLoop is one worker goroutine: block-read, process, repeat, until ctx is
// done. Storage failures are logged and left pending for redelivery.
func runLoop(ctx context.Context, sub broker.Subscriber, p *Processor, stream, group string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		msgs, err := sub.Read(ctx, stream, group, 1)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Warn("worker read failed; backing off", "error", err)
			sleepOneSecond(ctx)
			continue
		}
		for _, m := range msgs {
			if err := p.Handle(ctx, m); err != nil {
				slog.Warn("worker handle failed; message stays pending", "error", err)
				sleepOneSecond(ctx)
			}
		}
	}
}

func sleepOneSecond(ctx context.Context) {
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
	}
}
