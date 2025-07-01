#!/bin/sh

# Package name from argument
pkg="$1"

# Global variables for tracking
tmp_result=$(mktemp)
tmp_processed=$(mktemp)
tmp_includes=$(mktemp)

# Check if path should be skipped (virtual filesystems)
should_skip_path() {
    local path="$1"
    case "$path" in
        /proc/*|/dev/*|/sys/*|/run/*)
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Extract includes from a single file
extract_includes() {
    local file="$1"
    {
        # Find include directives with quotes
        grep -i "include.*[\"'].*[\"']" "$file" 2>/dev/null | grep -o "[\"'][^\"']*[\"']" | tr -d "'\"" || true
        # Find include directives without quotes
        grep -i "^[[:space:]]*include[[:space:]]*[^\"']" "$file" 2>/dev/null | sed 's/.*include[[:space:]]*//g' | sed 's/#.*//g' || true
        # Find configuration directories (.d pattern)
        grep -i -E "(conf|config)\.d/" "$file" 2>/dev/null | grep -o '\/[^[:space:]"]*' || true
        grep -i "\.d/" "$file" 2>/dev/null | grep -o '\/[^[:space:]"]*' || true
    } | while read -r inc_path; do
        # Skip empty paths
        [ -z "$inc_path" ] && continue

        # Convert relative paths to absolute
        if [ "${inc_path#/}" = "$inc_path" ]; then
            inc_path="$(dirname "$file")/$inc_path"
        fi

        # Skip virtual filesystem paths
        if should_skip_path "$inc_path"; then
            continue
        fi

        # Handle wildcard patterns
        if echo "$inc_path" | grep -q "[*]"; then
            find $(dirname "$inc_path" 2>/dev/null) -name "$(basename "$inc_path")" -type f 2>/dev/null | head -10
        else
            echo "$inc_path"
        fi
    done > "$tmp_includes"
}

# Process includes iteratively (not recursively)
process_includes() {
    local start_file="$1"
    local queue_file=$(mktemp)

    # Initialize queue with the starting file
    echo "$start_file" > "$queue_file"

    # Process queue until empty
    while [ -s "$queue_file" ]; do
        # Get next file from queue
        current_file=$(head -n1 "$queue_file")
        # Remove it from queue
        tail -n +2 "$queue_file" > "${queue_file}.tmp" && mv "${queue_file}.tmp" "$queue_file"

        # Skip if already processed
        real_current=$(readlink -f "$current_file" 2>/dev/null || echo "$current_file")
        if grep -Fxq "$real_current" "$tmp_processed" 2>/dev/null; then
            continue
        fi

        # Mark as processed
        echo "$real_current" >> "$tmp_processed"

        # Add to results if it's a real file
        if [ -f "$current_file" ]; then
            echo "$current_file" >> "$tmp_result"

            # Extract includes from this file
            extract_includes "$current_file"

            # Add new includes to queue
            while read -r new_include; do
                if [ -f "$new_include" ]; then
                    real_include=$(readlink -f "$new_include" 2>/dev/null || echo "$new_include")
                    # Only add if not already processed
                    if ! grep -Fxq "$real_include" "$tmp_processed" 2>/dev/null; then
                        echo "$new_include" >> "$queue_file"
                    fi
                fi
            done < "$tmp_includes"
        fi
    done

    rm -f "$queue_file"
}

# Check file status (Original/Modified/Custom)
check_config_status() {
    local file="$1"
    local real_file=$(readlink -f "$file")

    if command -v dpkg >/dev/null 2>&1; then
        # Find package that owns the file
        local owner_pkg=$(dpkg -S "$real_file" 2>/dev/null | cut -d: -f1)
        if [ -n "$owner_pkg" ]; then
            if dpkg -V "$owner_pkg" 2>/dev/null | grep -q "^..5.* $real_file"; then
                echo "$file [Modified]"
            else
                echo "$file"
            fi
        else
            if [ "$file" != "$real_file" ] && dpkg -S "$file" >/dev/null 2>&1; then
                echo "$file"
            else
                echo "$file [Custom]"
            fi
        fi
    elif command -v rpm >/dev/null 2>&1; then
        # Find package that owns the file
        local owner_pkg=$(rpm -qf "$real_file" 2>/dev/null)
        if [ -n "$owner_pkg" ]; then
            if rpm -V "$owner_pkg" | grep -q "^..5.* $real_file"; then
                echo "$file [Modified]"
            else
                echo "$file"
            fi
        else
            if [ "$file" != "$real_file" ] && rpm -qf "$file" >/dev/null 2>&1; then
                echo "$file"
            else
                echo "$file [Custom]"
            fi
        fi
    else
        echo "$file [Custom]"
    fi
}

# Cleanup function
cleanup() {
    rm -f "$tmp_result" "$tmp_processed" "$tmp_includes"
}

# Set trap for cleanup
trap cleanup EXIT

{
    # Search using package manager
    if command -v dpkg >/dev/null 2>&1; then
        dpkg -L "$pkg" 2>/dev/null
    elif command -v rpm >/dev/null 2>&1; then
        rpm -ql "$pkg" 2>/dev/null
    fi

    # Search for config files in process arguments
    ps aux | grep "$pkg" | grep -o '[[:space:]]/[^[:space:]]*'

    # Search common config directories
    for base in "/etc" "/usr/local/etc"; do
        find "$base" -type f -path "*/$pkg*" ! -path "/etc/apt/*" 2>/dev/null
        find "$base/$pkg" -type f ! -path "/etc/apt/*" 2>/dev/null 2>/dev/null
    done

    # Search systemd and init.d configurations
    if [ -d "/etc/systemd" ]; then
        find /etc/systemd -type f -exec grep -l "$pkg" {} \; 2>/dev/null
    fi

    if [ -d "/etc/init.d" ]; then
        find /etc/init.d -type f -exec grep -l "$pkg" {} \; 2>/dev/null
    fi

} | sort -u | while read -r file; do
    # Skip virtual filesystem paths
    if should_skip_path "$file"; then
        continue
    fi

    # Filter config files by common patterns and backup files
    if echo "$file" | grep -i -E '\.(conf|cfg|ini|yaml|yml|bak)$|conf\.d|\.d/|config$|[._]bak$' >/dev/null 2>&1; then
        if [ -f "$file" ]; then
            process_includes "$file"
        fi
    # Check content of non-standard named files for config-like content
    elif [ -f "$file" ] && file "$file" | grep -i "text" >/dev/null 2>&1; then
        if grep -i -E '(server|listen|port|host|config|ssl|virtual|root|location)' "$file" >/dev/null 2>&1; then
            process_includes "$file"
        fi
    fi
done

# Process and output unique results with status
sort -u "$tmp_result" | while read -r file; do
    check_config_status "$file"
done
