#!/bin/bash

set -euo pipefail

CURRENT_DIR="$(pwd)"
UTIL_DIR="$(dirname "$(realpath "$0")")"
REPO_DIR="$(dirname "$UTIL_DIR")"
SOURCE_DIR="$REPO_DIR/src"
CONFIG_FILE="$SOURCE_DIR/config.env"

usage() {
    cat <<EOF
Usage:
  $0 [OPTION]...

Options:
  -g, --generate      Generate compose file
  -u, --up            Start all user containers
  -d, --down          Stop all user containers
  -r, --restart       Restart all user containers
  -v, --volume-clean  Reset all user directories
  -h, --help          Show this help message

Examples:
  $0 -g
  $0 -g -u
  $0 --generate --up
EOF
}

die() {
    echo "Error: $*" >&2
    exit 1
}

load_config() {
    [ -f "$CONFIG_FILE" ] || die "config file not found: $CONFIG_FILE"

    set -o allexport
    # shellcheck disable=SC1090
    source "$CONFIG_FILE"
    set +o allexport
}

go_source_dir() {
    cd "$SOURCE_DIR"
}

generate_compose() {
    echo "[+] Generating compose file..."
    ./generate_compose.sh "$CONFIG_FILE"
}

compose_up() {
    echo "[+] Starting containers..."
    sudo docker compose -f "$OUTPUT_FILE" up -d --build
}

compose_down() {
    echo "[+] Stopping containers..."
    sudo docker compose -f "$OUTPUT_FILE" down --remove-orphans
}

compose_restart() {
    echo "[+] Restarting containers..."
    sudo docker compose -f "$OUTPUT_FILE" down --remove-orphans
    sudo docker compose -f "$OUTPUT_FILE" up -d --build
}

volume_clean() {
    echo "[+] Cleaning volumes..."

    sudo docker compose -f "$OUTPUT_FILE" down -v --remove-orphans || true

    find "$HOST_HOMES_DIR" -mindepth 1 -type d 2>/dev/null \
        | awk '{ print length, $0 }' \
        | sort -rn \
        | cut -d' ' -f2- \
        | xargs -r -n1 sudo umount 2>/dev/null || true

    sudo rm -rf "$HOST_HOMES_DIR" "$HOST_SHARE_DIR" "$HOST_READONLY_DIR"
    sudo mkdir -p "$HOST_HOMES_DIR" "$HOST_SHARE_DIR" "$HOST_READONLY_DIR"
    sudo chown "${CONTAINER_RUNTIME_UID}:${CONTAINER_RUNTIME_GID}" "$HOST_SHARE_DIR" "$HOST_READONLY_DIR"
}

main() {
    [ "$#" -gt 0 ] || {
        usage
        exit 1
    }

    trap 'cd "$CURRENT_DIR"' EXIT

    load_config
    go_source_dir

    while [ "$#" -gt 0 ]; do
        case "$1" in
            -g|--generate)
                generate_compose
                ;;
            -u|--up)
                compose_up
                ;;
            -d|--down)
                compose_down
                ;;
            -r|--restart)
                compose_restart
                ;;
            -v|--volume-clean)
                volume_clean
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                echo "Invalid parameter: $1" >&2
                usage
                exit 1
                ;;
        esac
        shift
    done
}

main "$@"