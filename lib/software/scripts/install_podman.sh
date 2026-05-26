#!/bin/bash

set -e

echo "[INFO] Starting Podman installation script"

# Check permissions
if ! sudo -n true 2>/dev/null; then
    echo "[ERROR] sudo privileges required."
    exit 1
fi

# Check OS
if [[ ! -f /etc/os-release ]]; then
    echo "[ERROR] /etc/os-release file not found."
    exit 1
fi

source /etc/os-release

case "$ID" in
    ubuntu|debian|rhel|fedora|centos|rocky)
        echo "[INFO] Detected OS: $PRETTY_NAME"
        ;;
    *)
        echo "[ERROR] Unsupported distribution: $ID"
        exit 1
        ;;
esac

# Remove old Podman packages
echo "[INFO] Removing old Podman packages..."

if [[ "$ID" == "ubuntu" || "$ID" == "debian" ]]; then
    sudo apt-get remove -y podman 2>/dev/null || true
else
    sudo dnf remove -y podman 2>/dev/null || true
fi

# Install Podman
echo "[INFO] Installing Podman..."

if [[ "$ID" == "ubuntu" || "$ID" == "debian" ]]; then
    sudo apt-get update
    sudo apt-get install -y podman
else
    sudo dnf install -y podman
fi

# Configure unqualified search registries
echo "[INFO] Configuring container registries..."
sudo bash -c 'cat >> /etc/containers/registries.conf << EOF

[registries.search]
registries = ["docker.io"]
EOF'

# Start and enable Podman socket
echo "[INFO] Starting Podman socket service..."
sudo systemctl enable --now podman.socket 2>/dev/null || true

# Install podman-compose
echo "[INFO] Installing podman-compose..."
if command -v pip3 &>/dev/null; then
    sudo pip3 install podman-compose 2>/dev/null || true
elif command -v pip &>/dev/null; then
    sudo pip install podman-compose 2>/dev/null || true
else
    echo "[WARN] pip not found, skipping podman-compose installation."
fi

# Verify installation
echo "[INFO] Verifying Podman installation..."
if sudo podman run --rm hello-world; then
    echo "[INFO] Podman installation completed successfully!"
    podman --version
else
    echo "[ERROR] Podman installation verification failed."
    exit 1
fi