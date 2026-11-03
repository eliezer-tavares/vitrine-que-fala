# ADR-0010: observability with slog JSON + pull-based Prometheus metrics

- Status: accepted (v2.0)
- Date: 2026
- Deciders: Eliezer Tavares de Oliveira

## Context

v1.0 logs were `log.Printf` lines — fine for one service, unqueryable for
five. The v2.0 promise includes "full observability", but the platform also
runs on an 8 GB notebook where agents and paid APMs are non-starters.

## Decision

- **Structured logging**: every Go service initializes `slog` with a JSON
  handler stamped with the service name (`internal/obs.Init`). The gateway
  and services log request outcomes and gRPC failures with fields, not
  prose.
- **Metrics**: an `internal/obs` package exposes a request counter and a
  latency histogram (`vitrine_http_requests_total`,
  `vitrine_http_request_seconds`) labeled by service and **route pattern**
  (bounded cardinality). Every service serves `/metrics` in Prometheus pull
  format next to `/healthz`; the AI service uses `prometheus-client`.
- **Optional local stack**: a compose `observability` profile runs
  Prometheus + Grafana with provisioned scrape/datasource config — off by
  default, one `make obs-up` away.
- **No agents, no sidecars, no paid APM.** Cloud Run covers request metrics
  natively; `/metrics` exists for managed Prometheus collection.

## Consequences

- Zero-cost observability floor: JSON logs + two metrics everywhere, same
  shape in dev and prod.
- What is deferred, honestly: distributed tracing (OpenTelemetry) and
  alerting rules. The metric/log shape is chosen so both can land later
  without renaming anything (`docs/OBSERVABILITY.md` keeps the honest list).
