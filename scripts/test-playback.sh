#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API="${API_URL:-http://localhost:8080}"
HLS="${HLS_URL:-http://localhost:8090}"

echo "== Playback API test =="

TS=$(date +%s)
EMAIL="playback_${TS}@test.local"
USER="pb_${TS}"
PASS="TestPass123!"
DISPLAY="Playback Tester"

curl -sf -X POST "${API}/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"username\":\"${USER}\",\"display_name\":\"${DISPLAY}\",\"password\":\"${PASS}\"}" >/dev/null 2>&1 || true

login=$(curl -sf -X POST "${API}/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${EMAIL}\",\"password\":\"${PASS}\"}")
TOKEN=$(echo "$login" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])")

channel=$(curl -sf -X POST "${API}/v1/channels" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"title":"Playback Channel","slug":"pb-'$(date +%s)'"}')
SLUG=$(echo "$channel" | python3 -c "import sys,json; print(json.load(sys.stdin)['slug'])")

stream=$(curl -sf -X POST "${API}/v1/streams" \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"channel_slug\":\"${SLUG}\",\"title\":\"Playback Stream Test\"}")
STREAM_ID=$(echo "$stream" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

curl -sf -X POST "${API}/v1/streams/${STREAM_ID}/start" \
  -H "Authorization: Bearer ${TOKEN}" > /dev/null

playback=$(curl -sf "${API}/v1/streams/${STREAM_ID}/playback")
URL=$(echo "$playback" | python3 -c "import sys,json; print(json.load(sys.stdin)['url'])")

echo "Playback URL: ${URL}"

if [[ "$URL" != *"master.m3u8"* ]] || [[ "$URL" != *"sig="* ]]; then
  echo "FAIL: playback URL must be signed HLS manifest"
  exit 1
fi

if curl -sf "${HLS}/health" > /dev/null; then
  echo "HLS origin: OK"
else
  echo "WARN: HLS origin not reachable at ${HLS}/health"
fi

echo "OK - playback endpoint works"
