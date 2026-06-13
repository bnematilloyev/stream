#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${HOME}/go/bin:/opt/homebrew/bin:${PATH}"
LOG_DIR="${ROOT}/.logs"
mkdir -p "${LOG_DIR}"

bash "${ROOT}/scripts/stop-services.sh"

echo "Starting Sahiy Stream services..."
echo "Logs: ${LOG_DIR}/"

start() {
  local name="$1"
  local path="$2"
  (cd "${ROOT}" && go run "${path}") >"${LOG_DIR}/${name}.log" 2>&1 &
  echo "  ${name} pid=$! -> .logs/${name}.log"
}

start auth-service    "./services/auth-service/cmd/server"
sleep 2
start user-service    "./services/user-service/cmd/server"
sleep 2
start stream-service       "./services/stream-service/cmd/server"
sleep 2
start media-orchestrator   "./services/media-orchestrator/cmd/server"
sleep 2
start api-gateway          "./services/api-gateway/cmd/server"

echo "Waiting for API gateway..."
bash "${ROOT}/scripts/wait-for-api.sh"

echo ""
echo "Ready: http://localhost:8080/health"
echo "Test:  bash scripts/test-platform.sh"
echo "Stop:  make stop"
