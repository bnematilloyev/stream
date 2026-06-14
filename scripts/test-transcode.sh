#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKER_ADDR="${WORKER_HTTP_ADDR:-:9086}"

echo "Transcode worker smoke test (${WORKER_ADDR})..."

curl -sf "http://localhost${WORKER_ADDR}/health" | grep -q '"status":"ok"' || {
  echo "FAIL: transcode-worker /health"
  exit 1
}

if curl -sf "http://localhost${WORKER_ADDR}/ready" | grep -q '"status":"ready"'; then
  echo "OK: transcode-worker ready (NATS connected)"
else
  echo "WARN: transcode-worker /ready not ready — is NATS running?"
fi

if [[ "${TRANSCODE_MODE:-local}" == "queue" ]]; then
  echo "OK: TRANSCODE_MODE=queue (orchestrator dispatches via NATS)"
else
  echo "INFO: TRANSCODE_MODE=${TRANSCODE_MODE:-local} (worker idle until queue mode)"
fi

echo "OK: transcode-worker healthy"
