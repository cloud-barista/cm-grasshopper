package software

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	softwaremodel "github.com/cloud-barista/cm-model/sw"
	"github.com/jollaman999/utils/logger"
)

type Execution struct {
	ExecutionID         string
	MigrationList       *softwaremodel.MigrationList
	MigrationStatusList []model.SoftwareMigrationStatus
	SourceClient        *ssh.Client
	TargetClient        *ssh.Client
}

func getVMId(sourceConnectionInfoID, nsID, mciID string) (string, error) {
	data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress+
		":"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort+
		"/honeybee/connection_info/"+sourceConnectionInfoID, "", "")
	if err != nil {
		return "", err
	}

	var encryptedConnectionInfo honeybee.ConnectionInfo
	err = json.Unmarshal(data, &encryptedConnectionInfo)
	if err != nil {
		return "", err
	}

	data, err = common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress+
		":"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort+
		"/honeybee/source_group/"+encryptedConnectionInfo.SourceGroupID+
		"/connection_info/"+sourceConnectionInfoID+"/infra", "", "")
	if err != nil {
		return "", err
	}

	var infraInfo infra.Infra
	err = json.Unmarshal(data, &infraInfo)
	if err != nil {
		return "", err
	}

	machineid := infraInfo.Compute.OS.Node.Machineid

	data, err = common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerAddress+
		":"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerPort+
		"/tumblebug/ns/"+nsID+"/mci/"+mciID,
		config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Username, config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Password)
	if err != nil {
		return "", err
	}

	var mciInfo model.TBMCIInfo
	err = json.Unmarshal(data, &mciInfo)
	if err != nil {
		return "", err
	}

	for _, vm := range mciInfo.VM {
		// cm-beetle - v0.3.7
		// https://github.com/cloud-barista/cm-beetle/commit/94e030cba5cf2dcdc055240ec5e46d24666da06d
		if vm.Label["sourceMachineId"] == machineid {
			return vm.Id, nil
		}
	}

	return "", errors.New("can't find matched target vm")
}

func PrepareSoftwareMigration(executionID string, targetServers []softwaremodel.MigrationServer, nsId string, mciId string) ([]Execution, []model.TargetMapping, error) {
	var sourceClient *ssh.Client
	var targetClient *ssh.Client
	var target *model.Target
	var err error

	var exList = make([]Execution, 0)
	var targetMappings []model.TargetMapping

	for _, server := range targetServers {
		var softwareMigrationStatusList []model.SoftwareMigrationStatus

		sourceClient, err = ssh.NewSSHClient(ssh.ConnectionTypeSource, server.SourceConnectionInfoID, "", "")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect to source host: %v", err)
		}

		vmId, err := getVMId(server.SourceConnectionInfoID, nsId, mciId)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get VM ID: %v", err)
		}

		target = &model.Target{
			NamespaceID: nsId,
			MCIID:       mciId,
			VMID:        vmId,
		}

		targetClient, err = ssh.NewSSHClient(ssh.ConnectionTypeTarget, target.VMID, target.NamespaceID, target.MCIID)
		if err != nil {
			_ = sourceClient.Close()

			return nil, nil, fmt.Errorf("failed to connect to target host: %v", err)
		}

		for _, execution := range server.MigrationList.Packages {
			softwareMigrationStatus := model.SoftwareMigrationStatus{
				ExecutionID:            executionID,
				SourceConnectionInfoID: server.SourceConnectionInfoID,
				Target:                 *target,
				Order:                  execution.Order,
				SoftwareName:           execution.Name,
				SoftwareVersion:        execution.Version,
				SoftwareInstallType:    softwaremodel.SoftwareTypePackage,
				Status:                 "ready",
				StartedAt:              time.Time{},
				UpdatedAt:              time.Time{},
				ErrorMessage:           "",
			}
			softwareMigrationStatusList = append(softwareMigrationStatusList, softwareMigrationStatus)
			_, err = dao.SoftwareMigrationStatusCreate(&softwareMigrationStatus)
			if err != nil {
				logger.Println(logger.ERROR, true, "Failed to create software migration status ("+
					"SoftwareName: "+softwareMigrationStatus.SoftwareName+", "+
					"SoftwareVersion: "+softwareMigrationStatus.SoftwareVersion+", "+
					"SoftwareInstallType: "+string(softwareMigrationStatus.SoftwareInstallType)+")")
			}
		}

		for _, execution := range server.MigrationList.Containers {
			softwareMigrationStatus := model.SoftwareMigrationStatus{
				ExecutionID:            executionID,
				SourceConnectionInfoID: server.SourceConnectionInfoID,
				Target:                 *target,
				Order:                  execution.Order,
				SoftwareName:           execution.Name,
				SoftwareVersion:        execution.ContainerImage.ImageVersion,
				SoftwareInstallType:    softwaremodel.SoftwareTypeContainer,
				Status:                 "ready",
				StartedAt:              time.Time{},
				UpdatedAt:              time.Time{},
				ErrorMessage:           "",
			}
			softwareMigrationStatusList = append(softwareMigrationStatusList, softwareMigrationStatus)
			_, err = dao.SoftwareMigrationStatusCreate(&softwareMigrationStatus)
			if err != nil {
				logger.Println(logger.ERROR, true, "Failed to create software migration status ("+
					"SoftwareName: "+softwareMigrationStatus.SoftwareName+", "+
					"SoftwareVersion: "+softwareMigrationStatus.SoftwareVersion+", "+
					"SoftwareInstallType: "+string(softwareMigrationStatus.SoftwareInstallType)+")")
			}
		}

		// TODO - Kubernetes Migration

		exList = append(exList, Execution{
			ExecutionID:         executionID,
			MigrationList:       &server.MigrationList,
			MigrationStatusList: softwareMigrationStatusList,
			SourceClient:        sourceClient,
			TargetClient:        targetClient,
		})

		targetMappings = append(targetMappings, model.TargetMapping{
			SourceConnectionInfoID: server.SourceConnectionInfoID,
			Target:                 *target,
		})
	}

	return exList, targetMappings, nil
}

