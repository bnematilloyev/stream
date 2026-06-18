#!/usr/bin/env bash
# Redis NOAUTH — .env dagi REDIS_URL va container mosligini tekshirish.
# Ishlatish: bash /opt/sahiy-stream/scripts/fix-redis-auth.sh
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
ENV_FILE="${REMOTE_DIR}/.env"

if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -qx sahiy-redis; then
  echo "sahiy-redis container ishlamayapti"
  exit 1
fi

REDIS_URL="$(grep -E '^REDIS_URL=' "${ENV_FILE}" 2>/dev/null | cut -d= -f2- | sed 's/\r$//' || true)"
REDIS_URL="${REDIS_URL:-redis://127.0.0.1:16379/0}"

extract_redis_password() {
  local url="$1"
  # redis://:password@host:port/db  yoki  redis://user:password@host:port/db
  if [[ "${url}" =~ redis://([^:@/]+):([^@/]+)@ ]]; then
    echo "${BASH_REMATCH[2]}"
  elif [[ "${url}" =~ redis://:([^@/]+)@ ]]; then
    echo "${BASH_REMATCH[1]}"
  fi
}

redis_ping() {
  local pass="${1:-}"
  if [[ -n "${pass}" ]]; then
    docker exec sahiy-redis redis-cli -a "${pass}" --no-auth-warning ping 2>/dev/null
  else
    docker exec sahiy-redis redis-cli ping 2>/dev/null
  fi
}

echo "==> Redis tekshiruv"
if redis_ping | grep -q PONG; then
  echo "  OK  parolsiz redis-cli ping"
  exit 0
fi

PASS="$(extract_redis_password "${REDIS_URL}")"
if [[ -n "${PASS}" ]] && redis_ping "${PASS}" | grep -q PONG; then
  echo "  OK  REDIS_URL dagi parol ishlayapti"
  exit 0
fi

echo "  Redis parol talab qilmoqda, lekin .env dagi REDIS_URL mos emas"
echo "  Hozirgi REDIS_URL: ${REDIS_URL%%@*}@***"

# Faqat localhost — eski volume dagi requirepass ni tozalash (cache yo'qolishi mumkin)
echo "  requirepass tozalashga urinilmoqda (faqat 127.0.0.1:16379)..."
if [[ -n "${PASS}" ]]; then
  docker exec sahiy-redis redis-cli -a "${PASS}" --no-auth-warning CONFIG SET requirepass "" 2>/dev/null || true
fi

if redis_ping | grep -q PONG; then
  echo "  OK  requirepass tozalandi"
  if grep -q '^REDIS_URL=' "${ENV_FILE}" 2>/dev/null; then
    sed -i.bak 's|^REDIS_URL=.*|REDIS_URL=redis://127.0.0.1:16379/0|' "${ENV_FILE}"
    echo "  .env REDIS_URL parolsiz qilib yangilandi"
  fi
  exit 0
fi

echo ""
echo "XATO: Redis parolini aniqlab bo'lmadi."
echo "Qo'lda:"
echo "  1) docker exec sahiy-redis redis-cli CONFIG GET requirepass"
echo "  2) .env ga qo'shing: REDIS_URL=redis://:PAROL@127.0.0.1:16379/0"
echo "  3) Yoki cache volume ni qayta yarating (session cache yo'qoladi):"
echo "     cd ${REMOTE_DIR}/infra/docker && docker compose -f docker-compose.prod.yml stop redis"
echo "     docker volume rm docker_redis_data  # nomini: docker volume ls | grep redis"
echo "     docker compose -f docker-compose.prod.yml up -d redis"
exit 1
