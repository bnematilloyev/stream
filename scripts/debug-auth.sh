#!/usr/bin/env bash
# Auth / refresh diagnostika (serverda: bash /opt/sahiy-stream/scripts/debug-auth.sh)
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
ENV_FILE="${REMOTE_DIR}/.env"
LOG="${REMOTE_DIR}/.logs"
GATEWAY_PORT=8080

if [[ -f "${REMOTE_DIR}/for-deploy.txt" ]]; then
  GATEWAY_PORT="$(grep -E '^Gateway port:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || echo 8080)"
fi

echo "=== Sahiy auth diagnostika ==="
echo "Vaqt: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo ""

echo "=== Go servislar ==="
for svc in auth-service user-service api-gateway; do
  if pgrep -f "${REMOTE_DIR}/bin/${svc}" >/dev/null 2>&1; then
    echo "  OK  ${svc}"
  else
    echo "  OFF ${svc}  <-- refresh 500 bo'lishi mumkin"
  fi
done
echo ""

echo "=== Portlar ==="
for port in 50051 "${GATEWAY_PORT}"; do
  if ss -ltn 2>/dev/null | grep -q ":${port} "; then
    echo "  OK  :${port} tinglanmoqda"
  else
    echo "  OFF :${port}"
  fi
done
echo ""

echo "=== Gateway health ==="
curl -sf "http://127.0.0.1:${GATEWAY_PORT}/health" | head -c 400 || echo "  FAIL"
echo ""
echo ""

if [[ -f "${LOG}/auth-service.log" ]]; then
  echo "=== auth-service log (so'nggi 25 qator) ==="
  tail -25 "${LOG}/auth-service.log" || true
  echo ""
fi

if [[ -f "${LOG}/api-gateway.log" ]]; then
  echo "=== api-gateway log (refresh/grpc) ==="
  grep -iE 'refresh|auth-service|grpc|internal' "${LOG}/api-gateway.log" 2>/dev/null | tail -15 || true
  echo ""
fi

if [[ -f "${ENV_FILE}" ]]; then
  DATABASE_URL="$(grep -E '^DATABASE_URL=' "${ENV_FILE}" | cut -d= -f2- | sed 's/\r$//')"
  JWT_ACCESS="$(grep -E '^JWT_ACCESS_SECRET=' "${ENV_FILE}" | cut -d= -f2- | sed 's/\r$//')"
  JWT_REFRESH="$(grep -E '^JWT_REFRESH_SECRET=' "${ENV_FILE}" | cut -d= -f2- | sed 's/\r$//')"
  echo "=== .env ==="
  echo "  DATABASE_URL: ${DATABASE_URL%%@*}@***"
  echo "  JWT_ACCESS_SECRET uzunligi: ${#JWT_ACCESS}"
  echo "  JWT_REFRESH_SECRET uzunligi: ${#JWT_REFRESH}"
  echo ""
fi

echo "=== Postgres (sessions jadvali) ==="
if docker exec sahiy-postgres psql -U sahiy -d sahiy_stream -t -c \
  "SELECT to_regclass('public.sessions');" 2>/dev/null | grep -q sessions; then
  docker exec sahiy-postgres psql -U sahiy -d sahiy_stream -t -c \
    "SELECT COUNT(*) AS session_count FROM sessions;" 2>/dev/null || true
else
  echo "  sessions jadvali YO'Q — migratsiya kerak:"
  echo "    bash ${REMOTE_DIR}/scripts/prod-migrate.sh up"
fi
echo ""

if command -v migrate >/dev/null 2>&1 && [[ -d "${REMOTE_DIR}/migrations" ]] && [[ -n "${DATABASE_URL:-}" ]]; then
  echo "=== Migration version ==="
  migrate -path "${REMOTE_DIR}/migrations" -database "${DATABASE_URL}" version 2>&1 || true
  echo ""
fi

echo "=== Refresh endpoint (noto'g'ri token — 401 bo'lishi kerak) ==="
REFRESH_CODE="$(curl -s -o /tmp/sahiy-refresh-test.json -w '%{http_code}' \
  -X POST "http://127.0.0.1:${GATEWAY_PORT}/v1/auth/refresh" \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"invalid"}')"
echo "  HTTP ${REFRESH_CODE}"
cat /tmp/sahiy-refresh-test.json 2>/dev/null || true
echo ""
echo ""

if [[ "${REFRESH_CODE}" == "500" ]]; then
  echo "DIAGNOZ: Noto'g'ri token ham 500 qaytaryapti."
  echo "  → auth-service ishlamayapti YOKI DB ulanishi buzilgan."
  echo "  Tuzatish:"
  echo "    1) tail -50 ${LOG}/auth-service.log"
  echo "    2) bash ${REMOTE_DIR}/scripts/prod-migrate.sh up"
  echo "    3) bash ${REMOTE_DIR}/scripts/deploy-remote-only.sh"
elif [[ "${REFRESH_CODE}" == "401" ]]; then
  echo "DIAGNOZ: Auth zanjiri ishlayapti (401 kutilgan)."
  echo "  Agar brauzerda 500 bo'lsa — JWT secret o'zgargan yoki eski refresh token."
  echo "  Tuzatish: chiqib qayta kiring (logout/login)."
fi
