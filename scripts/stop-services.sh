#!/usr/bin/env bash
set -euo pipefail

PORTS=(50051 50052 50053 9081 9082 9083 9084 8080)

echo "Stopping Sahiy Stream dev services..."

for port in "${PORTS[@]}"; do
  pids=$(lsof -ti :"${port}" 2>/dev/null || true)
  if [[ -n "${pids}" ]]; then
    echo "  Killing port ${port}: ${pids}"
    kill ${pids} 2>/dev/null || true
  fi
done

sleep 1
echo "Done."
