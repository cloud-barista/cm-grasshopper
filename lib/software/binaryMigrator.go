package software

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
)

// binaryUserInfo holds the resolved owner of a migrated binary on the source host.
type binaryUserInfo struct {
	uid   int32
	gid   int32
	uname string
	gname string
	home  string
	shell string
}

// envDenyList contains environment variables that are managed by the shell,
// the login session or systemd at runtime, so they must not be propagated to
// the target's /etc/environment (or to a command-based start line).
var envDenyList = map[string]bool{
	"USER":                     true,
	"LOGNAME":                  true,
	"HOME":                     true,
	"PWD":                      true,
	"OLDPWD":                   true,
	"SHELL":                    true,
	"SHLVL":                    true,
	"TERM":                     true,
	"MAIL":                     true,
	"_":                        true,
	"PATH":                     true,
	"HOSTNAME":                 true,
	"LS_COLORS":                true,
	"container":                true,
	"SYSTEMD_EXEC_PID":         true,
	"INVOCATION_ID":            true,
	"JOURNAL_STREAM":           true,
	"NOTIFY_SOCKET":            true,
	"LISTEN_PID":               true,
	"LISTEN_FDS":               true,
	"LISTEN_FDNAMES":           true,
	"MANAGERPID":               true,
	"SSH_CLIENT":               true,
	"SSH_CONNECTION":           true,
	"SSH_TTY":                  true,
	"XDG_RUNTIME_DIR":          true,
	"XDG_SESSION_ID":           true,
	"XDG_SESSION_TYPE":         true,
	"XDG_SESSION_CLASS":        true,
	"DBUS_SESSION_BUS_ADDRESS": true,
}

// runSSHCommand runs a command on the given host wrapped with sudo and returns
// its combined output as a string.
func runSSHCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSessionWithRetry()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	output, err := session.CombinedOutput(sudoWrapper(cmd, client.SSHTarget.Password))
	return string(output), err
}

// remotePathType reports whether the given path is a "dir", "file" or "none"
// (does not exist) on the remote host.
func remotePathType(client *ssh.Client, path string) string {
	out, _ := runSSHCommand(client, fmt.Sprintf("if [ -d '%s' ]; then echo dir; elif [ -e '%s' ]; then echo file; else echo none; fi", path, path))
	return strings.TrimSpace(out)
}

// isUnderCopiedPath returns true if path equals or is located under any path
// already copied to the target. Used to skip configs/data dirs that are already
// included in the binary or dependency directory copy.
func isUnderCopiedPath(copied map[string]bool, path string) bool {
	for cp := range copied {
		base := strings.TrimRight(cp, "/")
		if path == cp || path == base || strings.HasPrefix(path, base+"/") {
			return true
		}
	}
	return false
}

// remoteSymlinkTarget returns the raw link target of path and whether path is a
// symlink on the remote host. The target is exactly what `readlink` reports (it
// may be relative or absolute), so it can be recreated verbatim on the target.
func remoteSymlinkTarget(client *ssh.Client, path string) (string, bool) {
	out, _ := runSSHCommand(client, fmt.Sprintf("if [ -L '%s' ]; then readlink '%s'; else echo '__NOT_A_SYMLINK__'; fi", path, path))
	out = strings.TrimSpace(out)
	if out == "" || out == "__NOT_A_SYMLINK__" {
		return "", false
	}
	return out, true
}

// remoteRealPath returns the fully resolved (symlink-free) path on the remote host.
func remoteRealPath(client *ssh.Client, path string) string {
	out, _ := runSSHCommand(client, fmt.Sprintf("readlink -f '%s'", path))
	return strings.TrimSpace(out)
}

