# ADR-0008: Google Pub/Sub behind the same broker interface

- Status: accepted (v2.0)
- Date: 2026
- Deciders: Eliezer Tavares de Oliveira

## Context

ADR-0004 put the event bus behind an interface precisely so the transport
could change without touching business code: Redis Streams locally, Pub/Sub
in production. v2.0 delivers that promise.

## Decision

- A `broker` package defines `Publisher` and `Subscriber`; two
  implementations exist: `Redis*` (Streams, consumer groups, dead-letter
  stream) and `PubSub*` (topics, pull subscriptions, `<topic>.dead`).
- Selection is environment-driven: `BROKER=redis` (default) or
  `BROKER=pubsub` (+ `PUBSUB_PROJECT`; `PUBSUB_EMULATOR_HOST` works for
  local experiments and CI).
- Topic names are shared verbatim with the v1.0 streams: `images.pending`,
  `images.dead`, `orders.events` — no renaming ceremony when switching.
- Pub/Sub values travel as string attributes (plus JSON in `Data`), which
  maps cleanly onto the Redis stream field maps the processor already
  consumes; the `attempts` counter semantics survive unchanged.
- The catalog cache remains Redis even when the broker is Pub/Sub: a cache
  is not an event bus, and conflating the two would couple deployments.

## Consequences

- Cloud Run deployments can run with managed Pub/Sub (no Redis dependency
  for the event path); local runs stay on the zero-config Redis Streams.
- The worker's `Processor` logic is untouched by the choice — its tests run
  against fakes of the narrow interface, so both transports satisfy the
  same contract.
- Cost: Pub/Sub pricing applies in production; the emulator keeps CI free.
