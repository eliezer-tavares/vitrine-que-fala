# ADR-0004 — Redis Streams first, Google Pub/Sub at scale

Status: Accepted (v1.0)
Date: 2026
Deciders: Eliezer Tavares de Oliveira

## Context

Two async flows need a queue in v1.0: `OrderPlaced` events (order-service) and `images.pending` alt-text jobs (alttext-worker). The deployment reality is an 8 GB notebook (ADR-0005) and a GCP target where Pub/Sub is the managed option.

## Decision

Use **Redis Streams** (consumer groups, `XADD`/`XREADGROUP`/`XACK`, dead-letter stream for exhausted retries) with **all producers/consumers behind a `Broker` interface**. v2.0 swaps the implementation for **Google Pub/Sub** without touching business code; the interface becomes the compatibility boundary and the swap itself becomes a demonstrable migration.

## Consequences

- One less container: ~200 MB RAM saved and no extra broker to operate locally.
- Consumer-group semantics (at-least-once, pending list, `XACK`) already map to Pub/Sub acks — the v2.0 migration is mechanical, not a redesign.
- Trade-off: Redis is not a durable broker (AOF persistence optional). For v1.0's demo-scale volume this is acceptable; the retry + dead-letter design limits blast radius. Documented for v2.0 review.
- Rejected alternatives: RabbitMQ (extra container for no functional gain at this scale), direct HTTP calls (loses retry/backpressure semantics and the event-driven story).

## Links

- ADR-0002/0005 for the resource rationale.
