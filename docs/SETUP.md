# Publishing guide (PT) — como publicar este projeto no seu GitHub

Repositório de destino: <https://github.com/eliezer-tavares/vitrine-que-fala.git>

Autoria: **arquitetado por Eliezer Tavares de Oliveira** — <contact@eliezertavaresdeoliveira.com>

## 1. Enviar o código pela primeira vez

No diretório raiz deste projeto (a pasta que contém este `docs/`):

```bash
git init -b main
git add .
git commit -m "feat: v1.0 — MVP acessível bilíngue (pt-BR/en-US)"
git remote add origin https://github.com/eliezer-tavares/vitrine-que-fala.git
git push -u origin main
```

Dica: se o seu Git não estiver configurado ainda:

```bash
git config --global user.name "Eliezer Tavares de Oliveira"
git config --global user.email "contact@eliezertavaresdeoliveira.com"
```

## 2. Rodar o CI

O workflow `.github/workflows/ci.yml` roda automaticamente no push: lint + testes Go, lint + testes Python, testes unitários da web (com axe), E2E Playwright e build das imagens Docker. Acompanhe em **Actions**, no site do repositório.

Observação: o primeiro `go test ./...` no CI baixa as dependências automaticamente e gera o `go.sum`. Se preferir versionar o `go.sum` (recomendado), rode localmente:

```bash
cd backend && go mod tidy && git add go.sum && git commit -m "chore: add go.sum" && git push
```

## 3. Criar a Release v1.0

```bash
git tag -a v1.0 -m "v1.0 — MVP acessível bilíngue / bilingual accessible MVP"
git push origin v1.0

# Com a GitHub CLI (opcional, cria a página da Release):
gh release create v1.0 --title "v1.0 — MVP acessível bilíngue" --notes "Primeira release pública: 4 serviços em Go + serviço de IA open source (Florence-2 + Opus-MT + Tesseract), vitrine React WCAG 2.2 AA em pt-BR/en-US, modo Experiência Sem Olhos, pipeline de acessibilidade no CI (axe nos dois idiomas, E2E por teclado), deploy no Cloud Run."
```

O `workflow_dispatch`/tag do `deploy.yml` só executa o deploy quando as variáveis GCP existirem (passo 4); antes disso o CI continua verde.

## 4. (Opcional) Configurar o deploy no Google Cloud Run

Siga o passo a passo CLI-friendly em `infra/cloud-run.md`. Resumo dos segredos/variáveis do repositório (Settings > Secrets and variables > Actions):

- Secret `GCP_SA_KEY`: JSON da conta de serviço com papéis `roles/run.admin`, `roles/iam.serviceAccountUser` e `roles/artifactregistry.writer`.
- Variable `GCP_PROJECT`: id do projeto.
- Variable `GCP_REGION`: ex.: `southamerica-east1`.

Enquanto `GCP_PROJECT` não estiver definida, o job de deploy é pulado — nada quebra.

## 5. Rodar localmente (notebook 8 GB)

1. Windows: copie `.wslconfig.example` para `C:\Users\<voce>\.wslconfig` e rode `wsl --shutdown` antes de subir o Docker.
2. `cp .env.example .env`
3. `docker compose up -d`
4. `cd web && npm install && npm run dev` — abra <http://localhost:5173> com NVDA e teste a "Experiência Sem Olhos".

## 6. Checklist antes de compartilhar com o recrutador

- [ ] CI verde no GitHub Actions.
- [ ] Release `v1.0` criada com notas.
- [ ] Deploy no ar (ou o README apontando para o vídeo/print do modo Sem Olhos — prints com alt-text impecável).
- [ ] LinkedIn atualizado com o link do repositório.
- [ ] Teste final com NVDA na página publicada.