func MigrateSoftware(executionID string, executionList *softwaremodel.MigrationList,
	migrationStatusList []model.SoftwareMigrationStatus, sourceClient *ssh.Client, targetClient *ssh.Client) {
	defer func() {
		_ = sourceClient.Close()
		_ = targetClient.Close()
	}()

	// Cache system types once at the beginning
	var sourceSystemType, targetSystemType SystemType
	migrationLogger, loggerErr := initLoggerWithUUID(executionID)
	if loggerErr != nil {
		logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
			", Error=failed to initialize logger: "+loggerErr.Error())
		return
	}
	defer migrationLogger.Close()

	sourceSystemType, err := getSystemType(sourceClient, migrationLogger)
	if err != nil {
		logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
			", Error=failed to detect source system type: "+err.Error())
		return
	}

	targetSystemType, err = getSystemType(targetClient, migrationLogger)
	if err != nil {
		logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
			", Error=failed to detect target system type: "+err.Error())
		return
	}

	if sourceSystemType != targetSystemType {
		logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
			", Error=system type mismatch")
		return
	}

	var updateStatus = func(installType softwaremodel.SoftwareType, i int, status string, errMsg string, updateStartedTime bool) {
		migrationStatusList[i].Status = status
		migrationStatusList[i].ErrorMessage = errMsg
		now := time.Now()
		if updateStartedTime {
			migrationStatusList[i].StartedAt = now
		}
		migrationStatusList[i].UpdatedAt = now

		err := dao.SoftwareMigrationStatusUpdate(&migrationStatusList[i])
		if err != nil {
			logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
				", Error="+err.Error())
		}
	}

	var dockerRuntimeChecked bool
	var dockerInstallOk bool
	var podmanRuntimeChecked bool
	var podmanInstallOk bool

	for i, execution := range executionList.Containers {
		if execution.Runtime == string(softwaremodel.SoftwareContainerRuntimeTypeDocker) {
			if dockerRuntimeChecked {
				if !dockerInstallOk {
					continue
				}
			} else {
				err = runtimeInstaller(targetClient, softwaremodel.SoftwareContainerRuntimeTypeDocker, migrationLogger)
				if err != nil {
					logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
						", InstallType=container, Error="+err.Error())
					dockerInstallOk = false
				} else {
					dockerInstallOk = true
				}
				dockerRuntimeChecked = true
			}
		} else if execution.Runtime == string(softwaremodel.SoftwareContainerRuntimeTypePodman) {
			if podmanRuntimeChecked {
				if !podmanInstallOk {
					continue
				}
			} else {
				err = runtimeInstaller(targetClient, softwaremodel.SoftwareContainerRuntimeTypePodman, migrationLogger)
				if err != nil {
					logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
						", InstallType=container, Error="+err.Error())
					podmanInstallOk = false
				} else {
					podmanInstallOk = true
				}
				podmanRuntimeChecked = true
			}
		}

		updateStatus(softwaremodel.SoftwareTypeContainer, i, "installing", "", true)

		logger.Println(logger.DEBUG, true, "migrateSoftware: ExecutionID="+executionID+
			", InstallType=container, ContainerName="+execution.Name)

		for i, execution := range executionList.Containers {
			// 1. image pull
			imageRef := fmt.Sprintf("%s:%s", execution.ContainerImage.ImageName, execution.ContainerImage.ImageVersion)
			pulCmd := fmt.Sprintf("docker pull %s", imageRef)
			pullCmd := sudoWrapper(pulCmd, targetClient.SSHTarget.Password)
			if _, err := targetClient.Run(pullCmd); err != nil {
				updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", err.Error(), false)
				continue
			}

			// 2. port validation
			if err := listenPortsValidator(sourceClient, targetClient, execution.Name, migrationLogger); err != nil {
				updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", err.Error(), false)
				continue
			}

			// 3. copy docker compose
			if execution.DockerComposePath != "" {
				baseDir := filepath.Dir(execution.DockerComposePath) // => /home/ubuntu

				migrationLogger.Println(INFO, true,
					"ExecutionID="+executionID+", Info=Copying full directory: "+baseDir)

				if err := copyDirectoryWithChunks(sourceClient, targetClient, baseDir, executionID, migrationLogger); err != nil {
					updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", err.Error(), false)
					continue
				}

				// 3-1 volume
				for _, mount := range execution.MountPaths {
					hostPath := strings.Split(mount, ":")[0]
					if err := copyDirectoryWithChunks(sourceClient, targetClient, hostPath, executionID, migrationLogger); err != nil {
						updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", "failed to copy volume "+hostPath+": "+err.Error(), false)
						continue
					}
				}

				// 3-2 create compose network
				if execution.NetworkMode != "" && execution.NetworkMode != "bridge" {
					rmCmd := fmt.Sprintf("docker network rm %s || true", execution.NetworkMode)
					rmNetCmd := sudoWrapper(rmCmd, targetClient.SSHTarget.Password)
					_, _ = targetClient.Run(rmNetCmd)
				}

				copCmd := fmt.Sprintf("cd %s && docker compose -f docker-compose.yml up -d", baseDir)
				composeCmd := sudoWrapper(copCmd, targetClient.SSHTarget.Password)

				if _, err := targetClient.Run(composeCmd); err != nil {
					updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", err.Error(), false)
				} else {
					updateStatus(softwaremodel.SoftwareTypeContainer, i, "finished", "", true)
				}

				continue
			}

			// 4. generate network
			if execution.DockerComposePath == "" {
				if execution.NetworkMode != "" && execution.NetworkMode != "bridge" {
					checkCmd := fmt.Sprintf("docker network ls --format '{{.Name}}' | grep -w %s", execution.NetworkMode)
					checkNetCmd := sudoWrapper(checkCmd, targetClient.SSHTarget.Password)

					if _, err := targetClient.Run(checkNetCmd); err != nil {
						createCmd := fmt.Sprintf("docker network create %s", execution.NetworkMode)
						createNetCmd := sudoWrapper(createCmd, targetClient.SSHTarget.Password)

						if _, cerr := targetClient.Run(createNetCmd); cerr != nil {
							updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", cerr.Error(), false)
							continue
						}
					}
				}
			}

			// 5. docker run
			dockerRunCmd := fmt.Sprintf("docker run -d --name %s", execution.Name)
			runCmd := sudoWrapper(dockerRunCmd, targetClient.SSHTarget.Password)

			validatedPorts := make(map[int]bool)
			for _, port := range execution.ContainerPorts {
				if port.HostPort > 0 && !validatedPorts[port.HostPort] {
					runCmd += fmt.Sprintf(" -p %d:%d/%s", port.HostPort, port.ContainerPort, port.Protocol)
					validatedPorts[port.HostPort] = true
				}
			}

			for _, mount := range execution.MountPaths {
				hostPath := strings.Split(mount, ":")[0]

				if err := copyDirectoryWithChunks(sourceClient, targetClient, hostPath, executionID, migrationLogger); err != nil {
					updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", "failed to copy mount path "+hostPath+": "+err.Error(), false)
					continue
				}

				runCmd += fmt.Sprintf(" -v %s", mount)
			}

			for _, env := range execution.Envs {
				runCmd += fmt.Sprintf(" -e %s=%s", env.Name, env.Value)
			}
			if execution.NetworkMode != "" {
				runCmd += fmt.Sprintf(" --network %s", execution.NetworkMode)
			}
			if execution.RestartPolicy != "" {
				runCmd += fmt.Sprintf(" --restart %s", execution.RestartPolicy)
			}
			runCmd += " " + imageRef

			if _, err := targetClient.Run(runCmd); err != nil {
				updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", err.Error(), false)
				continue
			}

			updateStatus(softwaremodel.SoftwareTypeContainer, i, "finished", "", true)

		}

		if len(executionList.Packages) > 0 {
			logger.Println(logger.INFO, true, "migrateSoftware: Starting repository and GPG keys migration")

			// Migrate repository configuration
			if err := MigrateRepositoryConfiguration(sourceClient, targetClient, executionID, migrationLogger); err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: Repository migration failed: "+err.Error())
				// Continue with package migration even if repo migration fails
			} else {
				logger.Println(logger.INFO, true, "migrateSoftware: Repository migration completed successfully")
			}
		}

		for i, execution := range executionList.Packages {
			updateStatus(softwaremodel.SoftwareTypePackage, i, "installing", "", true)

			err := runPlaybook(executionID, "package", execution.Name, targetClient.SSHTarget)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", InstallType=package, Error="+err.Error())
				updateStatus(softwaremodel.SoftwareTypePackage, i, "failed", err.Error(), false)

				return
			}

			err = configCopier(sourceClient, targetClient, execution.Name, executionID, migrationLogger)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", InstallType=package, Error="+err.Error())
				updateStatus(softwaremodel.SoftwareTypePackage, i, "failed", err.Error(), false)
			}

			err = serviceMigrator(sourceClient, targetClient, execution.Name, executionID, migrationLogger)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", InstallType=package, Error="+err.Error())
				updateStatus(softwaremodel.SoftwareTypePackage, i, "failed", err.Error(), false)
			}

			updateStatus(softwaremodel.SoftwareTypePackage, i, "finished", "", true)
		}
	}
}
