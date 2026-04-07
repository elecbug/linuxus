#!/bin/bash

set -e

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

USERNAME="${USERNAME_PREFIX}${USER_ID}"
HOME_DIR="/home/${USERNAME}"

# mkdir -p "$HOME_DIR"
# mkdir -p "$SHARED_DIR"
# mkdir -p "$READONLY_DIR"

# chmod 755 "$HOME_DIR"
# chmod 777 "$SHARED_DIR"

# if [ "$IS_ADMIN" = "true" ]; then
#     chmod 777 "$READONLY_DIR"
# fi

BASHRC="$HOME_DIR/.bashrc"
if ! grep -q 'Welcome to the linuxus service shell' "$BASHRC" 2>/dev/null; then
    cat > "$BASHRC" <<EOF
echo "+---------------------------------------------------+"
echo "|       Welcome to the linuxus service shell.       |"
echo "+---------------------------------------------------+"
echo ""
echo "  - User ID            : $USER_ID"
echo "  - Linux user         : $(id -un)"
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
  --client-option "titleFixed=linuxus | $USERNAME" \
  bash --login