// order-service accepts checkouts and publishes OrderPlaced events.
package main

import (
	"log"
	"net/http"

	"github.com/eliezer-tavares/vitrine-que-fala/backend/internal/httputil"
	"github.com/redis/go-redis/v9"
)

func main() {
	rdb := redis.NewClient(&redis.Options{Addr: env("REDIS_ADDR", "redis:6379")})

	h := &Handler{
		Broker: &RedisBroker{R: rdb},
		Stream: env("ORDERS_STREAM", "orders.events"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/v1/orders", h.Create)

	addr := ":" + env("PORT", "8082")
	log.Printf("order-service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
