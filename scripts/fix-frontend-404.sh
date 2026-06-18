#!/usr/bin/env bash
# Frontend 404 (main-app-*.js) — aralash .next yoki build xatosi.
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
FRONTEND_DIR="${REMOTE_DIR}/frontend"
FRONTEND_PORT="${FRONTEND_PORT:-3002}"

echo "==> Frontend holatini tekshirish..."

if [[ ! -d "${FRONTEND_DIR}" ]]; then
  echo "Xato: ${FRONTEND_DIR} yo'q"
  exit 1
fi

if [[ -f "${FRONTEND_DIR}/.next/BUILD_ID" ]]; then
  echo "BUILD_ID: $(cat "${FRONTEND_DIR}/.next/BUILD_ID")"
  chunk_count="$(find "${FRONTEND_DIR}/.next/static/chunks" -name '*.js' 2>/dev/null | wc -l | tr -d ' ')"
  echo "Chunk fayllar: ${chunk_count}"
else
  echo "Ogohlantirish: .next/BUILD_ID yo'q — build kerak"
fi

if [[ -d "${FRONTEND_DIR}/src/app" ]]; then
  echo ""
  echo "==> Tozalab qayta build..."
  bash "${REMOTE_DIR}/scripts/build-frontend-server.sh"
  echo ""
  echo "Agar Cloudflare ishlatilsa — cache tozalang (Purge Everything)."
  exit 0
fi

echo ""
echo "src/app yo'q — serverda build qilib bo'lmaydi."
echo "Lokal mashinadan to'liq deploy qiling:"
echo "  bash scripts/deploy.sh"
exit 1
