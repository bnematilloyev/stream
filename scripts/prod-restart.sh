#!/usr/bin/env bash
# Production servislarni qayta ishga tushirish.
# Ishlatish: bash scripts/prod-restart.sh
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
SCRIPT="${REMOTE_DIR}/scripts/deploy-remote-only.sh"

if [[ ! -f "${SCRIPT}" ]]; then
  echo "deploy-remote-only.sh topilmadi: ${SCRIPT}"
  exit 1
fi

bash "${SCRIPT}"
