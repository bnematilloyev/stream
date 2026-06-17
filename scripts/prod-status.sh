#!/usr/bin/env bash
# Production holatini tekshirish.
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
ENV_FILE="${REMOTE_DIR}/.env"
GATEWAY_PORT=8080
FRONTEND_PORT=3002

if [[ -f "${REMOTE_DIR}/for-deploy.txt" ]]; then
  GATEWAY_PORT="$(grep -E '^Gateway port:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || echo 8080)"
  FRONTEND_PORT="$(grep -E '^Frontend port:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || echo 3002)"
fi

echo "==> Docker"
cd "${REMOTE_DIR}/infra/docker" 2>/dev/null && docker compose -f docker-compose.prod.yml ps || echo "docker compose topilmadi"

echo ""
echo "==> Go servislar (process)"
for svc in auth-service user-service stream-service chat-service media-orchestrator api-gateway; do
  if pgrep -f "${REMOTE_DIR}/bin/${svc}" >/dev/null 2>&1; then
    echo "  OK  ${svc}"
  else
    echo "  OFF ${svc}"
  fi
done

echo ""
echo "==> Frontend"
if pgrep -f "next start -p ${FRONTEND_PORT}" >/dev/null 2>&1; then
  echo "  OK  frontend :${FRONTEND_PORT}"
else
  echo "  OFF frontend :${FRONTEND_PORT}"
fi

echo ""
echo "==> Health"
curl -sf "http://127.0.0.1:${GATEWAY_PORT}/health" && echo "  API gateway OK" || echo "  API gateway FAIL"
curl -sf "http://127.0.0.1:${FRONTEND_PORT}/" >/dev/null && echo "  Frontend OK" || echo "  Frontend FAIL"

if [[ -f "${ENV_FILE}" ]] && command -v migrate >/dev/null 2>&1 && [[ -d "${REMOTE_DIR}/migrations" ]]; then
  echo ""
  echo "==> DB migration version"
  DATABASE_URL="$(grep -E '^DATABASE_URL=' "${ENV_FILE}" | cut -d= -f2- | sed 's/\r$//')"
  migrate -path "${REMOTE_DIR}/migrations" -database "${DATABASE_URL}" version 2>/dev/null || true
fi
