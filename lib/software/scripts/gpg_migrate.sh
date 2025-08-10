#!/bin/bash

# GPG Key Migration Script
# Usage: ./gpg_migrate.sh [collect|deploy|transfer] [target_host]

SCRIPT_DIR="$(dirname "$0")"
BACKUP_DIR="$SCRIPT_DIR/gpg_backup_$(date +%Y%m%d_%H%M%S)"
TRANSFER_DIR="$SCRIPT_DIR/gpg_transfer"

show_usage() {
    echo "Usage: $0 <command> [target_host]"
    echo ""
    echo "Commands:"
    echo "  collect                 - Collect GPG keys from current system"
    echo "  deploy                  - Deploy collected keys to current system"
    echo "  transfer <target_host>  - Collect and transfer keys to target host"
    echo ""
    echo "Examples:"
    echo "  $0 collect              # Collect keys from current system"
    echo "  $0 deploy               # Deploy keys to current system"
    echo "  $0 transfer user@server # Collect and transfer to remote server"
}

collect_gpg_keys() {
    local backup_dir="$1"

    echo "=== Collecting GPG keys from current system ==="
    mkdir -p "$backup_dir/keyrings"
    mkdir -p "$backup_dir/trusted"
    mkdir -p "$backup_dir/sources"

    # Collect keyring files from /usr/share/keyrings
    echo "Collecting keyrings from /usr/share/keyrings..."
    if [ -d "/usr/share/keyrings" ]; then
        find /usr/share/keyrings -name "*.gpg" -type f | while read -r keyfile; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")
                cp "$keyfile" "$backup_dir/keyrings/"

                # Get fingerprint and metadata
                fingerprint=$(gpg --with-fingerprint --with-colons "$keyfile" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
                keyinfo=$(gpg --with-fingerprint "$keyfile" 2>/dev/null | head -10)

                echo "File: $filename" >> "$backup_dir/keyrings_info.txt"
                echo "Fingerprint: $fingerprint" >> "$backup_dir/keyrings_info.txt"
                echo "Info: $keyinfo" >> "$backup_dir/keyrings_info.txt"
                echo "---" >> "$backup_dir/keyrings_info.txt"
            fi
        done
    fi

    # Collect trusted GPG keys from /etc/apt/trusted.gpg.d
    echo "Collecting trusted keys from /etc/apt/trusted.gpg.d..."
    if [ -d "/etc/apt/trusted.gpg.d" ]; then
        find /etc/apt/trusted.gpg.d -name "*.gpg" -type f | while read -r keyfile; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")
                cp "$keyfile" "$backup_dir/trusted/"

                # Get fingerprint and metadata
                fingerprint=$(gpg --with-fingerprint --with-colons "$keyfile" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
                keyinfo=$(gpg --with-fingerprint "$keyfile" 2>/dev/null | head -10)

                echo "File: $filename" >> "$backup_dir/trusted_info.txt"
                echo "Fingerprint: $fingerprint" >> "$backup_dir/trusted_info.txt"
                echo "Info: $keyinfo" >> "$backup_dir/trusted_info.txt"
                echo "---" >> "$backup_dir/trusted_info.txt"
            fi
        done
    fi

    # Collect legacy trusted.gpg if exists
    if [ -f "/etc/apt/trusted.gpg" ]; then
        echo "Collecting legacy trusted.gpg..."
        cp "/etc/apt/trusted.gpg" "$backup_dir/trusted/"

        fingerprint=$(gpg --with-fingerprint --with-colons "/etc/apt/trusted.gpg" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
        keyinfo=$(gpg --with-fingerprint "/etc/apt/trusted.gpg" 2>/dev/null | head -10)

        echo "File: trusted.gpg (legacy)" >> "$backup_dir/trusted_info.txt"
        echo "Fingerprint: $fingerprint" >> "$backup_dir/trusted_info.txt"
        echo "Info: $keyinfo" >> "$backup_dir/trusted_info.txt"
        echo "---" >> "$backup_dir/trusted_info.txt"
    fi

    # Collect repository sources for reference
    echo "Collecting repository sources for reference..."
    if [ -d "/etc/apt/sources.list.d" ]; then
        find /etc/apt/sources.list.d -name "*.list" -o -name "*.sources" | while read -r sourcefile; do
            if [ -f "$sourcefile" ]; then
                filename=$(basename "$sourcefile")
                cp "$sourcefile" "$backup_dir/sources/"
            fi
        done
    fi

    if [ -f "/etc/apt/sources.list" ]; then
        cp "/etc/apt/sources.list" "$backup_dir/sources/"
    fi

    # Create summary
    echo "=== GPG Key Collection Summary ===" > "$backup_dir/SUMMARY.txt"
    echo "Collection Date: $(date)" >> "$backup_dir/SUMMARY.txt"
    echo "Source System: $(hostname)" >> "$backup_dir/SUMMARY.txt"
    echo "" >> "$backup_dir/SUMMARY.txt"
    echo "Keyrings collected: $(ls -1 "$backup_dir/keyrings" 2>/dev/null | wc -l)" >> "$backup_dir/SUMMARY.txt"
    echo "Trusted keys collected: $(ls -1 "$backup_dir/trusted" 2>/dev/null | wc -l)" >> "$backup_dir/SUMMARY.txt"
    echo "Source files collected: $(ls -1 "$backup_dir/sources" 2>/dev/null | wc -l)" >> "$backup_dir/SUMMARY.txt"
    echo "" >> "$backup_dir/SUMMARY.txt"
    echo "Files in keyrings:" >> "$backup_dir/SUMMARY.txt"
    ls -la "$backup_dir/keyrings" >> "$backup_dir/SUMMARY.txt" 2>/dev/null
    echo "" >> "$backup_dir/SUMMARY.txt"
    echo "Files in trusted:" >> "$backup_dir/SUMMARY.txt"
    ls -la "$backup_dir/trusted" >> "$backup_dir/SUMMARY.txt" 2>/dev/null

    echo "GPG keys collected in: $backup_dir"
    echo "Summary:"
    cat "$backup_dir/SUMMARY.txt"
}

deploy_gpg_keys() {
    local source_dir="$1"

    if [ ! -d "$source_dir" ]; then
        echo "Error: Source directory $source_dir not found"
        exit 1
    fi

    echo "=== Deploying GPG keys to current system ==="

    # Create directories if they don't exist
    mkdir -p /usr/share/keyrings
    mkdir -p /etc/apt/trusted.gpg.d

    # Deploy keyrings
    if [ -d "$source_dir/keyrings" ]; then
        echo "Deploying keyrings..."
        for keyfile in "$source_dir/keyrings"/*.gpg; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")

                # Check if key already exists by fingerprint
                new_fingerprint=$(gpg --with-fingerprint --with-colons "$keyfile" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')

                existing_key_found=false
                if [ -n "$new_fingerprint" ]; then
                    # Check existing keyrings
                    for existing_key in /usr/share/keyrings/*.gpg; do
                        if [ -f "$existing_key" ]; then
                            existing_fingerprint=$(gpg --with-fingerprint --with-colons "$existing_key" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
                            if [ "$new_fingerprint" = "$existing_fingerprint" ]; then
                                echo "  Key $filename already exists (fingerprint match): $(basename "$existing_key")"
                                existing_key_found=true
                                break
                            fi
                        fi
                    done
                fi

                if [ "$existing_key_found" = false ]; then
                    echo "  Installing new key: $filename"
                    cp "$keyfile" "/usr/share/keyrings/"
                    chmod 644 "/usr/share/keyrings/$filename"
                fi
            fi
        done
    fi

    # Deploy trusted keys
    if [ -d "$source_dir/trusted" ]; then
        echo "Deploying trusted keys..."
        for keyfile in "$source_dir/trusted"/*.gpg; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")

                # Skip legacy trusted.gpg for now (needs special handling)
                if [ "$filename" = "trusted.gpg" ]; then
                    echo "  Skipping legacy trusted.gpg (manual intervention may be required)"
                    continue
                fi

                # Check if key already exists by fingerprint
                new_fingerprint=$(gpg --with-fingerprint --with-colons "$keyfile" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')

                existing_key_found=false
                if [ -n "$new_fingerprint" ]; then
                    # Check existing trusted keys
                    for existing_key in /etc/apt/trusted.gpg.d/*.gpg; do
                        if [ -f "$existing_key" ]; then
                            existing_fingerprint=$(gpg --with-fingerprint --with-colons "$existing_key" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
                            if [ "$new_fingerprint" = "$existing_fingerprint" ]; then
                                echo "  Key $filename already exists (fingerprint match): $(basename "$existing_key")"
                                existing_key_found=true
                                break
                            fi
                        fi
                    done
                fi

                if [ "$existing_key_found" = false ]; then
                    echo "  Installing new trusted key: $filename"
                    cp "$keyfile" "/etc/apt/trusted.gpg.d/"
                    chmod 644 "/etc/apt/trusted.gpg.d/$filename"
                fi
            fi
        done
    fi

    echo "GPG key deployment completed!"
    echo "You may want to run 'apt update' to refresh package lists."
}

transfer_to_remote() {
    local target_host="$1"
    local temp_dir=$(mktemp -d)

    echo "=== Transferring GPG keys to $target_host ==="

    # Collect keys locally
    collect_gpg_keys "$temp_dir"

    # Create deployment script
    cat > "$temp_dir/deploy.sh" << 'EOF'
#!/bin/bash
# Auto-generated deployment script

DEPLOY_DIR="$(dirname "$0")"

echo "=== Deploying GPG keys ==="

# Create directories
mkdir -p /usr/share/keyrings
mkdir -p /etc/apt/trusted.gpg.d

# Deploy keyrings
if [ -d "$DEPLOY_DIR/keyrings" ]; then
    echo "Deploying keyrings..."
    for keyfile in "$DEPLOY_DIR/keyrings"/*.gpg; do
        if [ -f "$keyfile" ]; then
            filename=$(basename "$keyfile")
            echo "  Installing: $filename"
            cp "$keyfile" "/usr/share/keyrings/"
            chmod 644 "/usr/share/keyrings/$filename"
        fi
    done
fi

# Deploy trusted keys
if [ -d "$DEPLOY_DIR/trusted" ]; then
    echo "Deploying trusted keys..."
    for keyfile in "$DEPLOY_DIR/trusted"/*.gpg; do
        if [ -f "$keyfile" ]; then
            filename=$(basename "$keyfile")
            if [ "$filename" != "trusted.gpg" ]; then
                echo "  Installing: $filename"
                cp "$keyfile" "/etc/apt/trusted.gpg.d/"
                chmod 644 "/etc/apt/trusted.gpg.d/$filename"
            fi
        fi
    done
fi

echo "Deployment completed!"
echo "Run 'apt update' to refresh package lists."
EOF

    chmod +x "$temp_dir/deploy.sh"

    # Transfer to remote host
    echo "Transferring files to $target_host..."
    scp -r "$temp_dir" "$target_host:/tmp/gpg_migration_$(date +%Y%m%d_%H%M%S)"

    if [ $? -eq 0 ]; then
        echo "Transfer completed successfully!"
        echo "On the target host, run:"
        echo "  sudo /tmp/gpg_migration_*/deploy.sh"
    else
        echo "Transfer failed!"
        exit 1
    fi

    # Cleanup
    rm -rf "$temp_dir"
}

# Main script logic
case "$1" in
    collect)
        if [ -z "$2" ]; then
            collect_gpg_keys "$BACKUP_DIR"
        else
            collect_gpg_keys "$2"
        fi
        ;;
    deploy)
        if [ -z "$2" ]; then
            echo "Error: Please specify source directory"
            echo "Usage: $0 deploy <source_directory>"
            exit 1
        fi
        deploy_gpg_keys "$2"
        ;;
    transfer)
        if [ -z "$2" ]; then
            echo "Error: Please specify target host"
            echo "Usage: $0 transfer <target_host>"
            exit 1
        fi
        transfer_to_remote "$2"
        ;;
    *)
        show_usage
        exit 1
        ;;
esac