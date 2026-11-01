# Vitrine que Fala — CLI-first developer workflow (screen-reader friendly).
# Run `make help` to list targets.

SHELL := /bin/bash
COMPOSE := docker compose

.PHONY: help up down logs seed test test-go test-py web-test web-build e2e build-images tag-v1 tidy

help: ## list available targets
        @grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

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
        cd ai-vision-service && python -m pytest tests -q

web-test: ## vitest unit tests incl. jest-axe assertions (web)
        cd web && npm run test

web-build: ## type-check + production build (web)
        cd web && npm run build

e2e: ## Playwright keyboard + axe E2E (web)
        cd web && npx playwright test

build-images: ## build the Docker images (stub pipeline default)
        $(COMPOSE) build

tag-v1: ## create the v1.0 tag locally (see docs/SETUP.md before pushing)
        git tag -a v1.0 -m "v1.0 — MVP acessível bilíngue / bilingual accessible MVP"

tidy: ## go mod tidy for the backend module
        cd backend && go mod tidy
