package software

import (
	"fmt"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
)

// snapStoreProbeURL is hit (with a short timeout) from the target to decide
// whether the snap store is reachable (online install) or the target is
// air-gapped (offline sideload from the source blob).
const snapStoreProbeURL = "https://api.snapcraft.io/"

// shellSingleQuote single-quotes an argument for safe shell use.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// runTargetCmd runs a sudo-wrapped command on the target host and returns its
// combined output.
func runTargetCmd(targetClient *ssh.Client, cmd string) (string, error) {
	return runClientCmd(targetClient, cmd)
}

// runClientCmd runs a sudo-wrapped command on the given host (source or target)
// and returns its combined output.
func runClientCmd(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSessionWithRetry()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer func() { _ = session.Close() }()

	out, err := session.CombinedOutput(sudoWrapper(cmd, client.SSHTarget.Password))
	return string(out), err
}

// storeReachable reports whether url is reachable from the given host. A false
// result (curl missing, DNS/timeout, non-2xx/3xx) is treated as "air-gapped" so
// the caller falls back to offline sideload.
func storeReachable(client *ssh.Client, url string) bool {
	q := shellSingleQuote(url)
	// Try curl, then wget; if neither tool exists we can't confirm reachability,
	// so treat the target as air-gapped and let the caller sideload.
	probe := "if command -v curl >/dev/null 2>&1; then " +
		"curl -sfI --max-time 8 -o /dev/null " + q + " && echo REACHABLE || echo UNREACHABLE; " +
		"elif command -v wget >/dev/null 2>&1; then " +
		"wget -q --timeout=8 --spider " + q + " && echo REACHABLE || echo UNREACHABLE; " +
		"else echo UNREACHABLE; fi"
	out, err := runClientCmd(client, probe)
	if err != nil {
		return false
	}
	return strings.Contains(out, "REACHABLE")
}

// ensureTargetCommand installs the package that provides cmd on the target when
// the command is missing (best-effort — for snap/flatpak tooling).
func ensureTargetCommand(targetClient *ssh.Client, cmd, pkg string, migrationLogger *Logger) error {
	if err := checkCommandExists(targetClient, cmd); err == nil {
		return nil
	}

	systemType, err := getSystemType(targetClient, migrationLogger)
	if err != nil {
		return fmt.Errorf("failed to detect target system type: %v", err)
	}

	var installCmd string
	switch systemType {
	case Debian:
		installCmd = "apt-get update -y >/dev/null 2>&1; DEBIAN_FRONTEND=noninteractive apt-get install -y " + pkg
	case RHEL:
		// snapd/flatpak live in EPEL on RHEL/Rocky/Alma; CSP base images ship
		// neither the tool nor the repo, so enable EPEL first (best-effort).
		epel := "(dnf install -y epel-release || yum install -y epel-release || true)"
		if err := checkCommandExists(targetClient, "dnf"); err == nil {
			installCmd = epel + "; dnf install -y " + pkg
		} else {
			installCmd = epel + "; yum install -y " + pkg
		}
	default:
		return fmt.Errorf("unsupported target system type for installing %s", pkg)
	}

	migrationLogger.Printf(INFO, "Installing %s (provides '%s') on target\n", pkg, cmd)
	out, err := runTargetCmd(targetClient, installCmd)
	migrationLogger.Printf(DEBUG, "%s install output: %s\n", pkg, out)
	if err != nil {
		return fmt.Errorf("failed to install %s: %s", pkg, out)
	}
	return nil
}

// ensureSnapd makes snapd usable on the target: installs it (EPEL on RHEL),
// enables the socket, and creates the /snap symlink classic snaps require on
// non-Ubuntu distros.
func ensureSnapd(targetClient *ssh.Client, migrationLogger *Logger) error {
	if err := ensureTargetCommand(targetClient, "snap", "snapd", migrationLogger); err != nil {
		return err
	}
	// Enable snapd, create the /snap symlink (classic snaps expect it on
	// RHEL-family), and wait for seeding so the first install doesn't race snapd.
	_, _ = runTargetCmd(targetClient, "systemctl enable --now snapd.socket snapd.service 2>/dev/null; "+
		"[ -e /snap ] || ln -s /var/lib/snapd/snap /snap 2>/dev/null; "+
		"snap wait system seed.loaded 2>/dev/null; true")
	return nil
}

