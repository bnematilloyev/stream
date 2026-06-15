#!/usr/bin/env bash
# Live stream diagnostika (lokal yoki serverda).
# Lokal: bash scripts/debug-live.sh [stream-id]
# Server: ssh root@server 'bash /opt/sahiy-stream/scripts/debug-live.sh STREAM_ID'
set -euo pipefail

SID="${1:-}"
REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
LOG="${REMOTE_DIR}/.logs"

echo "=== Sahiy Stream debug ==="
echo "Vaqt: $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo ""

echo "=== FFmpeg ==="
if command -v ffmpeg >/dev/null 2>&1; then
  ffmpeg -version 2>&1 | head -1
else
  echo "❌ ffmpeg topilmadi — ingest ishlamaydi!"
  echo "   Tuzatish: apt-get install -y ffmpeg"
fi
echo ""

echo "=== Servislar ==="
for svc in api-gateway stream-service chat-service media-orchestrator; do
  if pgrep -f "${REMOTE_DIR}/bin/${svc}" >/dev/null 2>&1; then
    echo "  ✅ ${svc}"
  else
    echo "  ❌ ${svc} ishlamayapti"
  fi
done
curl -sf "http://127.0.0.1:$(grep -E '^Gateway port:' "${REMOTE_DIR}/for-deploy.txt" 2>/dev/null | cut -d: -f2- | xargs || echo 18080)/health" >/dev/null 2>&1 && echo "  ✅ gateway /health" || echo "  ❌ gateway /health"
echo ""

echo "=== Docker ==="
docker ps --format '  {{.Names}}: {{.Status}}' 2>/dev/null | grep sahiy || echo "  docker yo'q"
echo ""

echo "=== MediaMTX hook ==="
grep -E 'runOnReady|runOnNotReady' "${REMOTE_DIR}/infra/docker/mediamtx/mediamtx.yml" 2>/dev/null | sed 's/^/  /' || true
if grep -q 'internal_secret=' "${REMOTE_DIR}/infra/docker/mediamtx/mediamtx.yml" 2>/dev/null; then
  echo "  ✅ internal_secret bor"
else
  echo "  ❌ internal_secret yo'q — bash scripts/sync-hook-secrets.sh"
fi
echo ""

if [[ -n "${SID}" ]]; then
  echo "=== Stream: ${SID} ==="
  if docker exec sahiy-postgres psql -U sahiy -d sahiy_stream -t -c \
    "SELECT status, started_at, ended_at FROM streams WHERE id='${SID}';" 2>/dev/null; then
    :
  else
    echo "  DB ulanmadi"
  fi
  echo "  stream_media:"
  docker exec sahiy-postgres psql -U sahiy -d sahiy_stream -c \
    "SELECT stream_id, status, ingest_name, started_at, stopped_at FROM stream_media WHERE stream_id='${SID}';" 2>/dev/null || true
  echo "  HLS fayllar:"
  ls -la "${REMOTE_DIR}/data/hls/${SID}/" 2>&1 | head -5 || true
  echo ""
  echo "  media-orchestrator (stream):"
  grep -i "${SID}" "${LOG}/media-orchestrator.log" 2>/dev/null | tail -8 || echo "  log yo'q"
  echo ""
fi

echo "=== So'nggi media log ==="
tail -12 "${LOG}/media-orchestrator.log" 2>/dev/null || echo "log yo'q"
echo ""

echo "=== Nima uchun stream o'chadi? ==="
echo "  1) FFmpeg yo'q → publish rejected → 90s dan keyin stale cleanup"
echo "  2) Broadcast sahifani refresh → WHIP uziladi → publish_done → ended"
echo "  3) Efini tugatish tugmasi → ended"
echo ""
echo "Tekshirish:"
echo "  curl -s https://api.stream.vibrant.uz/v1/streams/STREAM_ID"
echo "  curl -s https://api.stream.vibrant.uz/v1/streams/STREAM_ID/playback"
