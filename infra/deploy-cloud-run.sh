#!/usr/bin/env bash
# Deploys all v1.0 services to Cloud Run. Requires: gcloud auth + GCP_PROJECT/GCP_REGION.
# CLI-first and safe to re-run.
set -euo pipefail

: "${GCP_PROJECT:?export GCP_PROJECT=<project-id>}"
: "${GCP_REGION:?export GCP_REGION=<region>}"

PROJECT="$GCP_PROJECT"
REGION="$GCP_REGION"
REPO="${REGION}-docker.pkg.dev/${PROJECT}/vitrine"

echo "==> building and pushing images (stub pipeline for AI by default)"
for svc in api-gateway catalog order alttext-worker; do
  gcloud builds submit "./backend" \
    --project "$PROJECT" \
    --tag "${REPO}/${svc}:v1.0" \
    --substitutions _SERVICE="${svc}" 2>/dev/null \
    || { echo "gcloud builds failed for ${svc}; building locally"; 
         docker build --build-arg SERVICE="${svc}" -t "${REPO}/${svc}:v1.0" ./backend;
         docker push "${REPO}/${svc}:v1.0"; }
done

gcloud builds submit "./ai-vision-service" --project "$PROJECT" --tag "${REPO}/ai:v1.0" \
  || { docker build -t "${REPO}/ai:v1.0" ./ai-vision-service && docker push "${REPO}/ai:v1.0"; }

echo "==> deploying catalog"
gcloud run deploy catalog --image "${REPO}/catalog:v1.0" --region "$REGION" --project "$PROJECT" \
  --allow-unauthenticated --ingress internal-and-cloud-load-balancing \
  --set-env-vars "REDIS_ADDR=${REDIS_ADDR:-redis:6379}" \
  --min-instances 0 --cpu 1 --memory 256Mi

echo "==> deploying order"
gcloud run deploy order --image "${REPO}/order:v1.0" --region "$REGION" --project "$PROJECT" \
  --allow-unauthenticated --cpu 1 --memory 256Mi

echo "==> deploying alttext-worker (always-allocated CPU for a long-running worker)"
gcloud run deploy alttext-worker --image "${REPO}/alttext-worker:v1.0" --region "$REGION" --project "$PROJECT" \
  --no-allow-unauthenticated --cpu 1 --memory 512Mi --cpu-always

echo "==> deploying ai-vision-service"
gcloud run deploy ai --image "${REPO}/ai:v1.0" --region "$REGION" --project "$PROJECT" \
  --allow-unauthenticated --cpu 2 --memory 2Gi \
  --set-env-vars "PIPELINE=stub,TESSERACT_LANGS=por+eng"

echo "==> deploying gateway (public entry point)"
gcloud run deploy gateway --image "${REPO}/api-gateway:v1.0" --region "$REGION" --project "$PROJECT" \
  --allow-unauthenticated --cpu 1 --memory 256Mi \
  --set-env-vars "CATALOG_URL=$(gcloud run services describe catalog --region "$REGION" --project "$PROJECT" --format 'value(status.url)'),ORDER_URL=$(gcloud run services describe order --region "$REGION" --project "$PROJECT" --format 'value(status.url)'),AI_URL=$(gcloud run services describe ai --region "$REGION" --project "$PROJECT" --format 'value(status.url)')"

echo "==> done. Public URL:"
gcloud run services describe gateway --region "$REGION" --project "$PROJECT" --format 'value(status.url)'
