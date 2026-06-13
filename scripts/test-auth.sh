#!/usr/bin/env bash
set -euo pipefail

API="${API_URL:-http://localhost:8080}"

echo "==> Health check"
curl -sf "${API}/health" | python3 -m json.tool 2>/dev/null || curl -sf "${API}/health"
echo ""

echo "==> Login"
RESP=$(curl -sf -X POST "${API}/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@sahiy.stream","password":"password123"}')

echo "${RESP}" | python3 -m json.tool 2>/dev/null || echo "${RESP}"
echo ""

TOKEN=$(echo "${RESP}" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])" 2>/dev/null)

if [[ -z "${TOKEN}" ]]; then
  echo "Login failed or no token returned"
  exit 1
fi

echo "==> Me (with token)"
curl -sf "${API}/v1/auth/me" \
  -H "Authorization: Bearer ${TOKEN}" | python3 -m json.tool 2>/dev/null || \
curl -sf "${API}/v1/auth/me" -H "Authorization: Bearer ${TOKEN}"

echo ""
echo "OK - auth flow works"
