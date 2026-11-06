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
tmp_base=$(mktemp)
tmp_extra=$(mktemp)

cleanup() {
    rm -f "$tmp_all_files" "$tmp_deps" "$tmp_file_owners" "$tmp_verification" "$tmp_processed" "$tmp_base" "$tmp_extra"
}
trap cleanup EXIT

should_skip_path() {
    case "$1" in
        /proc/*|/dev/*|/sys/*|/run/*|*/doc/*|*/man/*|*copyright*|*LICENSE*|*README*|*CHANGELOG*)
            return 0 ;;
        /etc/passwd|/etc/group|/etc/shadow|/etc/shadow-|/etc/gshadow|/etc/gshadow-|/etc/sudoers|/etc/sudoers.d/*)
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

    {
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
    } > "$tmp_base"
    cat "$tmp_base"

    # Only paths that are not candidates already, so existing candidates keep their status.
    {
        if [ "$PKG_MANAGER" = "dpkg" ]; then
            while read -r p; do
                [ -n "$p" ] && get_ucf_configs "$p"
            done < "$tmp_deps"
        fi

        get_well_known_configs "$pkg"
    } | sort -u | grep -Fxv -f "$tmp_base" | tee "$tmp_extra"
}

# Config files that a package uses but does not list in its file list, keyed by
# package name. One line per package: "<package> <path or glob>...".
WELL_KNOWN_CONFIGS='
nfs-kernel-server /etc/exports /etc/exports.d/*
nfs-utils /etc/exports /etc/exports.d/*
'

get_well_known_configs() {
    echo "$WELL_KNOWN_CONFIGS" | awk -v p="$1" '$1 == p { for (i = 2; i <= NF; i++) print $i }' | \
    while read -r pattern; do
        for f in $pattern; do
            if [ -f "$f" ] || [ -L "$f" ]; then
                echo "$f"
            fi
        done
    done
}

# Debian packages may create files under /etc from a template with ucf instead of
# shipping them as dpkg conffiles (e.g. /etc/exports of nfs-kernel-server).
get_ucf_configs() {
    p="$1"
    {
        if command -v ucfq >/dev/null 2>&1; then
            ucfq -w "$p" 2>/dev/null | cut -d: -f1
        elif [ -r /var/lib/ucf/registry ]; then
            awk -v p="$p" '$1 == p { print $2 }' /var/lib/ucf/registry
        fi

        # Not every postinst registers its files with ucfr, so also read the
        # destination of "ucf [options] <source> <destination>" calls.
        for script in "/var/lib/dpkg/info/$p.postinst" /var/lib/dpkg/info/"$p":*.postinst; do
            [ -r "$script" ] || continue
            grep -E '(^|[[:space:];&|])ucf[[:space:]]' "$script" 2>/dev/null | \
            awk '{
                n = 0
                for (i = 1; i <= NF; i++) {
                    f = $i
                    gsub(/["'"'"']/, "", f)
                    if (f ~ /^\//) last[++n] = f
                }
                if (n >= 2) print last[n]
            }'
        done
    } | while read -r f; do
        case "$f" in
            /etc/*) [ -f "$f" ] && echo "$f" ;;
        esac
    done
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
        elif grep -Fxq "$file" "$tmp_extra" 2>/dev/null; then
            # Not judged by package verification: ucf files are not dpkg conffiles,
            # and RHEL "setup" excludes /etc/exports from rpm -V content checks.
            echo "$file [Custom]"
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
