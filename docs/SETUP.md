# Publishing guide (PT) — como publicar este projeto no seu GitHub

Repositório de destino: <https://github.com/eliezer-tavares/vitrine-que-fala.git>

Autoria: **arquitetado por Eliezer Tavares de Oliveira** — <contact@eliezertavaresdeoliveira.com>

## 1. Enviar a v2.0 (a v1.0 já está publicada)

No diretório raiz do projeto:

```bash
git add .
git commit -m "feat: v2.0 — seis idiomas, gRPC, Pub/Sub, pgvector, k6, observabilidade"
git push origin main
```

## 2. Rodar o CI

O workflow `.github/workflows/ci.yml` roda automaticamente no push: lint + testes Go, lint + testes Python (agora incluindo `/embed`), testes unitários da web com axe (em seis idiomas), E2E Playwright e build das imagens Docker. Acompanhe com `gh run list --limit 3`.

## 3. Criar a Release v2.0

```bash
git tag -a v2.0 -m "v2.0 — seis idiomas, gRPC, Pub/Sub, pgvector, k6"
git push origin v2.0

# Página da Release com a GitHub CLI:
gh release create v2.0 --title "v2.0 — seis idiomas, gRPC, Pub/Sub, pgvector" --notes-file notas-v2.md
```

O `deploy.yml` dispara na tag; enquanto `GCP_PROJECT` não existir, o job é pulado e o CI fica verde.

## 4. (Opcional) Configurar o deploy no Google Cloud Run

Siga o passo a passo CLI-friendly em `infra/cloud-run.md`. Resumo dos segredos/variáveis do repositório (Settings > Secrets and variables > Actions):

- Secret `GCP_SA_KEY`: JSON da conta de serviço com papéis `roles/run.admin`, `roles/iam.serviceAccountUser` e `roles/artifactregistry.writer`.
- Variable `GCP_PROJECT`: id do projeto.
- Variable `GCP_REGION`: ex.: `southamerica-east1`.
- Variable `REDIS_ADDR` e Secret `CLOUD_DATABASE_URL`: endereços do Memorystore/Cloud SQL (ou de provedores gratuitos compatíveis).

Enquanto `GCP_PROJECT` não estiver definida, o job de deploy é pulado — nada quebra.

Depois do primeiro deploy, rode os testes de carga e preencha a tabela de `docs/LOAD-TESTING.md` (ADR-0011: números reais ou nada).

## 5. Rodar localmente (notebook 8 GB)

1. Windows: copie `.wslconfig.example` para `C:\Users\<voce>\.wslconfig` e rode `wsl --shutdown` antes de subir o Docker.
2. `cp .env.example .env`
3. `docker compose up -d` (a imagem do Postgres agora é pgvector; os seeds 000–003 criam tudo, incluindo os índices HNSW)
4. `cd web && npm install && npm run dev` — abra <http://localhost:5173> com NVDA, troque para tailandês e teste a busca e a "Experiência Sem Olhos".
5. Opcional: `make obs-up` para Prometheus (:9090) e Grafana (:3000).

## 6. Checklist antes de compartilhar com o recrutador

- [ ] CI verde no GitHub Actions (seis idiomas no scan).
- [ ] Release `v2.0` criada com notas.
- [ ] Deploy no ar + números reais de k6 em `docs/LOAD-TESTING.md` (ou o README apontando para o vídeo/print do modo Sem Olhos).
- [ ] LinkedIn atualizado com o link do repositório.
- [ ] Teste final com NVDA na página publicada, incluindo tailandês e chinês tradicional.
