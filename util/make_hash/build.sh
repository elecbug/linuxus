#!/bin/bash

set -e

CURRENT_DIR=$(pwd)
SOURCE_DIR=$(dirname "$(realpath "$0")")
SOURCE="main.go"
OUTPUT="make_hash.out"

cd "$SOURCE_DIR"

go build -o "../$OUTPUT" "$SOURCE"

cd "$CURRENT_DIR"