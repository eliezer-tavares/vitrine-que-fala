# Observability — deliberately boring (ADR-0010)

Every service exposes the same two endpoints:

- `GET /healthz` — liveness (as in v1.0);
- `GET /metrics` — Prometheus pull format.

Logs are structured **JSON via `slog`** (Go) and uvicorn/JSON on the Python
side, always carrying a `service` field. `docker compose logs` and Cloud Run
logs both stay greppable without any agent.

## The design constraints

1. **No agents, no sidecars, no paid APM.** The 8 GB RAM budget rules the
   stack; pull-based scraping costs nothing when nobody scrapes.
2. **Bounded cardinality.** The HTTP middleware labels metrics with the mux
   *route pattern* (e.g. `GET /api/v1/products`), never the raw path.
3. **One metric namespace.** `vitrine_http_requests_total` and
   `vitrine_http_request_seconds` exist on every service; dashboards group by
   the `service` label.

## Metrics inventory

| Metric | Type | Labels | Meaning |
| --- | --- | --- | --- |
| `vitrine_http_requests_total` | counter | service, route, status | request count per route pattern |
| `vitrine_http_request_seconds` | histogram | service, route | latency distribution |

gRPC handlers log failures with `slog` (`code`, `message`) and surface as
HTTP 5xx through the gateway, so the same request metrics capture them.

## Local stack (optional profile)

```bash
make obs-up      # prometheus on :9090, grafana on :3000 (anonymous viewer)
make obs-down
```

`infra/prometheus.yml` scrapes gateway, catalog, order, worker and the AI
service; `infra/grafana-datasource.yml` provisions the datasource. The
profile is off by default — on an 8 GB machine you turn it on when you want
to look, and off when you don't.

## Cloud Run

Cloud Run provides request count/latency out of the box. The service
`/metrics` endpoints are intended for a Cloud Monitoring Prometheus scrape
(or managed collection) — configure once per environment, per
`infra/cloud-run.md`.

## What is NOT here (honest list)

- No distributed tracing yet (OpenTelemetry) — planned with the gRPC mesh
  maturity pass; the slog records carry enough context to trace by hand
  today.
- No alerting rules shipped; the metrics exist, the alerts belong to the
  environment that consumes them.
