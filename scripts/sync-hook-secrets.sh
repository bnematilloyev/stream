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
  sed -i "s|host.docker.internal:9084|172.17.0.1:9084|g" "${f}"
  # MediaMTX: .../hooks/publish -d  yoki .../hooks/publish?internal_secret=OLD -d
  sed -i "s|9084/hooks/publish?internal_secret=[^ \"&]*|9084/hooks/publish?internal_secret=${ESCAPED}|g" "${f}"
  sed -i "s|9084/hooks/publish_done?internal_secret=[^ \"&]*|9084/hooks/publish_done?internal_secret=${ESCAPED}|g" "${f}"
  sed -i "s|9084/hooks/publish -d|9084/hooks/publish?internal_secret=${ESCAPED} -d|g" "${f}"
  sed -i "s|9084/hooks/publish_done -d|9084/hooks/publish_done?internal_secret=${ESCAPED} -d|g" "${f}"
  # nginx-rtmp: .../hooks/publish; yoki .../hooks/publish?internal_secret=...
  sed -i "s|9084/hooks/publish;|9084/hooks/publish?internal_secret=${ESCAPED};|g" "${f}"
  sed -i "s|9084/hooks/publish_done;|9084/hooks/publish_done?internal_secret=${ESCAPED};|g" "${f}"
}

patch_file "${MTX}"
patch_file "${RTMP}"

echo "sync-hook-secrets: OK"
grep -E 'runOn|hooks/publish' "${MTX}" 2>/dev/null | head -3 || true

if command -v docker >/dev/null 2>&1 && [[ -d "${REMOTE_DIR}/infra/docker" ]]; then
  cd "${REMOTE_DIR}/infra/docker"
  docker compose -f docker-compose.prod.yml restart mediamtx nginx-rtmp 2>/dev/null || true
fi
