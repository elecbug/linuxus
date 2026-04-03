#!/bin/bash

set -e

# Usage:
#   ./generate_compose.sh <students.txt> [auth_port] [username_prefix]
# Example:
#   ./generate_compose.sh ./data/students.txt 8080 "stu-"

if [ $# -lt 1 ]; then
    echo "Usage: ./generate_compose.sh <students.txt> [auth_port] [username_prefix]"
    exit 1
fi

STUDENT_LIST_FILE="$1"
AUTH_PORT="${2:-8080}"
USERNAME_PREFIX="${3:-stu-}"
OUTPUT_FILE="docker-compose.generated.yml"

if [ ! -f "$STUDENT_LIST_FILE" ]; then
    echo "Error: file not found: $STUDENT_LIST_FILE"
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
    local student_id="$1"

    echo "${USERNAME_PREFIX}${student_id}"
}

declare -a STUDENT_IDS=()
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
    student_id="${line%%:*}"
    student_id="$(trim "$student_id")"

    if [ -z "$student_id" ]; then
        continue
    fi

    if [ -n "${SEEN[$student_id]}" ]; then
        echo "Warning: duplicate student ID skipped: $student_id"
        continue
    fi

    safe_id="$(sanitize_name "$student_id")"
    username="$(make_username "$student_id")"

    SEEN["$student_id"]=1
    STUDENT_IDS+=("$student_id")
    SAFE_IDS+=("$safe_id")
    USERNAMES+=("$username")
done < "$STUDENT_LIST_FILE"

if [ "${#STUDENT_IDS[@]}" -eq 0 ]; then
    echo "Error: no valid student IDs found in $STUDENT_LIST_FILE"
    exit 1
fi

cat > "$OUTPUT_FILE" <<EOF
version: "3.8"

services:
  linuxus_auth:
    build: ./auth
    container_name: linuxus_auth
    environment:
      - STUDENTS_FILE=/data/STUDENTS
      - SESSION_SECRET=change-this-secret-before-production
    volumes:
      - ${STUDENT_LIST_FILE}:/data/STUDENTS:ro
    ports:
      - "${AUTH_PORT}:8080"
    restart: unless-stopped

EOF

for ((i=0; i<${#STUDENT_IDS[@]}; i++)); do
    STUDENT_ID="${STUDENT_IDS[$i]}"
    SAFE_ID="${SAFE_IDS[$i]}"
    USERNAME="${USERNAMES[$i]}"

    cat >> "$OUTPUT_FILE" <<EOF
  linuxus_service_${SAFE_ID}:
    build: ./service
    container_name: linuxus_service_${SAFE_ID}
    hostname: linuxus
    environment:
      - STUDENT_ID=${STUDENT_ID}
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
echo "  http://localhost:${AUTH_PORT}/login"
echo
echo "Students:"
for ((i=0; i<${#STUDENT_IDS[@]}; i++)); do
    echo "  ID=${STUDENT_IDS[$i]} USER=${USERNAMES[$i]} SERVICE=student_${SAFE_IDS[$i]}"
done
echo
echo "Run:"
echo "  docker compose -f $OUTPUT_FILE up -d --build"