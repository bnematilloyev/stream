#!/usr/bin/env bash
# Serverda qolgan qadam: servislar + nginx (kod yuklangan bo'lsa).
# Ishlatish: bash scripts/finish-deploy.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_FILE="${ROOT}/for-deploy.txt"

if [[ ! -f "${DEPLOY_FILE}" ]]; then
  echo "for-deploy.txt topilmadi"
  exit 1
fi

read_deploy() {
  grep -E "^${1}:" "${DEPLOY_FILE}" | cut -d: -f2- | xargs || true
}

HOST=$(read_deploy "IP-manzil")
USER=$(read_deploy "Foydalanuvchi nomi")
PASS=$(read_deploy "Parol")
FRONTEND_PORT=$(read_deploy "Frontend port")
FRONTEND_PORT="${FRONTEND_PORT:-3010}"
GATEWAY_PORT=$(read_deploy "Gateway port")
GATEWAY_PORT="${GATEWAY_PORT:-18080}"
HLS_PORT=$(read_deploy "HLS port")
HLS_PORT="${HLS_PORT:-18090}"
FRONTEND_DOMAIN=$(read_deploy "Frontend domen")
API_DOMAIN=$(read_deploy "API domen")
CERTBOT_EMAIL=$(read_deploy "Certbot email")
FRONTEND_DOMAIN="${FRONTEND_DOMAIN:-stream.vibrant.uz}"
API_DOMAIN="${API_DOMAIN:-api.stream.vibrant.uz}"
CERTBOT_EMAIL="${CERTBOT_EMAIL:-admin@vibrant.uz}"

SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=15"
ssh_cmd() {
  sshpass -p "${PASS}" ssh ${SSH_OPTS} "${USER}@${HOST}" "$@"
}

scp_cmd() {
  sshpass -p "${PASS}" scp ${SSH_OPTS} "$@"
}

if ! command -v sshpass >/dev/null 2>&1; then
  echo "sshpass kerak: brew install hudochenkov/sshpass/sshpass"
  exit 1
fi

echo "==> Yangilangan skriptlarni yuklash..."
ssh_cmd "mkdir -p /opt/sahiy-stream/scripts /opt/sahiy-stream/bin"
if [[ -d "${ROOT}/bin" ]]; then
  scp_cmd "${ROOT}/bin/"* "${USER}@${HOST}:/opt/sahiy-stream/bin/"
fi
scp_cmd "${ROOT}/scripts/deploy-remote-only.sh" "${USER}@${HOST}:/opt/sahiy-stream/scripts/"
scp_cmd "${ROOT}/scripts/setup-nginx-ssl.sh" "${USER}@${HOST}:/opt/sahiy-stream/scripts/"
scp_cmd "${ROOT}/infra/nginx/api.stream.vibrant.uz.conf" "${USER}@${HOST}:/opt/sahiy-stream/infra/nginx/"
scp_cmd "${ROOT}/infra/nginx/stream.vibrant.uz.conf" "${USER}@${HOST}:/opt/sahiy-stream/infra/nginx/"
scp_cmd "${ROOT}/frontend/next.config.mjs" "${USER}@${HOST}:/opt/sahiy-stream/frontend/"
ssh_cmd "chmod +x /opt/sahiy-stream/scripts/deploy-remote-only.sh /opt/sahiy-stream/scripts/setup-nginx-ssl.sh"

echo "==> Serverda servislarni ishga tushirish..."
ssh_cmd bash -s <<REMOTE
set -euo pipefail
REMOTE_DIR="/opt/sahiy-stream"
if [[ ! -d "\${REMOTE_DIR}/bin" ]]; then
  echo "Xato: /opt/sahiy-stream/bin yo'q. Avval: bash scripts/deploy.sh"
  exit 1
fi
bash "\${REMOTE_DIR}/scripts/deploy-remote-only.sh"
REMOTE

echo "==> Nginx + SSL..."
ssh_cmd bash -s <<REMOTE
set -euo pipefail
REMOTE_DIR="/opt/sahiy-stream"
REMOTE_DIR="\${REMOTE_DIR}" FRONTEND_PORT="${FRONTEND_PORT}" \
  GATEWAY_PORT="${GATEWAY_PORT}" HLS_PORT="${HLS_PORT}" \
  API_DOMAIN="${API_DOMAIN}" FRONTEND_DOMAIN="${FRONTEND_DOMAIN}" \
  CERTBOT_EMAIL="${CERTBOT_EMAIL}" \
  bash "\${REMOTE_DIR}/scripts/setup-nginx-ssl.sh"
REMOTE

echo ""
echo "Tekshiruv:"
echo "  curl -s https://${API_DOMAIN}/health"
echo "  curl -sI https://${FRONTEND_DOMAIN}/"
echo ""
curl -s "https://${API_DOMAIN}/health" || true
echo ""
curl -sI "https://${FRONTEND_DOMAIN}/" | head -5 || true
