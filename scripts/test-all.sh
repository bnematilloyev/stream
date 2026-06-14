#!/usr/bin/env bash
# go.work modullarida test (root ./... ishlamaydi).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT}"

go work sync

while IFS= read -r dir; do
  dir="${dir#./}"
  echo "==> go test ${dir}/..."
  (cd "${dir}" && go test ./...)
done < <(grep -E '^\s+\./' go.work | sed 's/^[[:space:]]*//')
