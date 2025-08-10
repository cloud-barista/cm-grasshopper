package software

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
)

// MigrateRepositoryConfiguration migrates repository configuration from source to target
func MigrateRepositoryConfiguration(sourceClient *ssh.Client, targetClient *ssh.Client, uuid string, migrationLogger *Logger) error {
	migrationLogger.Printf(INFO, "Starting repository configuration migration\n")

	// Step 1: Collect from source with specified directory
	sourceTmpDir := fmt.Sprintf("/tmp/system_migration_%d", time.Now().Unix())
	migrationLogger.Printf(INFO, "Collecting repository configuration from source system\n")
	output, err := executeScript(sourceClient, migrationLogger, "system_migrate.sh", "collect", sourceTmpDir)
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to collect repository configuration: %v\n", err)
		migrationLogger.Printf(DEBUG, "Collection output: %s\n", string(output))
		return fmt.Errorf("repository configuration collection failed: %v", err)
	}
	migrationLogger.Printf(INFO, "Repository configuration collection completed\n")

	// Step 2: Copy to cm-grasshopper intermediate storage
	localBackupDir := fmt.Sprintf("./software_temp/%s/repo_backup", uuid)
	if err := copyFromRemoteToLocal(sourceClient, sourceTmpDir, localBackupDir, migrationLogger); err != nil {
		return fmt.Errorf("failed to copy repository config to local storage: %v", err)
	}

	// Step 3: Copy from cm-grasshopper to target
	targetTmpDir := fmt.Sprintf("/tmp/system_migration_deploy_%d", time.Now().Unix())
	if err := copyFromLocalToRemote(targetClient, localBackupDir, targetTmpDir, migrationLogger); err != nil {
		return fmt.Errorf("failed to copy repository config to target: %v", err)
	}

	// Step 4: Deploy on target
	migrationLogger.Printf(INFO, "Deploying repository configuration to target system\n")
	output, err = executeScript(targetClient, migrationLogger, "system_migrate.sh", "deploy", targetTmpDir)
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to deploy repository configuration: %v\n", err)
		migrationLogger.Printf(DEBUG, "Deployment output: %s\n", string(output))
		return fmt.Errorf("repository configuration deployment failed: %v", err)
	}

	migrationLogger.Printf(INFO, "Repository configuration deployment completed\n")

	// Step 5: Update package repositories after migration
	if err := updatePackageRepositories(targetClient, migrationLogger); err != nil {
		migrationLogger.Printf(WARN, "Repository update failed but migration was successful: %v\n", err)
	}

	// Step 6: Cleanup
	cleanupRemoteDir(sourceClient, sourceTmpDir, migrationLogger)
	cleanupRemoteDir(targetClient, targetTmpDir, migrationLogger)

	migrationLogger.Printf(INFO, "Repository configuration migration completed successfully\n")
	return nil
}

func updatePackageRepositories(client *ssh.Client, migrationLogger *Logger) error {
	migrationLogger.Printf(INFO, "Updating package repositories\n")

	// Use existing getSystemType function
	systemType, err := getSystemType(client, migrationLogger)
	if err != nil {
		return fmt.Errorf("failed to detect system type: %v", err)
	}

	var updateCmd string
	switch systemType {
	case Debian:
		updateCmd = "apt update"
		migrationLogger.Printf(INFO, "Using APT package manager for Debian-based system\n")
	case RHEL:
		// Check if dnf is available first, fallback to yum
		if err := checkCommandExists(client, "dnf"); err == nil {
			updateCmd = "dnf clean all && dnf makecache"
			migrationLogger.Printf(INFO, "Using DNF package manager for RHEL-based system\n")
		} else {
			updateCmd = "yum clean all && yum makecache"
			migrationLogger.Printf(INFO, "Using YUM package manager for RHEL-based system\n")
		}
	default:
		return fmt.Errorf("unsupported system type: %v", systemType)
	}

	migrationLogger.Printf(INFO, "Executing repository update command: %s\n", updateCmd)

	session, err := client.NewSessionWithRetry()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	wrappedCmd := sudoWrapper(updateCmd, client.SSHTarget.Password)
	output, err := session.CombinedOutput(wrappedCmd)

	migrationLogger.Printf(DEBUG, "Repository update output: %s\n", string(output))

	if err != nil {
		migrationLogger.Printf(ERROR, "Repository update failed: %v\n", err)
		return fmt.Errorf("repository update failed: %v", err)
	}

	migrationLogger.Printf(INFO, "Package repositories updated successfully\n")
	return nil
}


