package software

import (
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"path/filepath"
	"regexp"
	"strings"
)

type ConfigFile struct {
	Path   string `json:"path"`
	Status string `json:"status,omitempty"` // Modified, Custom, or empty for default
}

func sudoWrapper(cmd string, password string) string {
	return fmt.Sprintf("echo '%s' | sudo -S sh -c '%s'", password, strings.Replace(cmd, "'", "'\"'\"'", -1))
}

func findCertKeyPaths(client *ssh.Client, filePath string) ([]string, error) {
	migrationLogger.Printf(INFO, "Finding certificate and key paths in file: %s\n", filePath)

	cmd := fmt.Sprintf(`
        grep -i -E '(ssl|tls)_(certificate|key|trusted_certificate|client_certificate|dhparam)|key_file|cert_file|ca_file|private_key|public_key|certificate|keyfile|certfile|cafile' '%s' 2>/dev/null || true
    `, filePath)

	session, err := client.NewSession()
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to create SSH session: %v\n", err)
		return nil, fmt.Errorf("ssh session creation failed: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	migrationLogger.Printf(DEBUG, "Executing grep command for cert/key paths\n")
	output, err := session.CombinedOutput(sudoWrapper(cmd, client.SSHTarget.Password))
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to execute grep command: %v\n", err)
		return nil, fmt.Errorf("command execution failed: %v", err)
	}

	paths := make([]string, 0)
	seen := make(map[string]bool)

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?mi)(?:ssl_certificate|tls_certificate|cert_file|certfile|certificate|cert)\s*(?:=|:|\s)\s*["']?(/[^"'\s;]+|[^"'\s;]+)["']?`),
		regexp.MustCompile(`(?mi)(?:ssl_certificate_key|tls_key|key_file|keyfile|private_key|key)\s*(?:=|:|\s)\s*["']?(/[^"'\s;]+|[^"'\s;]+)["']?`),
		regexp.MustCompile(`(?mi)(?:ssl_trusted_certificate|ssl_client_certificate|ca_file|cafile)\s*(?:=|:|\s)\s*["']?(/[^"'\s;]+|[^"'\s;]+)["']?`),
		regexp.MustCompile(`(?mi)(?:ssl_dhparam|dhparam)\s*(?:=|:|\s)\s*["']?(/[^"'\s;]+|[^"'\s;]+)["']?`),
	}

	migrationLogger.Printf(DEBUG, "Processing grep output for pattern matches\n")
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		for _, pattern := range patterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) > 1 {
				path := matches[1]

				if !strings.HasPrefix(path, "/") {
					baseDir := filepath.Dir(filePath)
					path = filepath.Join(baseDir, path)
					migrationLogger.Printf(DEBUG, "Converted relative path to absolute: %s\n", path)
				}

				if !seen[path] {
					seen[path] = true
					paths = append(paths, path)
					migrationLogger.Printf(DEBUG, "Found unique cert/key path: %s\n", path)
				}
			}
		}
	}

	migrationLogger.Printf(INFO, "Found %d potential cert/key paths\n", len(paths))

	var validPaths []string
	for _, path := range paths {
		migrationLogger.Printf(DEBUG, "Verifying path existence: %s\n", path)

		checkCmd := fmt.Sprintf("test -f '%s'", path)
		session, err := client.NewSession()
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to create SSH session for path verification: %v\n", err)
			return nil, fmt.Errorf("ssh session creation failed: %v", err)
		}

		_, err = session.CombinedOutput(sudoWrapper(checkCmd, client.SSHTarget.Password))
		if err == nil {
			validPaths = append(validPaths, path)
			migrationLogger.Printf(INFO, "Verified valid cert/key path: %s\n", path)
		} else {
			migrationLogger.Printf(WARN, "Path does not exist or is not accessible: %s\n", path)
		}
		_ = session.Close()
	}

	migrationLogger.Printf(INFO, "Found %d valid cert/key paths\n", len(validPaths))
	return validPaths, nil
}

