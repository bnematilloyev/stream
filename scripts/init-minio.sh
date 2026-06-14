#!/usr/bin/env bash
set -euo pipefail

ENDPOINT="${MINIO_ENDPOINT:-localhost:9000}"
ACCESS_KEY="${MINIO_ACCESS_KEY:-sahiy_minio}"
SECRET_KEY="${MINIO_SECRET_KEY:-sahiy_minio_secret}"
BUCKET="${MINIO_BUCKET:-sahiy-media}"

echo "==> Ensuring MinIO bucket: ${BUCKET}"
docker run --rm --network host minio/mc:latest alias set local "http://${ENDPOINT}" "${ACCESS_KEY}" "${SECRET_KEY}"
docker run --rm --network host minio/mc:latest mb --ignore-existing "local/${BUCKET}"
docker run --rm --network host minio/mc:latest anonymous set download "local/${BUCKET}/hls" 2>/dev/null || true
echo "MinIO bucket ready"
