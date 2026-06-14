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
API_DOMAIN="${API_DOMAIN:-api.stream.vibrant.uz}"

CERTBOT_EMAIL="${CERTBOT_EMAIL:-admin@vibrant.uz}"
API_URL="https://${API_DOMAIN}"
FRONTEND_URL="https://${FRONTEND_DOMAIN}"

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
    export NEXT_PUBLIC_API_URL="${API_URL}"
    export NEXT_PUBLIC_WHIP_BASE_URL="${API_URL}"
    export NEXT_PUBLIC_HLS_BASE_URL="${API_URL}/hls"
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
  scripts/migrate.sh \
  scripts/wait-for-api.sh \
  scripts/deploy-remote-only.sh \
  scripts/setup-nginx-ssl.sh \
  scripts/check-server-ports.sh \
  for-deploy.txt.example \
  services/auth-service/migrations \
  frontend/.next \
  frontend/public \
  frontend/package.json \
  frontend/package-lock.json \
  frontend/next.config.ts

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

cd "\${REMOTE_DIR}"
tar -xzf deploy.tar.gz
rm -f deploy.tar.gz
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

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow 80/tcp comment "HTTP certbot" 2>/dev/null || true
  ufw allow 443/tcp comment "HTTPS" 2>/dev/null || true
  ufw allow "\${FRONTEND_PORT}/tcp" comment "Next.js" 2>/dev/null || true
  ufw allow from 172.16.0.0/12 to any port 9084 proto tcp 2>/dev/null || true
fi

sed -i 's/host.docker.internal:9084/172.17.0.1:9084/g' "\${REMOTE_DIR}/infra/docker/nginx-rtmp/nginx.conf" 2>/dev/null || true
sed -i 's/host.docker.internal:9084/172.17.0.1:9084/g' "\${REMOTE_DIR}/infra/docker/mediamtx/mediamtx.yml" 2>/dev/null || true
sed -i "s/__SERVER_IP__/\${HOST_IP}/g" "\${REMOTE_DIR}/infra/docker/mediamtx/mediamtx.yml"
sed -i "s/\"8090:8090\"/\"\${HLS_PORT}:8090\"/" "\${REMOTE_DIR}/infra/docker/docker-compose.prod.yml"

JWT_ACCESS_SECRET=\$(openssl rand -hex 32)
JWT_REFRESH_SECRET=\$(openssl rand -hex 32)
MEDIA_HOOK_SECRET=\$(openssl rand -hex 32)
PLAYBACK_SIGNING_SECRET=\$(openssl rand -hex 32)

sed -i "s|/hooks/publish_done;|/hooks/publish_done?internal_secret=\${MEDIA_HOOK_SECRET};|g" "\${REMOTE_DIR}/infra/docker/nginx-rtmp/nginx.conf" 2>/dev/null || true
sed -i "s|/hooks/publish;|/hooks/publish?internal_secret=\${MEDIA_HOOK_SECRET};|g" "\${REMOTE_DIR}/infra/docker/nginx-rtmp/nginx.conf" 2>/dev/null || true
sed -i "s|/hooks/publish_done\"|/hooks/publish_done?internal_secret=\${MEDIA_HOOK_SECRET}\"|g" "\${REMOTE_DIR}/infra/docker/mediamtx/mediamtx.yml" 2>/dev/null || true
sed -i "s|/hooks/publish\"|/hooks/publish?internal_secret=\${MEDIA_HOOK_SECRET}\"|g" "\${REMOTE_DIR}/infra/docker/mediamtx/mediamtx.yml" 2>/dev/null || true

cd "\${REMOTE_DIR}/infra/docker"
docker compose -f docker-compose.prod.yml down 2>/dev/null || true
docker compose -f docker-compose.prod.yml up -d --build
sleep 12

