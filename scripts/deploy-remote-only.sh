#!/usr/bin/env bash
# Serverda /opt/sahiy-stream ichida — Shopla ishlayveradi, alohida portda
set -euo pipefail

REMOTE_DIR="/opt/sahiy-stream"
FRONTEND_PORT="${FRONTEND_PORT:-3002}"
# IPv4 majburiy — IPv6 CORS ni buzadi
if [[ -f /opt/sahiy-stream/deploy-host.txt ]]; then
  HOST_IP=$(cat /opt/sahiy-stream/deploy-host.txt)
else
  HOST_IP="${HOST_IP:-$(hostname -I | tr " " "\n" | grep -E '^[0-9]+\.' | head -1)}"
fi
FRONTEND_PORT="${FRONTEND_PORT:-$(grep -E '^Frontend port:' /opt/sahiy-stream/for-deploy.txt 2>/dev/null | cut -d: -f2- | xargs)}"
FRONTEND_PORT="${FRONTEND_PORT:-3002}"

pkill -f "${REMOTE_DIR}/bin/" 2>/dev/null || true
for port in 50051 50052 50053 50054 8080 9084 9085 9086 "${FRONTEND_PORT}"; do
  fuser -k "${port}/tcp" 2>/dev/null || true
done
sleep 1

cd "${REMOTE_DIR}/infra/docker"
docker compose -f docker-compose.prod.yml down 2>/dev/null || true
docker rm -f sahiy-redis sahiy-postgres sahiy-minio sahiy-nats 2>/dev/null || true
docker compose -f docker-compose.prod.yml up -d
sleep 8

cat >"${REMOTE_DIR}/.env" <<ENVFILE
APP_ENV=production
LOG_LEVEL=info
DATABASE_URL=postgres://sahiy:sahiy_secret@127.0.0.1:15433/sahiy_stream?sslmode=disable
REDIS_URL=redis://127.0.0.1:16379/0
NATS_URL=nats://127.0.0.1:14222
AUTH_SERVICE_ADDR=localhost:50051
USER_SERVICE_ADDR=localhost:50052
STREAM_SERVICE_ADDR=localhost:50053
CHAT_SERVICE_ADDR=localhost:50054
CHAT_HTTP_ADDR=localhost:9085
GATEWAY_HTTP_ADDR=:8080
GATEWAY_CORS_ORIGINS=http://${HOST_IP}:${FRONTEND_PORT},http://${HOST_IP}:3000,http://${HOST_IP}
WHIP_BASE_URL=http://${HOST_IP}:8889
HLS_BASE_URL=http://${HOST_IP}:8090/hls
HLS_OUTPUT_DIR=${REMOTE_DIR}/data/hls
RTMP_INTERNAL_URL=rtmp://127.0.0.1:1935/live
RTSP_INTERNAL_URL=rtsp://127.0.0.1:8554
MEDIA_HTTP_ADDR=:9084
TRANSCODE_QUALITY=production
TRANSCODE_MODE=local
WORKER_MAX_JOBS=1
GATEWAY_RATE_LIMIT_RPM=500
ENVFILE

LOG="${REMOTE_DIR}/.logs"
mkdir -p "${LOG}" "${REMOTE_DIR}/data/hls"

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow "${FRONTEND_PORT}/tcp" comment "Sahiy Stream frontend" 2>/dev/null || true
  ufw allow 8080/tcp comment "Sahiy Stream API" 2>/dev/null || true
  ufw allow 8889/tcp comment "Sahiy WHIP" 2>/dev/null || true
  ufw allow 8090/tcp comment "Sahiy HLS" 2>/dev/null || true
  ufw allow 1935/tcp comment "Sahiy RTMP" 2>/dev/null || true
  ufw allow from 172.16.0.0/12 to any port 9084 proto tcp comment "Docker to media-orchestrator" 2>/dev/null || true
fi

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
npm install
nohup npx next start -p "${FRONTEND_PORT}" -H 0.0.0.0 >"${LOG}/frontend.log" 2>&1 &

echo "Sahiy Stream: http://${HOST_IP}:${FRONTEND_PORT}"
echo "Shopla: http://${HOST_IP}:3000"
