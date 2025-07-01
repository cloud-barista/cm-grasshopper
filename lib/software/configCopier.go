package software

import (
	"encoding/base64"
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type ConfigFile struct {
	Path   string `json:"path"`
	Status string `json:"status,omitempty"` // Modified, Custom, or empty for default
}

var webServerDataDirs = map[string][]string{
	"nginx": {
		"/var/www",
		"/usr/share/nginx",
		"/var/log/nginx",
		"/var/cache/nginx",
	},
	"apache2": {
		"/var/www",
		"/usr/share/apache2",
		"/var/log/apache2",
		"/var/cache/apache2",
	},
	"httpd": {
		"/var/www",
		"/usr/share/httpd",
		"/var/log/httpd",
		"/var/cache/httpd",
	},
}

func isWebServer(packageName string) bool {
	_, exists := webServerDataDirs[packageName]
	return exists
}

func checkDirectorySize(client *ssh.Client, dirPath string) (int64, error) {
	session, err := client.NewSession()
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = session.Close()
	}()

	cmd := fmt.Sprintf("du -sb '%s' 2>/dev/null | cut -f1 || echo 0", dirPath)
	output, err := session.CombinedOutput(sudoWrapper(cmd, client.SSHTarget.Password))
	if err != nil {
		return 0, err
	}

	sizeStr := strings.TrimSpace(string(output))
	if sizeStr == "" || sizeStr == "0" {
		return 0, nil
	}

	return strconv.ParseInt(sizeStr, 10, 64)
}

func formatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	} else if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.2f MB", float64(bytes)/(1024*1024))
	}

	return fmt.Sprintf("%.2f GB", float64(bytes)/(1024*1024*1024))
}

func sudoWrapper(cmd string, password string) string {
	return fmt.Sprintf("echo '%s' | sudo -S sh -c '%s'", password, strings.ReplaceAll(cmd, "'", "'\"'\"'"))
}

func getScript(scriptName string) (string, error) {
	content, err := scriptsFS.ReadFile(fmt.Sprintf("scripts/%s", scriptName))
	if err != nil {
		return "", fmt.Errorf("failed to read script %s: %v", scriptName, err)
	}
	return string(content), nil
}

