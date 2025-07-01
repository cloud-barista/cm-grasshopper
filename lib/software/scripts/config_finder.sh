#!/bin/sh

# Package name from argument
pkg="$1"

# Validate package name
if [ -z "$pkg" ]; then
    echo "Usage: $0 <package_name>" >&2
    exit 1
fi

# Check if package exists
if command -v dpkg >/dev/null 2>&1; then
    if ! dpkg -l "$pkg" 2>/dev/null | grep -q "^ii"; then
        echo "Package '$pkg' is not installed" >&2
        exit 1
    fi
elif command -v rpm >/dev/null 2>&1; then
    if ! rpm -q "$pkg" >/dev/null 2>&1; then
        echo "Package '$pkg' is not installed" >&2
        exit 1
    fi
else
    echo "Neither dpkg nor rpm found. Unsupported system." >&2
    exit 1
fi

# Global variables for tracking
tmp_result=$(mktemp)
tmp_processed=$(mktemp)
tmp_includes=$(mktemp)
tmp_dep_packages=$(mktemp)

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

# Get dependency packages for Debian-based systems
get_debian_dependencies() {
    local pkg="$1"

    # Get dependencies using dpkg-query
    dpkg-query -Wf='${Depends}\n${Pre-Depends}\n' "$pkg" 2>/dev/null | \
    tr ',' '\n' | \
    sed 's/|.*//' | \
    sed 's/(.*//' | \
    sed 's/^[[:space:]]*//' | \
    sed 's/[[:space:]]*$//' | \
    grep -v '^$' | \
    while read -r dep; do
        if dpkg -l "$dep" 2>/dev/null | grep -q "^ii"; then
            echo "$dep"
        fi
    done
}

# Get dependency packages for RPM-based systems
get_rpm_dependencies() {
    local pkg="$1"

    # Get requires using rpm
    rpm -qR "$pkg" 2>/dev/null | \
    grep -v 'rpmlib\|/bin/sh\|/usr/bin\|libc\|ld-linux' | \
    sed 's/(.*//' | \
    sed 's/>=.*//' | \
    sed 's/<=.*//' | \
    sed 's/=.*//' | \
    sed 's/^[[:space:]]*//' | \
    sed 's/[[:space:]]*$//' | \
    while read -r dep; do
        if rpm -q "$dep" >/dev/null 2>&1; then
            echo "$dep"
        fi
    done
}

# Get all dependency packages
get_dependency_packages() {
    local pkg="$1"

    if command -v dpkg >/dev/null 2>&1; then
        get_debian_dependencies "$pkg"
    elif command -v rpm >/dev/null 2>&1; then
        get_rpm_dependencies "$pkg"
    fi
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

        # Add to results if it's a real file or symlink
        if [ -f "$current_file" ] || [ -L "$current_file" ]; then
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

    # Symlink
    if [ -L "$file" ]; then
        echo "$file [Symlink]"
        return
    fi

    if command -v dpkg >/dev/null 2>&1; then
        # Find package that owns the file
        local owner_pkg=$(dpkg -S "$real_file" 2>/dev/null | cut -d: -f1)
        if [ -n "$owner_pkg" ]; then
            if dpkg -V "$owner_pkg" 2>/dev/null | grep -q "^..5.* $real_file"; then
                echo "$file [Modified]"
            else
                echo "$file [Unmodified]"
            fi
        else
            if [ "$file" != "$real_file" ] && dpkg -S "$file" >/dev/null 2>&1; then
                echo "$file [Unmodified]"
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
                echo "$file [Unmodified]"
            fi
        else
            if [ "$file" != "$real_file" ] && rpm -qf "$file" >/dev/null 2>&1; then
                echo "$file [Unmodified]"
            else
                echo "$file [Custom]"
            fi
        fi
    else
        echo "$file [Custom]"
    fi
}

# Find files in directories where config files are located
find_files_in_config_locations() {
    local pkg="$1"

    # Get actual config files from package first
    if command -v dpkg >/dev/null 2>&1; then
        config_files=$(dpkg -L "$pkg" 2>/dev/null | grep -E '\.(conf|cfg|ini|yaml|yml)$|/config$')
    elif command -v rpm >/dev/null 2>&1; then
        config_files=$(rpm -ql "$pkg" 2>/dev/null | grep -E '\.(conf|cfg|ini|yaml|yml)$|/config$')
    fi

    # Extract unique directories from config files
    echo "$config_files" | while read -r config_file; do
        if [ -n "$config_file" ] && [ -f "$config_file" ]; then
            config_dir=$(dirname "$config_file")
            echo "$config_dir"
        fi
    done | sort -u | while read -r dir; do
        if [ -n "$dir" ] && [ -d "$dir" ]; then
            # Find all files in this specific config directory
            find "$dir" -type f -o -type l 2>/dev/null
        fi
    done

    # Add common config directories for well-known packages
    case "$pkg" in
        nginx*|apache*|httpd*)
            for dir in /etc/nginx /etc/apache2 /etc/httpd; do
                if [ -d "$dir" ]; then
                    find "$dir" -type f -o -type l 2>/dev/null
                fi
            done
            ;;
        mysql*|mariadb*)
            for dir in /etc/mysql /etc/my.cnf.d; do
                if [ -d "$dir" ]; then
                    find "$dir" -type f -o -type l 2>/dev/null
                fi
            done
            ;;
        postgresql*)
            for dir in /etc/postgresql; do
                if [ -d "$dir" ]; then
                    find "$dir" -type f -o -type l 2>/dev/null
                fi
            done
            ;;
        ssh*|openssh*)
            for dir in /etc/ssh; do
                if [ -d "$dir" ]; then
                    find "$dir" -type f -o -type l 2>/dev/null
                fi
            done
            ;;
    esac
}

# Process config files for a single package
process_package_configs() {
    local pkg="$1"

    {
        # Search using package manager (get actual config files)
        if command -v dpkg >/dev/null 2>&1; then
            dpkg -L "$pkg" 2>/dev/null
        elif command -v rpm >/dev/null 2>&1; then
            rpm -ql "$pkg" 2>/dev/null
        fi

        # Find additional files in directories where config files are located
        find_files_in_config_locations "$pkg"

    } | sort -u | while read -r file; do
        # Skip virtual filesystem paths
        if should_skip_path "$file"; then
            continue
        fi

        # Skip if not a valid file path (starts with /, exists)
        if [ "${file#/}" = "$file" ] || [ ! -f "$file" ] && [ ! -L "$file" ]; then
            continue
        fi

        # Skip documentation and copyright files
        case "$file" in
            */doc/*|*/man/*|*copyright*|*LICENSE*|*README*|*CHANGELOG*)
                continue
                ;;
        esac

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
}

# Cleanup function
cleanup() {
    rm -f "$tmp_result" "$tmp_processed" "$tmp_includes" "$tmp_dep_packages"
}

# Set trap for cleanup
trap cleanup EXIT

# Get dependency packages silently
get_dependency_packages "$pkg" > "$tmp_dep_packages"

# Process main package
process_package_configs "$pkg"

# Process dependency packages
if [ -s "$tmp_dep_packages" ]; then
    while read -r dep_pkg; do
        [ -z "$dep_pkg" ] && continue
        process_package_configs "$dep_pkg"
    done < "$tmp_dep_packages"
fi

# Process and output unique results with status
sort -u "$tmp_result" | while read -r file; do
    check_config_status "$file"
done
