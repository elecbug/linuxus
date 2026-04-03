#!/bin/bash

set -e

# Validate required environment variable
if [ -z "$STUDENT_ID" ]; then
    echo "Error: STUDENT_ID is not set."
    exit 1
fi

if [ -z "$USERNAME_PREFIX" ]; then
    USERNAME_PREFIX="u"
fi

USERNAME="$USERNAME_PREFIX$STUDENT_ID"
HOME_DIR="/home/$USERNAME"
SHARE_DIR="/home/share"
TERMINAL_PATH="terminal"

# Create shared directory if missing
mkdir -p "$SHARE_DIR"
chmod 777 "$SHARE_DIR"

# Create user if it does not exist
if ! id "$USERNAME" >/dev/null 2>&1; then
    useradd -M -d "$HOME_DIR" -s /bin/bash "$USERNAME"
fi

# Ensure home directory exists
mkdir -p "$HOME_DIR"

# Ensure ownership
chown -R "$USERNAME:$USERNAME" "$HOME_DIR"
chmod 755 "$HOME_DIR"

# Add welcome message once
BASHRC="$HOME_DIR/.bashrc"
if ! grep -q 'Welcome to the linuxus service shell' "$BASHRC" 2>/dev/null; then
    cat > "$BASHRC" <<EOF
echo "+---------------------------------------------------+"
echo "|       Welcome to the linuxus service shell.       |"
echo "+---------------------------------------------------+"
echo ""
echo "  - Student ID       : $STUDENT_ID"
echo "  - Linux user       : $USERNAME"
echo "  - Home directory   : /home/$USERNAME"
echo "  - Shared directory : /home/share"
echo ""
echo "+---------------------------------------------------+"
echo ""
EOF
fi

chown "$USERNAME:$USERNAME" "$BASHRC"

# Ensure .bash_profile sources .bashrc
PROFILE="$HOME_DIR/.bash_profile"
if [ ! -f "$PROFILE" ]; then
    cat > "$PROFILE" <<EOF
if [ -f ~/.bashrc ]; then
    . ~/.bashrc
fi
EOF
elif ! grep -q '.bashrc' "$PROFILE" 2>/dev/null; then
    cat >> "$PROFILE" <<EOF

if [ -f ~/.bashrc ]; then
    . ~/.bashrc
fi
EOF
fi

chown "$USERNAME:$USERNAME" "$PROFILE"

# Start ttyd and launch bash as the student
# NOTE: do not use --base-path or it will break the terminal proxying in auth server
exec ttyd \
  --port 7681 \
  --client-option "titleFixed=linuxus | $USERNAME" \
  su - "$USERNAME"