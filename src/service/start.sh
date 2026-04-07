#!/bin/bash

set -e

echo "ENVIRONMENT VARIABLES:"
env | sort

HOME_DIR="/home/${CONTAINER_RUNTIME_USER}"

BASHRC="$HOME_DIR/.bashrc"
if ! grep -q 'Welcome to the linuxus service shell' "$BASHRC" 2>/dev/null; then
    cat > "$BASHRC" <<EOF
echo "+---------------------------------------------------+"
echo "|       Welcome to the linuxus service shell.       |"
echo "+---------------------------------------------------+"
echo ""
echo "  - Linux user         : $CONTAINER_RUNTIME_USER"
echo "  - User ID            : $USER_ID"
echo "  - User mode          : $( [ "$IS_ADMIN" = "true" ] && echo "ADMIN" || echo "DEFAULT" )"
echo ""
echo "  - Home directory     : $HOME_DIR"
echo "  - Shared directory   : $SHARED_DIR"
echo "  - Readonly directory : $READONLY_DIR"
echo ""
echo "+---------------------------------------------------+"
echo ""
EOF
fi

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

export HOME="$HOME_DIR"
cd "$HOME_DIR"

exec ttyd \
  --port 7681 \
  --client-option "titleFixed=linuxus | $USER_ID" \
  bash --login