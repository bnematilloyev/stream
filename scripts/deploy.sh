#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_FILE="${ROOT}/for-deploy.txt"

if [[ ! -f "${DEPLOY_FILE}" ]]; then
  echo "for-deploy.txt topilmadi"
  exit 1
fi

HOST=$(grep -E '^IP-manzil:' "${DEPLOY_FILE}" | cut -d: -f2- | xargs)
USER=$(grep -E '^Foydalanuvchi nomi:' "${DEPLOY_FILE}" | cut -d: -f2- | xargs)
PASS=$(grep -E '^Parol:' "${DEPLOY_FILE}" | cut -d: -f2- | xargs)
FRONTEND_PORT=$(grep -E '^Frontend port:' "${DEPLOY_FILE}" | cut -d: -f2- | xargs)
FRONTEND_PORT="${FRONTEND_PORT:-3002}"
DOMAIN=$(grep -E '^Domen:' "${DEPLOY_FILE}" | cut -d: -f2- | xargs)
PUBLIC_URL="${DOMAIN:+https://${DOMAIN}}"
PUBLIC_URL="${PUBLIC_URL:-http://${HOST}:${FRONTEND_PORT}}"

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

echo "==> SSH tekshiruvi..."
if ! ssh_cmd "echo ok" >/dev/null 2>&1; then
  echo "SSH ulanib bo'lmadi (${USER}@${HOST}:22). Server ishlayaptimi?"
  exit 1
fi

echo "==> Sahiy Stream port: ${FRONTEND_PORT} (Shopla :3000 tegmaslik)"

if [[ "${SKIP_BUILD:-}" != "1" ]]; then
  echo "==> Go servislarni build qilish (linux/amd64)..."
  export GOOS=linux GOARCH=amd64
  mkdir -p "${ROOT}/bin"
  (cd "${ROOT}/services/auth-service" && go build -o "${ROOT}/bin/auth-service" ./cmd/server)
  (cd "${ROOT}/services/user-service" && go build -o "${ROOT}/bin/user-service" ./cmd/server)
  (cd "${ROOT}/services/stream-service" && go build -o "${ROOT}/bin/stream-service" ./cmd/server)
  (cd "${ROOT}/services/media-orchestrator" && go build -o "${ROOT}/bin/media-orchestrator" ./cmd/server)
  (cd "${ROOT}/services/api-gateway" && go build -o "${ROOT}/bin/api-gateway" ./cmd/server)
else
  echo "==> SKIP_BUILD=1 — Go build o'tkazib yuborildi"
fi

echo "==> Frontend build (${PUBLIC_URL})..."
pkill -f "next dev" 2>/dev/null || true
(
  cd "${ROOT}/frontend"
  export NEXT_PUBLIC_API_URL="${PUBLIC_URL}"
  export NEXT_PUBLIC_WHIP_BASE_URL="${PUBLIC_URL}"
  export NEXT_PUBLIC_HLS_BASE_URL="${PUBLIC_URL}/hls"
  npm run build
)

ARCHIVE="/tmp/sahiy-stream-deploy.tar.gz"
echo "==> Arxiv yaratish..."
tar -czf "${ARCHIVE}" -C "${ROOT}" \
  bin \
  infra \
  scripts/migrate.sh \
  scripts/wait-for-api.sh \
  scripts/deploy-remote-only.sh \
  services/auth-service/migrations \
  frontend/.next \
  frontend/public \
  frontend/package.json \
  frontend/package-lock.json \
  frontend/next.config.ts

echo "==> Serverga yuklash..."
ssh_cmd "mkdir -p ${REMOTE_DIR} && echo '${HOST}' > ${REMOTE_DIR}/deploy-host.txt"
scp_cmd "${ARCHIVE}" "${USER}@${HOST}:${REMOTE_DIR}/deploy.tar.gz"

echo "==> Serverda sozlash..."
ssh_cmd bash -s <<REMOTE
set -euo pipefail
REMOTE_DIR="${REMOTE_DIR}"
HOST_IP="${HOST}"
FRONTEND_PORT="${FRONTEND_PORT}"
PUBLIC_URL="${PUBLIC_URL}"

cd "\${REMOTE_DIR}"
tar -xzf deploy.tar.gz
rm -f deploy.tar.gz
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

if [[ -f "\${REMOTE_DIR}/infra/nginx/stream.shopla.uz.conf" ]]; then
  cp "\${REMOTE_DIR}/infra/nginx/stream.shopla.uz.conf" /etc/nginx/sites-available/stream.shopla.uz
  ln -sf /etc/nginx/sites-available/stream.shopla.uz /etc/nginx/sites-enabled/stream.shopla.uz
  nginx -t && systemctl reload nginx
fi

