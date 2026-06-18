#!/usr/bin/env bash
# Production .env ni xavfsiz yuklash (CRLF va xargs muammolaridan qochish).
REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
ENV_FILE="${ENV_FILE:-${REMOTE_DIR}/.env}"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "load-prod-env: ${ENV_FILE} topilmadi" >&2
  return 1 2>/dev/null || exit 1
fi

set -a
# shellcheck disable=SC1090
source <(grep -v '^#' "${ENV_FILE}" | sed 's/\r$//')
set +a
