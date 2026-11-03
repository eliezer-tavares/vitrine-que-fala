# ADR-0007: gRPC for the internal service mesh, HTTP at the edge

- Status: accepted (v2.0)
- Date: 2026
- Deciders: Eliezer Tavares de Oliveira

## Context

v1.0 spoke HTTP everywhere: the gateway reverse-proxied catalog and order,
the worker wrote straight to PostgreSQL. Reading paths and business rules
were duplicated between the edge and the services, and "internal API" had no
contract stronger than a URL.

## Decision

- All **Go-to-Go** service communication moves to **gRPC**, with one proto
  module: `backend/api/proto/vitrine.proto`, generated stubs committed under
  `backend/gen/` (builds never require protoc; `make proto` regenerates).
- The **edge stays HTTP**: browsers cannot speak gRPC, and the storefront
  must remain a plain REST+JSON client (`docs/openapi/`).
- **Multipart uploads stay HTTP end-to-end** (product image enqueue, AI
  alt-text proxy): binary multipart over gRPC buys nothing and costs
  complexity.
- The worker **no longer connects to PostgreSQL**. It writes alt-text and
  embeddings through the catalog's `UpdateAltText`/`UpdateEmbedding` RPCs —
  the service that owns `product_translations` owns its writes. Data access
  gets a single owner.
- gRPC servers listen on their own ports (catalog :9081, order :9082) next
  to the HTTP servers (8081, 8082) — one binary, two transports.
- The AI service (Python/FastAPI) stays HTTP in v2.0. A Python gRPC server
  would add codegen surface for a single internal client; the typed HTTP
  client with explicit timeouts covers it. Revisit if the AI surface grows.

## Consequences

- Business rules live once: order validation/publishing is shared by the
  HTTP handler and the gRPC server; catalog list/get/search are shared the
  same way.
- The gateway test suite now runs against bufconn gRPC servers — the
  integration is real, not a mocked interface.
- `insecure` credentials are used inside the mesh; the trust boundary is the
  network (compose network / Cloud Run internal ingress). Documented in
  `infra/cloud-run.md`.
- Regenerating stubs is a deliberate, reviewable act (diff on `gen/`).
