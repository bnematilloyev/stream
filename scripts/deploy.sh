#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_FILE="${ROOT}/for-deploy.txt"

if [[ ! -f "${DEPLOY_FILE}" ]]; then
  echo "for-deploy.txt topilmadi. Nusxa: cp for-deploy.txt.example for-deploy.txt"
  exit 1
fi

read_deploy() {
  grep -E "^${1}:" "${DEPLOY_FILE}" | cut -d: -f2- | xargs || true
}

HOST=$(read_deploy "IP-manzil")
USER=$(read_deploy "Foydalanuvchi nomi")
PASS=$(read_deploy "Parol")
FRONTEND_PORT=$(read_deploy "Frontend port")
FRONTEND_PORT="${FRONTEND_PORT:-3002}"
GATEWAY_PORT=$(read_deploy "Gateway port")
GATEWAY_PORT="${GATEWAY_PORT:-8080}"
HLS_PORT=$(read_deploy "HLS port")
HLS_PORT="${HLS_PORT:-8090}"
FRONTEND_DOMAIN=$(read_deploy "Frontend domen")
API_DOMAIN=$(read_deploy "API domen")
CERTBOT_EMAIL=$(read_deploy "Certbot email")
# Eski format: Domen: -> frontend
if [[ -z "${FRONTEND_DOMAIN}" ]]; then
  FRONTEND_DOMAIN=$(read_deploy "Domen")
fi
FRONTEND_DOMAIN="${FRONTEND_DOMAIN:-stream.vibrant.uz}"
API_DOMAIN=$(read_deploy "API domen")
if [[ -z "${API_DOMAIN}" || "${API_DOMAIN}" == "${FRONTEND_DOMAIN}" ]]; then
  API_DOMAIN="${FRONTEND_DOMAIN}"
fi
GPU_TRANSCODE=$(read_deploy "GPU transcode")
GPU_TRANSCODE="${GPU_TRANSCODE:-no}"

CERTBOT_EMAIL="${CERTBOT_EMAIL:-admin@vibrant.uz}"
PUBLIC_URL="https://${FRONTEND_DOMAIN}"
FRONTEND_URL="${PUBLIC_URL}"
API_URL="${PUBLIC_URL}"

if [[ -z "${HOST}" || -z "${USER}" || -z "${PASS}" ]]; then
  echo "for-deploy.txt da IP, user yoki parol yo'q"
  exit 1
fi

REMOTE_DIR="/opt/sahiy-stream"
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=15"

ssh_cmd() {
  sshpass -p "${PASS}" ssh ${SSH_OPTS} "${USER}@${HOST}" "$@"
}

scp_cmd() {
  sshpass -p "${PASS}" scp ${SSH_OPTS} "$@"
}

echo "==> SSH tekshiruvi (${USER}@${HOST}:22)..."
if ! command -v sshpass >/dev/null 2>&1; then
  echo "sshpass topilmadi. Mac: brew install sshpass"
  echo "Yoki SSH kalit bilan: ssh ${USER}@${HOST}"
  exit 1
fi
if ! ssh_cmd "echo ok" >/dev/null 2>&1; then
  echo "SSH ulanib bo'lmadi (${USER}@${HOST}:22)"
  echo "Tekshiring:"
  echo "  1) for-deploy.txt — IP, Foydalanuvchi nomi, Parol to'g'rimi?"
  echo "  2) Qo'lda: ssh ${USER}@${HOST}"
  echo "  3) Contabo paneldagi root paroli (maxsus belgilar bo'lsa qo'shtirnoqda)"
  exit 1
fi

echo "==> Domenlar:"
echo "    Frontend: ${FRONTEND_URL}  (port ${FRONTEND_PORT})"
echo "    Gateway:  :${GATEWAY_PORT}"
echo "    HLS:      :${HLS_PORT}"

if [[ "${SKIP_BUILD:-}" != "1" ]]; then
  echo "==> Go servislarni build qilish (linux/amd64)..."
  export GOOS=linux GOARCH=amd64
  mkdir -p "${ROOT}/bin"
  for svc in auth-service user-service stream-service chat-service media-orchestrator transcode-worker api-gateway; do
    (cd "${ROOT}/services/${svc}" && go build -o "${ROOT}/bin/${svc}" ./cmd/server)
  done
