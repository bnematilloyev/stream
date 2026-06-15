#!/usr/bin/env bash
# RTMP ingest: firewall + RTMP_BASE_URL + servislar (lokal yoki serverda).
# Lokal: bash scripts/fix-rtmp-ingest.sh
# Remote: bash scripts/fix-rtmp-ingest.sh  (for-deploy.txt kerak)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DEPLOY_FILE="${ROOT}/for-deploy.txt"
REMOTE_DIR="/opt/sahiy-stream"

run_remote() {
  if [[ ! -f "${DEPLOY_FILE}" ]]; then
    echo "for-deploy.txt topilmadi"
    exit 1
  fi
  read_deploy() { grep -E "^${1}:" "${DEPLOY_FILE}" | cut -d: -f2- | xargs || true; }
  HOST=$(read_deploy "IP-manzil")
  USER=$(read_deploy "Foydalanuvchi nomi")
  PASS=$(read_deploy "Parol")
  if [[ -z "${HOST}" || -z "${USER}" || -z "${PASS}" ]]; then
    echo "for-deploy.txt da IP, user yoki parol yo'q"
    exit 1
  fi
  if ! command -v sshpass >/dev/null 2>&1; then
    echo "sshpass kerak: brew install sshpass"
    exit 1
  fi
  SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=15"
  sshpass -p "${PASS}" scp ${SSH_OPTS} \
    "${ROOT}/scripts/fix-rtmp-ingest.sh" \
    "${USER}@${HOST}:${REMOTE_DIR}/scripts/fix-rtmp-ingest.sh"
  sshpass -p "${PASS}" scp ${SSH_OPTS} \
    "${ROOT}/scripts/sync-hook-secrets.sh" \
    "${USER}@${HOST}:${REMOTE_DIR}/scripts/sync-hook-secrets.sh"
  sshpass -p "${PASS}" scp ${SSH_OPTS} \
    "${ROOT}/infra/docker/nginx-rtmp/nginx.conf" \
    "${USER}@${HOST}:${REMOTE_DIR}/infra/docker/nginx-rtmp/nginx.conf"
  sshpass -p "${PASS}" scp ${SSH_OPTS} \
    "${ROOT}/infra/docker/mediamtx/mediamtx.yml" \
    "${USER}@${HOST}:${REMOTE_DIR}/infra/docker/mediamtx/mediamtx.yml"
  sshpass -p "${PASS}" ssh ${SSH_OPTS} "${USER}@${HOST}" "chmod +x ${REMOTE_DIR}/scripts/fix-rtmp-ingest.sh ${REMOTE_DIR}/scripts/sync-hook-secrets.sh && REMOTE_ONLY=1 bash ${REMOTE_DIR}/scripts/fix-rtmp-ingest.sh"

  if [[ "${DEPLOY_FRONTEND:-}" == "1" && -d "${ROOT}/frontend/.next" ]]; then
    echo "==> Frontend yangilanishi..."
    local archive="/tmp/sahiy-frontend-patch.tar.gz"
    tar -czf "${archive}" -C "${ROOT}/frontend" .next
    sshpass -p "${PASS}" scp ${SSH_OPTS} "${archive}" "${USER}@${HOST}:${REMOTE_DIR}/frontend-patch.tar.gz"
    sshpass -p "${PASS}" ssh ${SSH_OPTS} "${USER}@${HOST}" bash -s <<REMOTE
set -euo pipefail
REMOTE_DIR="${REMOTE_DIR}"
FRONTEND_PORT="\$(grep -E '^Frontend port:' "\${REMOTE_DIR}/for-deploy.txt" 2>/dev/null | cut -d: -f2- | xargs || echo 3010)"
cd "\${REMOTE_DIR}/frontend"
tar -xzf ../frontend-patch.tar.gz
rm -f ../frontend-patch.tar.gz
pkill -f "next start -p \${FRONTEND_PORT}" 2>/dev/null || true
fuser -k "\${FRONTEND_PORT}/tcp" 2>/dev/null || true
sleep 1
nohup ./node_modules/.bin/next start -p "\${FRONTEND_PORT}" -H 0.0.0.0 >"\${REMOTE_DIR}/.logs/frontend.log" 2>&1 &
echo "  frontend qayta ishga tushirildi (port \${FRONTEND_PORT})"
REMOTE
    rm -f "${archive}"
  fi
}

