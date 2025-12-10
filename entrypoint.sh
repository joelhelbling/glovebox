#!/bin/bash
set -euo pipefail

# Fix ownership of mise directory if it's owned by root (new volume)
if [ -d "/home/ubuntu/.local/share/mise" ] && [ "$(stat -c '%u' /home/ubuntu/.local/share/mise)" = "0" ]; then
    sudo chown -R ubuntu:ubuntu /home/ubuntu/.local/share/mise
fi

# Execute the requested command (default: fish)
exec "$@"