else
  echo "==> SKIP_BUILD=1"
fi

if [[ "${SKIP_FRONTEND:-}" != "1" ]]; then
  echo "==> Frontend build..."
  pkill -f "next dev" 2>/dev/null || true
  (
    cd "${ROOT}/frontend"
    export NEXT_PUBLIC_API_URL="${FRONTEND_URL}"
    export NEXT_PUBLIC_WHIP_BASE_URL="${FRONTEND_URL}"
    export NEXT_PUBLIC_HLS_BASE_URL="${FRONTEND_URL}/hls"
    if [[ ! -x node_modules/.bin/next ]]; then
      echo "==> Frontend dependencies (npm ci)..."
      npm ci
    fi
    npm run build
  )
fi

ARCHIVE="/tmp/sahiy-stream-deploy.tar.gz"
echo "==> Arxiv..."
tar -czf "${ARCHIVE}" -C "${ROOT}" \
  bin \
  infra \
  Makefile.prod \
  scripts/migrate.sh \
  scripts/prod-migrate.sh \
  scripts/prod-restart.sh \
  scripts/prod-status.sh \
  scripts/wait-for-api.sh \
  scripts/deploy-remote-only.sh \
  scripts/build-frontend-server.sh \
  scripts/setup-nginx-ssl.sh \
  scripts/check-server-ports.sh \
  scripts/ensure-gpu-queue.sh \
  for-deploy.txt.example \
  services/auth-service/migrations \
  frontend/src \
  frontend/public \
  frontend/package.json \
  frontend/package-lock.json \
  frontend/next.config.mjs \
  frontend/tsconfig.json \
  frontend/postcss.config.mjs \
  frontend/.next

echo "==> Serverga yuklash..."
ssh_cmd "mkdir -p ${REMOTE_DIR}"
scp_cmd "${DEPLOY_FILE}" "${USER}@${HOST}:${REMOTE_DIR}/for-deploy.txt"
scp_cmd "${ARCHIVE}" "${USER}@${HOST}:${REMOTE_DIR}/deploy.tar.gz"

echo "==> Serverda sozlash..."
ssh_cmd bash -s <<REMOTE
set -euo pipefail
REMOTE_DIR="${REMOTE_DIR}"
HOST_IP="${HOST}"
FRONTEND_PORT="${FRONTEND_PORT}"
GATEWAY_PORT="${GATEWAY_PORT}"
HLS_PORT="${HLS_PORT}"
API_DOMAIN="${API_DOMAIN}"
FRONTEND_DOMAIN="${FRONTEND_DOMAIN}"
API_URL="${API_URL}"
FRONTEND_URL="${FRONTEND_URL}"
CERTBOT_EMAIL="${CERTBOT_EMAIL}"
GPU_TRANSCODE="${GPU_TRANSCODE}"

