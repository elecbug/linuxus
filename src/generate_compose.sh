#!/bin/bash

set -e

# Usage:
#   ./generate_compose.sh <auths.txt> [auth_port]
# Example:
#   ./generate_compose.sh ./data/auths.txt 8080

if [ $# -lt 1 ]; then
    echo "Usage: ./generate_compose.sh <auths.txt> [auth_port]"
    exit 1
fi

LOGIN_PATH="login"
LOGOUT_PATH="logout"
SERVICE_PATH="service"
TERMINAL_PATH="terminal"

AUTH_PORT="${2:-8080}"
AUTH_SOURCE_DIR="./auth"
AUTH_CONTAINER_NAME="linuxus_auth"

SERVICE_SOURCE_DIR="./service"
SERVICE_CONTAINER_NAME_PREFIX="linuxus_service_"
USERNAME_PREFIX="stu-"

AUTH_LIST_FILE="$1"

OUTPUT_FILE="docker-compose.generated.yml"

if [ ! -f "$AUTH_LIST_FILE" ]; then
    echo "Error: file not found: $AUTH_LIST_FILE"
    exit 1
fi

if ! [[ "$AUTH_PORT" =~ ^[0-9]+$ ]] || [ "$AUTH_PORT" -le 0 ]; then
    echo "Error: auth_port must be a positive integer."
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
    # Replace all non-alnum chars with underscore
    s="$(printf '%s' "$s" | sed 's/[^a-z0-9]/_/g')"
    # Collapse repeated underscores
    s="$(printf '%s' "$s" | sed 's/_\+/_/g')"
    # Remove leading/trailing underscores
    s="$(printf '%s' "$s" | sed 's/^_//; s/_$//')"

    if [ -z "$s" ]; then
        echo "invalid"
    else
        printf '%s' "$s"
    fi
}

make_username() {
    local user_id="$1"

    echo "${USERNAME_PREFIX}${user_id}"
}

declare -a USER_IDS=()
declare -a SAFE_IDS=()
declare -a USERNAMES=()
declare -A SEEN=()

while IFS= read -r line || [ -n "$line" ]; do
    line="$(trim "$line")"

    # Skip empty lines and comments
    [ -z "$line" ] && continue
    case "$line" in
        \#*) continue ;;
    esac

    # Split only at the first colon
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

cat > "$OUTPUT_FILE" <<EOF
version: "3.8"

services:
  $AUTH_CONTAINER_NAME:
    build: $AUTH_SOURCE_DIR
    container_name: $AUTH_CONTAINER_NAME
    environment:
      - AUTH_LIST=/data/AUTH_LIST
      - SESSION_SECRET=change-this-secret-before-production
      - LOGIN_PATH=${LOGIN_PATH}
      - LOGOUT_PATH=${LOGOUT_PATH}
      - SERVICE_PATH=${SERVICE_PATH}
      - TERMINAL_PATH=${TERMINAL_PATH}
    volumes:
      - ${AUTH_LIST_FILE}:/data/AUTH_LIST:ro
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
    build: $SERVICE_SOURCE_DIR
    container_name: ${SERVICE_CONTAINER_NAME_PREFIX}${SAFE_ID}
    hostname: linuxus
    environment:
      - USER_ID=${USER_ID}
      - USERNAME_PREFIX=${USERNAME_PREFIX}
    expose:
      - "7681"
    volumes:
      - home_${SAFE_ID}:/home/${USERNAME}
      - shared_data:/home/share
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
EOF
done

cat >> "$OUTPUT_FILE" <<EOF
volumes:
EOF

for SAFE_ID in "${SAFE_IDS[@]}"; do
    cat >> "$OUTPUT_FILE" <<EOF
  home_${SAFE_ID}:
EOF
done

cat >> "$OUTPUT_FILE" <<EOF
  shared_data:
EOF

echo "Generated $OUTPUT_FILE"
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