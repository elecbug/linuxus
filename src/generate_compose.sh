#!/bin/bash

set -e

# Usage:
#   ./generate_compose.sh <auths.txt> [config.env]

if [ $# -lt 1 ]; then
    echo "Usage: ./generate_compose.sh <auths.txt> [config.env]"
    exit 1
fi

AUTH_LIST_FILE="$1"
CONFIG_FILE="${2:-./config.env}"

if [ ! -f "$AUTH_LIST_FILE" ]; then
    echo "Error: file not found: $AUTH_LIST_FILE"
    exit 1
fi

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

# =========================
# Validation
# =========================
if ! [[ "$AUTH_PORT" =~ ^[0-9]+$ ]] || [ "$AUTH_PORT" -le 0 ]; then
    echo "Error: AUTH_PORT must be a positive integer."
    exit 1
fi

trim() {
    local s="$1"
    s="${s#"${s%%[![:space:]]*}"}"
    s="${s%"${s##*[![:space:]]}"}"
    printf '%s' "$s"
}

sanitize_name() {
    local s="$1"
    s="$(printf '%s' "$s" | tr '[:upper:]' '[:lower:]')"
    s="$(printf '%s' "$s" | sed 's/[^a-z0-9]/_/g')"
    s="$(printf '%s' "$s" | sed 's/_\+/_/g')"
    s="$(printf '%s' "$s" | sed 's/^_//; s/_$//')"

    if [ -z "$s" ]; then
        echo "invalid"
    else
        printf '%s' "$s"
    fi
}

make_username() {
    local user_id="$1"
    printf '%s%s' "$USERNAME_PREFIX" "$user_id"
}

declare -a USER_IDS=()
declare -a SAFE_IDS=()
declare -a USERNAMES=()
declare -A SEEN=()

while IFS= read -r line || [ -n "$line" ]; do
    line="$(trim "$line")"

    [ -z "$line" ] && continue
    case "$line" in
        \#*) continue ;;
    esac

    user_id="${line%%:*}"
    user_id="$(trim "$user_id")"

    if [ -z "$user_id" ]; then
        continue
    fi

    if [ -n "${SEEN[$user_id]}" ]; then
        echo "Warning: duplicate user ID skipped: $user_id"
        continue
    fi

    safe_id="$(sanitize_name "$user_id")"
    username="$(make_username "$user_id")"

    SEEN["$user_id"]=1
    USER_IDS+=("$user_id")
    SAFE_IDS+=("$safe_id")
    USERNAMES+=("$username")
done < "$AUTH_LIST_FILE"

if [ "${#USER_IDS[@]}" -eq 0 ]; then
    echo "Error: no valid user IDs found in $AUTH_LIST_FILE"
    exit 1
fi

# =========================
# Prepare volumes
# =========================
if [ -d "$SERVICE_SHARED_DIR" ]; then
    mkdir -p "$SERVICE_SHARED_DIR"
    chmod 755 "$SERVICE_SHARED_DIR"
fi

if [ -d "$SERVICE_READONLY_DIR" ]; then
    mkdir -p "$SERVICE_READONLY_DIR"
    chmod 755 "$SERVICE_READONLY_DIR"
fi

# =========================
# Generate compose
# =========================
cat > "$OUTPUT_FILE" <<EOF
version: "3.8"

services:
  ${AUTH_CONTAINER_NAME}:
    build: ${AUTH_SOURCE_DIR}
    container_name: ${AUTH_CONTAINER_NAME}
    environment:
      - AUTH_LIST=${AUTH_LIST_MOUNT_PATH}
      - SESSION_SECRET=${SESSION_SECRET}
      - LOGIN_PATH=${LOGIN_PATH}
      - LOGOUT_PATH=${LOGOUT_PATH}
      - SERVICE_PATH=${SERVICE_PATH}
      - TERMINAL_PATH=${TERMINAL_PATH}
      - MANAGER_LOGIN_ID=${MANAGER_LOGIN_ID}
      - MANAGER_LOGIN_PASSWORD=${MANAGER_LOGIN_PASSWORD}
    volumes:
      - ${AUTH_LIST_FILE}:${AUTH_LIST_MOUNT_PATH}:ro
    ports:
      - "${AUTH_PORT}:8080"
    restart: unless-stopped

EOF

for ((i=0; i<${#USER_IDS[@]}; i++)); do
    USER_ID="${USER_IDS[$i]}"
    SAFE_ID="${SAFE_IDS[$i]}"
    USERNAME="${USERNAMES[$i]}"

    cat >> "$OUTPUT_FILE" <<EOF
  ${SERVICE_CONTAINER_NAME_PREFIX}${SAFE_ID}:
    build: ${SERVICE_SOURCE_DIR}
    container_name: ${SERVICE_CONTAINER_NAME_PREFIX}${SAFE_ID}
    hostname: ${SERVICE_HOSTNAME}
    environment:
      - USER_ID=${USER_ID}
      - USERNAME_PREFIX=${USERNAME_PREFIX}
      - SHARED_DIR=${SERVICE_SHARED_DIR}
      - READONLY_DIR=${SERVICE_READONLY_DIR}
    expose:
      - "7681"
    volumes:
      - ${HOST_HOMES_DIR}/${USERNAME}:/home/${USERNAME}:rw
      - ${HOST_SHARE_DIR}:${SERVICE_SHARED_DIR}:rw
      - ${HOST_READONLY_DIR}:${SERVICE_READONLY_DIR}:ro
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true

EOF
done


cat >> "$OUTPUT_FILE" <<EOF
  ${MANAGER_CONTAINER_NAME}:
    build: ${SERVICE_SOURCE_DIR}
    container_name: ${SERVICE_CONTAINER_NAME_PREFIX}${MANAGER_CONTAINER_NAME}
    environment:
      - USER_ID=${MANAGER_CONTAINER_USERNAME}
      - USERNAME_PREFIX=""
      - SHARED_DIR=${SERVICE_SHARED_DIR}
      - READONLY_DIR=${SERVICE_READONLY_DIR}
    expose:
      - "7681"
    volumes:
      - ${HOST_HOMES_DIR}/${MANAGER_CONTAINER_USERNAME}:/home/${MANAGER_CONTAINER_USERNAME}:rw
      - ${HOST_SHARE_DIR}:${SERVICE_SHARED_DIR}:rw
      - ${HOST_READONLY_DIR}:${SERVICE_READONLY_DIR}:rw
    restart: unless-stopped

EOF

# =========================
# Output
# =========================
echo "Generated $OUTPUT_FILE"
echo
echo "Config file:"
echo "  $CONFIG_FILE"
echo
echo "Login URL:"
echo "  http://localhost:${AUTH_PORT}/${LOGIN_PATH}"
echo
echo "Users:"
for ((i=0; i<${#USER_IDS[@]}; i++)); do
    echo "  ID=${USER_IDS[$i]} USER=${USERNAMES[$i]} SERVICE=${SERVICE_CONTAINER_NAME_PREFIX}${SAFE_IDS[$i]}"
done
echo
echo "Run:"
echo "  docker compose -f $OUTPUT_FILE up -d --build"