// warnConfinementDrift logs a warning when a strict-confined snap lands on a
// target whose snapd cannot enforce strict confinement (e.g. Ubuntu/AppArmor ->
// RHEL/SELinux), where confinement silently degrades to partial. `snap debug
// confinement` reports strict/partial/none independent of the LSM.
func warnConfinementDrift(targetClient *ssh.Client, pkg *softwaremodel.PackageMigrationInfo, migrationLogger *Logger) {
	if strings.TrimSpace(pkg.Confinement) != "strict" && strings.TrimSpace(pkg.Confinement) != "" {
		return // classic/devmode snaps are unconfined by design; no drift to warn about
	}
	out, err := runTargetCmd(targetClient, "snap debug confinement 2>/dev/null || true")
	if err != nil {
		return
	}
	level := strings.TrimSpace(out)
	if level != "" && level != "strict" {
		migrationLogger.Printf(WARN, "snap %s is strict-confined on the source but the target enforces only '%s' "+
			"confinement (likely AppArmor->SELinux); its sandbox will be weaker than on the source\n", pkg.Name, level)
	}
}

// resolvePackageType classifies a package whose type was not set upstream and
// routes it to the right installer. A migration list produced before snap/flatpak
// support (and frozen into a stored cm-cicada workflow body) carries an empty type
// for snaps/flatpaks; installed via the OS package manager they fail as
// "No package matching '<name>' is available". Probing the source host recovers the
// real kind so the snap/flatpak installer is used instead. Only empty types are
// probed, so normally-classified deb/rpm packages pay no cost.
func resolvePackageType(sourceClient *ssh.Client, pkg *softwaremodel.PackageMigrationInfo, migrationLogger *Logger) {
	if pkg.Type != "" {
		return
	}

	name := shellSingleQuote(pkg.Name)

	if out, err := runClientCmd(sourceClient, "snap list "+name+" 2>/dev/null"); err == nil && strings.TrimSpace(out) != "" {
		pkg.Type = softwaremodel.SoftwarePackageTypeSnap
		migrationLogger.Printf(INFO, "Package %s has no type set; the source reports it as a snap, installing via snap\n", pkg.Name)
		return
	}

	if out, err := runClientCmd(sourceClient, "flatpak info "+name+" 2>/dev/null"); err == nil && strings.TrimSpace(out) != "" {
		pkg.Type = softwaremodel.SoftwarePackageTypeFlatpak
		migrationLogger.Printf(INFO, "Package %s has no type set; the source reports it as a flatpak, installing via flatpak\n", pkg.Name)
		return
	}
	// Left empty: handled by the OS package (deb/rpm) playbook path.
}

// snapMigrator installs a snap on the target. It boots snapd, warns on
// confinement drift, then installs from the store when reachable or sideloads the
// source blob when the target is air-gapped. Verification is presence/channel
// based (revisions drift on auto-refresh, so a revision mismatch is not a failure).
func snapMigrator(sourceClient, targetClient *ssh.Client, pkg *softwaremodel.PackageMigrationInfo,
	executionID string, migrationLogger *Logger) error {
	if err := ensureSnapd(targetClient, migrationLogger); err != nil {
		return err
	}
	warnConfinementDrift(targetClient, pkg, migrationLogger)

	var err error
	if storeReachable(targetClient, snapStoreProbeURL) {
		err = snapInstallOnline(targetClient, pkg, migrationLogger)
	} else if strings.TrimSpace(pkg.BlobPath) != "" {
		migrationLogger.Printf(INFO, "snap store unreachable from target; sideloading %s from source blob\n", pkg.Name)
		err = snapInstallOffline(sourceClient, targetClient, pkg, executionID, migrationLogger)
	} else {
		return fmt.Errorf("snap %s: store unreachable and no source blob available for offline install", pkg.Name)
	}
	if err != nil {
		return err
	}

	if verr := snapVerify(targetClient, pkg, migrationLogger); verr != nil {
		return verr
	}

	// Snap config/data lives outside /etc (/var/snap/<name>, ~/snap/<name>); copy
	// it so the app keeps its settings. Best-effort — the app is already installed.
	migrateDataDirs(sourceClient, targetClient, snapDataCandidates(sourceClient, pkg.Name),
		executionID, "snap "+pkg.Name, migrationLogger)
	return nil
}

