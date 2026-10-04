#!/bin/bash
set -euo pipefail

BUILD_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd -- "$BUILD_DIR/.." && pwd)"
TARGET_ARCH="${GOARCH:-$(go env GOARCH)}"

cd "$REPO_DIR/src"
# One static Linux binary provides the CLI and both container services.
CGO_ENABLED=0 GOOS=linux GOARCH="$TARGET_ARCH" go build -trimpath -o "$REPO_DIR/linuxusctl" ./cmd/ctl

echo "[+] Built linuxusctl -> $REPO_DIR/linuxusctl (linux/$TARGET_ARCH)"
