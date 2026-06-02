#!/bin/bash

set -e

echo "[INFO] Starting Wine installation script"

# Check permissions
if ! sudo -n true 2>/dev/null; then
    echo "[ERROR] sudo privileges required."
    exit 1
fi

# Already installed?
if command -v wine &>/dev/null; then
    echo "[INFO] Wine already installed: $(wine --version 2>/dev/null)"
    exit 0
fi

# Check OS
if [[ ! -f /etc/os-release ]]; then
    echo "[ERROR] /etc/os-release file not found."
    exit 1
fi

source /etc/os-release

echo "[INFO] Detected OS: $PRETTY_NAME"

case "$ID" in
    ubuntu|debian)
        # Wine needs 32-bit support.
        sudo dpkg --add-architecture i386
        sudo apt-get update
        # Prefer the full set; fall back to the meta package if split packages are unavailable.
        sudo apt-get install -y wine wine64 wine32 || sudo apt-get install -y wine
        ;;
    fedora)
        sudo dnf install -y wine
        ;;
    rhel|centos|rocky|almalinux)
        sudo dnf install -y epel-release 2>/dev/null || true
        sudo dnf install -y wine
        ;;
    *)
        echo "[ERROR] Unsupported distribution: $ID"
        exit 1
        ;;
esac

# Verify installation
if command -v wine &>/dev/null; then
    echo "[INFO] Wine installation completed successfully: $(wine --version 2>/dev/null)"
else
    echo "[ERROR] Wine installation verification failed."
    exit 1
fi
