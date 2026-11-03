# Vitrine que Fala — CLI-first developer workflow (screen-reader friendly).
# Run `make help` to list targets. v2.0 adds proto, observability and k6.

SHELL := /bin/bash
COMPOSE := docker compose

.PHONY: help up down logs seed test test-go test-py web-test web-build e2e build-images proto proto-install tag-v2 obs-up obs-down k6-browse k6-search tidy

help: ## list available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

up: ## start db, redis, ai, services and worker
	$(COMPOSE) up -d

down: ## stop everything
	$(COMPOSE) down

logs: ## follow gateway logs
	$(COMPOSE) logs -f gateway

seed: ## wait for the gateway and run smoke checks
	./infra/seed-wait.sh

test: test-go test-py ## run Go and Python test suites

test-go: ## Go unit tests (backend)
	cd backend && go test ./...

test-py: ## Python contract tests (ai-vision-service)
	cd ai-vision-service && python -m pytest -q

web-test: ## vitest unit tests incl. jest-axe assertions (web)
	cd web && npm run test

web-build: ## type-check + production build (web)
	cd web && npm run build

e2e: ## Playwright keyboard + axe E2E (web)
	cd web && npx playwright test

build-images: ## build the Docker images (stub pipeline default)
	$(COMPOSE) build

proto-install: ## install the protoc Go plugins (one-time)
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.4.0

proto: ## regenerate the gRPC stubs (committed to backend/gen — ADR-0007)
	cd backend && protoc -I api/proto \
	        --go_out=. --go_opt=module=github.com/eliezer-tavares/vitrine-que-fala/backend \
	        --go-grpc_out=. --go-grpc_opt=module=github.com/eliezer-tavares/vitrine-que-fala/backend \
	        api/proto/vitrine.proto

obs-up: ## start prometheus + grafana (optional profile, ADR-0010)
	$(COMPOSE) --profile observability up -d prometheus grafana

obs-down: ## stop the observability profile
	$(COMPOSE) --profile observability stop prometheus grafana

k6-browse: ## load test the browsing flow (BASE_URL required, see docs/LOAD-TESTING.md)
	k6 run k6/browse.js

k6-search: ## load test the semantic/lexical search flow
	k6 run k6/search.js

tag-v2: ## create the v2.0 tag locally (see docs/SETUP.md before pushing)
	git tag -a v2.0 -m "v2.0 — seis idiomas, gRPC, Pub/Sub, pgvector, k6 / six locales, gRPC, Pub/Sub, pgvector, k6"

tidy: ## go mod tidy for the backend module
	cd backend && go mod tidy
