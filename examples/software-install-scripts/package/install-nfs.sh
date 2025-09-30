#!/bin/bash

## NFS Install Script by ish
## Working with Ubuntu 22.04

if [ "$EUID" -ne 0 ]; then
  echo "[!] Please run as root or use sudo!"
  exit 1
fi

apt update -y

# Install NFS Server
apt install -y nfs-kernel-server

# Configure ports
cat << EOF > /etc/default/nfs-kernel-server
RPCMOUNTDOPTS="--manage-gids --port 20048"
EOF

cat << EOF > /etc/default/nfs-common
STATDOPTS="--port 32765 --outgoing-port 32766"
EOF

if ! grep -q "^fs.nfs.nlm_tcpport" /etc/sysctl.conf; then
    cat << EOF >> /etc/sysctl.conf

fs.nfs.nlm_tcpport = 32803
fs.nfs.nlm_udpport = 32803
EOF
    sysctl -p
fi

# Setting NFS directory
mkdir -p /nfs/share
chown nobody:nogroup -R /nfs
chmod 755 /nfs
chmod 755 /nfs/share

# Write exports file
if ! grep -q "/nfs/share" /etc/exports; then
    cat << EOF >> /etc/exports

/nfs/share *(rw,sync,no_subtree_check,no_root_squash)
EOF
fi

# Apply changes
exportfs -ra
