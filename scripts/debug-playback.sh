#!/usr/bin/env bash
# Playback diagnostika — serverda yoki lokal.
# Server: bash /opt/sahiy-stream/scripts/debug-playback.sh STREAM_ID
set -euo pipefail

SID="${1:-}"
REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
API_DOMAIN="${API_DOMAIN:-api.stream.vibrant.uz}"
GATEWAY_PORT="$(grep -E '^Gateway port:' "${REMOTE_DIR}/for-deploy.txt" 2>/dev/null | cut -d: -f2- | xargs || echo 18080)"

if [[ -z "${SID}" ]]; then
  echo "Ishlatish: bash scripts/debug-playback.sh STREAM_ID"
  exit 1
fi

echo "=== Playback debug: ${SID} ==="
echo ""

echo "=== HLS fayl ==="
HLS="${REMOTE_DIR}/data/hls/${SID}/master.m3u8"
if [[ -f "${HLS}" ]]; then
  echo "  OK  ${HLS} ($(wc -c <"${HLS}") bytes)"
  head -5 "${HLS}" | sed 's/^/    /'
else
  echo "  YO'Q  ${HLS}"
  echo "  Sabab: OBS stream yo'q yoki FFmpeg ishlamagan"
fi
echo ""

echo "=== DB ==="
docker exec sahiy-postgres psql -U sahiy -d sahiy_stream -t -c \
  "SELECT s.status, COALESCE(sm.status,'-') FROM streams s LEFT JOIN stream_media sm ON s.id=sm.stream_id WHERE s.id='${SID}';" 2>/dev/null | sed 's/^/  /' || echo "  DB ulanmadi"
echo ""

echo "=== FFmpeg ==="
if pgrep -af '[f]fmpeg' >/dev/null 2>&1; then
  pgrep -af '[f]fmpeg' | sed 's/^/  /'
else
  echo "  ffmpeg ishlamayapti"
fi
echo ""

echo "=== Playback API (gateway) ==="
PB_JSON="$(curl -sf "http://127.0.0.1:${GATEWAY_PORT}/v1/streams/${SID}/playback" 2>/dev/null || true)"
if [[ -z "${PB_JSON}" ]]; then
  echo "  API javob bermadi (stream live+ingesting emas yoki gateway o'chiq)"
else
  echo "${PB_JSON}" | python3 -m json.tool 2>/dev/null | sed 's/^/  /' || echo "  ${PB_JSON}"
  URL="$(echo "${PB_JSON}" | python3 -c "import sys,json; print(json.load(sys.stdin).get('url',''))" 2>/dev/null || true)"
  if [[ -n "${URL}" ]]; then
    echo ""
    echo "=== stream-service (to'g'ridan-to'g'ri :9083) ==="
    PATH_QS="${URL#https://${API_DOMAIN}}"
    CODE="$(curl -s -o /tmp/pb-test.m3u8 -w '%{http_code}' "http://127.0.0.1:9083${PATH_QS}")"
    echo "  HTTP ${CODE}"
    if [[ "${CODE}" == "200" ]]; then
      echo "  OK — manifest rewrite tekshiruvi:"
      grep -E 'playlist.m3u8|\.m4s|\.ts' /tmp/pb-test.m3u8 | head -3 | sed 's/^/    /'
      if grep -q 'sig=' /tmp/pb-test.m3u8; then
        echo "  OK  ichki URLlar imzolangan"
      else
        echo "  XATO  ichki URLlar imzosiz — stream-service yangilang"
      fi
    elif [[ "${CODE}" == "401" ]]; then
      echo "  XATO  unauthorized — PLAYBACK_SIGNING_SECRET mos emas"
    elif [[ "${CODE}" == "404" ]]; then
      echo "  XATO  not found — HLS fayl yo'q"
    fi
    rm -f /tmp/pb-test.m3u8
  fi
fi
echo ""

echo "=== PLAYBACK_SIGNING_SECRET ==="
SECRET="$(grep -E '^PLAYBACK_SIGNING_SECRET=' "${REMOTE_DIR}/.env" 2>/dev/null | cut -d= -f2- | xargs || true)"
if [[ -n "${SECRET}" ]]; then
  echo "  bor (${#SECRET} belgi)"
else
  echo "  YO'Q — .env da PLAYBACK_SIGNING_SECRET topilmadi"
fi
echo ""

echo "Eslatma: curl da & belgisini \\u0026 emas, to'g'ridan-to'g'ri ishlating:"
echo "  curl -s \"https://${API_DOMAIN}/playback/${SID}/master.m3u8?exp=EXP&sig=SIG\""