func findCertKeyPaths(client *ssh.Client, filePath string, migrationLogger *Logger) ([]string, error) {
	migrationLogger.Printf(INFO, "Finding certificate and key paths in file: %s\n", filePath)

	output, err := executeScript(client, migrationLogger, "cert_key_finder.sh", filePath)
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to execute cert key finder: %v\n", err)
		return nil, fmt.Errorf("cert key finder failed: %v", err)
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

func copyFile(sourceClient *ssh.Client, targetClient *ssh.Client, filePath string, migrationLogger *Logger) error {
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

func parseConfigLine(line string, migrationLogger *Logger) ConfigFile {
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

func findConfigs(client *ssh.Client, packageName string, migrationLogger *Logger) ([]ConfigFile, error) {
	migrationLogger.Printf(INFO, "Starting config search for package: %s\n", packageName)

	output, err := executeScript(client, migrationLogger, "config_finder.sh", packageName)
	if err != nil && len(output) == 0 {
		migrationLogger.Printf(ERROR, "Failed to execute config search: %v\n", err)
		return nil, fmt.Errorf("config search failed: %v", err)
	}

	var configs []ConfigFile
	seen := make(map[string]bool)

	migrationLogger.Printf(DEBUG, "Processing found config files\n")
	for _, line := range strings.Split(string(output), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			conf := parseConfigLine(line, migrationLogger)
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

type DirectoryRef struct {
	Path  string
	Perms string
	UID   string
	GID   string
}

func getDirInfo(client *ssh.Client, dirPath string, seen map[string]bool) DirectoryRef {
	seen[dirPath] = true

	statCmd := fmt.Sprintf("stat -c '%%a|%%u|%%g' '%s'", dirPath)
	session, err := client.NewSession()
	if err != nil {
		return DirectoryRef{Path: dirPath, Perms: "755", UID: "0", GID: "0"}
	}
	defer func() {
		_ = session.Close()
	}()

	statOutput, err := session.CombinedOutput(sudoWrapper(statCmd, client.SSHTarget.Password))
	if err != nil {
		return DirectoryRef{Path: dirPath, Perms: "755", UID: "0", GID: "0"}
	}

	parts := strings.Split(strings.TrimSpace(string(statOutput)), "|")
	if len(parts) == 3 {
		return DirectoryRef{
			Path:  dirPath,
			Perms: parts[0],
			UID:   parts[1],
			GID:   parts[2],
		}
	}

	return DirectoryRef{Path: dirPath, Perms: "755", UID: "0", GID: "0"}
}

func isNfsExportContent(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.HasPrefix(fields[0], "/") &&
			strings.Contains(line, "(") && strings.Contains(line, ")") {
			return true
		}
	}
	return false
}

func findReferencedDirectories(sourceClient *ssh.Client, filePath string) ([]DirectoryRef, error) {
	session, err := sourceClient.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ssh session creation failed: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	catCmd := fmt.Sprintf("cat '%s'", filePath)
	output, err := session.CombinedOutput(sudoWrapper(catCmd, sourceClient.SSHTarget.Password))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	content := string(output)
	var dirs []DirectoryRef
	seen := make(map[string]bool)

	if isNfsExportContent(content) {
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) >= 2 && strings.HasPrefix(fields[0], "/") {
				dirPath := fields[0]
				if !seen[dirPath] {
					dirs = append(dirs, getDirInfo(sourceClient, dirPath, seen))
				}
			}
		}
	} else {
		dirPattern := regexp.MustCompile(`(?i)(directory|path|root|DocumentRoot|datadir|log_path|socket)\s*[=:]\s*["']?(/[^"'\s]+)`)
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			matches := dirPattern.FindStringSubmatch(line)
			if len(matches) > 2 {
				dirPath := matches[2]
				if !seen[dirPath] {
					dirs = append(dirs, getDirInfo(sourceClient, dirPath, seen))
				}
			}
		}
	}

	return dirs, nil
}

func copyConfigFiles(sourceClient *ssh.Client, targetClient *ssh.Client, configs []ConfigFile, migrationLogger *Logger) error {
	for i, conf := range configs {
		migrationLogger.Printf(INFO, "Processing config file %d/%d: %s\n", i+1, len(configs), conf.Path)

		if strings.Contains(conf.Status, "Unmodified") {
			migrationLogger.Printf(DEBUG, "Skipping unmodified config: %s\n", conf.Path)
			continue
		}

		migrationLogger.Printf(INFO, "Finding referenced directories in %s\n", conf.Path)
		dirs, err := findReferencedDirectories(sourceClient, conf.Path)
		if err != nil {
			migrationLogger.Printf(WARN, "Error finding referenced directories in %s: %v\n", conf.Path, err)
		} else if len(dirs) > 0 {
			for _, dir := range dirs {
				cmd := fmt.Sprintf("mkdir -p '%s' && chown %s:%s '%s' && chmod %s '%s'",
					dir.Path, dir.UID, dir.GID, dir.Path, dir.Perms, dir.Path)

				migrationLogger.Printf(DEBUG, "Creating directory with permissions - Path: %s, UID: %s, GID: %s, Perms: %s\n",
					dir.Path, dir.UID, dir.GID, dir.Perms)

				session, err := targetClient.NewSession()
				if err != nil {
					continue
				}

				if output, err := session.CombinedOutput(sudoWrapper(cmd, targetClient.SSHTarget.Password)); err != nil {
					migrationLogger.Printf(WARN, "Failed to create directory %s: %s\n", dir.Path, string(output))
				} else {
					migrationLogger.Printf(INFO, "Successfully created directory  %s\n", dir.Path)
				}
				_ = session.Close()
			}
		}

		err = copyFile(sourceClient, targetClient, conf.Path, migrationLogger)
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to copy config file %s: %v\n", conf.Path, err)
			return fmt.Errorf("failed to copy file %s: %v", conf.Path, err)
		}

		migrationLogger.Printf(DEBUG, "Searching for associated cert/key files for: %s\n", conf.Path)
		certKeyPaths, err := findCertKeyPaths(sourceClient, conf.Path, migrationLogger)
		if err != nil {
			migrationLogger.Printf(WARN, "Error finding cert/key paths for %s: %v\n", conf.Path, err)
			continue
		}

		if len(certKeyPaths) > 0 {
			migrationLogger.Printf(INFO, "Found %d cert/key files for %s\n", len(certKeyPaths), conf.Path)
			for j, path := range certKeyPaths {
				migrationLogger.Printf(INFO, "Copying cert/key file %d/%d: %s\n", j+1, len(certKeyPaths), path)
				err := copyFile(sourceClient, targetClient, path, migrationLogger)
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

func transferChunk(sourceClient, targetClient *ssh.Client, sourceDir, targetDir, chunkName string, migrationLogger *Logger) error {
	readChunkCmd := fmt.Sprintf("cat '%s/%s'", sourceDir, chunkName)
	sourceSession, err := sourceClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create source session: %v", err)
	}
	defer func() {
		_ = sourceSession.Close()
	}()

	chunkData, err := sourceSession.CombinedOutput(sudoWrapper(readChunkCmd, sourceClient.SSHTarget.Password))
	if err != nil {
		return fmt.Errorf("failed to read chunk: %v", err)
	}

	migrationLogger.Printf(DEBUG, "Read %d bytes from source chunk %s\n", len(chunkData), chunkName)

	base64Data := base64.StdEncoding.EncodeToString(chunkData)

	const maxEchoLength = 100000 // 100KB
	targetPath := fmt.Sprintf("%s/%s", targetDir, chunkName)

	for i := 0; i < len(base64Data); i += maxEchoLength {
		end := i + maxEchoLength
		if end > len(base64Data) {
			end = len(base64Data)
		}
		part := base64Data[i:end]

		var writeCmd string
		if i == 0 {
			writeCmd = fmt.Sprintf("echo '%s' | base64 -d > '%s'", part, targetPath)
		} else {
			writeCmd = fmt.Sprintf("echo '%s' | base64 -d >> '%s'", part, targetPath)
		}

		targetSession, err := targetClient.NewSession()
		if err != nil {
			return fmt.Errorf("failed to create target session: %v", err)
		}

		if _, err := targetSession.CombinedOutput(sudoWrapper(writeCmd, targetClient.SSHTarget.Password)); err != nil {
			_ = targetSession.Close()
			return fmt.Errorf("failed to write chunk part: %v", err)
		}
		_ = targetSession.Close()
	}

	migrationLogger.Printf(DEBUG, "Wrote %d bytes to target chunk %s\n", len(chunkData), chunkName)
	return nil
}

func copyDirectoryWithChunks(sourceClient, targetClient *ssh.Client, dirPath string, uuid string, migrationLogger *Logger) error {
	chunkSize := "5M"
	tempDir := fmt.Sprintf("/tmp/grasshopper_%s", uuid)
	chunkPrefix := fmt.Sprintf("%s/chunk_", tempDir)

	migrationLogger.Printf(INFO, "Starting chunked copy of %s\n", dirPath)

	createChunksCmd := fmt.Sprintf(`
        mkdir -p '%s' &&
        cd '%s' &&
        tar --numeric-owner -czf - . | split -d -b %s - '%s'
    `, tempDir, dirPath, chunkSize, chunkPrefix)

	session1, err := sourceClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create source session: %v", err)
	}
	defer func() {
		_ = session1.Close()
	}()

	if _, err := session1.CombinedOutput(sudoWrapper(createChunksCmd, sourceClient.SSHTarget.Password)); err != nil {
		return fmt.Errorf("failed to create chunks: %v", err)
	}

	verifyChunksCmd := fmt.Sprintf("ls -la '%s'*", chunkPrefix)
	session2, err := sourceClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session2.Close()
	}()

	verifyOutput, err := session2.CombinedOutput(sudoWrapper(verifyChunksCmd, sourceClient.SSHTarget.Password))
	if err != nil {
		return fmt.Errorf("failed to verify chunks: %v", err)
	}
	migrationLogger.Printf(DEBUG, "Source chunks created:\n%s\n", string(verifyOutput))

	listChunksCmd := fmt.Sprintf("ls -1 '%s'* 2>/dev/null | wc -l", chunkPrefix)
	session3, err := sourceClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session3.Close()
	}()

	output, err := session3.CombinedOutput(sudoWrapper(listChunksCmd, sourceClient.SSHTarget.Password))
	if err != nil {
		return fmt.Errorf("failed to list chunks: %v", err)
	}

	chunkCount, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	migrationLogger.Printf(INFO, "Created %d chunks for transfer\n", chunkCount)

	if chunkCount == 0 {
		return fmt.Errorf("no chunks were created")
	}

	targetTempDir := fmt.Sprintf("/tmp/grasshopper_%s", uuid)
	createTargetDirCmd := fmt.Sprintf("mkdir -p '%s'", targetTempDir)

	session4, err := targetClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create target session: %v", err)
	}
	defer func() {
		_ = session4.Close()
	}()

	if _, err := session4.CombinedOutput(sudoWrapper(createTargetDirCmd, targetClient.SSHTarget.Password)); err != nil {
		return fmt.Errorf("failed to create target temp dir: %v", err)
	}

	for i := 0; i < chunkCount; i++ {
		chunkName := fmt.Sprintf("chunk_%02d", i)
		migrationLogger.Printf(DEBUG, "Transferring chunk %d/%d: %s\n", i+1, chunkCount, chunkName)

		checkChunkSizeCmd := fmt.Sprintf("stat -c '%%s' '%s/%s'", tempDir, chunkName)
		sourceSession, err := sourceClient.NewSession()
		if err != nil {
			migrationLogger.Printf(WARN, "Failed to create session for chunk size check: %v\n", err)
			continue
		}

		sizeOutput, err := sourceSession.CombinedOutput(sudoWrapper(checkChunkSizeCmd, sourceClient.SSHTarget.Password))
		_ = sourceSession.Close()
		if err != nil {
			migrationLogger.Printf(WARN, "Failed to get chunk size: %v\n", err)
			continue
		}

		chunkSize, _ := strconv.Atoi(strings.TrimSpace(string(sizeOutput)))
		migrationLogger.Printf(DEBUG, "Source chunk %s size: %d bytes\n", chunkName, chunkSize)

		if chunkSize == 0 {
			migrationLogger.Printf(WARN, "Source chunk %s is empty, skipping\n", chunkName)
			continue
		}

		if err := transferChunk(sourceClient, targetClient, tempDir, targetTempDir, chunkName, migrationLogger); err != nil {
			migrationLogger.Printf(WARN, "Failed to transfer chunk %s: %v\n", chunkName, err)
			continue
		}

		verifyTargetChunkCmd := fmt.Sprintf("stat -c '%%s' '%s/%s'", targetTempDir, chunkName)
		targetSession, err := targetClient.NewSession()
		if err != nil {
			migrationLogger.Printf(WARN, "Failed to create session for target chunk verify: %v\n", err)
			continue
		}

		targetSizeOutput, err := targetSession.CombinedOutput(sudoWrapper(verifyTargetChunkCmd, targetClient.SSHTarget.Password))
		_ = targetSession.Close()
		if err != nil {
			migrationLogger.Printf(WARN, "Failed to verify target chunk size: %v\n", err)
			continue
		}

		targetChunkSize, _ := strconv.Atoi(strings.TrimSpace(string(targetSizeOutput)))
		migrationLogger.Printf(DEBUG, "Target chunk %s size: %d bytes\n", chunkName, targetChunkSize)

		if targetChunkSize != chunkSize {
			migrationLogger.Printf(WARN, "Chunk size mismatch for %s: source=%d, target=%d\n", chunkName, chunkSize, targetChunkSize)
		}

		migrationLogger.Printf(DEBUG, "Successfully transferred chunk %d/%d\n", i+1, chunkCount)
	}

	reassembleCmd := fmt.Sprintf(`
        mkdir -p '%s' &&
        cd '%s' &&
        for i in $(seq -f "%%02g" 0 %d); do
            cat '%s'/chunk_$i
        done | tar --numeric-owner -xzf - &&
        rm -rf '%s'
    `, dirPath, dirPath, chunkCount-1, targetTempDir, targetTempDir)

	session5, err := targetClient.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create final session: %v", err)
	}
	defer func() {
		_ = session5.Close()
	}()

	if _, err := session5.CombinedOutput(sudoWrapper(reassembleCmd, targetClient.SSHTarget.Password)); err != nil {
		return fmt.Errorf("failed to reassemble chunks: %v", err)
	}

	cleanupCmd := fmt.Sprintf("rm -rf '%s'", tempDir)
	session6, err := sourceClient.NewSession()
	if err == nil {
		_, _ = session6.CombinedOutput(sudoWrapper(cleanupCmd, sourceClient.SSHTarget.Password))
		_ = session6.Close()
	}

	migrationLogger.Printf(INFO, "Successfully completed chunked copy of %s\n", dirPath)
	return nil
}

