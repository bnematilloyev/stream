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

CTRL_DIR="${TMPDIR:-/tmp}/sahiy-ssh-$$"
mkdir -p "${CTRL_DIR}"
CTRL_SOCK="${CTRL_DIR}/ctrl.sock"
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=15 -o ServerAliveInterval=15 -o ServerAliveCountMax=4 -o ControlMaster=auto -o ControlPath=${CTRL_SOCK} -o ControlPersist=120"

cleanup_ssh() {
  sshpass -p "${PASS}" ssh ${SSH_OPTS} -O exit "${USER}@${HOST}" 2>/dev/null || true
  rm -rf "${CTRL_DIR}"
}
trap cleanup_ssh EXIT

ssh_cmd() {
  sshpass -p "${PASS}" ssh ${SSH_OPTS} "${USER}@${HOST}" "$@"
}

scp_cmd() {
  sshpass -p "${PASS}" scp ${SSH_OPTS} "$@"
}

retry_cmd() {
  local attempt=1
  local max=3
  local delay=3
  while true; do
    if "$@"; then
      return 0
    fi
    if (( attempt >= max )); then
      return 1
    fi
    echo "    qayta urinish (${attempt}/${max})..."
    sleep "${delay}"
    attempt=$((attempt + 1))
    delay=$((delay * 2))
  done
}

prime_ssh() {
  retry_cmd ssh_cmd "true"
}

if ! command -v sshpass >/dev/null 2>&1; then
  echo "sshpass kerak: brew install hudochenkov/sshpass/sshpass"
  exit 1
fi

echo "==> Go binarylarni build qilish (linux/amd64)..."
export GOOS=linux GOARCH=amd64
mkdir -p "${ROOT}/bin"
for svc in auth-service user-service stream-service chat-service media-orchestrator transcode-worker api-gateway; do
  echo "    ${svc}..."
  (cd "${ROOT}/services/${svc}" && go build -o "${ROOT}/bin/${svc}" ./cmd/server)
done

prime_ssh

echo "==> Binarylarni yuklash (staging)..."
STAGING="/opt/sahiy-stream/bin-upload"
ssh_cmd "mkdir -p /opt/sahiy-stream/scripts ${STAGING}"
if [[ -d "${ROOT}/bin" ]] && compgen -G "${ROOT}/bin/"* >/dev/null; then
  echo "    Yuklanmoqda (6 ta binary, 1-3 daqiqa)..."
  scp_cmd "${ROOT}/bin/"* "${USER}@${HOST}:${STAGING}/"
  echo "    Yuklandi. Serverda almashtirilmoqda..."
  ssh_cmd bash -s "${FRONTEND_PORT}" "${GATEWAY_PORT}" <<'REMOTE'
set -euo pipefail
FRONTEND_PORT="$1"
GATEWAY_PORT="$2"
REMOTE_DIR="/opt/sahiy-stream"
STAGING="/opt/sahiy-stream/bin-upload"

echo "  servislarni to'xtatish..."
pkill -f "${REMOTE_DIR}/bin/" 2>/dev/null || true
pkill -f "next start -p ${FRONTEND_PORT}" 2>/dev/null || true
pkill -f "${REMOTE_DIR}/frontend/node_modules/.bin/next start" 2>/dev/null || true
sleep 3

echo "  binarylarni o'rnatish..."
mkdir -p "${REMOTE_DIR}/bin"
for f in "${STAGING}"/*; do
  name="$(basename "$f")"
  install -m 755 "$f" "${REMOTE_DIR}/bin/${name}.new"
  mv -f "${REMOTE_DIR}/bin/${name}.new" "${REMOTE_DIR}/bin/${name}"
done
rm -rf "${STAGING}"
echo "  binarylar yangilandi"
REMOTE
else
  echo "    bin/ bo'sh — faqat skriptlar yangilanadi"
fi
echo "==> Skriptlar va konfiguratsiya..."
AUX_ARCHIVE="$(mktemp /tmp/sahiy-deploy-aux.XXXXXX.tar.gz)"
tar -czf "${AUX_ARCHIVE}" \
  -C "${ROOT}" \
  scripts/deploy-remote-only.sh \
  scripts/setup-nginx-ssl.sh \
  scripts/sync-hook-secrets.sh \
  infra/nginx/api.stream.vibrant.uz.conf \
  infra/nginx/stream.vibrant.uz.conf \
  infra/docker/nginx-rtmp/nginx.conf \
  scripts/debug-playback.sh \
  frontend/next.config.mjs
retry_cmd scp_cmd "${AUX_ARCHIVE}" "${USER}@${HOST}:/tmp/sahiy-deploy-aux.tar.gz"
rm -f "${AUX_ARCHIVE}"
ssh_cmd bash -s <<'REMOTE'
set -euo pipefail
REMOTE_DIR="/opt/sahiy-stream"
tar -xzf /tmp/sahiy-deploy-aux.tar.gz -C "${REMOTE_DIR}"
rm -f /tmp/sahiy-deploy-aux.tar.gz
chmod +x "${REMOTE_DIR}/scripts/deploy-remote-only.sh" \
  "${REMOTE_DIR}/scripts/setup-nginx-ssl.sh" \
  "${REMOTE_DIR}/scripts/sync-hook-secrets.sh" \
  "${REMOTE_DIR}/scripts/debug-playback.sh"
REMOTE_DIR="${REMOTE_DIR}" bash "${REMOTE_DIR}/scripts/sync-hook-secrets.sh"
REMOTE

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
