#!/usr/bin/env bash
# Bitta buyruq: Docker + migrate + servislar + testlar
set -euo pipefail
cd "$(dirname "$0")"
make smoke
