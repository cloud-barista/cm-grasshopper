#!/bin/sh

# Package name from argument
pkg="$1"

# Check if package is likely a library package
is_library_package() {
     case "$pkg" in
         lib*[0-9]|*-dev|*-devel|*-headers|*-doc|*-man|*-common|*-locale|*-dbg|*-data)
             return 0
             ;;
         *)
             return 1
             ;;
     esac
 }

# Find included config files and referenced directories
check_includes() {
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
        # Convert relative paths to absolute
        if [ "${inc_path#/}" = "$inc_path" ]; then
            inc_path="$(dirname "$file")/$inc_path"
        fi
        # Handle wildcard patterns
        if echo "$inc_path" | grep -q "[*]"; then
            eval find $inc_path -type f 2>/dev/null || echo "$inc_path"
        else
            echo "$inc_path"
        fi
    done
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

# Early exit for library packages
if is_library_package; then
    exit 0
fi

# Temporary file for collecting results
tmp_result=$(mktemp)

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
    # Filter config files by common patterns and backup files
    if echo "$file" | grep -i -E '\.(conf|cfg|ini|yaml|yml|bak)$|conf\.d|\.d/|config$|[._]bak$' >/dev/null 2>&1; then
        if [ -f "$file" ]; then
            echo "$file" >> "$tmp_result"
            check_includes "$file" | while read -r inc_file; do
                if [ -f "$inc_file" ]; then
                    echo "$inc_file" >> "$tmp_result"
                fi
            done
        fi
    # Check content of non-standard named files for config-like content
    elif [ -f "$file" ] && file "$file" | grep -i "text" >/dev/null 2>&1; then
        if grep -i -E '(server|listen|port|host|config|ssl|virtual|root|location)' "$file" >/dev/null 2>&1; then
            echo "$file" >> "$tmp_result"
            check_includes "$file" | while read -r inc_file; do
                if [ -f "$inc_file" ]; then
                    echo "$inc_file" >> "$tmp_result"
                fi
            done
        fi
    fi
done

# Process and output unique results with status
sort -u "$tmp_result" | while read -r file; do
    check_config_status "$file"
done

rm -f "$tmp_result"