// copyPathWithChunks copies a file or directory from the source host to the same
// absolute path on the target host, preserving numeric ownership and permissions.
//
// If path is a symlink (e.g. catalina.home = /opt/tomcat/latest -> /opt/tomcat/
// apache-tomcat-10.x), it copies the resolved real target first (so the data
// exists on the target) and then recreates the symlink itself. A plain tar of a
// symlink would otherwise archive only the dangling link.
func copyPathWithChunks(sourceClient, targetClient *ssh.Client, path string, uuid string, migrationLogger *Logger) error {
	if linkTarget, isLink := remoteSymlinkTarget(sourceClient, path); isLink {
		realPath := remoteRealPath(sourceClient, path)
		migrationLogger.Printf(INFO, "%s is a symlink -> %s (real: %s)\n", path, linkTarget, realPath)

		if realPath != "" && realPath != path {
			if err := copyResolvedPath(sourceClient, targetClient, realPath, uuid, migrationLogger); err != nil {
				return err
			}
		}

		// Recreate the symlink on the target (mkdir parent first in case it differs).
		recreateCmd := fmt.Sprintf("mkdir -p \"$(dirname '%s')\" && ln -sfn '%s' '%s'", path, linkTarget, path)
		if out, err := runSSHCommand(targetClient, recreateCmd); err != nil {
			return fmt.Errorf("failed to recreate symlink %s -> %s: %s", path, linkTarget, out)
		}
		migrationLogger.Printf(INFO, "Recreated symlink %s -> %s on target\n", path, linkTarget)
		return nil
	}

	return copyResolvedPath(sourceClient, targetClient, path, uuid, migrationLogger)
}

// copyResolvedPath copies a real (non-symlink) file or directory from the source
// host to the same absolute path on the target host using a chunked, tar-based
// transfer. It is binary-safe and preserves numeric ownership and permissions, so
// it works for both directories (e.g. /opt/tomcat, /opt/jdk) and individual files.
func copyResolvedPath(sourceClient, targetClient *ssh.Client, path string, uuid string, migrationLogger *Logger) error {
	parent := filepath.Dir(path)
	base := filepath.Base(path)

	tempDir := fmt.Sprintf("/tmp/grasshopper_bin_%s", uuid)
	chunkPrefix := fmt.Sprintf("%s/chunk_", tempDir)
	chunkSize := "5M"

	migrationLogger.Printf(INFO, "Starting chunked copy of %s\n", path)

	createChunksCmd := fmt.Sprintf(`
        rm -rf '%s' &&
        mkdir -p '%s' &&
        cd '%s' &&
        tar --numeric-owner -czf - '%s' | split -d -b %s - '%s'
    `, tempDir, tempDir, parent, base, chunkSize, chunkPrefix)

	if out, err := runSSHCommand(sourceClient, createChunksCmd); err != nil {
		return fmt.Errorf("failed to create chunks for %s: %s", path, out)
	}

	listChunksCmd := fmt.Sprintf("ls -1 '%s'* 2>/dev/null | wc -l", chunkPrefix)
	out, err := runSSHCommand(sourceClient, listChunksCmd)
	if err != nil {
		return fmt.Errorf("failed to list chunks: %v", err)
	}

	chunkCount, _ := strconv.Atoi(strings.TrimSpace(out))
	migrationLogger.Printf(INFO, "Created %d chunks for transfer of %s\n", chunkCount, path)

	if chunkCount == 0 {
		return fmt.Errorf("no chunks were created for %s", path)
	}

	targetTempDir := tempDir
	if _, err := runSSHCommand(targetClient, fmt.Sprintf("rm -rf '%s' && mkdir -p '%s'", targetTempDir, targetTempDir)); err != nil {
		return fmt.Errorf("failed to create target temp dir: %v", err)
	}

	for i := 0; i < chunkCount; i++ {
		chunkName := fmt.Sprintf("chunk_%02d", i)
		migrationLogger.Printf(DEBUG, "Transferring chunk %d/%d: %s\n", i+1, chunkCount, chunkName)

		if err := transferChunk(sourceClient, targetClient, tempDir, targetTempDir, chunkName, migrationLogger); err != nil {
			return fmt.Errorf("failed to transfer chunk %s: %v", chunkName, err)
		}
	}

	reassembleCmd := fmt.Sprintf(`
        mkdir -p '%s' &&
        cd '%s' &&
        for i in $(seq -f "%%02g" 0 %d); do
            cat '%s'/chunk_$i
        done | tar --numeric-owner -xzf - &&
        rm -rf '%s'
    `, parent, parent, chunkCount-1, targetTempDir, targetTempDir)

	if out, err := runSSHCommand(targetClient, reassembleCmd); err != nil {
		return fmt.Errorf("failed to reassemble chunks for %s: %s", path, out)
	}

	_, _ = runSSHCommand(sourceClient, fmt.Sprintf("rm -rf '%s'", tempDir))

	migrationLogger.Printf(INFO, "Successfully copied %s\n", path)
	return nil
}

