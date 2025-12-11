#!/bin/bash
set -euo pipefail

# Fix ownership of home directory if it's owned by root (new volume)
# This handles the case where a fresh volume is mounted
if [ "$(stat -c '%u' /home/ubuntu)" = "0" ]; then
    sudo chown -R ubuntu:ubuntu /home/ubuntu
fi

# Execute the requested command (default shell)
exec "$@"
