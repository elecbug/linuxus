#!/bin/bash

set -e

# =========================
# Usage check
# =========================
if [ -z "$1" ]; then
    echo "Usage: ./generate_compose.sh <student_count> [start_port]"
    exit 1
fi

STUDENT_COUNT="$1"
START_PORT="${2:-7681}"
OUTPUT_FILE="docker-compose.generated.yml"

# Validate student count
if ! [[ "$STUDENT_COUNT" =~ ^[0-9]+$ ]] || [ "$STUDENT_COUNT" -le 0 ]; then
    echo "Error: student_count must be a positive integer."
    exit 1
fi

# Validate start port
if ! [[ "$START_PORT" =~ ^[0-9]+$ ]] || [ "$START_PORT" -le 0 ]; then
    echo "Error: start_port must be a positive integer."
    exit 1
fi

# =========================
# Write compose header
# =========================
cat > "$OUTPUT_FILE" <<EOF
version: "3.8"

services:
EOF

# =========================
# Generate student services
# =========================
for ((i=1; i<=STUDENT_COUNT; i++)); do
    PORT=$((START_PORT + i - 1))

    cat >> "$OUTPUT_FILE" <<EOF
  student$i:
    build: .
    container_name: ubuntu-shell-student$i
    environment:
      - STUDENT_NAME=student$i
    ports:
      - "$PORT:7681"
    volumes:
      - student${i}_workspace:/home/student/workspace
      - shared_data:/home/student/share
    restart: unless-stopped

EOF
done

# =========================
# Generate volumes section
# =========================
cat >> "$OUTPUT_FILE" <<EOF
volumes:
EOF

for ((i=1; i<=STUDENT_COUNT; i++)); do
    cat >> "$OUTPUT_FILE" <<EOF
  student${i}_workspace:
EOF
done

cat >> "$OUTPUT_FILE" <<EOF
  shared_data:
EOF

echo "Generated $OUTPUT_FILE for $STUDENT_COUNT students."
echo "Start port: $START_PORT"
echo
echo "Run:"
echo "  docker compose -f $OUTPUT_FILE up -d --build"