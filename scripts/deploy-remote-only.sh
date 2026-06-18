#!/usr/bin/env bash
# Serverda servislarni qayta ishga tushirish (/opt/sahiy-stream)
set -euo pipefail

REMOTE_DIR="/opt/sahiy-stream"
FRONTEND_PORT="${FRONTEND_PORT:-3002}"
GATEWAY_PORT="${GATEWAY_PORT:-8080}"
HLS_PORT="${HLS_PORT:-8090}"
FRONTEND_DOMAIN="${FRONTEND_DOMAIN:-stream.vibrant.uz}"
API_DOMAIN="${API_DOMAIN:-${FRONTEND_DOMAIN}}"

if [[ -f "${REMOTE_DIR}/deploy-host.txt" ]]; then
  HOST_IP=$(cat "${REMOTE_DIR}/deploy-host.txt")
else
  HOST_IP="$(hostname -I | tr ' ' '\n' | grep -E '^[0-9]+\.' | head -1)"
fi

if [[ -f "${REMOTE_DIR}/for-deploy.txt" ]]; then
  FRONTEND_PORT="$(grep -E '^Frontend port:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || true)"
  FRONTEND_PORT="${FRONTEND_PORT:-3002}"
  GATEWAY_PORT="$(grep -E '^Gateway port:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || true)"
  GATEWAY_PORT="${GATEWAY_PORT:-8080}"
  HLS_PORT="$(grep -E '^HLS port:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || true)"
  HLS_PORT="${HLS_PORT:-8090}"
  FRONTEND_DOMAIN="$(grep -E '^Frontend domen:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || true)"
  FRONTEND_DOMAIN="${FRONTEND_DOMAIN:-stream.vibrant.uz}"
  API_DOMAIN="$(grep -E '^API domen:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || true)"
  if [[ -z "${API_DOMAIN}" || "${API_DOMAIN}" == "${FRONTEND_DOMAIN}" ]]; then
    API_DOMAIN="${FRONTEND_DOMAIN}"
  fi
fi

PUBLIC_URL="https://${FRONTEND_DOMAIN}"
FRONTEND_URL="${PUBLIC_URL}"
API_URL="${PUBLIC_URL}"

ensure_env() {
  local env_file="${REMOTE_DIR}/.env"
  if [[ -f "${REMOTE_DIR}/.gpu-transcode" ]] || grep -q '^TRANSCODE_MODE=queue' "${env_file}" 2>/dev/null; then
    echo "GPU queue mode — transcode sozlamalari saqlanadi (.env transcode qismi o'zgartirilmaydi)"
    touch "${REMOTE_DIR}/.gpu-transcode"
    return 0
  fi
  local jwt_access jwt_refresh media_hook playback_signing service_token market_webhook_url market_webhook_secret
  if [[ -f "${env_file}" ]]; then
    jwt_access="$(grep -E '^JWT_ACCESS_SECRET=' "${env_file}" | cut -d= -f2- || true)"
    jwt_refresh="$(grep -E '^JWT_REFRESH_SECRET=' "${env_file}" | cut -d= -f2- || true)"
    media_hook="$(grep -E '^MEDIA_HOOK_SECRET=' "${env_file}" | cut -d= -f2- || true)"
    playback_signing="$(grep -E '^PLAYBACK_SIGNING_SECRET=' "${env_file}" | cut -d= -f2- || true)"
    service_token="$(grep -E '^SERVICE_TOKEN=' "${env_file}" | cut -d= -f2- || true)"
    market_webhook_url="$(grep -E '^MARKET_WEBHOOK_URL=' "${env_file}" | cut -d= -f2- || true)"
    market_webhook_secret="$(grep -E '^MARKET_WEBHOOK_SECRET=' "${env_file}" | cut -d= -f2- || true)"
  fi
  jwt_access="${jwt_access:-$(openssl rand -hex 32)}"
  jwt_refresh="${jwt_refresh:-$(openssl rand -hex 32)}"
  media_hook="${media_hook:-$(openssl rand -hex 32)}"
  playback_signing="${playback_signing:-$(openssl rand -hex 32)}"
  service_token="${service_token:-$(openssl rand -hex 32)}"
  market_webhook_secret="${market_webhook_secret:-${service_token}}"

  cat >"${env_file}" <<ENVFILE
APP_ENV=production
LOG_LEVEL=info
DATABASE_URL=postgres://sahiy:sahiy_secret@127.0.0.1:15433/sahiy_stream?sslmode=disable
REDIS_URL=redis://127.0.0.1:16379/0
NATS_URL=nats://127.0.0.1:14222
JWT_ACCESS_SECRET=${jwt_access}
JWT_REFRESH_SECRET=${jwt_refresh}
JWT_ACCESS_TTL=24h
JWT_REFRESH_TTL=720h
MEDIA_HOOK_SECRET=${media_hook}
PLAYBACK_SIGNING_SECRET=${playback_signing}
PLAYBACK_BASE_URL=${FRONTEND_URL}
HLS_STORAGE_BACKEND=local
FFMPEG_VIDEO_ENCODER=libx264
TRANSCODE_QUALITY=production
TRANSCODE_MODE=passthrough
WORKER_MAX_JOBS=1
AUTH_SERVICE_ADDR=localhost:50051
USER_SERVICE_ADDR=localhost:50052
STREAM_SERVICE_ADDR=localhost:50053
CHAT_SERVICE_ADDR=localhost:50054
CHAT_HTTP_ADDR=localhost:9085
GATEWAY_HTTP_ADDR=:${GATEWAY_PORT}
GATEWAY_CORS_ORIGINS=${FRONTEND_URL},https://${FRONTEND_DOMAIN}
WHIP_BASE_URL=${FRONTEND_URL}
HLS_BASE_URL=${FRONTEND_URL}/hls
HLS_OUTPUT_DIR=${REMOTE_DIR}/data/hls
RTMP_BASE_URL=rtmp://${HOST_IP}:1935/live
RTMP_INTERNAL_URL=rtmp://127.0.0.1:1935/live
RTSP_INTERNAL_URL=rtsp://127.0.0.1:8554
MEDIA_HTTP_ADDR=:9084
GATEWAY_RATE_LIMIT_RPM=500
SERVICE_TOKEN=${service_token}
MARKET_WEBHOOK_URL=${market_webhook_url}
MARKET_WEBHOOK_SECRET=${market_webhook_secret}
ENVFILE
}