// resolveSourceUser resolves the owner (user/group) of the binary on the source
// host from its collected UID/GID. Returns nil if no UID information is available.
func resolveSourceUser(client *ssh.Client, binary *softwaremodel.BinaryMigrationInfo, migrationLogger *Logger) *binaryUserInfo {
	if len(binary.UIDs) == 0 {
		migrationLogger.Printf(WARN, "No UID information for binary %s, skipping user creation\n", binary.Name)
		return nil
	}

	info := &binaryUserInfo{uid: binary.UIDs[0]}
	if len(binary.GIDs) > 0 {
		info.gid = binary.GIDs[0]
	}

	// name:x:uid:gid:gecos:home:shell
	if out, err := runSSHCommand(client, fmt.Sprintf("getent passwd %d", info.uid)); err == nil {
		line := strings.TrimSpace(out)
		parts := strings.Split(line, ":")
		if len(parts) >= 7 {
			info.uname = parts[0]
			info.home = parts[5]
			info.shell = parts[6]
			if g, gerr := strconv.Atoi(parts[3]); gerr == nil {
				info.gid = int32(g)
			}
		}
	}

	// name:x:gid:members
	if out, err := runSSHCommand(client, fmt.Sprintf("getent group %d", info.gid)); err == nil {
		gline := strings.TrimSpace(out)
		gparts := strings.Split(gline, ":")
		if len(gparts) >= 1 && gparts[0] != "" {
			info.gname = gparts[0]
		}
	}

	// Fallbacks when /etc/passwd lookup did not return an entry.
	if info.uname == "" {
		info.uname = binary.Name
	}
	if info.gname == "" {
		info.gname = info.uname
	}
	if info.home == "" {
		info.home = binary.BinaryPath
	}
	if info.home == "" {
		info.home = "/home/" + info.uname
	}
	if info.shell == "" {
		info.shell = "/bin/false"
	}

	return info
}

// createBinaryUserOnTarget recreates the binary's owner (group then user) on the
// target host, preserving the source UID/GID. Existing entries are left untouched.
func createBinaryUserOnTarget(client *ssh.Client, info *binaryUserInfo, migrationLogger *Logger) error {
	if info == nil || info.uname == "" {
		return nil
	}

	// Group
	if gout, _ := runSSHCommand(client, fmt.Sprintf("getent group %d", info.gid)); strings.TrimSpace(gout) == "" {
		if out, err := runSSHCommand(client, fmt.Sprintf("groupadd -g %d %s", info.gid, info.gname)); err != nil {
			migrationLogger.Printf(WARN, "Failed to create group %s (gid=%d): %s\n", info.gname, info.gid, out)
		} else {
			migrationLogger.Printf(INFO, "Created group %s (gid=%d)\n", info.gname, info.gid)
		}
	} else {
		migrationLogger.Printf(INFO, "Group with GID %d already exists on target, skipping groupadd\n", info.gid)
	}

	// User
	if uout, _ := runSSHCommand(client, fmt.Sprintf("getent passwd %d", info.uid)); strings.TrimSpace(uout) == "" {
		useraddCmd := fmt.Sprintf("useradd -m -u %d -g %d -d %s -s %s %s",
			info.uid, info.gid, info.home, info.shell, info.uname)
		if out, err := runSSHCommand(client, useraddCmd); err != nil {
			// useradd may exit non-zero on warnings (e.g. home already exists).
			// Treat it as fatal only if the user really was not created.
			if recheck, _ := runSSHCommand(client, fmt.Sprintf("getent passwd %d", info.uid)); strings.TrimSpace(recheck) == "" {
				return fmt.Errorf("failed to create user %s (uid=%d): %s", info.uname, info.uid, out)
			}
			migrationLogger.Printf(WARN, "useradd reported a warning for %s: %s\n", info.uname, out)
		}
		migrationLogger.Printf(INFO, "Created user %s (uid=%d, gid=%d, home=%s, shell=%s)\n",
			info.uname, info.uid, info.gid, info.home, info.shell)
	} else {
		migrationLogger.Printf(INFO, "User with UID %d already exists on target, skipping useradd\n", info.uid)
	}

	return nil
}

