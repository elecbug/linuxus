#!/bin/bash

set -euo pipefail

# Usage:
#   ./generate_compose.sh [config.env]

declare CONFIG_FILE=""
declare -a USER_IDS=()
declare -a SAFE_IDS=()
declare -A SEEN=()

trim() {
    local s="${1:-}"
    s="${s#"${s%%[![:space:]]*}"}"
    s="${s%"${s##*[![:space:]]}"}"
    printf '%s' "$s"
}

sanitize_name() {
    local s="${1:-}"
    s="$(printf '%s' "$s" | tr '[:upper:]' '[:lower:]')"
    s="$(printf '%s' "$s" | sed 's/[^a-z0-9]/_/g')"
    s="$(printf '%s' "$s" | sed 's/_\+/_/g')"
    s="$(printf '%s' "$s" | sed 's/^_//; s/_$//')"

    if [ -z "$s" ]; then
        printf 'invalid'
    else
        printf '%s' "$s"
    fi
}

die() {
    echo "Error: $*" >&2
    exit 1
}

parse_args() {
    if [ "$#" -lt 1 ]; then
        echo "Usage: ./generate_compose.sh [config.env]"
        exit 1
    fi

    CONFIG_FILE="${1:-./config.env}"
}

load_config() {
    [ -f "$CONFIG_FILE" ] || die "config file not found: $CONFIG_FILE"

    set -o allexport
    # shellcheck disable=SC1090
    source "$CONFIG_FILE"
    set +o allexport
}

validate_positive_integer() {
    local name="$1"
    local value="${2:-}"

    if ! [[ "$value" =~ ^[0-9]+$ ]] || [ "$value" -le 0 ]; then
        die "$name must be a positive integer."
    fi
}

validate_config() {
    validate_positive_integer "AUTH_EXTERNAL_PORT" "${AUTH_EXTERNAL_PORT:-}"
}

get_ip() {
    local base_ip="$1"
    local index="$2"
    local o1 o2 o3 o4

    IFS='.' read -r o1 o2 o3 o4 <<< "$base_ip"

    if ! [[ "$o1" =~ ^[0-9]+$ && "$o2" =~ ^[0-9]+$ && "$o3" =~ ^[0-9]+$ && "$o4" =~ ^[0-9]+$ ]]; then
        echo "Error: invalid base IP format" >&2
        return 1
    fi

    if ! [[ "$index" =~ ^[0-9]+$ ]]; then
        echo "Error: index must be a non-negative integer" >&2
        return 1
    fi

    local third_octet_offset=$(( index / 16 ))
    local fourth_octet_offset=$(( (index % 16) * 16 ))

    local new_o3=$(( o3 + third_octet_offset ))
    local new_o4=$fourth_octet_offset

    if [ "$new_o3" -gt 255 ]; then
        echo "Error: subnet overflow (3rd octet > 255)" >&2
        return 1
    fi

    echo "${o1}.${o2}.${new_o3}.${new_o4}/28"
}

create_user_disk() {
    local user_id="$1"
    local size="${USER_DISK_LIMIT}"

    if [ "$user_id" = "${ADMIN_USER_ID}" ]; then
        size="${ADMIN_DISK_LIMIT}"
    fi

    local img="${HOST_HOMES_DIR}/${user_id}.img"
    local mount_point="${HOST_HOMES_DIR}/${user_id}"

    if mountpoint -q "$mount_point"; then
        echo "[=] Already mounted: $mount_point"
        return 0
    fi

    if [ ! -f "$img" ]; then
        echo "[+] Creating disk for $user_id (${size}MB)"
        sudo dd if=/dev/zero of="$img" bs=1M count="$size"
        sudo mkfs.ext4 -F "$img"
    fi

    sudo mkdir -p "$mount_point"

    local loopdev
    loopdev=$(sudo losetup -f --show "$img")
    sudo mount "$loopdev" "$mount_point"

    sudo chown "${CONTAINER_RUNTIME_UID}:${CONTAINER_RUNTIME_GID}" "$mount_point"
    sudo chmod 755 "$mount_point"
}

load_users() {
    [ -f "$AUTH_LIST_FILE" ] || die "AUTH_LIST_FILE not found: $AUTH_LIST_FILE"

    USER_IDS=()
    SAFE_IDS=()
    SEEN=()

    while IFS= read -r line || [ -n "$line" ]; do
        line="$(trim "$line")"

        [ -z "$line" ] && continue
        case "$line" in
            \#*) continue ;;
        esac

        local user_id
        user_id="${line%%:*}"
        user_id="$(trim "$user_id")"

        [ -z "$user_id" ] && continue

        if [ "$user_id" = "$ADMIN_USER_ID" ]; then
            continue
        fi

        if [ -n "${SEEN[$user_id]:-}" ]; then
            echo "Warning: duplicate user ID skipped: $user_id"
            continue
        fi

        local safe_id
        safe_id="$(sanitize_name "$user_id")"

        SEEN["$user_id"]=1
        USER_IDS+=("$user_id")
        SAFE_IDS+=("$safe_id")
    done < "$AUTH_LIST_FILE"

    if [ "${#USER_IDS[@]}" -eq 0 ]; then
        die "no valid user IDs found in $AUTH_LIST_FILE"
    fi
}

write_file_header() {
    cat > "$OUTPUT_FILE" <<EOF
version: "3.8"

services:
EOF
}

emit_auth_service() {
    local admin_safe_name="$1"

    cat >> "$OUTPUT_FILE" <<EOF
  ${AUTH_CONTAINER_NAME}:
    build: 
      context: ${AUTH_SOURCE_DIR}
      args:
        - TIMEZONE=${AUTH_TIMEZONE}
    container_name: ${AUTH_CONTAINER_NAME}

    environment:
      - TZ=${AUTH_TIMEZONE}
      - AUTH_LIST=${AUTH_LIST_MOUNT_PATH}
      - SESSION_SECRET=${AUTH_SESSION_SECRET}
      - LOGIN_PATH=${URL_LOGIN_PATH}
      - LOGOUT_PATH=${URL_LOGOUT_PATH}
      - SERVICE_PATH=${URL_SERVICE_PATH}
      - TERMINAL_PATH=${URL_TERMINAL_PATH}
      - ADMIN_USER_ID=${ADMIN_USER_ID}
      - USER_CONTAINER_NAME_PREFIX=${USER_CONTAINER_NAME_PREFIX}
      - TRUSTED_PROXIES=${AUTH_TRUSTED_PROXIES}
    volumes:
      - ${AUTH_LIST_FILE}:${AUTH_LIST_MOUNT_PATH}:rw
    ports:
      - "${AUTH_EXTERNAL_PORT}:8080"
    restart: unless-stopped
    networks:
EOF

    local safe_id
    for safe_id in "${SAFE_IDS[@]}"; do
        cat >> "$OUTPUT_FILE" <<EOF
      - ${USER_NETWORK_PREFIX}${safe_id}
EOF
    done

    cat >> "$OUTPUT_FILE" <<EOF
      - ${USER_NETWORK_PREFIX}${admin_safe_name}

EOF
}

emit_user_service() {
    local user_id="$1"
    local safe_id="$2"

    cat >> "$OUTPUT_FILE" <<EOF
  ${USER_CONTAINER_NAME_PREFIX}${safe_id}:
    user: ${CONTAINER_RUNTIME_UID}:${CONTAINER_RUNTIME_GID}
    build:
      context: ${USER_SOURCE_DIR}
      args:
        - CONTAINER_RUNTIME_USER=${CONTAINER_RUNTIME_USER}
        - TIMEZONE=${CONTAINER_TIMEZONE}
    container_name: ${USER_CONTAINER_NAME_PREFIX}${safe_id}
    hostname: ${CONTAINER_RUNTIME_HOSTNAME}
    working_dir: /home/${CONTAINER_RUNTIME_USER}
    read_only: true
    tmpfs:
      - /tmp:rw,noexec,nosuid,nodev,size=64m
      - /run:rw,noexec,nosuid,nodev,size=16m
      - /var/tmp:rw,noexec,nosuid,nodev,size=64m
    environment:
      - TZ=${CONTAINER_TIMEZONE}
      - CONTAINER_RUNTIME_USER=${CONTAINER_RUNTIME_USER}
      - USER_ID=${user_id}
      - SHARED_DIR=${CONTAINER_SHARE_DIR}
      - READONLY_DIR=${CONTAINER_READONLY_DIR}
      - IS_ADMIN=false
    volumes:
      - ${HOST_HOMES_DIR}/${user_id}:/home/${CONTAINER_RUNTIME_USER}:rw
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
      - ${USER_NETWORK_PREFIX}${safe_id}

EOF
}

emit_admin_service() {
    local admin_safe_name="$1"

    cat >> "$OUTPUT_FILE" <<EOF
  ${USER_CONTAINER_NAME_PREFIX}${ADMIN_USER_ID}:
    user: ${CONTAINER_RUNTIME_UID}:${CONTAINER_RUNTIME_GID}
    build:
      context: ${USER_SOURCE_DIR}
      args:
        - CONTAINER_RUNTIME_USER=${CONTAINER_RUNTIME_USER}
        - TIMEZONE=${CONTAINER_TIMEZONE}
    container_name: ${USER_CONTAINER_NAME_PREFIX}${ADMIN_USER_ID}
    hostname: ${CONTAINER_RUNTIME_HOSTNAME}
    working_dir: /home/${CONTAINER_RUNTIME_USER}
    read_only: true
    tmpfs:
      - /tmp:rw,noexec,nosuid,nodev,size=64m
      - /run:rw,noexec,nosuid,nodev,size=16m
      - /var/tmp:rw,noexec,nosuid,nodev,size=64m
    environment:
      - TZ=${CONTAINER_TIMEZONE}
      - CONTAINER_RUNTIME_USER=${CONTAINER_RUNTIME_USER}
      - USER_ID=${ADMIN_USER_ID}
      - SHARED_DIR=${CONTAINER_SHARE_DIR}
      - READONLY_DIR=${CONTAINER_READONLY_DIR}
      - IS_ADMIN=true
    volumes:
      - ${HOST_HOMES_DIR}/${ADMIN_USER_ID}:/home/${CONTAINER_RUNTIME_USER}:rw
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
      - ${USER_NETWORK_PREFIX}${admin_safe_name}

EOF
}

emit_networks() {
    local admin_safe_name="$1"
    local seq_i=0
    local safe_id

    cat >> "$OUTPUT_FILE" <<EOF
networks:
EOF

    for safe_id in "${SAFE_IDS[@]}"; do
        cat >> "$OUTPUT_FILE" <<EOF
  ${USER_NETWORK_PREFIX}${safe_id}:
    driver: bridge
    ipam:
      config:
        - subnet: $(get_ip "$USER_BASE_IP" "$seq_i")
EOF
        seq_i=$((seq_i + 1))
    done

    cat >> "$OUTPUT_FILE" <<EOF
  ${USER_NETWORK_PREFIX}${admin_safe_name}:
    driver: bridge
    ipam:
      config:
        - subnet: $(get_ip "$USER_BASE_IP" "$seq_i")
EOF
}

prepare_user_disks() {
    local i
    for ((i=0; i<${#USER_IDS[@]}; i++)); do
        create_user_disk "${USER_IDS[$i]}"
    done

    create_user_disk "${ADMIN_USER_ID}"
}

emit_user_services() {
    local i
    for ((i=0; i<${#USER_IDS[@]}; i++)); do
        emit_user_service "${USER_IDS[$i]}" "${SAFE_IDS[$i]}"
    done
}

generate_compose() {
    local admin_safe_name
    admin_safe_name="$(sanitize_name "$ADMIN_USER_ID")"

    write_file_header
    emit_auth_service "$admin_safe_name"
    emit_user_services
    emit_admin_service "$admin_safe_name"
    emit_networks "$admin_safe_name"
}

print_summary() {
    local admin_safe_name
    admin_safe_name="$(sanitize_name "$ADMIN_USER_ID")"

    echo "Generated $OUTPUT_FILE"
    echo
    echo "Config file:"
    echo "  $CONFIG_FILE"
    echo
    echo "Login URL:"
    echo "  http://localhost:${AUTH_EXTERNAL_PORT}/${URL_LOGIN_PATH}"
    echo
    echo "Users:"

    local i
    for ((i=0; i<${#USER_IDS[@]}; i++)); do
        echo "  ID=${USER_IDS[$i]} SERVICE=${USER_CONTAINER_NAME_PREFIX}${SAFE_IDS[$i]} NET=${USER_NETWORK_PREFIX}${SAFE_IDS[$i]}"
    done

    echo "  ADMIN=${ADMIN_USER_ID} NET=${USER_NETWORK_PREFIX}${admin_safe_name}"
    echo
    echo "Run:"
    echo "  sudo docker compose -f $OUTPUT_FILE up -d --build"
}

main() {
    parse_args "$@"
    load_config
    validate_config
    load_users
    prepare_user_disks
    generate_compose
    print_summary
}

main "$@"