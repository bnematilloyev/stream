#!/usr/bin/env bash
# Lokal mashinadan server portlarini tekshirish (for-deploy.txt orqali SSH).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_FILE="${ROOT}/for-deploy.txt"

if [[ ! -f "${DEPLOY_FILE}" ]]; then
  echo "for-deploy.txt topilmadi. Serverda qo'lda:"
  echo "  bash scripts/check-server-ports.sh"
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

SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=15"

echo "==> Server: ${USER}@${HOST}"
sshpass -p "${PASS}" ssh ${SSH_OPTS} "${USER}@${HOST}" \
  "FRONTEND_PORT=${FRONTEND_PORT} GATEWAY_PORT=${GATEWAY_PORT} HLS_PORT=${HLS_PORT} bash -s" \
  < "${ROOT}/scripts/check-server-ports.sh"
