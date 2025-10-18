#!/bin/bash

pkg="$1"

if [ -z "$pkg" ]; then
    echo "Usage: $0 <package_name>" >&2
    exit 1
fi

if command -v dpkg >/dev/null 2>&1; then
    if ! dpkg -l "$pkg" 2>/dev/null | grep -q "^ii"; then
        echo "Package '$pkg' is not installed" >&2
        exit 1
    fi
    PKG_MANAGER="dpkg"
elif command -v rpm >/dev/null 2>&1; then
    if ! rpm -q "$pkg" >/dev/null 2>&1; then
        echo "Package '$pkg' is not installed" >&2
        exit 1
    fi
    PKG_MANAGER="rpm"
else
    echo "Neither dpkg nor rpm found. Unsupported system." >&2
    exit 1
fi

tmp_all_files=$(mktemp)
tmp_deps=$(mktemp)
tmp_file_owners=$(mktemp)
tmp_verification=$(mktemp)
tmp_processed=$(mktemp)

cleanup() {
    rm -f "$tmp_all_files" "$tmp_deps" "$tmp_file_owners" "$tmp_verification" "$tmp_processed"
}
trap cleanup EXIT

should_skip_path() {
    case "$1" in
        /proc/*|/dev/*|/sys/*|/run/*|*/doc/*|*/man/*|*copyright*|*LICENSE*|*README*|*CHANGELOG*)
            return 0 ;;
        *) return 1 ;;
    esac
}

get_all_dependencies() {
    if [ "$PKG_MANAGER" = "dpkg" ]; then
        dpkg-query -Wf='${Depends}\n${Pre-Depends}\n' "$pkg" 2>/dev/null | \
        tr ',' '\n' | sed 's/|.*//' | sed 's/(.*//' | \
        sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//' | \
        grep -v '^$' | sort -u | \
        xargs -I {} sh -c 'dpkg -l "{}" 2>/dev/null | grep -q "^ii" && echo "{}"' 2>/dev/null
    else
        rpm -qR "$pkg" 2>/dev/null | \
        grep -v 'rpmlib\|/bin/sh\|/usr/bin\|libc\|ld-linux' | \
        sed 's/(.*//' | sed 's/[><=].*//' | \
        sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//' | \
        sort -u | \
        xargs -I {} sh -c 'rpm -q "{}" >/dev/null 2>&1 && echo "{}"' 2>/dev/null
    fi
}

collect_all_files() {
    {
        echo "$pkg"
        get_all_dependencies
    } > "$tmp_deps"

    if [ "$PKG_MANAGER" = "dpkg" ]; then
        xargs -I {} dpkg -L {} 2>/dev/null < "$tmp_deps" | \
        grep -E '\.(conf|cfg|ini|yaml|yml|bak)$|/etc/|conf\.d|\.d/|config$|[._]bak$' | \
        sort -u
    else
        xargs -I {} rpm -ql {} 2>/dev/null < "$tmp_deps" | \
        grep -E '\.(conf|cfg|ini|yaml|yml|bak)$|/etc/|conf\.d|\.d/|config$|[._]bak$' | \
        sort -u
    fi

    if [ -d "/etc/$pkg" ]; then
        find "/etc/$pkg" -type f -o -type l 2>/dev/null
    fi
}

build_file_owner_cache() {
    if [ "$PKG_MANAGER" = "dpkg" ]; then
        xargs -I {} dpkg -S {} 2>/dev/null < "$tmp_all_files" | \
        awk -F': ' '{print $2 "\t" $1}' > "$tmp_file_owners"

        xargs -I {} sh -c 'pkg=$(dpkg -S "{}" 2>/dev/null | cut -d: -f1); [ -n "$pkg" ] && dpkg -V "$pkg" 2>/dev/null | grep "^..5.* {}" && echo "{}"' < "$tmp_all_files" > "$tmp_verification"
    else
        xargs -I {} rpm -qf {} 2>/dev/null < "$tmp_all_files" | \
        paste "$tmp_all_files" - > "$tmp_file_owners"

        xargs -I {} sh -c 'pkg=$(rpm -qf "{}" 2>/dev/null); [ -n "$pkg" ] && rpm -V "$pkg" 2>/dev/null | grep "^..5.* {}" && echo "{}"' < "$tmp_all_files" > "$tmp_verification"
    fi
}

extract_includes_parallel() {
    xargs -P 4 -I {} sh -c '
        file="{}"
        [ ! -f "$file" ] && exit 0
        [ ! -r "$file" ] && exit 0

        grep -E "^[[:space:]]*include" "$file" 2>/dev/null | \
        sed -E "s/.*include[[:space:]]+//; s/[\"'"'"';].*//; s/^[\"'"'"']//; s/[\"'"'"']$//" | \
        while read -r inc_path; do
            [ -z "$inc_path" ] && continue

            if [ "${inc_path#/}" = "$inc_path" ]; then
                inc_path="$(dirname "$file")/$inc_path"
            fi

            case "$inc_path" in
                /proc/*|/dev/*|/sys/*|/run/*|*/doc/*|*/man/*|*copyright*|*LICENSE*|*README*|*CHANGELOG*)
                    continue ;;
            esac

            if echo "$inc_path" | grep -q "[*]"; then
                dirname_path=$(dirname "$inc_path" 2>/dev/null)
                [ -d "$dirname_path" ] && find "$dirname_path" -maxdepth 1 -name "$(basename "$inc_path")" -type f 2>/dev/null
            else
                [ -f "$inc_path" ] && echo "$inc_path"
            fi
        done
    ' < "$tmp_all_files"
}

process_includes_iterative() {
    sort -u "$tmp_all_files" > "$tmp_processed"

    for iteration in 1 2; do
        new_files=$(mktemp)
        extract_includes_parallel | sort -u > "${new_files}.includes"

        sort -u "$tmp_processed" > "${tmp_processed}.sorted"
        comm -23 "${new_files}.includes" "${tmp_processed}.sorted" > "$new_files"

        if [ ! -s "$new_files" ]; then
            rm -f "$new_files" "${new_files}.includes" "${tmp_processed}.sorted"
            break
        fi

        cat "$new_files" >> "$tmp_processed"
        cat "$new_files" >> "$tmp_all_files"
        rm -f "$new_files" "${new_files}.includes" "${tmp_processed}.sorted"
    done
}

check_file_status() {
    sort -u "$tmp_all_files" | while read -r file; do
        [ -z "$file" ] && continue

        if should_skip_path "$file"; then
            continue
        fi

        if [ ! -f "$file" ] && [ ! -L "$file" ]; then
            continue
        fi

        if [ -L "$file" ]; then
            echo "$file [Symlink]"
        elif grep -Fq "$file" "$tmp_verification" 2>/dev/null; then
            echo "$file [Modified]"
        elif grep -F "$file" "$tmp_file_owners" >/dev/null 2>&1; then
            echo "$file [Unmodified]"
        else
            echo "$file [Custom]"
        fi
    done
}

collect_all_files > "$tmp_all_files"

build_file_owner_cache &
CACHE_PID=$!

process_includes_iterative

wait $CACHE_PID

check_file_status
