#!/bin/bash

# System Migration Script
# Usage: ./system_migrate.sh [collect|deploy|transfer] [target_host]

SCRIPT_DIR="$(dirname "$0")"
BACKUP_DIR="$SCRIPT_DIR/system_backup_$(date +%Y%m%d_%H%M%S)"
TRANSFER_DIR="$SCRIPT_DIR/system_transfer"

show_usage() {
    echo "Usage: $0 <command> [target_host]"
    echo ""
    echo "Commands:"
    echo "  collect                 - Collect system configuration from current system"
    echo "  deploy                  - Deploy collected configuration to current system" 
    echo "  transfer <target_host>  - Collect and transfer configuration to target host"
    echo ""
    echo "Examples:"
    echo "  $0 collect              # Collect from current system"
    echo "  $0 deploy <source_dir>  # Deploy to current system"
    echo "  $0 transfer user@server # Collect and transfer to remote server"
}

detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_FAMILY=$ID
        OS_VERSION=$VERSION_ID
        
        case $OS_FAMILY in
            ubuntu|debian)
                PACKAGE_MANAGER="apt"
                REPO_DIR="/etc/apt/sources.list.d"
                MAIN_REPO_FILE="/etc/apt/sources.list"
                GPG_TRUSTED_DIR="/etc/apt/trusted.gpg.d"
                GPG_KEYRING_DIR="/usr/share/keyrings"
                GPG_ETC_KEYRING_DIR="/etc/apt/keyrings"
                ;;
            rhel|centos|fedora|rocky|almalinux)
                PACKAGE_MANAGER="yum"
                REPO_DIR="/etc/yum.repos.d"
                MAIN_REPO_FILE=""
                GPG_TRUSTED_DIR="/etc/pki/rpm-gpg"
                GPG_KEYRING_DIR=""
                ;;
            *)
                echo "Unsupported OS family: $OS_FAMILY"
                exit 1
                ;;
        esac
    else
        echo "Cannot detect OS. /etc/os-release not found."
        exit 1
    fi
}

