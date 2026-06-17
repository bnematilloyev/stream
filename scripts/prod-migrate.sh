#!/usr/bin/env bash
# Production migration (/opt/sahiy-stream).
# Ishlatish: bash scripts/prod-migrate.sh
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
ENV_FILE="${REMOTE_DIR}/.env"
MIGRATIONS_DIR="${REMOTE_DIR}/migrations"

if [[ ! -d "${MIGRATIONS_DIR}" ]]; then
  echo "Migrations topilmadi: ${MIGRATIONS_DIR}"
  echo "Deploy qiling yoki: mv services/auth-service/migrations migrations"
  exit 1
fi

if [[ -f "${ENV_FILE}" ]]; then
  # shellcheck disable=SC1090
  set -a
  source <(grep -E '^(DATABASE_URL|APP_ENV)=' "${ENV_FILE}" | sed 's/\r$//')
  set +a
fi

DATABASE_URL="${DATABASE_URL:-postgres://sahiy:sahiy_secret@127.0.0.1:15433/sahiy_stream?sslmode=disable}"

MIGRATE_BIN="${MIGRATE_BIN:-migrate}"
if ! command -v "${MIGRATE_BIN}" >/dev/null 2>&1; then
  echo "migrate binary yo'q. O'rnatish:"
  echo "  curl -fsSL -o /tmp/migrate.tgz https://github.com/golang-migrate/migrate/releases/download/v4.17.1/migrate.linux-amd64.tar.gz"
  echo "  tar -xzf /tmp/migrate.tgz -C /tmp migrate && mv /tmp/migrate /usr/local/bin/migrate"
  exit 1
fi

ACTION="${1:-up}"
case "${ACTION}" in
  up)
    echo "==> Migration up (${MIGRATIONS_DIR})"
    "${MIGRATE_BIN}" -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" up
    ;;
  down)
    STEPS="${2:-1}"
    echo "==> Migration down ${STEPS} step"
    "${MIGRATE_BIN}" -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" down "${STEPS}"
    ;;
  version)
    "${MIGRATE_BIN}" -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" version
    ;;
  *)
    echo "Usage: prod-migrate.sh [up|down|version] [steps]"
    exit 1
    ;;
esac

echo "OK"
