#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PROTO_DIR="${ROOT}/proto"
OUT_DIR="${ROOT}/proto/gen"

if ! command -v protoc &>/dev/null; then
  echo "protoc not found. Install Protocol Buffers compiler."
  exit 1
fi

mkdir -p "${OUT_DIR}/auth/v1" "${OUT_DIR}/user/v1" "${OUT_DIR}/stream/v1"

protoc \
  --proto_path="${PROTO_DIR}" \
  --go_out="${OUT_DIR}" --go_opt=paths=source_relative \
  --go-grpc_out="${OUT_DIR}" --go-grpc_opt=paths=source_relative \
  "${PROTO_DIR}/auth/v1/auth.proto" \
  "${PROTO_DIR}/user/v1/user.proto" \
  "${PROTO_DIR}/stream/v1/stream.proto"

echo "Proto generated in ${OUT_DIR}"
# stream proto only - media hooks are HTTP on media-orchestrator