cd "\${REMOTE_DIR}"
tar -xzf deploy.tar.gz
rm -f deploy.tar.gz
mv -f Makefile.prod Makefile 2>/dev/null || true
chmod +x scripts/*.sh 2>/dev/null || true
chmod +x scripts/setup-nginx-ssl.sh
mv services/auth-service/migrations migrations 2>/dev/null || true

export DEBIAN_FRONTEND=noninteractive

if ! command -v docker >/dev/null 2>&1; then
  apt-get update -qq
  apt-get install -y -qq docker.io docker-compose-plugin ffmpeg curl wget
  systemctl enable --now docker
fi

if ! command -v node >/dev/null 2>&1; then
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
  apt-get install -y -qq nodejs
fi

if ! command -v migrate >/dev/null 2>&1; then
  curl -fsSL -o /tmp/migrate.tgz \
    https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz
  tar -xzf /tmp/migrate.tgz -C /tmp migrate
  mv /tmp/migrate /usr/local/bin/migrate
  chmod +x /usr/local/bin/migrate
  rm -f /tmp/migrate.tgz
fi

mkdir -p "\${REMOTE_DIR}/data/hls" "\${REMOTE_DIR}/.logs"
echo "\${HOST_IP}" >"\${REMOTE_DIR}/deploy-host.txt"

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow 80/tcp comment "HTTP certbot" 2>/dev/null || true
  ufw allow 443/tcp comment "HTTPS" 2>/dev/null || true
  ufw allow 1935/tcp comment "RTMP ingest" 2>/dev/null || true
  ufw allow "\${FRONTEND_PORT}/tcp" comment "Next.js" 2>/dev/null || true
  ufw allow from 172.16.0.0/12 to any port 9084 proto tcp 2>/dev/null || true
fi

sed -i "s/__SERVER_IP__/\${HOST_IP}/g" "\${REMOTE_DIR}/infra/docker/mediamtx/mediamtx.yml"
sed -i "s/\"8090:8090\"/\"\${HLS_PORT}:8090\"/" "\${REMOTE_DIR}/infra/docker/docker-compose.prod.yml"

JWT_ACCESS_SECRET=\$(openssl rand -hex 32)
JWT_REFRESH_SECRET=\$(openssl rand -hex 32)
MEDIA_HOOK_SECRET=\$(openssl rand -hex 32)
PLAYBACK_SIGNING_SECRET=\$(openssl rand -hex 32)
SERVICE_TOKEN=\$(openssl rand -hex 32)

if [[ -f "\${REMOTE_DIR}/.env" ]]; then
  _read_env() { grep "^\$1=" "\${REMOTE_DIR}/.env" 2>/dev/null | cut -d= -f2- || true; }
  _saved=\$(_read_env JWT_ACCESS_SECRET); [[ -n "\${_saved}" ]] && JWT_ACCESS_SECRET="\${_saved}"
  _saved=\$(_read_env JWT_REFRESH_SECRET); [[ -n "\${_saved}" ]] && JWT_REFRESH_SECRET="\${_saved}"
  _saved=\$(_read_env MEDIA_HOOK_SECRET); [[ -n "\${_saved}" ]] && MEDIA_HOOK_SECRET="\${_saved}"
  _saved=\$(_read_env PLAYBACK_SIGNING_SECRET); [[ -n "\${_saved}" ]] && PLAYBACK_SIGNING_SECRET="\${_saved}"
  _saved=\$(_read_env SERVICE_TOKEN); [[ -n "\${_saved}" ]] && SERVICE_TOKEN="\${_saved}"
fi

cat >"\${REMOTE_DIR}/.env" <<ENVFILE
APP_ENV=production
LOG_LEVEL=info
DATABASE_URL=postgres://sahiy:sahiy_secret@127.0.0.1:15433/sahiy_stream?sslmode=disable
REDIS_URL=redis://127.0.0.1:16379/0
NATS_URL=nats://127.0.0.1:14222
JWT_ACCESS_SECRET=\${JWT_ACCESS_SECRET}
JWT_REFRESH_SECRET=\${JWT_REFRESH_SECRET}
JWT_ACCESS_TTL=24h
JWT_REFRESH_TTL=720h
MEDIA_HOOK_SECRET=\${MEDIA_HOOK_SECRET}
PLAYBACK_SIGNING_SECRET=\${PLAYBACK_SIGNING_SECRET}
PLAYBACK_BASE_URL=\${FRONTEND_URL}
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
GATEWAY_HTTP_ADDR=:\${GATEWAY_PORT}
GATEWAY_CORS_ORIGINS=\${FRONTEND_URL},https://\${FRONTEND_DOMAIN}
WHIP_BASE_URL=\${FRONTEND_URL}
HLS_BASE_URL=\${FRONTEND_URL}/hls
HLS_OUTPUT_DIR=\${REMOTE_DIR}/data/hls
RTMP_BASE_URL=rtmp://\${HOST_IP}:1935/live
RTMP_INTERNAL_URL=rtmp://127.0.0.1:1935/live
RTSP_INTERNAL_URL=rtsp://127.0.0.1:8554
MEDIA_HTTP_ADDR=:9084
GATEWAY_RATE_LIMIT_RPM=500
SERVICE_TOKEN=\${SERVICE_TOKEN}
MARKET_WEBHOOK_URL=
MARKET_WEBHOOK_SECRET=\${SERVICE_TOKEN}
ENVFILE

echo "SERVICE_TOKEN (market STREAM_SERVICE_TOKEN bilan bir xil qiling):"
grep '^SERVICE_TOKEN=' "\${REMOTE_DIR}/.env"

bash "\${REMOTE_DIR}/scripts/sync-hook-secrets.sh" 2>/dev/null || true

cd "\${REMOTE_DIR}/infra/docker"
docker compose -f docker-compose.prod.yml down 2>/dev/null || true
docker compose -f docker-compose.prod.yml up -d --build
sleep 12

export DATABASE_URL="postgres://sahiy:sahiy_secret@127.0.0.1:15433/sahiy_stream?sslmode=disable"
bash "\${REMOTE_DIR}/scripts/prod-migrate.sh" up 2>/dev/null || \
  migrate -path "\${REMOTE_DIR}/migrations" -database "\${DATABASE_URL}" up 2>/dev/null || true

pkill -f "\${REMOTE_DIR}/bin/" 2>/dev/null || true
for port in 50051 50052 50053 50054 "\${GATEWAY_PORT}" 9084 9085 "\${FRONTEND_PORT}"; do
  fuser -k "\${port}/tcp" 2>/dev/null || true
done
sleep 2

# .env allaqachon yozilgan (sync-hook-secrets uchun)

LOG="\${REMOTE_DIR}/.logs"
start_svc() {
  nohup env \$(grep -v '^#' "\${REMOTE_DIR}/.env" | xargs) "\${REMOTE_DIR}/bin/\$1" >"\${LOG}/\$1.log" 2>&1 &
  echo "  \$1 pid=\$!"
}

start_svc auth-service; sleep 2
start_svc user-service; sleep 2
start_svc stream-service; sleep 2
start_svc chat-service; sleep 2
start_svc media-orchestrator; sleep 2
start_svc api-gateway; sleep 3

cd "\${REMOTE_DIR}/frontend"
if [[ ! -d .next ]]; then
  echo "Xato: frontend/.next yo'q"
  exit 1
fi
npm ci --omit=dev 2>/dev/null || npm ci
pkill -f "next start -p \${FRONTEND_PORT}" 2>/dev/null || true
fuser -k "\${FRONTEND_PORT}/tcp" 2>/dev/null || true
sleep 1
nohup ./node_modules/.bin/next start -p "\${FRONTEND_PORT}" -H 0.0.0.0 >"\${LOG}/frontend.log" 2>&1 &
echo "  frontend pid=\$! port=\${FRONTEND_PORT}"
for i in \$(seq 1 30); do
  if curl -sf "http://127.0.0.1:\${FRONTEND_PORT}/" >/dev/null 2>&1; then
    echo "  frontend tayyor"
    break
  fi
  sleep 1
done

chmod +x "\${REMOTE_DIR}/scripts/ensure-gpu-queue.sh" 2>/dev/null || true
use_gpu=0
case "\$(echo "\${GPU_TRANSCODE}" | tr '[:upper:]' '[:lower:]')" in
  yes|ha|true|1) use_gpu=1 ;;
esac
[[ -f "\${REMOTE_DIR}/.gpu-transcode" ]] && use_gpu=1
if [[ "\${use_gpu}" -eq 1 ]]; then
  echo "==> GPU queue mode (transcode RunPod da, VPS da emas)..."
  bash "\${REMOTE_DIR}/scripts/ensure-gpu-queue.sh"
fi

if [[ "\${SETUP_SSL:-1}" == "1" ]]; then
  echo "==> Nginx + SSL..."
  REMOTE_DIR="\${REMOTE_DIR}" FRONTEND_PORT="\${FRONTEND_PORT}" \
    GATEWAY_PORT="\${GATEWAY_PORT}" HLS_PORT="\${HLS_PORT}" \
    API_DOMAIN="\${API_DOMAIN}" FRONTEND_DOMAIN="\${FRONTEND_DOMAIN}" \
    CERTBOT_EMAIL="\${CERTBOT_EMAIL:-admin@vibrant.uz}" \
    bash "\${REMOTE_DIR}/scripts/setup-nginx-ssl.sh" || echo "SSL xato — keyin: SETUP_SSL=1 bash scripts/setup-nginx-ssl.sh"
fi

echo "Frontend: \${FRONTEND_URL}"
echo "API:      \${FRONTEND_URL}/health"
REMOTE

rm -f "${ARCHIVE}"
echo ""
echo "Deploy tugadi!"
echo "  Panel:    ${FRONTEND_URL}"
echo "  API:      ${FRONTEND_URL}/health"
echo "  Broadcast: ${FRONTEND_URL}/studio/broadcast"
