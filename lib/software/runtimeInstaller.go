package software

import (
	"fmt"

	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
)

func runtimeInstaller(client *ssh.Client, runtime softwaremodel.SoftwareContainerRuntimeType, migrationLogger *Logger) error {
	migrationLogger.Printf(INFO, "Starting runtime installation... (Runtime: %s)\n", runtime)

	var output []byte
	var err error

	switch runtime {
	case softwaremodel.SoftwareContainerRuntimeTypeDocker:
		migrationLogger.Printf(INFO, "Installing Docker...\n")
		output, err = executeScript(client, migrationLogger, "install_docker.sh")
		if err != nil && len(output) == 0 {
			migrationLogger.Printf(ERROR, "Failed to execute docker installation script: %v\n", err)
			return fmt.Errorf("docker installation failed: %v", err)
		}
	case softwaremodel.SoftwareContainerRuntimeTypePodman:
		migrationLogger.Printf(INFO, "Installing Podman...\n")
		output, err = executeScript(client, migrationLogger, "install_podman.sh")
		if err != nil && len(output) == 0 {
			migrationLogger.Printf(ERROR, "Failed to execute podman installation script: %v\n", err)
			return fmt.Errorf("podman installation failed: %v", err)
		}
	}

	migrationLogger.Println(DEBUG, output)
	return nil
}