fix_on_host() {
  local host_ip="${HOST_IP:-}"
  if [[ -z "${host_ip}" && -f "${REMOTE_DIR}/deploy-host.txt" ]]; then
    host_ip="$(cat "${REMOTE_DIR}/deploy-host.txt")"
  fi
  if [[ -z "${host_ip}" && -f "${REMOTE_DIR}/for-deploy.txt" ]]; then
    host_ip="$(grep -E '^IP-manzil:' "${REMOTE_DIR}/for-deploy.txt" | cut -d: -f2- | xargs || true)"
  fi
  if [[ -z "${host_ip}" ]]; then
    host_ip="$(hostname -I | tr ' ' '\n' | grep -E '^[0-9]+\.' | head -1)"
  fi
  echo "==> RTMP ingest tuzatish (IP: ${host_ip})"

  if command -v ufw >/dev/null 2>&1; then
    ufw allow 1935/tcp comment "RTMP ingest" 2>/dev/null || true
    ufw reload 2>/dev/null || true
    echo "  ufw: 1935/tcp"
  fi

  local env_file="${REMOTE_DIR}/.env"
  if [[ -f "${env_file}" ]]; then
    if grep -q '^RTMP_BASE_URL=' "${env_file}"; then
      sed -i "s|^RTMP_BASE_URL=.*|RTMP_BASE_URL=rtmp://${host_ip}:1935/live|" "${env_file}"
    else
      echo "RTMP_BASE_URL=rtmp://${host_ip}:1935/live" >>"${env_file}"
    fi
    echo "  RTMP_BASE_URL=rtmp://${host_ip}:1935/live"
  fi

  if [[ -f "${REMOTE_DIR}/scripts/sync-hook-secrets.sh" ]]; then
    REMOTE_DIR="${REMOTE_DIR}" bash "${REMOTE_DIR}/scripts/sync-hook-secrets.sh"
  fi

  cd "${REMOTE_DIR}/infra/docker" 2>/dev/null && {
    docker compose -f docker-compose.prod.yml up -d nginx-rtmp
    sleep 3
    docker ps --filter name=nginx-rtmp --format '  container: {{.Names}} {{.Status}}'
    docker logs sahiy-nginx-rtmp 2>&1 | tail -5 | sed 's/^/  log: /' || true
    ss -tlnp 2>/dev/null | grep 1935 | sed 's/^/  /' || echo "  :1935 hostda tinglanmayapti"
  } || echo "  ogohlantirish: docker compose yo'q"

  if pgrep -f "${REMOTE_DIR}/bin/user-service" >/dev/null 2>&1; then
    pkill -f "${REMOTE_DIR}/bin/user-service" || true
    sleep 1
    LOG="${REMOTE_DIR}/.logs"
    mkdir -p "${LOG}"
    nohup env $(grep -v '^#' "${env_file}" | xargs) "${REMOTE_DIR}/bin/user-service" >"${LOG}/user-service.log" 2>&1 &
    echo "  user-service qayta ishga tushirildi"
  fi

  if ss -tlnp 2>/dev/null | grep -q ':1935 '; then
    echo "  nginx-rtmp :1935 tinglanmoqda"
  else
    echo "  ogohlantirish: :1935 hali tinglanmayapti — docker ps | grep nginx-rtmp"
  fi

  echo ""
  echo "OBS sozlamalari:"
  echo "  Server:     rtmp://${host_ip}:1935/live"
  echo "  Stream Key: SahiyStream → Key yangilash"
  echo ""
  echo "Cloud panelda ham TCP 1935 ochilganini tekshiring (Contabo/Hetzner firewall)."
}

if [[ "${REMOTE_ONLY:-}" == "1" ]]; then
  fix_on_host
else
  if [[ -f "${DEPLOY_FILE}" ]]; then
    run_remote
  else
    REMOTE_DIR="${ROOT}" HOST_IP="127.0.0.1" fix_on_host
  fi
fi