collect_debian_config() {
    local backup_dir="$1"
    
    echo "=== Collecting Debian/Ubuntu configuration ==="
    
    # Create directories
    mkdir -p "$backup_dir/repos"
    mkdir -p "$backup_dir/keyrings"
    mkdir -p "$backup_dir/etc_keyrings"
    mkdir -p "$backup_dir/trusted"
    
    # Collect repository sources
    echo "Collecting repository sources..."
    if [ -d "$REPO_DIR" ]; then
        find "$REPO_DIR" -name "*.list" -o -name "*.sources" | while read -r sourcefile; do
            if [ -f "$sourcefile" ]; then
                filename=$(basename "$sourcefile")
                cp "$sourcefile" "$backup_dir/repos/"
                echo "Collected: $filename" >> "$backup_dir/repos_info.txt"
            fi
        done
    fi
    
    if [ -f "$MAIN_REPO_FILE" ]; then
        cp "$MAIN_REPO_FILE" "$backup_dir/repos/"
        echo "Collected: $(basename "$MAIN_REPO_FILE")" >> "$backup_dir/repos_info.txt"
    fi
    
    # Collect GPG keyrings
    echo "Collecting keyrings from $GPG_KEYRING_DIR..."
    if [ -d "$GPG_KEYRING_DIR" ]; then
        find "$GPG_KEYRING_DIR" -name "*.gpg" -type f | while read -r keyfile; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")
                cp "$keyfile" "$backup_dir/keyrings/"
                
                # Get fingerprint and metadata
                fingerprint=$(gpg --with-fingerprint --with-colons "$keyfile" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
                keyinfo=$(gpg --with-fingerprint "$keyfile" 2>/dev/null | head -5)
                
                echo "File: $filename" >> "$backup_dir/keyrings_info.txt"
                echo "Fingerprint: $fingerprint" >> "$backup_dir/keyrings_info.txt" 
                echo "Info: $keyinfo" >> "$backup_dir/keyrings_info.txt"
                echo "---" >> "$backup_dir/keyrings_info.txt"
            fi
        done
    fi
    
    # Collect /etc/apt/keyrings
    echo "Collecting /etc/apt/keyrings from $GPG_ETC_KEYRING_DIR..."
    if [ -d "$GPG_ETC_KEYRING_DIR" ]; then
        find "$GPG_ETC_KEYRING_DIR" \( -name "*.gpg" -o -name "*.asc" \) -type f | while read -r keyfile; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")
                cp "$keyfile" "$backup_dir/etc_keyrings/"
                
                # Get fingerprint and metadata
                fingerprint=$(gpg --with-fingerprint --with-colons "$keyfile" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
                keyinfo=$(gpg --with-fingerprint "$keyfile" 2>/dev/null | head -5)
                
                echo "File: $filename" >> "$backup_dir/etc_keyrings_info.txt"
                echo "Fingerprint: $fingerprint" >> "$backup_dir/etc_keyrings_info.txt" 
                echo "Info: $keyinfo" >> "$backup_dir/etc_keyrings_info.txt"
                echo "---" >> "$backup_dir/etc_keyrings_info.txt"
            fi
        done
    fi
    
    # Collect trusted GPG keys
    echo "Collecting trusted keys from $GPG_TRUSTED_DIR..."
    if [ -d "$GPG_TRUSTED_DIR" ]; then
        find "$GPG_TRUSTED_DIR" -name "*.gpg" -type f | while read -r keyfile; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")
                cp "$keyfile" "$backup_dir/trusted/"
                
                # Get fingerprint and metadata
                fingerprint=$(gpg --with-fingerprint --with-colons "$keyfile" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
                keyinfo=$(gpg --with-fingerprint "$keyfile" 2>/dev/null | head -5)
                
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
        keyinfo=$(gpg --with-fingerprint "/etc/apt/trusted.gpg" 2>/dev/null | head -5)
        
        echo "File: trusted.gpg (legacy)" >> "$backup_dir/trusted_info.txt"
        echo "Fingerprint: $fingerprint" >> "$backup_dir/trusted_info.txt"
        echo "Info: $keyinfo" >> "$backup_dir/trusted_info.txt"
        echo "---" >> "$backup_dir/trusted_info.txt"
    fi
}

collect_rhel_config() {
    local backup_dir="$1"
    
    echo "=== Collecting RHEL/CentOS configuration ==="
    
    # Create directories
    mkdir -p "$backup_dir/repos"
    mkdir -p "$backup_dir/gpg_keys"
    
    # Collect repository files
    echo "Collecting repository files from $REPO_DIR..."
    if [ -d "$REPO_DIR" ]; then
        find "$REPO_DIR" -name "*.repo" | while read -r repofile; do
            if [ -f "$repofile" ]; then
                filename=$(basename "$repofile")
                cp "$repofile" "$backup_dir/repos/"
                echo "Collected: $filename" >> "$backup_dir/repos_info.txt"
            fi
        done
    fi
    
    # Collect GPG keys
    echo "Collecting GPG keys from $GPG_TRUSTED_DIR..."
    if [ -d "$GPG_TRUSTED_DIR" ]; then
        find "$GPG_TRUSTED_DIR" -type f | while read -r keyfile; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")
                cp "$keyfile" "$backup_dir/gpg_keys/"
                
                # Try to get key info (may not always work for RPM keys)
                keyinfo=$(gpg --with-fingerprint "$keyfile" 2>/dev/null | head -5 || echo "RPM key - fingerprint not available via gpg")
                
                echo "File: $filename" >> "$backup_dir/gpg_keys_info.txt"
                echo "Info: $keyinfo" >> "$backup_dir/gpg_keys_info.txt"
                echo "---" >> "$backup_dir/gpg_keys_info.txt"
            fi
        done
    fi
    
    # Collect installed RPM GPG keys info
    echo "Collecting installed RPM GPG keys info..."
    rpm -qa gpg-pubkey* > "$backup_dir/installed_gpg_keys.txt" 2>/dev/null || true
}

deploy_debian_config() {
    local source_dir="$1"
    
    echo "=== Deploying Debian/Ubuntu configuration ==="
    
    # Create directories if they don't exist
    mkdir -p "$REPO_DIR"
    mkdir -p "$GPG_KEYRING_DIR" 
    mkdir -p "$GPG_ETC_KEYRING_DIR"
    mkdir -p "$GPG_TRUSTED_DIR"
    
    # Deploy repository sources
    if [ -d "$source_dir/repos" ]; then
        echo "Deploying repository sources..."
        for repofile in "$source_dir/repos"/*; do
            if [ -f "$repofile" ]; then
                filename=$(basename "$repofile")
                
                # Check if it's the main sources.list
                if [ "$filename" = "sources.list" ]; then
                    target_path="$MAIN_REPO_FILE"
                else
                    target_path="$REPO_DIR/$filename"
                fi
                
                # Check if file already exists
                if [ -f "$target_path" ]; then
                    echo "  Backing up existing: $filename -> $filename.bak"
                    cp "$target_path" "$target_path.bak"
                fi
                
                echo "  Installing: $filename"
                cp "$repofile" "$target_path"
                chmod 644 "$target_path"
            fi
        done
    fi
    
    # Deploy keyrings
    if [ -d "$source_dir/keyrings" ]; then
        echo "Deploying keyrings..."
        for keyfile in "$source_dir/keyrings"/*.gpg; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")
                target_path="$GPG_KEYRING_DIR/$filename"
                
                # Check if key already exists by fingerprint
                new_fingerprint=$(gpg --with-fingerprint --with-colons "$keyfile" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
                
                existing_key_found=false
                if [ -n "$new_fingerprint" ]; then
                    for existing_key in "$GPG_KEYRING_DIR"/*.gpg; do
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
                    cp "$keyfile" "$target_path"
                    chmod 644 "$target_path"
                fi
            fi
        done
    fi
    
    # Deploy /etc/apt/keyrings
    if [ -d "$source_dir/etc_keyrings" ]; then
        echo "Deploying /etc/apt/keyrings..."
        for keyfile in "$source_dir/etc_keyrings"/*; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")
                # Skip if not a key file
                case "$filename" in
                    *.gpg|*.asc) ;;
                    *) continue ;;
                esac
                target_path="$GPG_ETC_KEYRING_DIR/$filename"
                
                # Check if key already exists by fingerprint
                new_fingerprint=$(gpg --with-fingerprint --with-colons "$keyfile" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
                
                existing_key_found=false
                if [ -n "$new_fingerprint" ]; then
                    for existing_key in "$GPG_ETC_KEYRING_DIR"/*; do
                        if [ -f "$existing_key" ]; then
                            case "$(basename "$existing_key")" in
                                *.gpg|*.asc) ;;
                                *) continue ;;
                            esac
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
                    echo "  Installing new /etc/apt/keyrings key: $filename"
                    cp "$keyfile" "$target_path"
                    chmod 644 "$target_path"
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
                
                target_path="$GPG_TRUSTED_DIR/$filename"
                
                # Check if key already exists by fingerprint
                new_fingerprint=$(gpg --with-fingerprint --with-colons "$keyfile" 2>/dev/null | awk -F ':' '/fpr:/ { print $10; exit }')
                
                existing_key_found=false
                if [ -n "$new_fingerprint" ]; then
                    for existing_key in "$GPG_TRUSTED_DIR"/*.gpg; do
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
                    cp "$keyfile" "$target_path"
                    chmod 644 "$target_path"
                fi
            fi
        done
    fi
}

deploy_rhel_config() {
    local source_dir="$1"
    
    echo "=== Deploying RHEL/CentOS configuration ==="
    
    # Create directories if they don't exist
    mkdir -p "$REPO_DIR"
    mkdir -p "$GPG_TRUSTED_DIR"
    
    # Deploy repository files
    if [ -d "$source_dir/repos" ]; then
        echo "Deploying repository files..."
        for repofile in "$source_dir/repos"/*.repo; do
            if [ -f "$repofile" ]; then
                filename=$(basename "$repofile")
                target_path="$REPO_DIR/$filename"
                
                # Check if file already exists
                if [ -f "$target_path" ]; then
                    echo "  Backing up existing: $filename -> $filename.bak"
                    cp "$target_path" "$target_path.bak"
                fi
                
                echo "  Installing: $filename"
                cp "$repofile" "$target_path"
                chmod 644 "$target_path"
            fi
        done
    fi
    
    # Deploy GPG keys
    if [ -d "$source_dir/gpg_keys" ]; then
        echo "Deploying GPG keys..."
        for keyfile in "$source_dir/gpg_keys"/*; do
            if [ -f "$keyfile" ]; then
                filename=$(basename "$keyfile")
                target_path="$GPG_TRUSTED_DIR/$filename"
                
                # Check if file already exists
                if [ -f "$target_path" ]; then
                    echo "  Key $filename already exists, skipping"
                    continue
                fi
                
                echo "  Installing: $filename"
                cp "$keyfile" "$target_path"
                chmod 644 "$target_path"
                
                # Try to import the key
                rpm --import "$target_path" 2>/dev/null || echo "    Note: Could not import $filename as RPM key"
            fi
        done
    fi
}

collect_system_config() {
    local backup_dir="$1"
    
    detect_os
    
    echo "=== Collecting System Configuration ==="
    echo "OS Family: $OS_FAMILY"
    echo "Package Manager: $PACKAGE_MANAGER"
    
    # Create main backup directory
    mkdir -p "$backup_dir"
    
    # Create system info file
    echo "=== System Configuration Collection ===" > "$backup_dir/SYSTEM_INFO.txt"
    echo "Collection Date: $(date)" >> "$backup_dir/SYSTEM_INFO.txt"
    echo "Source System: $(hostname)" >> "$backup_dir/SYSTEM_INFO.txt"
    echo "OS Family: $OS_FAMILY" >> "$backup_dir/SYSTEM_INFO.txt"
    echo "OS Version: $OS_VERSION" >> "$backup_dir/SYSTEM_INFO.txt"
    echo "Package Manager: $PACKAGE_MANAGER" >> "$backup_dir/SYSTEM_INFO.txt"
    echo "" >> "$backup_dir/SYSTEM_INFO.txt"
    
    case $PACKAGE_MANAGER in
        apt)
            collect_debian_config "$backup_dir"
            ;;
        yum)
            collect_rhel_config "$backup_dir"
            ;;
        *)
            echo "Unsupported package manager: $PACKAGE_MANAGER"
            exit 1
            ;;
    esac
    
    # Add collection summary
    echo "Repository files collected: $(find "$backup_dir/repos" -type f 2>/dev/null | wc -l)" >> "$backup_dir/SYSTEM_INFO.txt"
    if [ -d "$backup_dir/keyrings" ]; then
        echo "Keyrings collected: $(find "$backup_dir/keyrings" -type f 2>/dev/null | wc -l)" >> "$backup_dir/SYSTEM_INFO.txt"
    fi
    if [ -d "$backup_dir/etc_keyrings" ]; then
        echo "/etc/apt/keyrings collected: $(find "$backup_dir/etc_keyrings" -type f 2>/dev/null | wc -l)" >> "$backup_dir/SYSTEM_INFO.txt"
    fi
    if [ -d "$backup_dir/trusted" ]; then
        echo "Trusted keys collected: $(find "$backup_dir/trusted" -type f 2>/dev/null | wc -l)" >> "$backup_dir/SYSTEM_INFO.txt"
    fi
    if [ -d "$backup_dir/gpg_keys" ]; then
        echo "GPG keys collected: $(find "$backup_dir/gpg_keys" -type f 2>/dev/null | wc -l)" >> "$backup_dir/SYSTEM_INFO.txt"
    fi
    
    echo ""
    echo "System configuration collected in: $backup_dir"
    echo "Summary:"
    cat "$backup_dir/SYSTEM_INFO.txt"
}

deploy_system_config() {
    local source_dir="$1"
    
    if [ ! -d "$source_dir" ]; then
        echo "Error: Source directory $source_dir not found"
        exit 1
    fi
    
    # Read system info to determine deployment strategy
    if [ -f "$source_dir/SYSTEM_INFO.txt" ]; then
        source_os_family=$(grep "OS Family:" "$source_dir/SYSTEM_INFO.txt" | cut -d: -f2 | tr -d ' ')
        source_package_manager=$(grep "Package Manager:" "$source_dir/SYSTEM_INFO.txt" | cut -d: -f2 | tr -d ' ')
    else
        echo "Warning: No system info file found, attempting to detect from directory structure"
        if [ -d "$source_dir/repos" ] && [ -d "$source_dir/keyrings" ]; then
            source_package_manager="apt"
        elif [ -d "$source_dir/repos" ] && [ -d "$source_dir/gpg_keys" ]; then
            source_package_manager="yum"
        else
            echo "Error: Cannot determine source system type"
            exit 1
        fi
    fi
    
    detect_os
    
    echo "=== Deploying System Configuration ==="
    echo "Source Package Manager: $source_package_manager"
    echo "Target Package Manager: $PACKAGE_MANAGER"
    
    # Check compatibility
    if [ "$source_package_manager" != "$PACKAGE_MANAGER" ]; then
        echo "Warning: Package managers don't match. This may require manual intervention."
        echo "Source: $source_package_manager, Target: $PACKAGE_MANAGER"
    fi
    
    case $PACKAGE_MANAGER in
        apt)
            deploy_debian_config "$source_dir"
            echo ""
            echo "Deployment completed! Run 'apt update' to refresh package lists."
            ;;
        yum)
            deploy_rhel_config "$source_dir"
            echo ""
            echo "Deployment completed! Run 'yum clean all && yum makecache' to refresh package lists."
            ;;
        *)
            echo "Unsupported target package manager: $PACKAGE_MANAGER"
            exit 1
            ;;
    esac
}

# Main script logic
case "$1" in
    collect)
        if [ -z "$2" ]; then
            collect_system_config "$BACKUP_DIR"
        else
            collect_system_config "$2"
        fi
        ;;
    deploy)
        if [ -z "$2" ]; then
            echo "Error: Please specify source directory"
            echo "Usage: $0 deploy <source_directory>"
            exit 1
        fi
        deploy_system_config "$2"
        ;;
    *)
        show_usage
        exit 1
        ;;
esac