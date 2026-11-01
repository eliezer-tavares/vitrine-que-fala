#!/usr/bin/env bash
# Waits for the gateway, then runs smoke checks against the seeded catalog.
# Usage: ./infra/seed-wait.sh  (or `make seed`)
set -euo pipefail

GW_URL="${GW_URL:-http://localhost:8080}"
ATTEMPTS="${ATTEMPTS:-30}"

echo "==> waiting for gateway at ${GW_URL}"
for i in $(seq 1 "${ATTEMPTS}"); do
  if curl -fsS "${GW_URL}/healthz" >/dev/null 2>&1; then
    echo "==> gateway is up"
    break
  fi
  if [ "${i}" -eq "${ATTEMPTS}" ]; then
    echo "gateway did not come up in ${ATTEMPTS} attempts" >&2
    exit 1
  fi
  sleep 2
done

echo "==> healthz: $(curl -fsS "${GW_URL}/healthz")"
echo "==> products (pt-BR):"
curl -fsS -H "Accept-Language: pt-BR,pt;q=0.9" "${GW_URL}/api/v1/products" | head -c 600
echo
echo "==> products (en-US):"
curl -fsS -H "Accept-Language: en-US,en;q=0.9" "${GW_URL}/api/v1/products" | head -c 600
echo
echo "==> seed smoke checks passed"