// snapInstallOnline installs a snap from the store. --classic is passed up front
// for classic-confined snaps and also retried if the store reports it is needed.
func snapInstallOnline(targetClient *ssh.Client, pkg *softwaremodel.PackageMigrationInfo, migrationLogger *Logger) error {
	installCmd := "snap install " + shellSingleQuote(pkg.Name)
	if channel := strings.TrimSpace(pkg.Channel); channel != "" {
		installCmd += " --channel=" + shellSingleQuote(channel)
	}
	if strings.TrimSpace(pkg.Confinement) == "classic" {
		installCmd += " --classic"
	}

	migrationLogger.Printf(INFO, "Installing snap (online): %s\n", pkg.Name)
	out, err := runTargetCmd(targetClient, installCmd)
	if err != nil && !strings.Contains(installCmd, "--classic") && strings.Contains(strings.ToLower(out), "classic") {
		migrationLogger.Printf(INFO, "snap %s requires classic confinement, retrying with --classic\n", pkg.Name)
		out, err = runTargetCmd(targetClient, installCmd+" --classic")
	}
	migrationLogger.Printf(DEBUG, "snap install output: %s\n", out)
	if err != nil {
		return fmt.Errorf("snap install failed for %s: %s", pkg.Name, out)
	}
	return nil
}

// snapInstallOffline sideloads a snap on an air-gapped target: it stages the app
// blob (and its base snap blob) on the source, copies the staging dir to the
// target, and installs base-first with `snap install --dangerous` (assertion
// checks skipped — the store can't be reached to verify them). Its base is
// installed first because the app snap won't run without it.
func snapInstallOffline(sourceClient, targetClient *ssh.Client, pkg *softwaremodel.PackageMigrationInfo,
	executionID string, migrationLogger *Logger) error {
	stageDir := "/tmp/grasshopper_snap_" + executionID + "_" + sanitizeName(pkg.Name)

	// Stage the app blob and (if any) the base snap blob on the source.
	stage := "rm -rf " + stageDir + " && mkdir -p " + stageDir + " && " +
		"cp -f " + shellSingleQuote(pkg.BlobPath) + " " + stageDir + "/"
	if base := sanitizeName(strings.TrimSpace(pkg.Base)); base != "" && base != "none" {
		// Base is a separate snap filtered out of the migration list; grab its
		// blob by globbing snapd's blob dir (revision-agnostic). The glob is left
		// unquoted so the shell expands it; base is sanitized to safe chars.
		stage += " && (cp -f /var/lib/snapd/snaps/" + base + "_*.snap " + stageDir + "/ 2>/dev/null || true)"
	}
	if out, err := runClientCmd(sourceClient, stage); err != nil {
		return fmt.Errorf("failed to stage snap blob on source: %s", out)
	}

	if err := copyDirectoryWithChunks(sourceClient, targetClient, stageDir, executionID, migrationLogger); err != nil {
		return fmt.Errorf("failed to transfer snap blob to target: %v", err)
	}
	defer func() {
		_, _ = runClientCmd(sourceClient, "rm -rf "+stageDir)
		_, _ = runTargetCmd(targetClient, "rm -rf "+stageDir)
	}()

	classic := ""
	if strings.TrimSpace(pkg.Confinement) == "classic" {
		classic = " --classic"
	}

	// Install the base snap(s) first (all blobs except the app's), then the app.
	appBlob := stageDir + "/" + baseName(pkg.BlobPath)
	install := "set -e; " +
		"for b in " + stageDir + "/*.snap; do " +
		"[ \"$b\" = " + shellSingleQuote(appBlob) + " ] && continue; " +
		"snap install --dangerous \"$b\" || true; done; " +
		"snap install --dangerous" + classic + " " + shellSingleQuote(appBlob)

	migrationLogger.Printf(INFO, "Installing snap (offline sideload): %s\n", pkg.Name)
	out, err := runTargetCmd(targetClient, install)
	migrationLogger.Printf(DEBUG, "snap sideload output: %s\n", out)
	if err != nil {
		return fmt.Errorf("snap sideload failed for %s: %s", pkg.Name, out)
	}
	return nil
}

