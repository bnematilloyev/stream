#!/usr/bin/env bash
# Serverda servislarni qayta ishga tushirish (/opt/sahiy-stream)
set -euo pipefail

REMOTE_DIR="/opt/sahiy-stream"
FRONTEND_PORT="${FRONTEND_PORT:-3002}"
API_DOMAIN="${API_DOMAIN:-api.stream.vibrant.uz}"
FRONTEND_DOMAIN="${FRONTEND_DOMAIN:-stream.vibrant.uz}"

if [[ -f "${REMOTE_DIR}/deploy-host.txt" ]]; then
  HOST_IP=$(cat "${REMOTE_DIR}/deploy-host.txt")
else
  HOST_IP="$(hostname -I | tr ' ' '\n' | grep -E '^[0-9]+\.' | head -1)"
fi

if [[ -f "${REMOTE_DIR}/for-deploy.txt" ]]; then
  FRONTEND_PORT="$(grep -E '^Frontend port:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || true)"
  FRONTEND_PORT="${FRONTEND_PORT:-3002}"
  API_DOMAIN="$(grep -E '^API domen:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || true)"
  FRONTEND_DOMAIN="$(grep -E '^Frontend domen:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || true)"
fi

API_URL="https://${API_DOMAIN:-api.stream.vibrant.uz}"
FRONTEND_URL="https://${FRONTEND_DOMAIN:-stream.vibrant.uz}"

pkill -f "${REMOTE_DIR}/bin/" 2>/dev/null || true
for port in 50051 50052 50053 50054 8080 9084 9085 "${FRONTEND_PORT}"; do
  fuser -k "${port}/tcp" 2>/dev/null || true
done
sleep 1

cd "${REMOTE_DIR}/infra/docker"
docker compose -f docker-compose.prod.yml up -d
sleep 8

LOG="${REMOTE_DIR}/.logs"
mkdir -p "${LOG}" "${REMOTE_DIR}/data/hls"

start_svc() {
  nohup env $(grep -v '^#' "${REMOTE_DIR}/.env" | xargs) "${REMOTE_DIR}/bin/$1" >"${LOG}/$1.log" 2>&1 &
  echo "  $1 pid=$!"
}

start_svc auth-service; sleep 2
start_svc user-service; sleep 2
start_svc stream-service; sleep 2
start_svc chat-service; sleep 2
start_svc media-orchestrator; sleep 2
start_svc api-gateway; sleep 3

cd "${REMOTE_DIR}/frontend"
npm install --omit=dev 2>/dev/null || npm install
nohup npx next start -p "${FRONTEND_PORT}" -H 0.0.0.0 >"${LOG}/frontend.log" 2>&1 &

echo "Panel:    ${FRONTEND_URL}"
echo "API:      ${API_URL}/health"
