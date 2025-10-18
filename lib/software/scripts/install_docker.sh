#!/bin/bash

set -e

echo "[INFO] Starting Docker installation script"

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
    ubuntu|debian|rhel|fedora|centos)
        echo "[INFO] Detected OS: $PRETTY_NAME"
        ;;
    *)
        echo "[ERROR] Unsupported distribution: $ID"
        exit 1
        ;;
esac

# Remove old Docker packages
echo "[INFO] Removing old Docker packages..."

if [[ "$ID" == "ubuntu" || "$ID" == "debian" ]]; then
    sudo apt-get remove -y docker docker-engine docker.io containerd runc 2>/dev/null || true
else
    sudo dnf remove -y docker docker-client docker-client-latest docker-common \
        docker-latest docker-latest-logrotate docker-logrotate docker-selinux \
        docker-engine-selinux docker-engine 2>/dev/null || true
fi

# Install Docker
echo "[INFO] Installing Docker..."

if [[ "$ID" == "ubuntu" || "$ID" == "debian" ]]; then
    # APT-based installation
    sudo apt-get update
    sudo apt-get install -y ca-certificates curl
    sudo install -m 0755 -d /etc/apt/keyrings
    sudo curl -fsSL https://download.docker.com/linux/$ID/gpg -o /etc/apt/keyrings/docker.asc
    sudo chmod a+r /etc/apt/keyrings/docker.asc

    if [[ "$ID" == "ubuntu" ]]; then
        CODENAME="${UBUNTU_CODENAME:-$VERSION_CODENAME}"
    else
        CODENAME="$VERSION_CODENAME"
    fi

    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/$ID $CODENAME stable" | \
        sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

    sudo apt-get update
    sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

else
    # DNF-based installation
    sudo dnf install -y dnf-plugins-core

    if [[ "$ID" == "fedora" ]] && command -v dnf-3 &> /dev/null; then
        sudo dnf-3 config-manager --add-repo https://download.docker.com/linux/$ID/docker-ce.repo
    else
        sudo dnf config-manager --add-repo https://download.docker.com/linux/$ID/docker-ce.repo
    fi

    sudo dnf install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi

# Start Docker service
echo "[INFO] Starting Docker service..."
sudo systemctl start docker
sudo systemctl enable docker

# Add user to docker group
echo "[INFO] Adding user to docker group..."
sudo usermod -aG docker $USER

# Verify installation
echo "[INFO] Verifying Docker installation..."
if sudo docker run --rm hello-world; then
    echo "[INFO] Docker installation completed successfully!"
    docker --version
    echo "[WARN] Please logout and login again or run 'newgrp docker' to apply group permissions."
else
    echo "[ERROR] Docker installation verification failed."
    exit 1
fi