// snapVerify checks the snap is present on the target. Presence (not a revision
// match) is the success criterion: snaps auto-refresh, so the installed revision
// legitimately differs from the source's.
func snapVerify(targetClient *ssh.Client, pkg *softwaremodel.PackageMigrationInfo, migrationLogger *Logger) error {
	out, err := runTargetCmd(targetClient, "snap list "+shellSingleQuote(pkg.Name)+" 2>&1")
	if err != nil || !strings.Contains(out, pkg.Name) {
		return fmt.Errorf("snap %s not present after install: %s", pkg.Name, strings.TrimSpace(out))
	}
	if rev := strings.TrimSpace(pkg.Revision); rev != "" && !strings.Contains(out, rev) {
		migrationLogger.Printf(INFO, "snap %s installed at a different revision than the source (source rev %s); "+
			"expected — snaps auto-refresh\n", pkg.Name, rev)
	}
	return nil
}

// flatpakMigrator installs a flatpak app on the target: online via the source's
// remote when reachable, otherwise offline by exporting the app+runtime from the
// source with `flatpak create-usb` and installing with --sideload-repo.
func flatpakMigrator(sourceClient, targetClient *ssh.Client, pkg *softwaremodel.PackageMigrationInfo,
	executionID string, migrationLogger *Logger) error {
	if err := ensureTargetCommand(targetClient, "flatpak", "flatpak", migrationLogger); err != nil {
		return err
	}

	origin := strings.TrimSpace(pkg.Origin)
	if origin == "" {
		origin = "flathub"
	}
	appID := strings.TrimSpace(pkg.ApplicationID)
	if appID == "" {
		appID = pkg.Name
	}
	url := strings.TrimSpace(pkg.OriginURL)

	// Add the remote using the URL collected from the source (not a hard-coded
	// one), so custom remotes work too. Source-trusted -> gpg verification off.
	if url != "" {
		migrationLogger.Printf(INFO, "Ensuring flatpak remote %s -> %s\n", origin, url)
		_, _ = runTargetCmd(targetClient, "flatpak remote-add --if-not-exists --no-gpg-verify "+
			shellSingleQuote(origin)+" "+shellSingleQuote(url))
	}

	var err error
	if url != "" && storeReachable(targetClient, url) {
		err = flatpakInstallOnline(targetClient, origin, appID, migrationLogger)
	} else {
		migrationLogger.Printf(INFO, "flatpak remote unreachable from target; sideloading %s from source export\n", appID)
		err = flatpakInstallOffline(sourceClient, targetClient, origin, appID, executionID, migrationLogger)
	}
	if err != nil {
		return err
	}

	if verr := flatpakVerify(targetClient, appID, migrationLogger); verr != nil {
		return verr
	}

	// Flatpak per-user data lives in ~/.var/app/<app-id>; copy it so the app keeps
	// its settings. Best-effort — the app is already installed.
	migrateDataDirs(sourceClient, targetClient, flatpakDataCandidates(sourceClient, appID),
		executionID, "flatpak "+appID, migrationLogger)
	return nil
}

// flatpakInstallOnline installs the app (and its runtime, auto-resolved) from the
// remote.
func flatpakInstallOnline(targetClient *ssh.Client, origin, appID string, migrationLogger *Logger) error {
	migrationLogger.Printf(INFO, "Installing flatpak (online): %s (from %s)\n", appID, origin)
	out, err := runTargetCmd(targetClient,
		"flatpak install -y --noninteractive "+shellSingleQuote(origin)+" "+shellSingleQuote(appID))
	migrationLogger.Printf(DEBUG, "flatpak install output: %s\n", out)
	if err != nil {
		return fmt.Errorf("flatpak install failed for %s: %s", appID, out)
	}
	return nil
}

// flatpakInstallOffline exports the app and its runtime dependencies from the
// source with `flatpak create-usb` (which bundles runtimes so the app can launch
// air-gapped), copies the export to the target, and installs with
// --sideload-repo so no network access is needed.
func flatpakInstallOffline(sourceClient, targetClient *ssh.Client, origin, appID, executionID string,
	migrationLogger *Logger) error {
	stageDir := "/tmp/grasshopper_flatpak_" + executionID + "_" + sanitizeName(appID)

	// create-usb writes an ostree repo (with runtimes) under stageDir/.ostree/repo.
	stage := "rm -rf " + stageDir + " && mkdir -p " + stageDir + " && " +
		"flatpak create-usb --allow-partial " + stageDir + " " + shellSingleQuote(appID)
	if out, err := runClientCmd(sourceClient, stage); err != nil {
		return fmt.Errorf("failed to export flatpak on source (create-usb): %s", out)
	}

	if err := copyDirectoryWithChunks(sourceClient, targetClient, stageDir, executionID, migrationLogger); err != nil {
		return fmt.Errorf("failed to transfer flatpak export to target: %v", err)
	}
	defer func() {
		_, _ = runClientCmd(sourceClient, "rm -rf "+stageDir)
		_, _ = runTargetCmd(targetClient, "rm -rf "+stageDir)
	}()

	repoPath := stageDir + "/.ostree/repo"
	migrationLogger.Printf(INFO, "Installing flatpak (offline sideload): %s\n", appID)
	out, err := runTargetCmd(targetClient,
		"flatpak install -y --noninteractive --sideload-repo="+shellSingleQuote(repoPath)+" "+
			shellSingleQuote(origin)+" "+shellSingleQuote(appID))
	migrationLogger.Printf(DEBUG, "flatpak sideload output: %s\n", out)
	if err != nil {
		return fmt.Errorf("flatpak sideload failed for %s: %s", appID, out)
	}
	return nil
}

