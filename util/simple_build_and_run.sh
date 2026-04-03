#!/bin/bash

param=$1

if [ -z "$param" ]; then
    echo "Usage: $0 [--clear-volume|--only-down|--only-up|--restart]"
    exit 1
fi

CURRENT_DIR=$(pwd)
UTIL_DIR=$(dirname "$(realpath "$0")")
REPO_DIR=$(dirname "$UTIL_DIR")
SOURCE_DIR=$REPO_DIR/src
CONFIG_FILE="$SOURCE_DIR/config.env"

if [ ! -f "$CONFIG_FILE" ]; then
    echo "Error: config file not found: $CONFIG_FILE"
    exit 1
fi

# =========================
# Load config
# =========================
set -o allexport
source "$CONFIG_FILE"
set +o allexport

cd "$SOURCE_DIR"

./generate_compose.sh "$CONFIG_FILE"

if [ "$param" == "--clear-volume" ]; then
    sudo rm -rf "$HOST_HOMES_DIR" "$HOST_SHARE_DIR" "$HOST_READONLY_DIR"
    mkdir -p "$HOST_HOMES_DIR" "$HOST_SHARE_DIR" "$HOST_READONLY_DIR"
    sudo docker compose -f "$OUTPUT_FILE" down -v --remove-orphans
    sudo docker compose -f "$OUTPUT_FILE" up -d --build
elif [ "$param" == "--only-down" ]; then
    sudo docker compose -f "$OUTPUT_FILE" down --remove-orphans
elif [ "$param" == "--only-up" ]; then
    sudo docker compose -f "$OUTPUT_FILE" up -d --build
elif [ "$param" == "--restart" ]; then
    sudo docker compose -f "$OUTPUT_FILE" down --remove-orphans
    sudo docker compose -f "$OUTPUT_FILE" up -d --build
fi

cd "$CURRENT_DIR"