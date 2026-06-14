#!/usr/bin/env bash
# Sahiy Stream port audit — serverda yoki lokal ishlatish.
# Boshqa loyihalar bilan to'qnashuvni ko'rsatadi.
set -euo pipefail

FRONTEND_PORT="${FRONTEND_PORT:-3002}"
GATEWAY_PORT="${GATEWAY_PORT:-8080}"
HLS_PORT="${HLS_PORT:-8090}"

declare -A PORTS=(
  [80]="nginx HTTP (umumiy)"
  [443]="nginx HTTPS (umumiy)"
  [${FRONTEND_PORT}]="Sahiy Next.js frontend"
  [${GATEWAY_PORT}]="Sahiy API gateway"
  [50051]="auth-service gRPC"
  [50052]="user-service gRPC"
  [50053]="stream-service gRPC"
  [50054]="chat-service gRPC"
  [9084]="media-orchestrator"
  [9085]="chat-service HTTP"
  [15433]="Postgres (localhost only)"
  [16379]="Redis (localhost only)"
  [14222]="NATS (localhost only)"
  [1935]="RTMP ingest (public)"
  [8554]="RTSP (public)"
  [8889]="WHIP/WebRTC (public)"
  [8189]="WebRTC ICE TCP/UDP (public)"
  [${HLS_PORT}]="Sahiy HLS origin (host map)"
  [8090]="Sahiy HLS default"
  [8088]="nginx-rtmp stats"
  [9997]="MediaMTX API"
)

port_in_use() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -tuln | grep -qE ":${port}\b"
    return $?
  fi
  if command -v lsof >/dev/null 2>&1; then
    lsof -i ":${port}" -sTCP:LISTEN >/dev/null 2>&1
    return $?
  fi
  return 1
}

port_process() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -tulpn 2>/dev/null | grep -E ":${port}\b" | head -1 || true
    return
  fi
  lsof -i ":${port}" -sTCP:LISTEN 2>/dev/null | tail -n +2 | head -1 || true
}

echo "Sahiy Stream — port tekshiruvi"
echo "Frontend port: ${FRONTEND_PORT}  Gateway port: ${GATEWAY_PORT}"
echo ""

busy=0
free=0

for port in $(printf '%s\n' "${!PORTS[@]}" | sort -n); do
  label="${PORTS[$port]}"
  if port_in_use "$port"; then
    busy=$((busy + 1))
    proc=$(port_process "$port")
    echo "  BUSY  :${port}  ${label}"
    [[ -n "${proc}" ]] && echo "         ${proc}"
  else
    free=$((free + 1))
    echo "  FREE  :${port}  ${label}"
  fi
done

echo ""
echo "Jami: ${free} bo'sh, ${busy} band"

if port_in_use "${FRONTEND_PORT}"; then
  echo ""
  echo "⚠ Frontend port ${FRONTEND_PORT} band — for-deploy.txt da boshqa port tanlang (masalan 3010)"
fi
if port_in_use "${GATEWAY_PORT}"; then
  echo "⚠ Gateway port ${GATEWAY_PORT} band — GATEWAY_HTTP_ADDR o'zgartirish kerak (deploy hali default 8080)"
fi
if port_in_use 8090; then
  echo "⚠ HLS :8090 band — docker-compose.prod.yml da hls-origin portini o'zgartiring"
fi
if port_in_use 8889; then
  echo "⚠ WHIP :8889 band — mediamtx portini o'zgartirish kerak"
fi
if port_in_use 1935; then
  echo "⚠ RTMP :1935 band — nginx-rtmp portini o'zgartirish kerak"
fi

if [[ "${busy}" -gt 0 ]]; then
  exit 1
fi
