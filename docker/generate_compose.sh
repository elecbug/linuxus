#!/bin/bash

set -e

# =========================
# Usage check
# =========================
if [ $# -lt 1 ]; then
    echo "Usage: ./generate_compose.sh <student_list.txt> [start_port]"
    exit 1
fi

STUDENT_LIST_FILE="$1"
START_PORT="${2:-7681}"
OUTPUT_FILE="docker-compose.generated.yml"

if [ ! -f "$STUDENT_LIST_FILE" ]; then
    echo "Error: file not found: $STUDENT_LIST_FILE"
    exit 1
fi

if ! [[ "$START_PORT" =~ ^[0-9]+$ ]] || [ "$START_PORT" -le 0 ]; then
    echo "Error: start_port must be a positive integer."
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
    s="$(printf '%s' "$s" | sed 's/[^a-z0-9_.-]/_/g')"
    s="$(printf '%s' "$s" | sed 's/^[_.-]\+//')"
    if [ -z "$s" ]; then
        echo "invalid"
    else
        printf '%s' "$s"
    fi
}

declare -a STUDENT_IDS=()
declare -a SAFE_IDS=()
declare -A SEEN=()

while IFS= read -r line || [ -n "$line" ]; do
    line="$(trim "$line")"
    [ -z "$line" ] && continue

    case "$line" in
        \#*) continue ;;
    esac

    if [ -n "${SEEN[$line]}" ]; then
        echo "Warning: duplicate student ID skipped: $line"
        continue
    fi

    SAFE_ID="$(sanitize_name "$line")"

    SEEN["$line"]=1
    STUDENT_IDS+=("$line")
    SAFE_IDS+=("$SAFE_ID")
done < "$STUDENT_LIST_FILE"

if [ "${#STUDENT_IDS[@]}" -eq 0 ]; then
    echo "Error: no valid student IDs found in $STUDENT_LIST_FILE"
    exit 1
fi

cat > "$OUTPUT_FILE" <<EOF
version: "3.8"

services:
EOF

PORT="$START_PORT"

for ((i=0; i<${#STUDENT_IDS[@]}; i++)); do
    STUDENT_ID="${STUDENT_IDS[$i]}"
    SAFE_ID="${SAFE_IDS[$i]}"

    cat >> "$OUTPUT_FILE" <<EOF
  student_${SAFE_ID}:
    build: .
    container_name: ubuntu-shell-${SAFE_ID}
    environment:
      - STUDENT_ID=${STUDENT_ID}
    ports:
      - "${PORT}:7681"
    volumes:
      - ${SAFE_ID}_home:/home/${STUDENT_ID}
      - shared_data:/home/share
    restart: unless-stopped

EOF

    PORT=$((PORT + 1))
done

cat >> "$OUTPUT_FILE" <<EOF
volumes:
EOF

for SAFE_ID in "${SAFE_IDS[@]}"; do
    cat >> "$OUTPUT_FILE" <<EOF
  ${SAFE_ID}_home:
EOF
done

cat >> "$OUTPUT_FILE" <<EOF
  shared_data:
EOF

echo "Generated $OUTPUT_FILE"
echo "Student count: ${#STUDENT_IDS[@]}"
echo "Start port   : $START_PORT"
echo
echo "Port mapping:"
PORT="$START_PORT"
for ((i=0; i<${#STUDENT_IDS[@]}; i++)); do
    echo "  ${STUDENT_IDS[$i]} -> ${PORT}"
    PORT=$((PORT + 1))
done
echo
echo "Run:"
echo "  docker compose -f $OUTPUT_FILE up -d --build"