// applyBinaryEnvironment writes the binary's application-relevant environment
// variables (e.g. JAVA_HOME) to the target's /etc/environment so they survive
// reboots. Shell/session/systemd-managed variables are filtered out.
func applyBinaryEnvironment(client *ssh.Client, binary *softwaremodel.BinaryMigrationInfo, migrationLogger *Logger) error {
	var script strings.Builder
	script.WriteString("touch /etc/environment\n")

	var applied []string
	for _, env := range binary.Envs {
		env = strings.TrimSpace(env)
		if env == "" {
			continue
		}

		idx := strings.Index(env, "=")
		if idx <= 0 {
			continue
		}
		key := env[:idx]
		value := env[idx+1:]

		if envDenyList[key] {
			continue
		}

		escapedValue := strings.ReplaceAll(value, `"`, `\"`)
		script.WriteString(fmt.Sprintf("sed -i '/^%s=/d' /etc/environment\n", key))
		script.WriteString(fmt.Sprintf("echo '%s=\"%s\"' >> /etc/environment\n", key, escapedValue))
		applied = append(applied, key)
	}

	if len(applied) == 0 {
		migrationLogger.Printf(INFO, "No application environment variables to apply for %s\n", binary.Name)
		return nil
	}

	if out, err := runSSHCommand(client, script.String()); err != nil {
		return fmt.Errorf("failed to apply environment variables: %s", out)
	}

	migrationLogger.Printf(INFO, "Applied environment variables to /etc/environment: %s\n", strings.Join(applied, ", "))
	return nil
}

// findBinaryServiceFile looks for a systemd unit file related to the binary on
// the source host. Returns the resolved unit file path, the unit name and whether
// the unit is enabled. An empty path means no systemd unit was found.
func findBinaryServiceFile(client *ssh.Client, name string, migrationLogger *Logger) (string, string, bool) {
	findCmd := fmt.Sprintf("find /etc/systemd/system /usr/lib/systemd/system /lib/systemd/system -maxdepth 2 -name '*%s*.service' 2>/dev/null", name)
	out, _ := runSSHCommand(client, findCmd)

	var candidate string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip the *.wants/ symlinks; we want the real unit file.
		if strings.Contains(line, ".wants/") {
			continue
		}
		candidate = line
		// Prefer locally-defined units in /etc/systemd/system.
		if strings.HasPrefix(line, "/etc/systemd/system/") {
			break
		}
	}

	if candidate == "" {
		return "", "", false
	}

	realPath := candidate
	if out, _ := runSSHCommand(client, fmt.Sprintf("readlink -f '%s'", candidate)); strings.TrimSpace(out) != "" {
		realPath = strings.TrimSpace(out)
	}

	unitName := filepath.Base(realPath)

	enabled := false
	if out, _ := runSSHCommand(client, fmt.Sprintf("systemctl is-enabled %s 2>/dev/null", unitName)); strings.TrimSpace(out) == "enabled" {
		enabled = true
	}

	migrationLogger.Printf(INFO, "Found systemd unit for %s: %s (enabled=%v)\n", name, realPath, enabled)
	return realPath, unitName, enabled
}

