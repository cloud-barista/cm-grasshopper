package software

import (
	"fmt"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
)

// shellSingleQuote single-quotes an argument for safe shell use.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// runTargetCmd runs a sudo-wrapped command on the target host and returns its
// combined output.
func runTargetCmd(targetClient *ssh.Client, cmd string) (string, error) {
	session, err := targetClient.NewSessionWithRetry()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer func() { _ = session.Close() }()

	out, err := session.CombinedOutput(sudoWrapper(cmd, targetClient.SSHTarget.Password))
	return string(out), err
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
		installCmd = "DEBIAN_FRONTEND=noninteractive apt-get install -y " + pkg
	case RHEL:
		if err := checkCommandExists(targetClient, "dnf"); err == nil {
			installCmd = "dnf install -y " + pkg
		} else {
			installCmd = "yum install -y " + pkg
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

// snapMigrator installs a snap package on the target (installing snapd first if
// needed). Classic-confined snaps are retried with --classic.
func snapMigrator(targetClient *ssh.Client, pkg *softwaremodel.PackageMigrationInfo, migrationLogger *Logger) error {
	if err := ensureTargetCommand(targetClient, "snap", "snapd", migrationLogger); err != nil {
		return err
	}
	// Make sure snapd is running; on RHEL the /snap symlink is also required.
	_, _ = runTargetCmd(targetClient, "systemctl enable --now snapd.socket 2>/dev/null; "+
		"[ -e /snap ] || ln -s /var/lib/snapd/snap /snap 2>/dev/null; true")

	installCmd := "snap install " + shellSingleQuote(pkg.Name)
	if channel := strings.TrimSpace(pkg.Channel); channel != "" {
		installCmd += " --channel=" + shellSingleQuote(channel)
	}

	migrationLogger.Printf(INFO, "Installing snap: %s\n", pkg.Name)
	out, err := runTargetCmd(targetClient, installCmd)
	if err != nil && strings.Contains(strings.ToLower(out), "classic") {
		migrationLogger.Printf(INFO, "snap %s requires classic confinement, retrying with --classic\n", pkg.Name)
		out, err = runTargetCmd(targetClient, installCmd+" --classic")
	}
	migrationLogger.Printf(DEBUG, "snap install output: %s\n", out)
	if err != nil {
		return fmt.Errorf("snap install failed for %s: %s", pkg.Name, out)
	}
	return nil
}

// flatpakMigrator installs a flatpak application on the target (installing
// flatpak and the flathub remote first if needed).
func flatpakMigrator(targetClient *ssh.Client, pkg *softwaremodel.PackageMigrationInfo, migrationLogger *Logger) error {
	if err := ensureTargetCommand(targetClient, "flatpak", "flatpak", migrationLogger); err != nil {
		return err
	}

	origin := strings.TrimSpace(pkg.Origin)
	if origin == "" || origin == "flathub" {
		origin = "flathub"
		// Flathub is the common default remote; other remotes are assumed present.
		_, _ = runTargetCmd(targetClient,
			"flatpak remote-add --if-not-exists flathub https://flathub.org/repo/flathub.flatpakrepo")
	}

	appID := strings.TrimSpace(pkg.ApplicationID)
	if appID == "" {
		appID = pkg.Name
	}

	migrationLogger.Printf(INFO, "Installing flatpak: %s (from %s)\n", appID, origin)
	out, err := runTargetCmd(targetClient,
		"flatpak install -y --noninteractive "+shellSingleQuote(origin)+" "+shellSingleQuote(appID))
	migrationLogger.Printf(DEBUG, "flatpak install output: %s\n", out)
	if err != nil {
		return fmt.Errorf("flatpak install failed for %s: %s", appID, out)
	}
	return nil
}
