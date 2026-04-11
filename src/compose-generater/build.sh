#!/bin/bash

set -e

CURRENT_DIR=$(pwd)
SOURCE_DIR=$(dirname "$(realpath "$0")")
SOURCE="cmd/main.go"
OUTPUT="linuxusctl"

cd "$SOURCE_DIR"

go build -o "../$OUTPUT" "$SOURCE"

cd "$CURRENT_DIR"