// reloadEnableStart runs daemon-reload, optionally enables, then starts the unit
// on the target and verifies it became active.
func reloadEnableStart(targetClient *ssh.Client, unitName string, enabled bool, migrationLogger *Logger) error {
	if out, err := runSSHCommand(targetClient, "systemctl daemon-reload"); err != nil {
		migrationLogger.Printf(WARN, "systemctl daemon-reload failed: %s\n", out)
	}

	if enabled {
		if out, err := runSSHCommand(targetClient, fmt.Sprintf("systemctl enable %s", unitName)); err != nil {
			migrationLogger.Printf(WARN, "Failed to enable %s: %s\n", unitName, out)
		} else {
			migrationLogger.Printf(INFO, "Enabled service %s on target\n", unitName)
		}
	}

	if out, err := runSSHCommand(targetClient, fmt.Sprintf("systemctl start %s", unitName)); err != nil {
		return fmt.Errorf("failed to start service %s: %s", unitName, out)
	}

	if out, _ := runSSHCommand(targetClient, fmt.Sprintf("systemctl is-active %s", unitName)); strings.TrimSpace(out) != "active" {
		return fmt.Errorf("service %s is not active after start (state: %s)", unitName, strings.TrimSpace(out))
	}

	migrationLogger.Printf(INFO, "Service %s started and active on target\n", unitName)
	return nil
}

// copyAndStartUnit copies an existing systemd unit file from the source to the
// same path on the target and starts it.
func copyAndStartUnit(sourceClient, targetClient *ssh.Client, unitPath, unitName string, enabled bool, migrationLogger *Logger) error {
	migrationLogger.Printf(INFO, "Migrating systemd unit %s (%s)\n", unitName, unitPath)
	if err := copyFile(sourceClient, targetClient, unitPath, migrationLogger); err != nil {
		return fmt.Errorf("failed to copy systemd unit file %s: %v", unitPath, err)
	}
	return reloadEnableStart(targetClient, unitName, enabled, migrationLogger)
}

// synthesizeAndStartUnit generates a systemd unit for a binary that was NOT
// managed by systemd on the source (started directly from the command line). The
// captured command line becomes ExecStart of a Type=simple service, supervised by
// systemd on the target — a modern replacement for the legacy /etc/rc.local trick.
//
// Known limitation: assumes the captured process is the long-running foreground
// daemon (true for most servers, e.g. java). A forking wrapper would need
// Type=forking with a PIDFile, which we cannot infer generically.
func synthesizeAndStartUnit(targetClient *ssh.Client, binary *softwaremodel.BinaryMigrationInfo, info *binaryUserInfo, migrationLogger *Logger) error {
	if len(binary.CmdlineSlice) == 0 {
		return fmt.Errorf("cannot synthesize a systemd unit for %s: no command line captured", binary.Name)
	}

	unitName := binary.Name + ".service"
	unitPath := "/etc/systemd/system/" + unitName

	var userName, groupName string
	if info != nil {
		userName = info.uname
		groupName = info.gname
	}

	var unit strings.Builder
	unit.WriteString("[Unit]\n")
	unit.WriteString(fmt.Sprintf("Description=%s (migrated by cm-grasshopper)\n", binary.Name))
	unit.WriteString("After=network.target\n\n")
	unit.WriteString("[Service]\n")
	unit.WriteString("Type=simple\n")
	if userName != "" {
		unit.WriteString(fmt.Sprintf("User=%s\n", userName))
	}
	if groupName != "" {
		unit.WriteString(fmt.Sprintf("Group=%s\n", groupName))
	}
	if binary.WorkingDirectory != "" {
		unit.WriteString(fmt.Sprintf("WorkingDirectory=%s\n", binary.WorkingDirectory))
	}
	for _, env := range binary.Envs {
		env = strings.TrimSpace(env)
		idx := strings.Index(env, "=")
		if idx <= 0 {
			continue
		}
		key := env[:idx]
		value := env[idx+1:]
		if envDenyList[key] {
			continue
		}
		escaped := strings.ReplaceAll(value, `"`, `\"`)
		unit.WriteString(fmt.Sprintf("Environment=\"%s=%s\"\n", key, escaped))
	}
	unit.WriteString(fmt.Sprintf("ExecStart=%s\n", strings.Join(binary.CmdlineSlice, " ")))
	unit.WriteString("Restart=on-failure\n\n")
	unit.WriteString("[Install]\n")
	unit.WriteString("WantedBy=multi-user.target\n")

	migrationLogger.Printf(INFO, "Synthesizing systemd unit at %s\n", unitPath)

	writeCmd := fmt.Sprintf("cat > '%s' << 'GRASSHOPPER_UNIT_EOF'\n%s\nGRASSHOPPER_UNIT_EOF\nchmod 644 '%s'",
		unitPath, unit.String(), unitPath)

	if out, err := runSSHCommand(targetClient, writeCmd); err != nil {
		return fmt.Errorf("failed to write synthesized unit: %s", out)
	}

	// A synthesized unit is always enabled so the service survives reboots.
	return reloadEnableStart(targetClient, unitName, true, migrationLogger)
}

