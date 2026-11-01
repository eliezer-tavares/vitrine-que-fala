# Deploying to Google Cloud Run (CLI-first, screen-reader friendly)

All commands use `gcloud` and `git` — no console clicking required.

## One-time setup

```bash
# 1. Variables you will reuse
export GCP_PROJECT=<your-project-id>
export GCP_REGION=southamerica-east1

# 2. Enable APIs
gcloud services enable run.googleapis.com sqladmin.googleapis.com \
  artifactregistry.googleapis.com secretmanager.googleapis.com \
  --project "$GCP_PROJECT"

# 3. Create an Artifact Registry repo
gcloud artifacts repositories create vitrine \
  --repository-format=docker --location="$GCP_REGION" --project "$GCP_PROJECT"

# 4. Cloud SQL instance (PostgreSQL 16) + database
gcloud sql instances create vitrine-db \
  --database-version=POSTGRES_16 --tier=db-f1-micro \
  --region="$GCP_REGION" --storage-auto-increase --project "$GCP_PROJECT"
gcloud sql databases create vitrine --instance vitrine-db --project "$GCP_PROJECT"
gcloud sql users create vitrine --instance vitrine-db \
  --password="<strong-password>" --project "$GCP_PROJECT"

# 5. Secrets
printf '%s' "<strong-password>" | gcloud secrets create db-password \
  --data-file=- --project "$GCP_PROJECT"
printf '%s' "<long-random-string>" | gcloud secrets create jwt-secret \
  --data-file=- --project "$GCP_PROJECT"
```

## Deploy the services

`infra/deploy-cloud-run.sh` automates the six services. Run after `gcloud` auth:

```bash
bash infra/deploy-cloud-run.sh
```

Each service deploys with minimal CPU/RAM (Cloud Run scale-to-zero keeps free-tier costs near zero). The AI service uses `PIPELINE=stub` by default; switch to `florence` once you validate memory headroom on the chosen instance.

## Connect GitHub Actions

1. Create a service account with `roles/run.admin`, `roles/iam.serviceAccountUser`, `roles/artifactregistry.writer` and download its JSON key.
2. In the repository: Settings > Secrets and variables > Actions:
   - Secret `GCP_SA_KEY` = the JSON key.
   - Variable `GCP_PROJECT` = your project id.
   - Variable `GCP_REGION` = e.g. `southamerica-east1`.
3. Push a `v1.0`-style tag — `.github/workflows/deploy.yml` builds and deploys.

Until `GCP_PROJECT` exists as a repo variable, the deploy job skips itself, so CI stays green.

## Security notes (v1.0 honest list)

- The gateway trusts `X-Client-Locale` from clients but re-negotiates `Accept-Language` itself; internal services must not be exposed publicly (Cloud Run ingress: internal + gateway only in production).
- The alt-text endpoint is public in v1.0 for recruiter demo friction; v2.0 adds rate limiting/auth.
- Rotate `JWT_SECRET` via Secret Manager; never commit real values (`.env` is git-ignored).
