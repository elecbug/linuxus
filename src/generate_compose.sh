#!/bin/bash

set -e

# Usage:
#   ./generate_compose.sh [config.env]

if [ $# -lt 1 ]; then
    echo "Usage: ./generate_compose.sh [config.env]"
    exit 1
fi

CONFIG_FILE="${1:-./config.env}"

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
if ! [[ "$AUTH_EXTERNAL_PORT" =~ ^[0-9]+$ ]] || [ "$AUTH_EXTERNAL_PORT" -le 0 ]; then
    echo "Error: AUTH_EXTERNAL_PORT must be a positive integer."
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

get_ip() {
    local base_ip="$1"
    local i="$2"

    # Split the base IP into octets
    IFS='.' read -r o1 o2 o3 o4 <<< "$base_ip"

    # Validate basic input
    if ! [[ "$o1" =~ ^[0-9]+$ && "$o2" =~ ^[0-9]+$ && "$o3" =~ ^[0-9]+$ && "$o4" =~ ^[0-9]+$ ]]; then
        echo "Error: invalid base IP format" >&2
        return 1
    fi

    if ! [[ "$i" =~ ^[0-9]+$ ]]; then
        echo "Error: index must be a non-negative integer" >&2
        return 1
    fi

    # /28 means one subnet per 16 addresses
    # 16 subnets fit into one /24
    local third_octet_offset=$(( i / 16 ))
    local fourth_octet_offset=$(( (i % 16) * 16 ))

    local new_o3=$(( o3 + third_octet_offset ))
    local new_o4=$fourth_octet_offset

    # Check overflow
    if [ "$new_o3" -gt 255 ]; then
        echo "Error: subnet overflow (3rd octet > 255)" >&2
        return 1
    fi

    echo "${o1}.${o2}.${new_o3}.${new_o4}/28"
}

create_user_disk() {
    local username="$1"
    local size="${USER_DISK_LIMIT:-1024}"

    if [ "$username" == "${ADMIN_PREFIX}${ADMIN_USER_ID}" ]; then
        size="${ADMIN_DISK_LIMIT:-1024}"
    fi

    local uid="${CONTAINER_RUNTIME_UID:-1000}"
    local gid="${CONTAINER_RUNTIME_GID:-1000}"

    local img="${HOST_HOMES_DIR}/${username}.img"
    local mount_point="${HOST_HOMES_DIR}/${username}"

    if mountpoint -q "$mount_point"; then
        echo "[=] Already mounted: $mount_point"
        return 0
    fi

    if [ ! -f "$img" ]; then
        echo "[+] Creating disk for $username (${size}MB)"
        sudo dd if=/dev/zero of="$img" bs=1M count="$size"
        sudo mkfs.ext4 -F "$img"
    fi

    sudo mkdir -p "$mount_point"

    local loopdev
    loopdev=$(sudo losetup -f --show "$img")
    sudo mount "$loopdev" "$mount_point"

    sudo chown ${CONTAINER_RUNTIME_UID}:${CONTAINER_RUNTIME_GID} "$mount_point"
    sudo chmod 755 "$mount_point"
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
    username="${USER_PREFIX}${user_id}"

    if [ "$user_id" == "$ADMIN_USER_ID" ]; then
        continue
    fi

    SEEN["$user_id"]=1
    USER_IDS+=("$user_id")
    SAFE_IDS+=("$safe_id")
    USERNAMES+=("$username")
done < "$AUTH_LIST_FILE"

if [ "${#USER_IDS[@]}" -eq 0 ]; then
    echo "Error: no valid user IDs found in $AUTH_LIST_FILE"
    exit 1
fi

ADMIN_SAFE_NAME="$(sanitize_name "$ADMIN_USER_ID")"

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
      - SESSION_SECRET=${AUTH_SESSION_SECRET}
      - LOGIN_PATH=${URL_LOGIN_PATH}
      - LOGOUT_PATH=${URL_LOGOUT_PATH}
      - SERVICE_PATH=${URL_SERVICE_PATH}
      - TERMINAL_PATH=${URL_TERMINAL_PATH}
      - ADMIN_USER_ID=${ADMIN_USER_ID}
    volumes:
      - ${AUTH_LIST_FILE}:${AUTH_LIST_MOUNT_PATH}:rw
    ports:
      - "${AUTH_EXTERNAL_PORT}:8080"
    restart: unless-stopped
    networks:
EOF

# Connect auth to every student-private network
for SAFE_ID in "${SAFE_IDS[@]}"; do
    cat >> "$OUTPUT_FILE" <<EOF
      - ${USER_NETWORK_PREFIX}${SAFE_ID}
EOF
done

# Connect auth to admin-private network
cat >> "$OUTPUT_FILE" <<EOF
      - ${ADMIN_NETWORK_PREFIX}${ADMIN_SAFE_NAME}

EOF

for ((i=0; i<${#USER_IDS[@]}; i++)); do
    USER_ID="${USER_IDS[$i]}"
    SAFE_ID="${SAFE_IDS[$i]}"
    USERNAME="${USERNAMES[$i]}"
    
    create_user_disk "$USERNAME"

    cat >> "$OUTPUT_FILE" <<EOF
  ${USER_CONTAINER_NAME_PREFIX}${SAFE_ID}:
    user: ${CONTAINER_RUNTIME_UID}:${CONTAINER_RUNTIME_GID}
    build: ${USER_SOURCE_DIR}
    container_name: ${USER_CONTAINER_NAME_PREFIX}${SAFE_ID}
    hostname: ${USER_HOSTNAME}
    working_dir: /home/${USERNAME}
    read_only: true
    tmpfs:
      - /tmp:rw,noexec,nosuid,nodev,size=64m
      - /run:rw,noexec,nosuid,nodev,size=16m
      - /var/tmp:rw,noexec,nosuid,nodev,size=64m
    environment:
      - USER_ID=${USER_ID}
      - USERNAME_PREFIX=${USER_PREFIX}
      - SHARED_DIR=${CONTAINER_SHARE_DIR}
      - READONLY_DIR=${CONTAINER_READONLY_DIR}
      - IS_ADMIN=false
    expose:
      - "7681"
    volumes:
      - ${HOST_HOMES_DIR}/${USERNAME}:/home/${USERNAME}:rw
      - ${HOST_SHARE_DIR}:${CONTAINER_SHARE_DIR}:rw
      - ${HOST_READONLY_DIR}:${CONTAINER_READONLY_DIR}:ro
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    mem_limit: ${USER_MEMORY_LIMIT}
    cpus: ${USER_CPU_LIMIT}
    pids_limit: ${USER_PID_LIMIT}
    ulimits:
      nofile:
        soft: ${USER_ULIMITS_NOFILE_SOFT}
        hard: ${USER_ULIMITS_NOFILE_HARD}
    networks:
      - ${USER_NETWORK_PREFIX}${SAFE_ID}

EOF
done

create_user_disk "${ADMIN_PREFIX}${ADMIN_USER_ID}"

cat >> "$OUTPUT_FILE" <<EOF
  ${ADMIN_CONTAINER_NAME_PREFIX}${ADMIN_USER_ID}:
    user: ${CONTAINER_RUNTIME_UID}:${CONTAINER_RUNTIME_GID}
    build: ${ADMIN_SOURCE_DIR}
    container_name: ${ADMIN_CONTAINER_NAME_PREFIX}${ADMIN_USER_ID}
    hostname: ${ADMIN_HOSTNAME}
    working_dir: /home/${ADMIN_PREFIX}${ADMIN_USER_ID}
    read_only: true
    tmpfs:
      - /tmp:rw,noexec,nosuid,nodev,size=64m
      - /run:rw,noexec,nosuid,nodev,size=16m
      - /var/tmp:rw,noexec,nosuid,nodev,size=64m
    environment:
      - USER_ID=${ADMIN_USER_ID}
      - USERNAME_PREFIX=${ADMIN_PREFIX}
      - SHARED_DIR=${CONTAINER_SHARE_DIR}
      - READONLY_DIR=${CONTAINER_READONLY_DIR}
      - IS_ADMIN=true
    expose:
      - "7681"
    volumes:
      - ${HOST_HOMES_DIR}/${ADMIN_PREFIX}${ADMIN_USER_ID}:/home/${ADMIN_PREFIX}${ADMIN_USER_ID}:rw
      - ${HOST_SHARE_DIR}:${CONTAINER_SHARE_DIR}:rw
      - ${HOST_READONLY_DIR}:${CONTAINER_READONLY_DIR}:rw
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    mem_limit: ${ADMIN_MEMORY_LIMIT}
    cpus: ${ADMIN_CPU_LIMIT}
    pids_limit: ${ADMIN_PID_LIMIT}
    ulimits:
      nofile:
        soft: ${ADMIN_ULIMITS_NOFILE_SOFT}
        hard: ${ADMIN_ULIMITS_NOFILE_HARD}
    networks:
      - ${ADMIN_NETWORK_PREFIX}${ADMIN_SAFE_NAME}

networks:
EOF

seq_i="0"

# One private network per student
for SAFE_ID in "${SAFE_IDS[@]}"; do
    cat >> "$OUTPUT_FILE" <<EOF
  ${USER_NETWORK_PREFIX}${SAFE_ID}:
    driver: bridge
    ipam:
      config:
        - subnet: $(get_ip "$USER_BASE_IP" "$seq_i")
EOF
    seq_i=$((seq_i + 1))
done

# One private network for admin
cat >> "$OUTPUT_FILE" <<EOF
  ${ADMIN_NETWORK_PREFIX}${ADMIN_SAFE_NAME}:
    driver: bridge
    ipam:
      config:
        - subnet: $(get_ip "$ADMIN_BASE_IP" "$seq_i")
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
echo "  http://localhost:${AUTH_EXTERNAL_PORT}/${URL_LOGIN_PATH}"
echo
echo "Users:"
for ((i=0; i<${#USER_IDS[@]}; i++)); do
    echo "  ID=${USER_IDS[$i]} USER=${USERNAMES[$i]} SERVICE=${USER_CONTAINER_NAME_PREFIX}${SAFE_IDS[$i]} NET=${USER_NETWORK_PREFIX}${SAFE_IDS[$i]}"
done
echo "  ADMIN=${ADMIN_USER_ID} NET=${ADMIN_NETWORK_PREFIX}${ADMIN_SAFE_NAME}"
echo
echo "Run:"
echo "  docker compose -f $OUTPUT_FILE up -d --build"