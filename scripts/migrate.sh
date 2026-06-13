#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="${ROOT}/services/auth-service/migrations"
DATABASE_URL="${DATABASE_URL:-postgres://sahiy:sahiy_secret@localhost:5433/sahiy_stream?sslmode=disable}"

MIGRATE_BIN="${MIGRATE_BIN:-migrate}"
if ! command -v "${MIGRATE_BIN}" &>/dev/null; then
  if command -v "${HOME}/go/bin/migrate" &>/dev/null; then
    MIGRATE_BIN="${HOME}/go/bin/migrate"
  else
    echo "golang-migrate not found. Install:"
    echo "  go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    echo "  export PATH=\"\$HOME/go/bin:\$PATH\""
    exit 1
  fi
fi

ACTION="${1:-up}"
case "${ACTION}" in
  up)
    "${MIGRATE_BIN}" -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" up
    ;;
  down)
    STEPS="${2:-1}"
    "${MIGRATE_BIN}" -path "${MIGRATIONS_DIR}" -database "${DATABASE_URL}" down "${STEPS}"
    ;;
  *)
    echo "Usage: migrate.sh [up|down] [steps]"
    exit 1
    ;;
esac