func copyDataDirectories(sourceClient *ssh.Client, targetClient *ssh.Client, packageName string, uuid string, migrationLogger *Logger) error {
	var isDataCopyNeeded = false

	if isWebServer(packageName) {
		isDataCopyNeeded = true
	}

	if !isDataCopyNeeded {
		migrationLogger.Printf(DEBUG, "Package %s is not need to copy data directory, skipping data directory copy\n", packageName)
		return nil
	}

	dataDirs := webServerDataDirs[packageName]
	migrationLogger.Printf(INFO, "Starting data directory copy for web server: %s\n", packageName)

	const MaxSizeBytes = 10 * 1024 * 1024 * 1024 // Limit Max to 10GB

	for _, dirPath := range dataDirs {
		migrationLogger.Printf(INFO, "Processing data directory: %s\n", dirPath)

		checkCmd := fmt.Sprintf("test -d '%s'", dirPath)
		session, err := sourceClient.NewSession()
		if err != nil {
			continue
		}

		_, err = session.CombinedOutput(sudoWrapper(checkCmd, sourceClient.SSHTarget.Password))
		_ = session.Close()

		if err != nil {
			migrationLogger.Printf(DEBUG, "Directory %s does not exist, skipping\n", dirPath)
			continue
		}

		size, err := checkDirectorySize(sourceClient, dirPath)
		if err != nil {
			migrationLogger.Printf(WARN, "Failed to check size of %s: %v\n", dirPath, err)
			continue
		}

		if size > MaxSizeBytes {
			migrationLogger.Printf(WARN, "Directory %s is too large (%s), skipping\n", dirPath, formatBytes(size))
			continue
		}

		if size == 0 {
			migrationLogger.Printf(DEBUG, "Directory %s is empty, skipping\n", dirPath)
			continue
		}

		migrationLogger.Printf(INFO, "Copying directory %s (size: %s)\n", dirPath, formatBytes(size))

		err = copyDirectoryWithChunks(sourceClient, targetClient, dirPath, uuid, migrationLogger)
		if err != nil {
			migrationLogger.Printf(ERROR, "Failed to copy directory %s: %v\n", dirPath, err)
			// Continue other directories even if error occurred
			continue
		}

		migrationLogger.Printf(INFO, "Successfully copied directory: %s\n", dirPath)
	}

	return nil
}

func configCopier(sourceClient *ssh.Client, targetClient *ssh.Client, packageName, uuid string, migrationLogger *Logger) error {
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
	configs, err := findConfigs(sourceClient, packageName, migrationLogger)
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to find configs for package %s: %v\n", packageName, err)
		return fmt.Errorf("failed to find configs: %v", err)
	}

	if len(configs) == 0 {
		migrationLogger.Printf(WARN, "No configuration files found for package: %s\n", packageName)
		return nil
	}

	migrationLogger.Printf(INFO, "Found %d configuration files for package %s\n", len(configs), packageName)

	if err := copyConfigFiles(sourceClient, targetClient, configs, migrationLogger); err != nil {
		migrationLogger.Printf(ERROR, "Failed to copy config files for package %s: %v\n", packageName, err)
		return fmt.Errorf("failed to copy config files: %v", err)
	}

	if err := copyDataDirectories(sourceClient, targetClient, packageName, uuid, migrationLogger); err != nil {
		migrationLogger.Printf(WARN, "Failed to copy data directories for package %s: %v\n", packageName, err)
	}

	migrationLogger.Printf(INFO, "Successfully completed config copy process for package: %s\n", packageName)
	return nil
}