ensure_env

if command -v ufw >/dev/null 2>&1; then
  ufw allow 1935/tcp comment "RTMP ingest" 2>/dev/null || true
  ufw reload 2>/dev/null || true
fi

pkill -f "${REMOTE_DIR}/bin/" 2>/dev/null || true
# VPS da transcode ffmpeg bo'lmasligi kerak (GPU worker da)
pkill -9 -f "ffmpeg -hide_banner" 2>/dev/null || true
for port in 50051 50052 50053 50054 "${GATEWAY_PORT}" 9084 9085 "${FRONTEND_PORT}"; do
  fuser -k "${port}/tcp" 2>/dev/null || true
done
sleep 1

if ! command -v ffmpeg >/dev/null 2>&1; then
  echo "==> FFmpeg o'rnatish..."
  export DEBIAN_FRONTEND=noninteractive
  apt-get update -qq
  apt-get install -y -qq ffmpeg
fi

cd "${REMOTE_DIR}/infra/docker"
docker compose -f docker-compose.prod.yml up -d
sleep 8

if [[ -x "${REMOTE_DIR}/scripts/prod-migrate.sh" ]]; then
  bash "${REMOTE_DIR}/scripts/prod-migrate.sh" up || true
elif [[ -d "${REMOTE_DIR}/migrations" ]] && command -v migrate >/dev/null 2>&1; then
  DATABASE_URL="$(grep -E '^DATABASE_URL=' "${REMOTE_DIR}/.env" | cut -d= -f2- | sed 's/\r$//')"
  migrate -path "${REMOTE_DIR}/migrations" -database "${DATABASE_URL}" up 2>/dev/null || true
fi

if [[ -x "${REMOTE_DIR}/scripts/sync-hook-secrets.sh" ]]; then
  bash "${REMOTE_DIR}/scripts/sync-hook-secrets.sh"
elif [[ -f "${REMOTE_DIR}/scripts/sync-hook-secrets.sh" ]]; then
  bash "${REMOTE_DIR}/scripts/sync-hook-secrets.sh"
fi

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
start_svc api-gateway
for i in $(seq 1 20); do
  if curl -sf "http://127.0.0.1:${GATEWAY_PORT}/health" >/dev/null 2>&1; then
    echo "  api-gateway tayyor (port ${GATEWAY_PORT})"
    break
  fi
  sleep 1
done
if ! curl -sf "http://127.0.0.1:${GATEWAY_PORT}/health" >/dev/null 2>&1; then
  echo "  api-gateway ishga tushmadi (port ${GATEWAY_PORT}) — log:"
  tail -20 "${LOG}/api-gateway.log" 2>&1 || true
  tail -10 "${LOG}/chat-service.log" 2>&1 || true
  exit 1
fi

cd "${REMOTE_DIR}/frontend"
if [[ ! -d .next ]]; then
  if [[ -d src/app ]]; then
    echo "==> .next yo'q — serverda build..."
    bash "${REMOTE_DIR}/scripts/build-frontend-server.sh"
  else
    echo "Xato: frontend/.next va frontend/src/app yo'q."
    echo "Lokalda deploy qiling: bash scripts/deploy.sh"
    exit 1
  fi
elif [[ "${REBUILD_FRONTEND:-}" == "1" && -d src/app ]]; then
  echo "==> REBUILD_FRONTEND=1 — qayta build..."
  bash "${REMOTE_DIR}/scripts/build-frontend-server.sh"
fi
npm ci --omit=dev 2>/dev/null || npm ci
pkill -f "next start -p ${FRONTEND_PORT}" 2>/dev/null || true
fuser -k "${FRONTEND_PORT}/tcp" 2>/dev/null || true
sleep 1
nohup ./node_modules/.bin/next start -p "${FRONTEND_PORT}" -H 0.0.0.0 >"${LOG}/frontend.log" 2>&1 &
echo "  frontend pid=$! port=${FRONTEND_PORT}"
for i in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:${FRONTEND_PORT}/" >/dev/null 2>&1; then
    echo "  frontend tayyor"
    break
  fi
  sleep 1
done
if ! curl -sf "http://127.0.0.1:${FRONTEND_PORT}/" >/dev/null 2>&1; then
  echo "  frontend ishga tushmadi — log:"
  tail -15 "${LOG}/frontend.log" || true
  exit 1
fi

echo "Panel:    ${FRONTEND_URL}"
echo "API:      ${FRONTEND_URL}/health"
