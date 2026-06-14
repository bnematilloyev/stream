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
  printf '%s' "$1" | sed -e 's/[\/&]/\\&/g'
}
ESCAPED="$(escape_sed "${SECRET}")"

for f in "${MTX}" "${RTMP}"; do
  [[ -f "${f}" ]] || continue
  sed -i "s|host.docker.internal:9084|172.17.0.1:9084|g" "${f}"
  sed -i "s|/hooks/publish?internal_secret=[^\"&]*|/hooks/publish?internal_secret=${ESCAPED}|g" "${f}"
  sed -i "s|/hooks/publish_done?internal_secret=[^\";&]*|/hooks/publish_done?internal_secret=${ESCAPED}|g" "${f}"
  sed -i "s|/hooks/publish\"|/hooks/publish?internal_secret=${ESCAPED}\"|g" "${f}"
  sed -i "s|/hooks/publish;|/hooks/publish?internal_secret=${ESCAPED};|g" "${f}"
  sed -i "s|/hooks/publish_done\"|/hooks/publish_done?internal_secret=${ESCAPED}\"|g" "${f}"
  sed -i "s|/hooks/publish_done;|/hooks/publish_done?internal_secret=${ESCAPED};|g" "${f}"
done

echo "sync-hook-secrets: OK"

if command -v docker >/dev/null 2>&1 && [[ -d "${REMOTE_DIR}/infra/docker" ]]; then
  cd "${REMOTE_DIR}/infra/docker"
  docker compose -f docker-compose.prod.yml restart mediamtx nginx-rtmp 2>/dev/null || true
fi
