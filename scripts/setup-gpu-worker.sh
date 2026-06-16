#!/usr/bin/env bash
# VPS ni GPU transcode-worker (RunPod) uchun sozlash.
# Lokalda: bash scripts/setup-gpu-worker.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_FILE="${ROOT}/for-deploy.txt"

if [[ ! -f "${DEPLOY_FILE}" ]]; then
  echo "for-deploy.txt topilmadi."
  echo "  cp for-deploy.txt.example for-deploy.txt"
  echo "  IP, user, parolni to'ldiring"
  exit 1
fi

read_deploy() {
  grep -E "^${1}:" "${DEPLOY_FILE}" | cut -d: -f2- | xargs || true
}

HOST=$(read_deploy "IP-manzil")
USER=$(read_deploy "Foydalanuvchi nomi")
PASS=$(read_deploy "Parol")
GATEWAY_PORT=$(read_deploy "Gateway port")
GATEWAY_PORT="${GATEWAY_PORT:-18080}"

if [[ -z "${HOST}" || -z "${USER}" || -z "${PASS}" ]]; then
  echo "for-deploy.txt da IP, user yoki parol yo'q"
  exit 1
fi

REMOTE_DIR="/opt/sahiy-stream"
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=20"

ssh_cmd() {
  sshpass -p "${PASS}" ssh ${SSH_OPTS} "${USER}@${HOST}" "$@"
}

scp_cmd() {
  sshpass -p "${PASS}" scp ${SSH_OPTS} "$@"
}

if ! command -v sshpass >/dev/null 2>&1; then
  echo "sshpass kerak: brew install sshpass"
  exit 1
fi

echo "==> transcode-worker build (linux/amd64)..."
export GOOS=linux GOARCH=amd64
mkdir -p "${ROOT}/bin"
(cd "${ROOT}/services/transcode-worker" && go build -o "${ROOT}/bin/transcode-worker" ./cmd/server)

echo "==> VPS ga yuklash..."
ssh_cmd "mkdir -p ${REMOTE_DIR}/bin ${REMOTE_DIR}/infra/docker ${REMOTE_DIR}/scripts"
scp_cmd "${ROOT}/bin/transcode-worker" "${USER}@${HOST}:${REMOTE_DIR}/bin/transcode-worker"
scp_cmd "${ROOT}/infra/docker/docker-compose.gpu-worker.yml" "${USER}@${HOST}:${REMOTE_DIR}/infra/docker/"
scp_cmd "${ROOT}/scripts/init-minio.sh" "${USER}@${HOST}:${REMOTE_DIR}/scripts/"

echo "==> VPS sozlash (queue mode + NATS/MinIO)..."
ssh_cmd bash -s <<REMOTE
set -euo pipefail
REMOTE_DIR="${REMOTE_DIR}"
HOST_IP="${HOST}"
GATEWAY_PORT="${GATEWAY_PORT}"

ENV_FILE="\${REMOTE_DIR}/.env"
touch "\${ENV_FILE}"

patch_env() {
  local key="\$1" val="\$2"
  if grep -q "^\${key}=" "\${ENV_FILE}"; then
    sed -i "s|^\${key}=.*|\${key}=\${val}|" "\${ENV_FILE}"
  else
    echo "\${key}=\${val}" >>"\${ENV_FILE}"
  fi
}

patch_env TRANSCODE_MODE queue
patch_env HLS_STORAGE_BACKEND s3
patch_env FFMPEG_VIDEO_ENCODER h264_nvenc
patch_env WORKER_MAX_JOBS 4
patch_env MINIO_ENDPOINT 127.0.0.1:19000
patch_env MINIO_ACCESS_KEY sahiy_minio
patch_env MINIO_SECRET_KEY sahiy_minio_secret
patch_env MINIO_BUCKET sahiy-media
patch_env MINIO_USE_SSL false
patch_env RTMP_INTERNAL_URL "rtmp://\${HOST_IP}:1935/live"
patch_env RTSP_INTERNAL_URL "rtsp://\${HOST_IP}:8554"
patch_env NATS_URL nats://127.0.0.1:14222

chmod +x "\${REMOTE_DIR}/bin/transcode-worker" "\${REMOTE_DIR}/scripts/init-minio.sh"