if command -v ufw >/dev/null 2>&1 && ufw status 2>/dev/null | grep -q "Status: active"; then
  ufw allow "\${FRONTEND_PORT}/tcp" comment "Sahiy Stream frontend" 2>/dev/null || true
  ufw allow 8080/tcp comment "Sahiy Stream API" 2>/dev/null || true
  ufw allow 8889/tcp comment "Sahiy WHIP" 2>/dev/null || true
  ufw allow 8090/tcp comment "Sahiy HLS" 2>/dev/null || true
  ufw allow 1935/tcp comment "Sahiy RTMP" 2>/dev/null || true
  ufw allow 8189/tcp comment "Sahiy WebRTC ICE" 2>/dev/null || true
  ufw allow 8189/udp comment "Sahiy WebRTC ICE" 2>/dev/null || true
  ufw allow from 172.16.0.0/12 to any port 9084 proto tcp comment "Docker to media-orchestrator" 2>/dev/null || true
fi

sed -i 's/host.docker.internal:9084/172.17.0.1:9084/g' "\${REMOTE_DIR}/infra/docker/nginx-rtmp/nginx.conf" 2>/dev/null || true
sed -i 's/host.docker.internal:9084/172.17.0.1:9084/g' "\${REMOTE_DIR}/infra/docker/mediamtx/mediamtx.yml" 2>/dev/null || true

cd "\${REMOTE_DIR}/infra/docker"
docker compose -f docker-compose.prod.yml down 2>/dev/null || true
docker rm -f sahiy-redis sahiy-postgres sahiy-minio 2>/dev/null || true
docker compose -f docker-compose.prod.yml up -d
sleep 10

export DATABASE_URL="postgres://sahiy:sahiy_secret@127.0.0.1:15433/sahiy_stream?sslmode=disable"
migrate -path "\${REMOTE_DIR}/migrations" -database "\${DATABASE_URL}" up 2>/dev/null || true

docker exec sahiy-postgres psql -U sahiy -d sahiy_stream -c "
  UPDATE streams SET status='ended', ended_at=NOW()
  WHERE status='live' AND NOT EXISTS (
    SELECT 1 FROM stream_media sm WHERE sm.stream_id=streams.id AND sm.status='ingesting'
  );
  UPDATE channels SET is_live=false WHERE is_live=true;
" 2>/dev/null || true

# Faqat Sahiy Stream (Shopla 3000 ga tegmaslik)
pkill -f "\${REMOTE_DIR}/bin/" 2>/dev/null || true
for port in 50051 50052 50053 8080 9084 "\${FRONTEND_PORT}"; do
  fuser -k "\${port}/tcp" 2>/dev/null || true
done
sleep 2

cat >"\${REMOTE_DIR}/.env" <<ENVFILE
APP_ENV=production
LOG_LEVEL=info
DATABASE_URL=postgres://sahiy:sahiy_secret@127.0.0.1:15433/sahiy_stream?sslmode=disable
REDIS_URL=redis://127.0.0.1:16379/0
AUTH_SERVICE_ADDR=localhost:50051
USER_SERVICE_ADDR=localhost:50052
STREAM_SERVICE_ADDR=localhost:50053
GATEWAY_HTTP_ADDR=:8080
GATEWAY_CORS_ORIGINS=\${PUBLIC_URL},http://\${HOST_IP}:\${FRONTEND_PORT},http://\${HOST_IP}:3000,http://\${HOST_IP}
WHIP_BASE_URL=\${PUBLIC_URL}
HLS_BASE_URL=\${PUBLIC_URL}/hls
HLS_OUTPUT_DIR=\${REMOTE_DIR}/data/hls
RTMP_INTERNAL_URL=rtmp://127.0.0.1:1935/live
RTSP_INTERNAL_URL=rtsp://127.0.0.1:8554
MEDIA_HTTP_ADDR=:9084
ENVFILE

LOG="\${REMOTE_DIR}/.logs"
start_svc() {
  nohup env \$(grep -v '^#' "\${REMOTE_DIR}/.env" | xargs) "\${REMOTE_DIR}/bin/\$1" >"\${LOG}/\$1.log" 2>&1 &
  echo "  \$1 pid=\$!"
}

start_svc auth-service
sleep 2
start_svc user-service
sleep 2
start_svc stream-service
sleep 2
start_svc media-orchestrator
sleep 2
start_svc api-gateway
sleep 3

cd "\${REMOTE_DIR}/frontend"
npm install
nohup npx next start -p "\${FRONTEND_PORT}" -H 0.0.0.0 >"\${LOG}/frontend.log" 2>&1 &

echo "Sahiy Stream: \${PUBLIC_URL}"
echo "Shopla: http://\${HOST_IP}:3000"
REMOTE

rm -f "${ARCHIVE}"
echo ""
echo "Deploy muvaffaqiyatli!"
echo "  Sahiy Stream: ${PUBLIC_URL}"
echo "  Kamera efir:  ${PUBLIC_URL}/studio/broadcast"
echo "  Shopla:       http://${HOST}:3000 (o'zgarmagan)"
