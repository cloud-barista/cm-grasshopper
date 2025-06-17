#!/bin/sh

# Certificate and key path finder script
# Usage: cert_key_finder.sh <config_file_path>

config_file="$1"

if [ -z "$config_file" ]; then
    echo "Usage: $0 <config_file_path>"
    exit 1
fi

if [ ! -f "$config_file" ]; then
    echo "Error: Config file '$config_file' not found"
    exit 1
fi

# Find certificate and key paths
grep -i -E '(ssl|tls)_(certificate|key|trusted_certificate|client_certificate|dhparam)|key_file|cert_file|ca_file|private_key|public_key|certificate|keyfile|certfile|cafile' "$config_file" 2>/dev/null || true