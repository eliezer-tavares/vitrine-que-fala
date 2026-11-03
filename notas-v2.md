# v2.0 — seis idiomas, gRPC, Pub/Sub, pgvector, k6

> **Arquitetado por Eliezer Tavares de Oliveira** — [contact@eliezertavaresdeoliveira.com](mailto:contact@eliezertavaresdeoliveira.com)
> Eliezer é cego, cursa Engenharia de Computação na Univesp (Universidade Virtual do Estado de São Paulo) e arquitetou esta aplicação para demonstrar sua qualificação técnica, buscando uma oportunidade profissional como engenheiro de software.

---

## Português (Brasil)

A **Vitrine que Fala** cresceu de MVP bilíngue para plataforma completa — e continua 100% open source, projetada para um notebook de 8 GB de RAM. A IA gera alt-text para **100% do catálogo nos seis idiomas dos mercados da Shopee** (pt-BR, en-US, bahasa Indonésio, tailandês, vietnamita, chinês tradicional), a busca semântica funciona **entre idiomas**, e toda a experiência — inclusive a busca — continua perfeita com leitor de tela e teclado.

### O que esta release entrega

- **Seis locales como dados, não como código** (ADR-0003) — traduções do catálogo, recibos, interface, Web Speech e OCR entram por seed + JSON de i18n; adicionar o sétimo idioma não muda arquitetura.
- **gRPC entre serviços** (ADR-0007) — contrato único em `backend/api/proto/vitrine.proto` com stubs commitados; a borda continua HTTP/JSON; o worker parou de falar com o Postgres e escreve pelo serviço dono dos dados.
- **Google Pub/Sub atrás da interface de broker** (ADR-0008) — `BROKER=redis|pubsub`; local roda Redis Streams, a nuvem pode rodar Pub/Sub, e o código de serviço não sabe a diferença.
- **Busca semântica multilíngue** (ADR-0009) — pgvector + HNSW no PostgreSQL, embeddings do multilingual-e5-small (MIT, lazy); uma busca em tailandês encontra o produto em português; a resposta informa `strategy: semantic|lexical`.
- **Moedas por mercado** (ADR-0012) — Rp, ฿, ₫ e NT$ com as convenções de cada país (zero-decimais, sufixo, separadores).
- **Observabilidade sem agentes** (ADR-0010) — logs JSON com `slog` e `/metrics` Prometheus em todos os serviços; perfil opcional com Prometheus + Grafana.
- **Harness de carga honesto** (ADR-0011) — scripts k6 para navegação, busca e checkout com thresholds; a tabela de resultados nasce vazia: números só entram de execuções reais.
- **Acessibilidade em seis idiomas no CI** — varreduras axe na home e no modo Sem Olhos nos seis idiomas; E2E por teclado troca para tailandês; busca anuncia contagem de resultados por `aria-live`.

### Como experimentar

```bash
cp .env.example .env && docker compose up -d
cd web && npm install && npm run dev   # http://localhost:5173
```

Troque o idioma para tailandês, busque "เสื้อกันฝน", ative a **"Experiência Sem Olhos"** — tudo com o seu leitor de tela ligado.

---

## English (US)

**Vitrine que Fala** (Speaking Storefront) grew from a bilingual MVP into the complete platform — still 100% open source, still engineered for an 8 GB RAM notebook. AI generates alt-text for **100% of the catalog in all six Shopee-market languages** (pt-BR, en-US, Bahasa Indonesia, Thai, Vietnamese, Traditional Chinese), semantic search works **across languages**, and the whole experience — search included — still works flawlessly with a screen reader and keyboard.

### What this release delivers

- **Six locales as data, not code** (ADR-0003) — catalog translations, receipts, UI, Web Speech and OCR arrive via seed + i18n JSON; adding a seventh locale changes no architecture.
- **gRPC between services** (ADR-0007) — one contract in `backend/api/proto/vitrine.proto` with committed stubs; the edge stays HTTP/JSON; the worker no longer talks to Postgres and writes through the service that owns the data.
- **Google Pub/Sub behind the broker interface** (ADR-0008) — `BROKER=redis|pubsub`; local runs use Redis Streams, the cloud can run Pub/Sub, and service code can't tell the difference.
- **Multilingual semantic search** (ADR-0009) — pgvector + HNSW in PostgreSQL, multilingual-e5-small embeddings (MIT, lazy); a Thai query finds the Portuguese product; the response reports `strategy: semantic|lexical`.
- **Per-market currencies** (ADR-0012) — Rp, ฿, ₫ and NT$ with each country's conventions (zero-decimals, suffix, separators).
- **Agent-free observability** (ADR-0010) — slog JSON logs and Prometheus `/metrics` on every service; an optional profile brings up Prometheus + Grafana.
- **An honest load-test harness** (ADR-0011) — k6 scripts for browse/search/checkout with thresholds; the results table ships empty: numbers only ever come from real runs.
- **Accessibility in six locales in CI** — axe scans on the home page and blind mode in all six languages; keyboard E2E switches to Thai; search announces result counts via `aria-live`.

### Try it

```bash
cp .env.example .env && docker compose up -d
cd web && npm install && npm run dev   # http://localhost:5173
```

Switch the language to Thai, search "เสื้อกันฝน", turn on the **"Experiência Sem Olhos" (No-Eyes Experience)** — ideally with your screen reader on.

---

## What's next

- First Cloud Run deployment with the real k6 numbers published in `docs/LOAD-TESTING.md`;
- OpenTelemetry tracing across the gRPC mesh and alerting rules (`docs/OBSERVABILITY.md` keeps the honest list).

Found a barrier or want to talk engineering? **[contact@eliezertavaresdeoliveira.com](mailto:contact@eliezertavaresdeoliveira.com)**
