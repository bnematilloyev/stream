#!/usr/bin/env bash
# RunPod Web Terminal da ishga tushiring.
# Masalan: VPS_IP=38.242.216.233 bash runpod-worker-bootstrap.sh
set -euo pipefail

VPS_IP="${VPS_IP:-}"
if [[ -z "${VPS_IP}" ]]; then
  echo "VPS_IP kerak. Masalan: VPS_IP=38.242.216.233 bash $0"
  exit 1
fi

echo "==> GPU tekshiruv..."
nvidia-smi --query-gpu=name,memory.total --format=csv,noheader
ffmpeg -hide_banner -encoders 2>/dev/null | grep -E 'h264_nvenc|hevc_nvenc' || {
  echo "NVENC topilmadi — ffmpeg o'rnatilmoqda..."
  apt-get update -qq && apt-get install -y -qq ffmpeg
}

mkdir -p /opt/transcode-worker /tmp/hls
cd /opt/transcode-worker

echo "==> transcode-worker yuklanmoqda..."
curl -fsSL -o transcode-worker "http://${VPS_IP}:19999/transcode-worker"
chmod +x transcode-worker

cat > .env <<EOF
APP_ENV=production
LOG_LEVEL=info
WORKER_HTTP_ADDR=:9086
WORKER_ID=runpod-gpu-1
WORKER_MAX_JOBS=4
NATS_URL=nats://${VPS_IP}:14222
FFMPEG_PATH=ffmpeg
FFMPEG_VIDEO_ENCODER=h264_nvenc
TRANSCODE_QUALITY=production
HLS_OUTPUT_DIR=/tmp/hls
HLS_STORAGE_BACKEND=s3
MINIO_ENDPOINT=${VPS_IP}:19000
MINIO_ACCESS_KEY=sahiy_minio
MINIO_SECRET_KEY=sahiy_minio_secret
MINIO_BUCKET=sahiy-media
MINIO_USE_SSL=false
EOF

pkill -f '/opt/transcode-worker/transcode-worker' 2>/dev/null || true
nohup env $(grep -v '^#' .env | xargs) ./transcode-worker >worker.log 2>&1 &
sleep 3

echo "==> Health..."
curl -sf http://localhost:9086/health && echo ""
curl -sf http://localhost:9086/ready && echo "" || echo "WARN: NATS ulanmadi — VPS setup-gpu-worker.sh ishlaganini tekshiring"

echo "OK: transcode-worker ishlayapti"
echo "Log: tail -f /opt/transcode-worker/worker.log"
