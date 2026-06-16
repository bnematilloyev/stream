#!/usr/bin/env bash
# VPS: transcode faqat RunPod GPU worker da. VPS da ffmpeg ishlamasligi kerak.
# Ishlatish (VPS da): bash /opt/sahiy-stream/scripts/ensure-gpu-queue.sh
set -euo pipefail

REMOTE_DIR="${REMOTE_DIR:-/opt/sahiy-stream}"
ENV_FILE="${REMOTE_DIR}/.env"
LOG="${REMOTE_DIR}/.logs"
HOST_IP="${HOST_IP:-$(hostname -I 2>/dev/null | awk '{print $1}')}"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Xato: ${ENV_FILE} topilmadi"
  exit 1
fi

patch_env() {
  local key="$1" val="$2"
  if grep -q "^${key}=" "${ENV_FILE}"; then
    sed -i "s|^${key}=.*|${key}=${val}|" "${ENV_FILE}"
  else
    echo "${key}=${val}" >>"${ENV_FILE}"
  fi
}

echo "==> VPS ffmpeg tozalash (transcode GPU da bo'lishi kerak)..."
pkill -9 -f "ffmpeg -hide_banner" 2>/dev/null || true
sleep 1

echo "==> Queue mode (GPU worker) sozlash..."
patch_env TRANSCODE_MODE queue
patch_env HLS_STORAGE_BACKEND s3
patch_env FFMPEG_VIDEO_ENCODER h264_nvenc
patch_env MINIO_ENDPOINT 127.0.0.1:19000
patch_env MINIO_ACCESS_KEY sahiy_minio
patch_env MINIO_SECRET_KEY sahiy_minio_secret
patch_env MINIO_BUCKET sahiy-media
patch_env MINIO_USE_SSL false
patch_env NATS_URL nats://127.0.0.1:14222
patch_env RTMP_INTERNAL_URL "rtmp://127.0.0.1:1935/live"
patch_env RTMP_BASE_URL "rtmp://${HOST_IP}:1935/live"
patch_env RTSP_INTERNAL_URL "rtsp://127.0.0.1:8554"
patch_env RTSP_WORKER_URL "rtsp://${HOST_IP}:8554"

touch "${REMOTE_DIR}/.gpu-transcode"

echo "==> media-orchestrator qayta ishga tushirish..."
mkdir -p "${LOG}"
pkill -f "${REMOTE_DIR}/bin/media-orchestrator" 2>/dev/null || true
fuser -k 9084/tcp 2>/dev/null || true
sleep 1
nohup env $(grep -v '^#' "${ENV_FILE}" | xargs) "${REMOTE_DIR}/bin/media-orchestrator" >"${LOG}/media-orchestrator.log" 2>&1 &
sleep 2

echo ""
echo "==> Tekshiruv"
echo -n "  TRANSCODE_MODE: "
grep '^TRANSCODE_MODE=' "${ENV_FILE}"
echo -n "  VPS ffmpeg: "
if pgrep -af "ffmpeg -hide_banner" 2>/dev/null | grep -v ffprobe | head -1; then
  echo "  ❌ VPS da transcode ffmpeg bor — bu noto'g'ri!"
  exit 1
else
  echo "  yo'q (to'g'ri — faqat ffprobe ingest uchun bo'lishi mumkin)"
fi

if grep -q "transcode mode: queue" "${LOG}/media-orchestrator.log" 2>/dev/null; then
  echo "  ✅ orchestrator: queue (NATS) — transcode GPU worker ga yuboriladi"
else
  echo "  ⚠️  orchestrator logda 'queue (NATS)' ko'rinmadi:"
  tail -5 "${LOG}/media-orchestrator.log" 2>/dev/null || true
fi

echo ""
echo "VPS vazifasi: RTMP ingest + NATS job dispatch + MinIO storage"
echo "GPU vazifasi: ffmpeg h264_nvenc + HLS segmentlar → MinIO"
echo ""
echo "RunPod tekshiruv:"
echo "  curl -s http://localhost:9086/ready"
echo "  tail -f /opt/transcode-worker/worker.log   # transcode started"
echo "  nvidia-smi   # efir paytida GPU >0%"