// startBinaryService reproduces the source's start mechanism on the target, based
// on the launch provenance collected by cm-honeybee:
//   - systemd-managed  -> copy the original unit file and start it
//   - command-started  -> synthesize a systemd unit from the captured command line
//   - unknown          -> prefer an existing unit if one is found, else synthesize
func startBinaryService(sourceClient, targetClient *ssh.Client, binary *softwaremodel.BinaryMigrationInfo, info *binaryUserInfo, migrationLogger *Logger) error {
	switch binary.LaunchType {
	case "systemd":
		unitPath := binary.SystemdUnitPath
		unitName := binary.SystemdUnitName
		enabled := binary.SystemdEnabled
		if unitPath == "" {
			// Provenance says systemd but no path was captured; locate it on the source.
			unitPath, unitName, enabled = findBinaryServiceFile(sourceClient, binary.Name, migrationLogger)
		}
		if unitPath == "" {
			migrationLogger.Printf(WARN, "LaunchType=systemd but no unit file available for %s; synthesizing one\n", binary.Name)
			return synthesizeAndStartUnit(targetClient, binary, info, migrationLogger)
		}
		return copyAndStartUnit(sourceClient, targetClient, unitPath, unitName, enabled, migrationLogger)

	case "command":
		migrationLogger.Printf(INFO, "Source %s was started by command; synthesizing a systemd unit on target\n", binary.Name)
		return synthesizeAndStartUnit(targetClient, binary, info, migrationLogger)

	default: // "unknown" or empty: prefer an existing unit, otherwise synthesize.
		unitPath, unitName, enabled := findBinaryServiceFile(sourceClient, binary.Name, migrationLogger)
		if unitPath != "" {
			return copyAndStartUnit(sourceClient, targetClient, unitPath, unitName, enabled, migrationLogger)
		}
		migrationLogger.Printf(INFO, "No launch provenance and no unit found for %s; synthesizing a systemd unit\n", binary.Name)
		return synthesizeAndStartUnit(targetClient, binary, info, migrationLogger)
	}
}

