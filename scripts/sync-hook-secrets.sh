#!/usr/bin/env bash
# MediaMTX va nginx-rtmp hook URLlariga MEDIA_HOOK_SECRET qo'shish.
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
ENV_FILE="${REMOTE_DIR}/.env"
MTX="${REMOTE_DIR}/infra/docker/mediamtx/mediamtx.yml"
RTMP="${REMOTE_DIR}/infra/docker/nginx-rtmp/nginx.conf"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "sync-hook-secrets: .env topilmadi (${ENV_FILE})"
  exit 1
fi

SECRET="$(grep -E '^MEDIA_HOOK_SECRET=' "${ENV_FILE}" | cut -d= -f2- | xargs || true)"
if [[ -z "${SECRET}" ]]; then
  echo "sync-hook-secrets: MEDIA_HOOK_SECRET yo'q"
  exit 1
fi

escape_sed() {
  printf '%s' "$1" | sed -e 's/[\/&|]/\\&/g'
}
ESCAPED="$(escape_sed "${SECRET}")"

patch_file() {
  local f="$1"
  [[ -f "${f}" ]] || return 0
  sed -i "s|__MEDIA_HOOK_SECRET__|${ESCAPED}|g" "${f}"
  sed -i "s|host.docker.internal:9084|172.17.0.1:9084|g" "${f}"
}

patch_file "${MTX}"
patch_file "${RTMP}"

echo "sync-hook-secrets: OK"
grep -E 'runOn|on_publish' "${MTX}" 2>/dev/null | head -2 || true
grep -E 'on_publish' "${RTMP}" 2>/dev/null | head -2 || true

if command -v docker >/dev/null 2>&1 && [[ -d "${REMOTE_DIR}/infra/docker" ]]; then
  cd "${REMOTE_DIR}/infra/docker"
  docker compose -f docker-compose.prod.yml restart mediamtx nginx-rtmp 2>/dev/null || true
fi
