#!/bin/bash

set -e

# Validate required environment variable
if [ -z "$USER_ID" ]; then
    echo "Error: USER_ID is not set."
    exit 1
fi

if [ -z "$USERNAME_PREFIX" ]; then
    echo "Error: USERNAME_PREFIX is not set."
    exit 1
fi

if [ -z "$SHARED_DIR" ]; then
    echo "Error: SHARED_DIR is not set."
    exit 1
fi

if [ -z "$READONLY_DIR" ]; then
    echo "Error: READONLY_DIR is not set."
    exit 1
fi

# Validate directories
if [ ! -d "$SHARED_DIR" ]; then
    mkdir -p "$SHARED_DIR"
fi
if [ ! -d "$READONLY_DIR" ]; then
    mkdir -p "$READONLY_DIR"
fi

USERNAME="$USERNAME_PREFIX$USER_ID"
HOME_DIR="/home/$USERNAME"

# Create user if it does not exist
if ! id "$USERNAME" >/dev/null 2>&1; then
    useradd -M -d "$HOME_DIR" -s /bin/bash "$USERNAME"
fi

# Ensure directories exist
mkdir -p "$HOME_DIR"
mkdir -p "$SHARED_DIR"
mkdir -p "$READONLY_DIR"

# Ensure ownership
chown -R "$USERNAME:$USERNAME" "$HOME_DIR"

chmod 755 "$HOME_DIR"
chmod 777 "$SHARED_DIR"

if [ "$IS_ADMIN" = "true" ]; then
    chmod 777 "$READONLY_DIR"
fi

# Add welcome message once
BASHRC="$HOME_DIR/.bashrc"
if ! grep -q 'Welcome to the linuxus service shell' "$BASHRC" 2>/dev/null; then
    cat > "$BASHRC" <<EOF
echo "+---------------------------------------------------+"
echo "|       Welcome to the linuxus service shell.       |"
echo "+---------------------------------------------------+"
echo ""
echo "  - User ID            : $USER_ID"
echo "  - Linux user         : $USERNAME"
echo "  - User mode          : $( [ "$IS_ADMIN" = "true" ] && echo "Admin" || echo "Default" )"
echo ""
echo "  - Home directory     : /home/$USERNAME"
echo "  - Shared directory   : $SHARED_DIR"
echo "  - Readonly directory : $READONLY_DIR"
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