// binaryMigrator migrates a legacy software binary (e.g. Tomcat) from the source
// to the target host. It copies the binary, its runtime dependencies, configs and
// data directories, recreates the owning user, applies the relevant environment
// variables and starts the software, reproducing the source's launch mechanism
// (systemd unit copy, or a synthesized systemd unit for command-started software).
func binaryMigrator(sourceClient, targetClient *ssh.Client, binary *softwaremodel.BinaryMigrationInfo, uuid string, migrationLogger *Logger) error {
	migrationLogger.Printf(INFO, "Starting binary migration for %s (version: %s)\n", binary.Name, binary.Version)

	if binary.IsWine {
		return fmt.Errorf("wine binary migration is not supported yet (binary: %s)", binary.Name)
	}

	// Tracks paths already transferred to avoid re-copying nested configs/data.
	copiedPaths := make(map[string]bool)

	// 1. Copy runtime dependency paths (e.g. /opt/jdk for Java).
	for _, dep := range binary.NeededLibraries {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			continue
		}
		if !strings.HasPrefix(dep, "/") {
			// A bare SONAME (no absolute path) is expected to be provided by the
			// base system or another dependency; nothing to copy.
			migrationLogger.Printf(DEBUG, "Skipping non-absolute dependency: %s\n", dep)
			continue
		}
		if isUnderCopiedPath(copiedPaths, dep) {
			continue
		}

		migrationLogger.Printf(INFO, "Copying dependency path: %s\n", dep)
		if err := copyPathWithChunks(sourceClient, targetClient, dep, uuid, migrationLogger); err != nil {
			return fmt.Errorf("failed to copy dependency %s: %v", dep, err)
		}
		copiedPaths[dep] = true
	}

	// 2. Copy the binary path itself (e.g. /opt/tomcat).
	if bp := strings.TrimSpace(binary.BinaryPath); bp != "" && strings.HasPrefix(bp, "/") {
		migrationLogger.Printf(INFO, "Copying binary path: %s\n", bp)
		if err := copyPathWithChunks(sourceClient, targetClient, bp, uuid, migrationLogger); err != nil {
			return fmt.Errorf("failed to copy binary path %s: %v", bp, err)
		}
		copiedPaths[bp] = true
	}

	// 3. Copy custom config files (skip ones already inside a copied directory).
	for _, cfg := range binary.CustomConfigs {
		cfg = strings.TrimSpace(cfg)
		if cfg == "" || !strings.HasPrefix(cfg, "/") {
			continue
		}
		if isUnderCopiedPath(copiedPaths, cfg) {
			migrationLogger.Printf(INFO, "Config %s already included in a copied directory, skipping\n", cfg)
			continue
		}

		migrationLogger.Printf(INFO, "Copying config: %s\n", cfg)
		switch remotePathType(sourceClient, cfg) {
		case "dir":
			if err := copyPathWithChunks(sourceClient, targetClient, cfg, uuid, migrationLogger); err != nil {
				return fmt.Errorf("failed to copy config directory %s: %v", cfg, err)
			}
		case "file":
			if err := copyFile(sourceClient, targetClient, cfg, migrationLogger); err != nil {
				return fmt.Errorf("failed to copy config file %s: %v", cfg, err)
			}
		default:
			migrationLogger.Printf(WARN, "Config path does not exist on source, skipping: %s\n", cfg)
		}
	}

	// 4. Copy custom data directories (skip ones already inside a copied directory).
	for _, dataPath := range binary.CustomDataPaths {
		dataPath = strings.TrimSpace(dataPath)
		if dataPath == "" || !strings.HasPrefix(dataPath, "/") {
			continue
		}
		if isUnderCopiedPath(copiedPaths, dataPath) {
			migrationLogger.Printf(INFO, "Data path %s already included in a copied directory, skipping\n", dataPath)
			continue
		}

		migrationLogger.Printf(INFO, "Copying data path: %s\n", dataPath)
		if err := copyPathWithChunks(sourceClient, targetClient, dataPath, uuid, migrationLogger); err != nil {
			return fmt.Errorf("failed to copy data path %s: %v", dataPath, err)
		}
		copiedPaths[dataPath] = true
	}

	// 5. Recreate the owning user/group on the target (non-fatal on failure).
	userInfo := resolveSourceUser(sourceClient, binary, migrationLogger)
	if err := createBinaryUserOnTarget(targetClient, userInfo, migrationLogger); err != nil {
		migrationLogger.Printf(WARN, "User creation issue for %s: %v\n", binary.Name, err)
	}

	// 6. Apply application environment variables (non-fatal on failure).
	if err := applyBinaryEnvironment(targetClient, binary, migrationLogger); err != nil {
		migrationLogger.Printf(WARN, "Environment application issue for %s: %v\n", binary.Name, err)
	}

	// 7. Migrate the service definition and start the software.
	if err := startBinaryService(sourceClient, targetClient, binary, userInfo, migrationLogger); err != nil {
		return fmt.Errorf("failed to start binary %s: %v", binary.Name, err)
	}

	migrationLogger.Printf(INFO, "Successfully migrated binary: %s\n", binary.Name)
	return nil
}
