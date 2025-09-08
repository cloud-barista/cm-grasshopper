package software

import (
	"fmt"

	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	softwaremodel "github.com/cloud-barista/cm-model/sw"
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
		migrationLogger.Printf(ERROR, "Podman installation is not supported.\n")
		return fmt.Errorf("podman installation is not supported")
	}

	migrationLogger.Println(DEBUG, output)
	return nil
}