// flatpakVerify checks the app is installed on the target.
func flatpakVerify(targetClient *ssh.Client, appID string, migrationLogger *Logger) error {
	out, err := runTargetCmd(targetClient, "flatpak info "+shellSingleQuote(appID)+" >/dev/null 2>&1 && echo OK || echo MISSING")
	if err != nil || !strings.Contains(out, "OK") {
		return fmt.Errorf("flatpak %s not present after install", appID)
	}
	return nil
}

// sanitizeName makes a package name safe for use in a filesystem path.
func sanitizeName(s string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, s)
}

// baseName returns the final path element of p.
func baseName(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// sourceHomeDirs returns /root plus every /home/* directory on the source, so
// per-user data paths (~/snap, ~/.var/app) can be enumerated for all users.
func sourceHomeDirs(sourceClient *ssh.Client) []string {
	out, err := runClientCmd(sourceClient, "echo /root; ls -d /home/*/ 2>/dev/null || true")
	homes := []string{}
	if err != nil {
		return []string{"/root"}
	}
	for _, line := range strings.Split(out, "\n") {
		if s := strings.TrimRight(strings.TrimSpace(line), "/"); s != "" {
			homes = append(homes, s)
		}
	}
	return homes
}

// snapDataCandidates lists a snap's writable-data directories on the source:
// system data (/var/snap/<name>) and each user's ~/snap/<name>.
func snapDataCandidates(sourceClient *ssh.Client, name string) []string {
	candidates := []string{"/var/snap/" + name}
	for _, home := range sourceHomeDirs(sourceClient) {
		candidates = append(candidates, home+"/snap/"+name)
	}
	return existingSourceDirs(sourceClient, candidates)
}

// flatpakDataCandidates lists a flatpak app's per-user data directories on the
// source (~/.var/app/<app-id>).
func flatpakDataCandidates(sourceClient *ssh.Client, appID string) []string {
	candidates := make([]string, 0)
	for _, home := range sourceHomeDirs(sourceClient) {
		candidates = append(candidates, home+"/.var/app/"+appID)
	}
	return existingSourceDirs(sourceClient, candidates)
}

// existingSourceDirs filters candidates to those that exist as directories on
// the source host.
func existingSourceDirs(sourceClient *ssh.Client, candidates []string) []string {
	if len(candidates) == 0 {
		return nil
	}
	var b strings.Builder
	for _, p := range candidates {
		b.WriteString("[ -d " + shellSingleQuote(p) + " ] && echo " + shellSingleQuote(p) + "; ")
	}
	out, err := runClientCmd(sourceClient, b.String()+"true")
	if err != nil {
		return nil
	}
	var res []string
	for _, line := range strings.Split(out, "\n") {
		if s := strings.TrimSpace(line); s != "" {
			res = append(res, s)
		}
	}
	return res
}

// migrateDataDirs copies each source directory to the target (preserving its
// absolute path). Every step is best-effort: a failure is logged but does not
// fail the migration, since the software itself is already installed.
func migrateDataDirs(sourceClient, targetClient *ssh.Client, dirs []string, executionID, label string,
	migrationLogger *Logger) {
	for _, dir := range dirs {
		migrationLogger.Printf(INFO, "Copying %s data directory: %s\n", label, dir)
		if err := copyDirectoryWithChunks(sourceClient, targetClient, dir, executionID, migrationLogger); err != nil {
			migrationLogger.Printf(WARN, "failed to copy %s data %s (non-fatal): %v\n", label, dir, err)
		}
	}
}
