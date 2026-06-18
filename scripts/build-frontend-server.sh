#!/usr/bin/env bash
# Serverda frontend build (src/ mavjud bo'lganda).
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
FRONTEND_DIR="${REMOTE_DIR}/frontend"
FRONTEND_PORT="${FRONTEND_PORT:-3002}"

if [[ -f "${REMOTE_DIR}/for-deploy.txt" ]]; then
  _d="$(grep -E '^Frontend domen:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || true)"
  if [[ -n "${_d}" ]]; then
    export NEXT_PUBLIC_API_URL="https://${_d}"
    export NEXT_PUBLIC_WHIP_BASE_URL="https://${_d}"
    export NEXT_PUBLIC_HLS_BASE_URL="https://${_d}/hls"
  fi
fi

if [[ ! -d "${FRONTEND_DIR}/src/app" ]]; then
  echo "Xato: ${FRONTEND_DIR}/src/app yo'q."
  echo "Serverda faqat .next yuborilgan bo'lishi mumkin."
  echo "Lokalda: bash scripts/deploy.sh"
  exit 1
fi

if ! command -v node >/dev/null 2>&1; then
  echo "Node.js yo'q. O'rnatish: curl -fsSL https://deb.nodesource.com/setup_20.x | bash - && apt-get install -y nodejs"
  exit 1
fi

cd "${FRONTEND_DIR}"
echo "==> npm ci..."
npm ci
echo "==> next build..."
npm run build
echo "==> Tayyor: ${FRONTEND_DIR}/.next"

if [[ "${RESTART_FRONTEND:-1}" == "1" ]]; then
  LOG="${REMOTE_DIR}/.logs"
  mkdir -p "${LOG}"
  pkill -f "next start -p ${FRONTEND_PORT}" 2>/dev/null || true
  fuser -k "${FRONTEND_PORT}/tcp" 2>/dev/null || true
  sleep 1
  nohup ./node_modules/.bin/next start -p "${FRONTEND_PORT}" -H 0.0.0.0 >"${LOG}/frontend.log" 2>&1 &
  echo "frontend qayta ishga tushirildi (port ${FRONTEND_PORT})"
fi
