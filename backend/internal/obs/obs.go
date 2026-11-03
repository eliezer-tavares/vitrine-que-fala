// Package obs centralizes v2.0 observability (ADR-0010): structured slog
// JSON logging plus pull-based Prometheus metrics. Deliberately boring and
// agent-free so the 8 GB RAM budget (docs/RAM-BUDGET.md) stays intact —
// every service simply exposes /metrics next to /healthz.
package obs

import (
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	requestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vitrine_http_requests_total",
		Help: "HTTP requests handled, by service, route pattern and status.",
	}, []string{"service", "route", "status"})

	requestSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "vitrine_http_request_seconds",
		Help:    "Request latency by service and route pattern.",
		Buckets: prometheus.DefBuckets,
	}, []string{"service", "route"})
)

// Init sets the default slog logger to JSON on stdout, stamped with the
// service name — the format Cloud Run and `docker compose logs` both read.
func Init(service string) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", service))
}

// MetricsHandler serves the Prometheus exposition format on /metrics.
func MetricsHandler() http.Handler { return promhttp.Handler() }

// statusRecorder captures the status code so the middleware can count it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Instrument adds request metrics to any handler. routePattern is the mux
// pattern (e.g. "GET /api/v1/products"), never the raw path — cardinality
// stays bounded (ADR-0010).
func Instrument(service, routePattern string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: 200}
		next.ServeHTTP(rec, r)
		requestsTotal.WithLabelValues(service, routePattern, strconv.Itoa(rec.status)).Inc()
		requestSeconds.WithLabelValues(service, routePattern).Observe(time.Since(start).Seconds())
	})
}
