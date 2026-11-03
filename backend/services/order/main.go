// order-service accepts checkouts over HTTP (edge/dev) and gRPC (internal
// mesh — ADR-0007) and publishes OrderPlaced events through the broker
// interface: Redis Streams locally, Google Pub/Sub when BROKER=pubsub
// (ADR-0008).
package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/gen/vitrine"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/broker"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/httputil"
	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/obs"
	"google.golang.org/grpc"
)

const serviceName = "order"

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	obs.Init(serviceName)
	ctx := context.Background()

	pub, err := broker.NewPublisher(ctx)
	if err != nil {
		log.Fatalf("order: broker publisher: %v", err)
	}

	h := &Handler{
		Broker: pub,
		Stream: env("ORDERS_STREAM", "orders.events"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.Handle("GET /metrics", obs.MetricsHandler())
	mux.Handle("POST /api/v1/orders", obs.Instrument(serviceName, "POST /api/v1/orders", http.HandlerFunc(h.Create)))

	lis, err := net.Listen("tcp", ":"+env("GRPC_PORT", "9082"))
	if err != nil {
		log.Fatalf("order: grpc listen: %v", err)
	}
	gs := grpc.NewServer()
	vitrine.RegisterOrderServiceServer(gs, &OrderGRPC{Handler: h})
	go func() {
		slog.Info("order grpc serving", "addr", lis.Addr().String())
		if err := gs.Serve(lis); err != nil {
			slog.Error("order grpc stopped", "error", err)
		}
	}()

	addr := ":" + env("PORT", "8082")
	slog.Info("order http listening", "addr", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
