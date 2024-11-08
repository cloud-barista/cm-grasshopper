package software

import (
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"github.com/jollaman999/utils/logger"
	"strings"
)

type ConfigFile struct {
	Path   string `json:"path"`
	Status string `json:"status,omitempty"` // Modified, Custom, or empty for default
}

func sudoWrapper(cmd string, password string) string {
	return fmt.Sprintf("echo '%s' | sudo -S sh -c '%s'", password, strings.Replace(cmd, "'", "'\"'\"'", -1))
}

func copyConfigFiles(sourceClient *ssh.Client, targetClient *ssh.Client, configs []ConfigFile) error {
	for _, config := range configs {
		if config.Status == "" {
			continue
		}

		err := copyFile(sourceClient, targetClient, config.Path)
		if err != nil {
			return fmt.Errorf("failed to copy file %s: %v", config.Path, err)
		}
	}
	return nil
}

func copyFile(sourceClient *ssh.Client, targetClient *ssh.Client, filePath string) error {
	session, err := sourceClient.NewSession()
	if err != nil {
		return err
	}
	defer func() {
		_ = session.Close()
	}()

	sourcePassword := sourceClient.SSHTarget.Password
	targetPassword := targetClient.SSHTarget.Password

	cmd := fmt.Sprintf(`
		# Permission|UID|GID|FileType
		stat -c "%%a|%%u|%%g|%%F" "%s"
		
		# Check symlinks
		if [ -L "%s" ]; then
			readlink -f "%s"
			realpath "%s"
			cat "%s"
		fi
	`, filePath, filePath, filePath, filePath, filePath)

	output, err := session.CombinedOutput(sudoWrapper(cmd, sourcePassword))
	if err != nil {
		return fmt.Errorf("failed to get file info: %s", string(output))
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	fileInfo := strings.Split(lines[0], "|")
	isSymlink := len(lines) > 1

	var fileContent string
	var symlinkTarget string

	if isSymlink {
		symlinkTarget = strings.TrimSpace(lines[1])
		fileContent = strings.Join(lines[3:], "\n")
	} else {
		session, err = sourceClient.NewSession()
		if err != nil {
			return err
		}
		defer func() {
			_ = session.Close()
		}()

		catCmd := fmt.Sprintf("cat '%s'", filePath)
		content, err := session.CombinedOutput(sudoWrapper(catCmd, sourcePassword))
		if err != nil {
			return fmt.Errorf("failed to read file content: %v", string(content))
		}
		fileContent = string(content)
	}

	var targetExists bool
	if isSymlink {
		session, err = targetClient.NewSession()
		if err != nil {
			return err
		}
		defer func() {
			_ = session.Close()
		}()

		checkCmd := fmt.Sprintf("test -f '%s'", symlinkTarget)
		_, err = session.CombinedOutput(sudoWrapper(checkCmd, targetPassword))
		targetExists = err == nil
	}

	session, err = targetClient.NewSession()
	if err != nil {
		return err
	}
	defer func() {
		_ = session.Close()
	}()

	var copyCmd string
	if isSymlink && targetExists {
		copyCmd = fmt.Sprintf(`
			cat > '%s' << 'EOL'
%s
EOL
			chown %s:%s '%s'
			chmod %s '%s'
			
			mkdir -p "$(dirname '%s')"
			ln -sf '%s' '%s'
		`, symlinkTarget, fileContent,
			fileInfo[1], fileInfo[2], symlinkTarget,
			fileInfo[0], symlinkTarget,
			filePath, symlinkTarget, filePath)
	} else {
		copyCmd = fmt.Sprintf(`
			mkdir -p "$(dirname '%s')"
			cat > '%s' << 'EOL'
%s
EOL
			chown %s:%s '%s'
			chmod %s '%s'
		`, filePath, filePath, fileContent,
			fileInfo[1], fileInfo[2], filePath,
			fileInfo[0], filePath)

		if isSymlink {
			logger.Println(logger.WARN, true, fmt.Sprintf("Symlink target '%s' not found on target system. Created regular file at '%s' instead.\n",
				symlinkTarget, filePath))
		}
	}

	if output, err := session.CombinedOutput(sudoWrapper(copyCmd, targetPassword)); err != nil {
		return fmt.Errorf("failed to copy file: %s", string(output))
	}

	return nil
}

func findConfigs(client *ssh.Client, packageName string) ([]string, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	cmd := `#!/bin/sh
pkg="%s"

check_includes() {
    local file="$1"
    # Find common include patterns
    {
        # Cases with quotes
        grep -i "include.*[\"'].*[\"']" "$file" 2>/dev/null | grep -o "[\"'][^\"']*[\"']" | tr -d "'\"" || true
        # Cases without quotes
        grep -i "^[[:space:]]*include[[:space:]]*[^\"']" "$file" 2>/dev/null | sed 's/.*include[[:space:]]*//g' | sed 's/#.*//g' || true
        # conf.d style directory pattern
        grep -i "conf.d" "$file" 2>/dev/null | grep -o '\/[^[:space:]"]*' || true
        # Directories ending with .d
        grep -i "\.d" "$file" 2>/dev/null | grep -o '\/[^[:space:]"]*' || true
    } | while read -r inc_path; do
        # Convert to absolute path if it's a relative path
        if [ "${inc_path#/}" = "$inc_path" ]; then
            inc_path="$(dirname "$file")/$inc_path"
        fi
        # Handle paths with wildcards
        if echo "$inc_path" | grep -q "[*]"; then
            eval find $inc_path -type f 2>/dev/null || echo "$inc_path"
        else
            echo "$inc_path"
        fi
    done
}

check_config_status() {
    local file="$1"
    # Get the actual file path (in case of symbolic links)
    local real_file=$(readlink -f "$file")
    
    if command -v dpkg >/dev/null 2>&1; then
        # Debian/Ubuntu
        if dpkg -S "$real_file" >/dev/null 2>&1; then
            if dpkg -V "$pkg" 2>/dev/null | grep -q "^..5.*$real_file$"; then
                echo "$file [Modified]"
            else
                echo "$file"
            fi
        else
            # Check the original file if it's a symbolic link
            if [ "$file" != "$real_file" ] && dpkg -S "$file" >/dev/null 2>&1; then
                echo "$file"
            else
                echo "$file [Custom]"
            fi
        fi
    elif command -v rpm >/dev/null 2>&1; then
        # RedHat/CentOS
        if rpm -qf "$real_file" >/dev/null 2>&1; then
            if rpm -V "$pkg" | grep -q "^..5.*$real_file$"; then
                echo "$file [Modified]"
            else
                echo "$file"
            fi
        else
            # Check the original file if it's a symbolic link
            if [ "$file" != "$real_file" ] && rpm -qf "$file" >/dev/null 2>&1; then
                echo "$file"
            else
                echo "$file [Custom]"
            fi
        fi
    else
        echo "$file"
    fi
}

# Create temp file for results
tmp_result=$(mktemp)

{
    # 1. Package manager search
    if command -v dpkg >/dev/null 2>&1; then
        dpkg -L "$pkg" 2>/dev/null | grep -E 'conf$|config|\.cfg$|\.ini$|\.yaml$|\.yml$' | grep -v '\-available'
    fi

    if command -v rpm >/dev/null 2>&1; then
        rpm -ql "$pkg" 2>/dev/null | grep -E 'conf$|config|\.cfg$|\.ini$|\.yaml$|\.yml$' | grep -v '\-available'
    fi

    # 2. Process config arguments
    ps aux | grep "$pkg" | grep -o '[[:space:]]/[^[:space:]]*conf[^[:space:]]*'

    # 3. Common config locations
    for basepath in "/etc/$pkg" "/etc/$pkg.conf" "/etc/$pkg.d" "/usr/local/etc/$pkg" "/opt/$pkg/etc" "/var/lib/$pkg" "/usr/share/$pkg"; do
        if [ -f "$basepath" ]; then
            echo "$basepath"
        elif [ -d "$basepath" ]; then
            find "$basepath" -type f \( -name "*.conf" -o -name "*.cfg" -o -name "*.ini" -o -name "*.yaml" -o -name "*.yml" \) \
            ! -path "*/available/*" \
            ! -path "*/*-available/*" 2>/dev/null
        fi
    done

    # 4. Systemd config references
    if [ -d "/etc/systemd/system" ]; then
        grep -r "^ExecStart.*$pkg\|^ExecStartPre.*$pkg\|^EnvironmentFile.*$pkg" /etc/systemd/system/ 2>/dev/null | \
        grep -o '/[^[:space:]]*\.conf\|/[^[:space:]]*\.cfg\|/[^[:space:]]*\.ini\|/[^[:space:]]*\.yaml\|/[^[:space:]]*\.yml'
        find /etc/systemd/system -type f -exec grep -l "conf=$\|config=" {} \; 2>/dev/null
    fi

    # 5. Init.d scripts
    if [ -f "/etc/init.d/$pkg" ]; then
        grep -o '/[^[:space:]]*\.conf\|/[^[:space:]]*\.cfg\|/[^[:space:]]*\.ini\|/[^[:space:]]*\.yaml\|/[^[:space:]]*\.yml' "/etc/init.d/$pkg" 2>/dev/null
    fi
} | sort -u | while read -r file; do
    if [ -f "$file" ]; then
        echo "$file" >> "$tmp_result"
        # Find included files within each config file
        check_includes "$file" | while read -r inc_file; do
            if [ -f "$inc_file" ]; then
                echo "$inc_file" >> "$tmp_result"
                # Recursively check for includes
                check_includes "$inc_file" | while read -r sub_inc; do
                    if [ -f "$sub_inc" ]; then
                        echo "$sub_inc" >> "$tmp_result"
                    fi
                done
            fi
        done
    fi
done

# Output results (remove duplicates)
sort -u "$tmp_result" | while read -r file; do
    check_config_status "$file"
done

rm -f "$tmp_result"`

	password := client.SSHTarget.Password
	output, err := session.CombinedOutput(sudoWrapper(fmt.Sprintf(cmd, packageName), password))
	if err != nil {
		if len(output) == 0 {
			return nil, fmt.Errorf("failed to run command: %s", string(output))
		}
	}

	var results []string
	seen := make(map[string]bool)

	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" && !seen[line] {
			seen[line] = true
			results = append(results, line)
		}
	}

	return results, nil
}

func configCopier(sourceClient *ssh.Client, targetClient *ssh.Client, packageName string) error {
	configs, err := findConfigs(sourceClient, packageName)
	if err != nil {
		return fmt.Errorf("failed to find configs: %v", err)
	}

	var configFiles []ConfigFile
	for _, config := range configs {
		configFiles = append(configFiles, parseConfigLine(config))
	}

	if err := copyConfigFiles(sourceClient, targetClient, configFiles); err != nil {
		return fmt.Errorf("failed to copy config files: %v", err)
	}

	return nil
}

func parseConfigLine(line string) ConfigFile {
	parts := strings.SplitN(line, " [", 2)
	config := ConfigFile{
		Path: strings.TrimSpace(parts[0]),
	}

	if len(parts) > 1 {
		config.Status = strings.TrimSuffix(parts[1], "]")
	}

	return config
}
