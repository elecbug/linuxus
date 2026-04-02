#!/bin/bash

set -e

# Validate required environment variable
if [ -z "$STUDENT_ID" ]; then
    echo "Error: STUDENT_ID is not set."
    exit 1
fi

# Determine username
if [[ "$STUDENT_ID" =~ ^[0-9] ]]; then
    USERNAME="u$STUDENT_ID"
else
    USERNAME="$STUDENT_ID"
fi

HOME_DIR="/home/$USERNAME"
SHARE_DIR="/home/share"

# Create shared directory if missing
mkdir -p "$SHARE_DIR"
chmod 777 "$SHARE_DIR"

# Create user if it does not exist
if ! id "$USERNAME" >/dev/null 2>&1; then
    useradd -m -d "$HOME_DIR" -s /bin/bash "$USERNAME"
    echo "$USERNAME ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
fi

# Ensure home directory exists
mkdir -p "$HOME_DIR"

# Ensure ownership
chown -R "$USERNAME:$USERNAME" "$HOME_DIR"
chmod 755 "$HOME_DIR"

# Add welcome message once
BASHRC="$HOME_DIR/.bashrc"
if ! grep -q 'linuxus service shell' "$BASHRC" 2>/dev/null; then
    cat >> "$BASHRC" <<EOF
    
echo "Welcome to the linuxus service shell."
echo "   Student ID: $STUDENT_ID"
echo "  Home folder: /home/$USERNAME"
echo "Shared folder: /home/share"
EOF
fi

# Start ttyd and launch bash as the student
exec ttyd \
  --writable \
  --port 7681 \
  sudo -u "$USERNAME" -H bash -lc "cd '$HOME_DIR' && exec bash"