func copyFile(sourceClient *ssh.Client, targetClient *ssh.Client, filePath string) error {
	migrationLogger.Printf(INFO, "Starting file copy process for: %s\n", filePath)

	sourcePassword := sourceClient.SSHTarget.Password
	targetPassword := targetClient.SSHTarget.Password

	cmd := fmt.Sprintf(`
               # Permission|UID|GID|FileType
               stat -c "%%a|%%u|%%g|%%F" "%s"
               
               # Check symlinks
               if [ -L "%s" ]; then
                 readlink -f "%s"
               fi
       `, filePath, filePath, filePath)

	migrationLogger.Printf(DEBUG, "Getting file statistics\n")

	sourceSession, err := sourceClient.NewSession()
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to create source SSH session: %v\n", err)
		return fmt.Errorf("source ssh session creation failed: %v", err)
	}
	defer func() {
		_ = sourceSession.Close()
	}()

	output, err := sourceSession.CombinedOutput(sudoWrapper(cmd, sourcePassword))
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to get file stats: %v\n", err)
		return fmt.Errorf("failed to get file stats: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		migrationLogger.Printf(ERROR, "Failed to get file stats: no output lines")
		return fmt.Errorf("failed to get file stats: no output lines")
	}
	fileInfo := strings.Split(lines[0], "|")
	if len(fileInfo) != 4 {
		migrationLogger.Printf(ERROR, "Failed to get file stats: wrong file stats: %s", lines[0])
		return fmt.Errorf("failed to get file stats: wrong file stats: %s", lines[0])
	}
	migrationLogger.Printf(DEBUG, "File stats - Permissions: %s, UID: %s, GID: %s, Type: %s\n",
		fileInfo[0], fileInfo[1], fileInfo[2], fileInfo[3])

	isSymlink := len(lines) > 1
	var symlinkTarget string
	var fileContent string

	if isSymlink {
		symlinkTarget = strings.TrimSpace(lines[1])

		migrationLogger.Printf(INFO, "File is a symbolic link\n")

		migrationLogger.Printf(DEBUG, "Symlink target: %s\n", symlinkTarget)
		contentCmd := fmt.Sprintf("cat '%s'", symlinkTarget)
		session, err := sourceClient.NewSession()
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to create SSH session for reading symlink target: %v\n", err)
			return fmt.Errorf("ssh session creation failed: %v", err)
		}

		content, err := session.CombinedOutput(sudoWrapper(contentCmd, sourcePassword))
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to read symlink target content: %v\n", err)
			return fmt.Errorf("failed to read symlink target: %v", err)
		}
		defer func() {
			_ = session.Close()
		}()

		fileContent = string(content)
	} else {
		migrationLogger.Printf(DEBUG, "Reading regular file content\n")

		contentCmd := fmt.Sprintf("cat '%s'", filePath)
		session, err := sourceClient.NewSession()
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to create SSH session for reading file: %v\n", err)
			return fmt.Errorf("ssh session creation failed: %v", err)
		}

		content, err := session.CombinedOutput(sudoWrapper(contentCmd, sourcePassword))
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to read file content: %v\n", err)
			return fmt.Errorf("failed to read file content: %v", err)
		}
		defer func() {
			_ = session.Close()
		}()

		fileContent = string(content)
	}

	migrationLogger.Printf(INFO, "Copying file to target system\n")

	var targetExists bool
	if isSymlink {
		checkCmd := fmt.Sprintf("test -f '%s'", symlinkTarget)

		targetSession, err := targetClient.NewSession()
		if err != nil {
			return err
		}
		defer func() {
			_ = targetSession.Close()
		}()
		_, err = targetSession.CombinedOutput(sudoWrapper(checkCmd, targetPassword))
		targetExists = err == nil
	}

	var copyCmd string
	if isSymlink && targetExists {
		migrationLogger.Printf(DEBUG, "Creating symlink on target system\n")

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
			migrationLogger.Println(WARN, true, fmt.Sprintf("Symlink target '%s' not found on target system. Created regular file at '%s' instead.",
				symlinkTarget, filePath))
		}
	}

	targetSession, err := targetClient.NewSession()
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to create target SSH session: %v\n", err)
		return fmt.Errorf("target ssh session creation failed: %v", err)
	}
	defer func() {
		_ = targetSession.Close()
	}()

	if output, err := targetSession.CombinedOutput(sudoWrapper(copyCmd, targetPassword)); err != nil {
		migrationLogger.Printf(ERROR, "Failed to execute copy command: %v\n", err)
		return fmt.Errorf("failed to copy file: %s", string(output))
	}

	migrationLogger.Printf(INFO, "Successfully copied file: %s\n", filePath)
	return nil
}

func findConfigs(client *ssh.Client, packageName string) ([]ConfigFile, error) {
	migrationLogger.Printf(INFO, "Starting config search for package: %s\n", packageName)

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

	migrationLogger.Printf(DEBUG, "Executing config search command\n")
	session, err := client.NewSession()
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to create SSH session: %v\n", err)
		return nil, fmt.Errorf("ssh session creation failed: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	output, err := session.CombinedOutput(sudoWrapper(fmt.Sprintf(cmd, packageName), client.SSHTarget.Password))
	if err != nil && len(output) == 0 {
		migrationLogger.Printf(ERROR, "Failed to execute config search: %v\n", err)
		return nil, fmt.Errorf("config search failed: %v", err)
	}

	var configs []ConfigFile
	seen := make(map[string]bool)

	migrationLogger.Printf(DEBUG, "Processing found config files\n")
	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			conf := parseConfigLine(line)
			if !seen[conf.Path] {
				seen[conf.Path] = true
				configs = append(configs, conf)
				migrationLogger.Printf(DEBUG, "Found config file: %s [%s]\n", conf.Path, conf.Status)
			}
		}
	}

	migrationLogger.Printf(INFO, "Found %d unique config files\n", len(configs))
	return configs, nil
}

