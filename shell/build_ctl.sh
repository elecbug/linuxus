#!/bin/bash

set -e

CURRENT_DIR="$(pwd)"
BUILD_DIR="$(dirname "$0")"
REPO_DIR="$(dirname "$BUILD_DIR")"
SRC_DIR="$REPO_DIR/src"

cd "$SRC_DIR"

SOURCE="cmd/ctl/main.go"
OUTPUT="linuxusctl"

go build -o "../$OUTPUT" "$SOURCE"

echo "[+] Built $OUTPUT successfully -> $REPO_DIR/$OUTPUT"

cd "$CURRENT_DIR"