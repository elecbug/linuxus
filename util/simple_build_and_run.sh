#!/bin/bash

param=$1

if [ -z "$param" ]; then
    echo "Usage: $0 [--clear|--down|--up|--restart]"
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

if [ "$param" == "--clear" ]; then
    sudo docker compose -f "$OUTPUT_FILE" down -v --remove-orphans

    find "$HOST_HOMES_DIR" -mindepth 1 -type d 2>/dev/null \
        | awk '{ print length, $0 }' \
        | sort -rn \
        | cut -d' ' -f2- \
        | xargs -r -n1 sudo umount 2>/dev/null || true

    sudo rm -rf "$HOST_HOMES_DIR" "$HOST_SHARE_DIR" "$HOST_READONLY_DIR"
    sudo mkdir -p "$HOST_HOMES_DIR" "$HOST_SHARE_DIR" "$HOST_READONLY_DIR"
    sudo chown ${CONTAINER_RUNTIME_UID}:${CONTAINER_RUNTIME_GID} "$HOST_SHARE_DIR" "$HOST_READONLY_DIR"
elif [ "$param" == "--down" ]; then
    sudo docker compose -f "$OUTPUT_FILE" down --remove-orphans
elif [ "$param" == "--up" ]; then
    ./generate_compose.sh "$CONFIG_FILE"
    
    sudo docker compose -f "$OUTPUT_FILE" up -d --build
elif [ "$param" == "--restart" ]; then
    sudo docker compose -f "$OUTPUT_FILE" down --remove-orphans

    ./generate_compose.sh "$CONFIG_FILE"
    
    sudo docker compose -f "$OUTPUT_FILE" up -d --build
fi

cd "$CURRENT_DIR"