func parseConfigLine(line string) ConfigFile {
	migrationLogger.Printf(DEBUG, "Parsing config line: %s\n", line)

	parts := strings.SplitN(line, " [", 2)
	conf := ConfigFile{
		Path: strings.TrimSpace(parts[0]),
	}

	if len(parts) > 1 {
		conf.Status = strings.TrimSuffix(parts[1], "]")
	}

	migrationLogger.Printf(DEBUG, "Parsed config - Path: %s, Status: %s\n", conf.Path, conf.Status)
	return conf
}
func copyConfigFiles(sourceClient *ssh.Client, targetClient *ssh.Client, configs []ConfigFile) error {
	migrationLogger.Printf(INFO, "Starting config files copy process for %d files\n", len(configs))

	for i, conf := range configs {
		migrationLogger.Printf(INFO, "Processing config file %d/%d: %s\n", i+1, len(configs), conf.Path)

		if conf.Status == "" {
			migrationLogger.Printf(DEBUG, "Skipping unmodified config: %s\n", conf.Path)
			continue
		}

		migrationLogger.Printf(INFO, "Copying config file: %s [Status: %s]\n", conf.Path, conf.Status)
		err := copyFile(sourceClient, targetClient, conf.Path)
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to copy config file %s: %v\n", conf.Path, err)
			return fmt.Errorf("failed to copy file %s: %v", conf.Path, err)
		}

		migrationLogger.Printf(DEBUG, "Searching for associated cert/key files for: %s\n", conf.Path)
		certKeyPaths, err := findCertKeyPaths(sourceClient, conf.Path)
		if err != nil {
			migrationLogger.Printf(WARN, "Error finding cert/key paths for %s: %v\n", conf.Path, err)
			continue
		}

		if len(certKeyPaths) > 0 {
			migrationLogger.Printf(INFO, "Found %d cert/key files for %s\n", len(certKeyPaths), conf.Path)
			for j, path := range certKeyPaths {
				migrationLogger.Printf(INFO, "Copying cert/key file %d/%d: %s\n", j+1, len(certKeyPaths), path)
				err := copyFile(sourceClient, targetClient, path)
				if err != nil {
					migrationLogger.Printf(WARN, "Failed to copy cert/key file %s: %v\n", path, err)
					continue
				}
				migrationLogger.Printf(INFO, "Successfully copied cert/key file: %s\n", path)
			}
		} else {
			migrationLogger.Printf(DEBUG, "No cert/key files found for: %s\n", conf.Path)
		}

		migrationLogger.Printf(INFO, "Successfully processed config file: %s\n", conf.Path)
	}

	migrationLogger.Printf(INFO, "Completed config files copy process\n")
	return nil
}

func configCopier(sourceClient *ssh.Client, targetClient *ssh.Client, packageName, uuid string) error {
	if err := initLoggerWithUUID(uuid); err != nil {
		return fmt.Errorf("failed to initialize logger: %v", err)
	}
	defer migrationLogger.Close()

	migrationLogger.Printf(INFO, "Starting config copier for package: %s (UUID: %s)\n", packageName, uuid)

	if sourceClient == nil || targetClient == nil {
		migrationLogger.Printf(ERROR, "Invalid client connections provided\n")
		return fmt.Errorf("source or target client is nil")
	}

	if packageName == "" {
		migrationLogger.Printf(ERROR, "Empty package name provided\n")
		return fmt.Errorf("package name cannot be empty")
	}

	migrationLogger.Printf(INFO, "Finding configuration files for package: %s\n", packageName)
	configs, err := findConfigs(sourceClient, packageName)
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to find configs for package %s: %v\n", packageName, err)
		return fmt.Errorf("failed to find configs: %v", err)
	}

	if len(configs) == 0 {
		migrationLogger.Printf(WARN, "No configuration files found for package: %s\n", packageName)
		return nil
	}

	migrationLogger.Printf(INFO, "Found %d configuration files for package %s\n", len(configs), packageName)

	if err := copyConfigFiles(sourceClient, targetClient, configs); err != nil {
		migrationLogger.Printf(ERROR, "Failed to copy config files for package %s: %v\n", packageName, err)
		return fmt.Errorf("failed to copy config files: %v", err)
	}

	migrationLogger.Printf(INFO, "Successfully completed config copy process for package: %s\n", packageName)
	return nil
}
