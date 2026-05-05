#!/bin/bash

set -e

CURRENT_DIR="$(pwd)"
SRC_DIR="$CURRENT_DIR/src"

cd "$SRC_DIR"

SOURCE="cmd/ctl/main.go"
OUTPUT="linuxusctl"

go build -o "$CURRENT_DIR/$OUTPUT" "$SOURCE"

cd "$CURRENT_DIR"