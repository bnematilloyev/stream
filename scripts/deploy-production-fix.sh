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
scp_cmd "${ROOT}/infra/nginx/api.stream.vibrant.uz.conf" "${USER}@${HOST}:/tmp/api.stream.vibrant.uz.conf"

echo "==> Serverda yangilash..."
ssh_cmd bash -s <<REMOTE
set -euo pipefail
REMOTE_DIR="${REMOTE_DIR}"
GATEWAY_PORT="${GATEWAY_PORT}"
HLS_PORT="${HLS_PORT}"

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

# Nginx CORS fix
for f in \
  "/etc/nginx/sites-enabled/api.stream.vibrant.uz" \
  "/etc/nginx/sites-enabled/api.stream.vibrant.uz.conf" \
  "/etc/nginx/sites-available/api.stream.vibrant.uz" \
  "/etc/nginx/sites-available/api.stream.vibrant.uz.conf"; do
  if [[ -f "\${f}" ]]; then
    cp /tmp/api.stream.vibrant.uz.conf "\${f}"
    sed -i "s/__GATEWAY_PORT__/\${GATEWAY_PORT}/g" "\${f}"
    sed -i "s/__HLS_PORT__/\${HLS_PORT}/g" "\${f}"
    echo "  nginx updated: \${f}"
  fi
done
if nginx -t 2>/dev/null; then
  systemctl reload nginx
  echo "  nginx reloaded"
fi

curl -sf http://127.0.0.1:9084/health && echo "media-orchestrator ok"
curl -sf http://127.0.0.1:9083/health && echo "stream-service ok"
REMOTE

echo ""
echo "Deploy tugadi. OBS: Stop Streaming -> Start Streaming"
