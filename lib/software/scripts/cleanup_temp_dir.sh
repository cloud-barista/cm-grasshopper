#!/bin/bash

# Cleanup temporary directory script
# Usage: ./cleanup_temp_dir.sh [directory_path]

if [ -z "$1" ]; then
    echo "Usage: $0 <directory_path>"
    exit 1
fi

TARGET_DIR="$1"

if [ ! -d "$TARGET_DIR" ]; then
    echo "Directory does not exist: $TARGET_DIR"
    exit 0
fi

echo "Cleaning up temporary directory: $TARGET_DIR"

# Safety check: only remove directories in /tmp that match our pattern
if [[ "$TARGET_DIR" == /tmp/system_migration_* ]] || [[ "$TARGET_DIR" == /tmp/gpg_migration_* ]]; then
    rm -rf "$TARGET_DIR"
    if [ $? -eq 0 ]; then
        echo "Successfully cleaned up: $TARGET_DIR"
    else
        echo "Failed to clean up: $TARGET_DIR"
        exit 1
    fi
else
    echo "Safety check failed: Will not remove directory outside of expected patterns"
    echo "Expected: /tmp/system_migration_* or /tmp/gpg_migration_*"
    echo "Received: $TARGET_DIR"
    exit 1
fi