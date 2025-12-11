#!/bin/bash
set -euo pipefail

# Fix ownership of home directory if it's owned by root (new volume)
# This handles the case where a fresh volume is mounted
if [ "$(stat -c '%u' /home/ubuntu)" = "0" ]; then
    sudo chown -R ubuntu:ubuntu /home/ubuntu
fi

# Run post-install script on first boot
MARKER_FILE="$HOME/.glovebox-initialized"
POST_INSTALL_SCRIPT="/usr/local/lib/glovebox/post-install.sh"

if [ ! -f "$MARKER_FILE" ] && [ -f "$POST_INSTALL_SCRIPT" ]; then
    # Run post-install script
    if bash "$POST_INSTALL_SCRIPT"; then
        # Create marker file on success
        touch "$MARKER_FILE"
    else
        echo ""
        echo "WARNING: Post-install script failed. Will retry on next start."
        echo ""
    fi
fi

# Execute the requested command (default shell)
exec "$@"
