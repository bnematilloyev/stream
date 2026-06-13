#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API="${API_URL:-http://localhost:8080}"
EMAIL="${TEST_EMAIL:-creator@sahiy.stream}"
PASS="${TEST_PASS:-password123}"
USER="${TEST_USER:-creator1}"
CHANNEL_SLUG="${TEST_CHANNEL:-creator1}"

bash "${ROOT}/scripts/wait-for-api.sh"

json_get() {
  local key="$1"
  python3 -c "import sys,json; print(json.load(sys.stdin)['${key}'])"
}

echo "==> Register (ignore if exists)"
curl -s -X POST "${API}/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"username\":\"${USER}\",\"display_name\":\"Creator\",\"password\":\"${PASS}\"}" >/dev/null 2>&1 || true

echo "==> Login"
LOGIN=$(curl -s -X POST "${API}/v1/auth/login" -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASS}\"}")
if ! echo "${LOGIN}" | python3 -c "import sys,json; json.load(sys.stdin)['access_token']" >/dev/null 2>&1; then
  echo "Login failed:"
  echo "${LOGIN}"
  exit 1
fi
TOKEN=$(echo "${LOGIN}" | json_get access_token)
AUTH="Authorization: Bearer ${TOKEN}"

echo "==> Create channel (skip if exists)"
CHANNEL_RESP=$(curl -s -w "\n%{http_code}" -X POST "${API}/v1/channels" -H "Content-Type: application/json" -H "${AUTH}" \
  -d "{\"slug\":\"${CHANNEL_SLUG}\",\"title\":\"Creator Channel\",\"description\":\"Live streams\"}")
CHANNEL_BODY=$(echo "${CHANNEL_RESP}" | sed '$d')
CHANNEL_CODE=$(echo "${CHANNEL_RESP}" | tail -n1)
if [[ "${CHANNEL_CODE}" == "201" ]]; then
  echo "${CHANNEL_BODY}" | python3 -m json.tool
elif [[ "${CHANNEL_CODE}" == "409" ]]; then
  echo "Channel already exists, continuing..."
  curl -s "${API}/v1/channels/${CHANNEL_SLUG}" | python3 -m json.tool
else
  echo "Create channel failed (${CHANNEL_CODE}):"
  echo "${CHANNEL_BODY}"
  exit 1
fi

echo "==> Rotate ingest key"
INGEST=$(curl -s -X POST "${API}/v1/channels/${CHANNEL_SLUG}/key/rotate" -H "${AUTH}")
if ! echo "${INGEST}" | python3 -c "import sys,json; json.load(sys.stdin)['stream_key']" >/dev/null 2>&1; then
  echo "Rotate key failed:"
  echo "${INGEST}"
  exit 1
fi
echo "${INGEST}" | python3 -m json.tool

echo "==> Create stream"
STREAM=$(curl -s -X POST "${API}/v1/streams" -H "Content-Type: application/json" -H "${AUTH}" \
  -d "{\"channel_slug\":\"${CHANNEL_SLUG}\",\"title\":\"My First Live\",\"visibility\":\"public\"}")
if ! echo "${STREAM}" | python3 -c "import sys,json; json.load(sys.stdin)['id']" >/dev/null 2>&1; then
  echo "Create stream failed:"
  echo "${STREAM}"
  exit 1
fi
STREAM_ID=$(echo "${STREAM}" | json_get id)
echo "${STREAM}" | python3 -m json.tool

echo "==> Start stream"
curl -s -X POST "${API}/v1/streams/${STREAM_ID}/start" -H "${AUTH}" | python3 -m json.tool

echo "==> List live streams"
curl -s "${API}/v1/streams/live" | python3 -m json.tool

echo "OK - platform flow works"