func checkCommandExists(client *ssh.Client, command string) error {
	session, err := client.NewSessionWithRetry()
	if err != nil {
		return err
	}
	defer func() {
		_ = session.Close()
	}()

	cmd := fmt.Sprintf("command -v %s", command)
	_, err = session.CombinedOutput(cmd)
	return err
}

// Helper functions for file transfer operations

func copyFromRemoteToLocal(client *ssh.Client, remotePath, localPath string, migrationLogger *Logger) error {
	migrationLogger.Printf(INFO, "Copying from remote %s to local %s\n", remotePath, localPath)

	// Create local directory
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create local directory: %v", err)
	}

	// Create tar archive on remote
	session, err := client.NewSessionWithRetry()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	// Create tar archive and stream it
	tarCmd := fmt.Sprintf("cd %s && tar -czf - .", remotePath)
	wrappedCmd := sudoWrapper(tarCmd, client.SSHTarget.Password)

	output, err := session.CombinedOutput(wrappedCmd)
	if err != nil {
		migrationLogger.Printf(ERROR, "Failed to create tar archive: %v\n", err)
		return fmt.Errorf("tar archive creation failed: %v", err)
	}

	// Save tar content to local file
	tarFile := filepath.Join(localPath, "backup.tar.gz")
	if err := os.WriteFile(tarFile, output, 0644); err != nil {
		return fmt.Errorf("failed to write tar file: %v", err)
	}

	migrationLogger.Printf(INFO, "Successfully copied remote data to local storage\n")
	return nil
}

func copyFromLocalToRemote(client *ssh.Client, localPath, remotePath string, migrationLogger *Logger) error {
	migrationLogger.Printf(INFO, "Copying from local %s to remote %s\n", localPath, remotePath)

	tarFile := filepath.Join(localPath, "backup.tar.gz")

	// Read local tar file
	tarData, err := os.ReadFile(tarFile)
	if err != nil {
		return fmt.Errorf("failed to read local tar file: %v", err)
	}

	// Create remote directory
	session, err := client.NewSessionWithRetry()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer func() {
		_ = session.Close()
	}()

	mkdirCmd := fmt.Sprintf("mkdir -p %s", remotePath)
	wrappedCmd := sudoWrapper(mkdirCmd, client.SSHTarget.Password)
	_, err = session.CombinedOutput(wrappedCmd)
	if err != nil {
		return fmt.Errorf("failed to create remote directory: %v", err)
	}

	// Transfer and extract tar file
	session2, err := client.NewSessionWithRetry()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %v", err)
	}
	defer func() {
		_ = session2.Close()
	}()

	// Create a script that reads stdin and extracts it
	extractScript := fmt.Sprintf(`
		cat > %s/backup.tar.gz
		cd %s && tar -xzf backup.tar.gz
		rm %s/backup.tar.gz
	`, remotePath, remotePath, remotePath)

	wrappedExtractCmd := sudoWrapper(fmt.Sprintf("sh -c '%s'", extractScript), client.SSHTarget.Password)

	stdin, err := session2.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	if err := session2.Start(wrappedExtractCmd); err != nil {
		return fmt.Errorf("failed to start extract command: %v", err)
	}

	// Write tar data to stdin
	if _, err := stdin.Write(tarData); err != nil {
		return fmt.Errorf("failed to write tar data: %v", err)
	}
	_ = stdin.Close()

	if err := session2.Wait(); err != nil {
		return fmt.Errorf("extract command failed: %v", err)
	}

	migrationLogger.Printf(INFO, "Successfully copied local data to remote storage\n")
	return nil
}

func cleanupRemoteDir(client *ssh.Client, remotePath string, migrationLogger *Logger) {
	if remotePath == "" {
		return
	}

	migrationLogger.Printf(DEBUG, "Cleaning up remote directory: %s\n", remotePath)

	session, err := client.NewSessionWithRetry()
	if err != nil {
		migrationLogger.Printf(WARN, "Failed to create SSH session for cleanup: %v\n", err)
		return
	}
	defer func() {
		_ = session.Close()
	}()

	cleanupCmd := fmt.Sprintf("rm -rf %s", remotePath)
	wrappedCmd := sudoWrapper(cleanupCmd, client.SSHTarget.Password)
	_, err = session.CombinedOutput(wrappedCmd)
	if err != nil {
		migrationLogger.Printf(WARN, "Failed to cleanup remote directory %s: %v\n", remotePath, err)
	} else {
		migrationLogger.Printf(DEBUG, "Successfully cleaned up remote directory: %s\n", remotePath)
	}
}