#!/usr/bin/env bash
set -euo pipefail

API="${API_URL:-http://localhost:8080/health}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-30}"

for i in $(seq 1 "${MAX_ATTEMPTS}"); do
  if curl -sf "${API}" >/dev/null 2>&1; then
    echo "API ready (${API})"
    exit 0
  fi
  sleep 1
done

echo "API not ready after ${MAX_ATTEMPTS}s: ${API}"
echo "Check: make stop && make start"
echo "Logs: .logs/"
exit 1