export DATABASE_URL="postgres://sahiy:sahiy_secret@127.0.0.1:15433/sahiy_stream?sslmode=disable"
migrate -path "\${REMOTE_DIR}/migrations" -database "\${DATABASE_URL}" up 2>/dev/null || true

pkill -f "\${REMOTE_DIR}/bin/" 2>/dev/null || true
for port in 50051 50052 50053 50054 "\${GATEWAY_PORT}" 9084 9085 "\${FRONTEND_PORT}"; do
  fuser -k "\${port}/tcp" 2>/dev/null || true
done
sleep 2

cat >"\${REMOTE_DIR}/.env" <<ENVFILE
APP_ENV=production
LOG_LEVEL=info
DATABASE_URL=postgres://sahiy:sahiy_secret@127.0.0.1:15433/sahiy_stream?sslmode=disable
REDIS_URL=redis://127.0.0.1:16379/0
NATS_URL=nats://127.0.0.1:14222
JWT_ACCESS_SECRET=\${JWT_ACCESS_SECRET}
JWT_REFRESH_SECRET=\${JWT_REFRESH_SECRET}
MEDIA_HOOK_SECRET=\${MEDIA_HOOK_SECRET}
PLAYBACK_SIGNING_SECRET=\${PLAYBACK_SIGNING_SECRET}
PLAYBACK_BASE_URL=\${API_URL}
HLS_STORAGE_BACKEND=local
FFMPEG_VIDEO_ENCODER=libx264
TRANSCODE_QUALITY=production
TRANSCODE_MODE=local
WORKER_MAX_JOBS=1
AUTH_SERVICE_ADDR=localhost:50051
USER_SERVICE_ADDR=localhost:50052
STREAM_SERVICE_ADDR=localhost:50053
CHAT_SERVICE_ADDR=localhost:50054
CHAT_HTTP_ADDR=localhost:9085
GATEWAY_HTTP_ADDR=:\${GATEWAY_PORT}
GATEWAY_CORS_ORIGINS=\${FRONTEND_URL},https://\${FRONTEND_DOMAIN}
WHIP_BASE_URL=\${API_URL}
HLS_BASE_URL=\${API_URL}/hls
HLS_OUTPUT_DIR=\${REMOTE_DIR}/data/hls
RTMP_INTERNAL_URL=rtmp://127.0.0.1:1935/live
RTSP_INTERNAL_URL=rtsp://127.0.0.1:8554
MEDIA_HTTP_ADDR=:9084
GATEWAY_RATE_LIMIT_RPM=500
ENVFILE

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
npm install --omit=dev 2>/dev/null || npm install
nohup npx next start -p "\${FRONTEND_PORT}" -H 0.0.0.0 >"\${LOG}/frontend.log" 2>&1 &

if [[ "\${SETUP_SSL:-1}" == "1" ]]; then
  echo "==> Nginx + SSL..."
  REMOTE_DIR="\${REMOTE_DIR}" FRONTEND_PORT="\${FRONTEND_PORT}" \
    GATEWAY_PORT="\${GATEWAY_PORT}" HLS_PORT="\${HLS_PORT}" \
    API_DOMAIN="\${API_DOMAIN}" FRONTEND_DOMAIN="\${FRONTEND_DOMAIN}" \
    CERTBOT_EMAIL="\${CERTBOT_EMAIL:-admin@vibrant.uz}" \
    bash "\${REMOTE_DIR}/scripts/setup-nginx-ssl.sh" || echo "SSL xato — keyin: SETUP_SSL=1 bash scripts/setup-nginx-ssl.sh"
fi

echo "Frontend: \${FRONTEND_URL}"
echo "API:      \${API_URL}/health"
REMOTE

rm -f "${ARCHIVE}"
echo ""
echo "Deploy tugadi!"
echo "  Panel:    ${FRONTEND_URL}"
echo "  API:      ${API_URL}/health"
echo "  Broadcast: ${FRONTEND_URL}/studio/broadcast"