cd "\${REMOTE_DIR}/infra/docker"
sed -i 's|"127.0.0.1:14222:4222"|"14222:4222"|g' docker-compose.prod.yml
docker compose -f docker-compose.prod.yml -f docker-compose.gpu-worker.yml down nats 2>/dev/null || true
docker compose -f docker-compose.prod.yml -f docker-compose.gpu-worker.yml up -d
sleep 10

MINIO_ENDPOINT=127.0.0.1:19000 bash "\${REMOTE_DIR}/scripts/init-minio.sh" || echo "WARN: init-minio (bucket orchestrator yaratadi)"

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow 14222/tcp comment "NATS GPU worker" 2>/dev/null || true
  ufw allow 19000/tcp comment "MinIO GPU worker" 2>/dev/null || true
  ufw allow 1935/tcp comment "RTMP ingest" 2>/dev/null || true
  ufw allow 8554/tcp comment "RTSP ingest" 2>/dev/null || true
fi

LOG="\${REMOTE_DIR}/.logs"
mkdir -p "\${LOG}"

pkill -f "\${REMOTE_DIR}/bin/media-orchestrator" 2>/dev/null || true
fuser -k 9084/tcp 2>/dev/null || true
sleep 1

nohup env \$(grep -v '^#' "\${ENV_FILE}" | xargs) "\${REMOTE_DIR}/bin/media-orchestrator" >"\${LOG}/media-orchestrator.log" 2>&1 &
echo "  media-orchestrator pid=\$!"

pkill -f "\${REMOTE_DIR}/bin/stream-service" 2>/dev/null || true
fuser -k 50053/tcp 9083/tcp 2>/dev/null || true
sleep 1
nohup env \$(grep -v '^#' "\${ENV_FILE}" | xargs) "\${REMOTE_DIR}/bin/stream-service" >"\${LOG}/stream-service.log" 2>&1 &
echo "  stream-service pid=\$!"

# RunPod dan binary yuklab olish uchun vaqtinchalik HTTP
pkill -f "http.server 19999" 2>/dev/null || true
nohup python3 -m http.server 19999 --directory "\${REMOTE_DIR}/bin" --bind 0.0.0.0 >"\${LOG}/worker-download.log" 2>&1 &
if command -v ufw >/dev/null 2>&1; then
  ufw allow 19999/tcp comment "transcode-worker download temp" 2>/dev/null || true
fi

sleep 2
curl -sf "http://127.0.0.1:\${GATEWAY_PORT}/health" >/dev/null && echo "API OK" || echo "API tekshiring"
REMOTE

echo ""
echo "=========================================="
echo "VPS tayyor (queue mode)."
echo ""
echo "RunPod Web Terminal da ishga tushiring:"
echo ""
cat <<RUNPOD
mkdir -p /opt/transcode-worker /tmp/hls
cd /opt/transcode-worker
curl -fsSL -o transcode-worker http://${HOST}:19999/transcode-worker
chmod +x transcode-worker
cat > .env << 'EOF'
APP_ENV=production
LOG_LEVEL=info
WORKER_HTTP_ADDR=:9086
WORKER_ID=runpod-gpu-1
WORKER_MAX_JOBS=4
NATS_URL=nats://${HOST}:14222
FFMPEG_PATH=ffmpeg
FFMPEG_VIDEO_ENCODER=h264_nvenc
TRANSCODE_QUALITY=production
HLS_OUTPUT_DIR=/tmp/hls
HLS_STORAGE_BACKEND=s3
MINIO_ENDPOINT=${HOST}:19000
MINIO_ACCESS_KEY=sahiy_minio
MINIO_SECRET_KEY=sahiy_minio_secret
MINIO_BUCKET=sahiy-media
MINIO_USE_SSL=false
EOF
nohup env \$(grep -v '^#' .env | xargs) ./transcode-worker > worker.log 2>&1 &
sleep 2
curl -s http://localhost:9086/health
curl -s http://localhost:9086/ready
RUNPOD
echo ""
echo "Test: https://stream.vibrant.uz/studio/broadcast"
echo "Log (RunPod): tail -f /opt/transcode-worker/worker.log"
echo "Log (VPS):    tail -f /opt/sahiy-stream/.logs/media-orchestrator.log"
echo "=========================================="
