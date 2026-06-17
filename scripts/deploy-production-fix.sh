#!/usr/bin/env bash
# Production fix: media + stream + chat servislari va nginx CORS.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_FILE="${ROOT}/for-deploy.txt"
REMOTE_DIR="/opt/sahiy-stream"

if [[ ! -f "${DEPLOY_FILE}" ]]; then
  echo "for-deploy.txt topilmadi"
  exit 1
fi

read_deploy() { grep -E "^${1}:" "${DEPLOY_FILE}" | cut -d: -f2- | xargs || true; }
HOST=$(read_deploy "IP-manzil")
USER=$(read_deploy "Foydalanuvchi nomi")
PASS=$(read_deploy "Parol")
GATEWAY_PORT=$(read_deploy "Gateway port")
GATEWAY_PORT="${GATEWAY_PORT:-18080}"
HLS_PORT=$(read_deploy "HLS port")
HLS_PORT="${HLS_PORT:-18090}"
FRONTEND_PORT=$(read_deploy "Frontend port")
FRONTEND_PORT="${FRONTEND_PORT:-3010}"

SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=15"
ssh_cmd() { sshpass -p "${PASS}" ssh ${SSH_OPTS} "${USER}@${HOST}" "$@"; }
scp_cmd() { sshpass -p "${PASS}" scp ${SSH_OPTS} "$@"; }

echo "==> Build (linux/amd64)..."
export GOOS=linux GOARCH=amd64
mkdir -p "${ROOT}/bin"
for svc in media-orchestrator stream-service chat-service; do
  (cd "${ROOT}/services/${svc}" && go build -o "${ROOT}/bin/${svc}" ./cmd/server)
done

echo "==> Upload binaries..."
for svc in media-orchestrator stream-service chat-service; do
  scp_cmd "${ROOT}/bin/${svc}" "${USER}@${HOST}:/tmp/${svc}.new"
done
scp_cmd "${ROOT}/infra/nginx/stream.vibrant.uz.conf" "${USER}@${HOST}:/tmp/stream.vibrant.uz.conf"
scp_cmd "${ROOT}/infra/nginx/api.stream.vibrant.uz.conf" "${USER}@${HOST}:/tmp/api.stream.vibrant.uz.conf"

echo "==> Serverda yangilash..."
ssh_cmd bash -s <<REMOTE
set -euo pipefail
REMOTE_DIR="${REMOTE_DIR}"
GATEWAY_PORT="${GATEWAY_PORT}"
HLS_PORT="${HLS_PORT}"
FRONTEND_PORT="${FRONTEND_PORT}"

update_stream_nginx() {
  local target="$1"
  cp /tmp/stream.vibrant.uz.conf "\${target}"
  sed -i "s/__FRONTEND_PORT__/\${FRONTEND_PORT}/g" "\${target}"
  sed -i "s/__GATEWAY_PORT__/\${GATEWAY_PORT}/g" "\${target}"
  sed -i "s/__HLS_PORT__/\${HLS_PORT}/g" "\${target}"
  echo "  nginx updated: \${target}"
}

for f in \
  "/etc/nginx/sites-enabled/stream.vibrant.uz" \
  "/etc/nginx/sites-enabled/stream.vibrant.uz.conf" \
  "/etc/nginx/sites-available/stream.vibrant.uz" \
  "/etc/nginx/sites-available/stream.vibrant.uz.conf"; do
  if [[ -f "\${f}" ]]; then
    update_stream_nginx "\${f}"
  fi
done

for f in \
  "/etc/nginx/sites-enabled/api.stream.vibrant.uz" \
  "/etc/nginx/sites-enabled/api.stream.vibrant.uz.conf" \
  "/etc/nginx/sites-available/api.stream.vibrant.uz" \
  "/etc/nginx/sites-available/api.stream.vibrant.uz.conf"; do
  if [[ -f "\${f}" ]]; then
    cp /tmp/api.stream.vibrant.uz.conf "\${f}"
    echo "  nginx redirect updated: \${f}"
  fi
done

if [[ -f "\${REMOTE_DIR}/.env" ]]; then
  sed -i 's|^PLAYBACK_BASE_URL=.*|PLAYBACK_BASE_URL=https://stream.vibrant.uz|' "\${REMOTE_DIR}/.env"
  sed -i 's|^WHIP_BASE_URL=.*|WHIP_BASE_URL=https://stream.vibrant.uz|' "\${REMOTE_DIR}/.env"
  sed -i 's|^HLS_BASE_URL=.*|HLS_BASE_URL=https://stream.vibrant.uz/hls|' "\${REMOTE_DIR}/.env"
  echo "  .env playback URL yangilandi"
fi

for svc in media-orchestrator stream-service chat-service; do
  fuser -k \$(case \$svc in
    media-orchestrator) echo 9084;;
    stream-service) echo 9083 50053;;
    chat-service) echo 9085 50054;;
  esac)/tcp 2>/dev/null || true
  pkill -f "\${REMOTE_DIR}/bin/\$svc" 2>/dev/null || true
done
sleep 2

for svc in media-orchestrator stream-service chat-service; do
  mv "/tmp/\${svc}.new" "\${REMOTE_DIR}/bin/\${svc}"
  chmod +x "\${REMOTE_DIR}/bin/\${svc}"
done

LOG="\${REMOTE_DIR}/.logs"
mkdir -p "\${LOG}"
ENV="\$(grep -v '^#' \${REMOTE_DIR}/.env | xargs)"
start_svc() {
  nohup env \${ENV} "\${REMOTE_DIR}/bin/\$1" >"\${LOG}/\$1.log" 2>&1 &
  echo "  \$1 pid=\$!"
}
start_svc stream-service; sleep 2
start_svc chat-service; sleep 2
start_svc media-orchestrator; sleep 2

if nginx -t 2>/dev/null; then
  systemctl reload nginx
  echo "  nginx reloaded"
fi

curl -sf http://127.0.0.1:9084/health && echo "media-orchestrator ok"
curl -sf http://127.0.0.1:9083/health && echo "stream-service ok"
REMOTE

echo ""
echo "Deploy tugadi. OBS: Stop Streaming -> Start Streaming"
