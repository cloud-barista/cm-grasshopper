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
	ExecutionID            string
	MigrationList          *softwaremodel.MigrationList
	MigrationStatusList    []model.SoftwareMigrationStatus
	Target                 model.Target
	SourceConnectionInfoID string
	SourceClient           *ssh.Client
	TargetClient           *ssh.Client
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
				NamespaceID:            nsId,
				MCIID:                  mciId,
				VMID:                   vmId,
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
				NamespaceID:            nsId,
				MCIID:                  mciId,
				VMID:                   vmId,
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
			ExecutionID:            executionID,
			MigrationList:          &server.MigrationList,
			MigrationStatusList:    softwareMigrationStatusList,
			SourceConnectionInfoID: server.SourceConnectionInfoID,
			Target:                 *target,
			SourceClient:           sourceClient,
			TargetClient:           targetClient,
		})

		targetMappings = append(targetMappings, model.TargetMapping{
			SourceConnectionInfoID: server.SourceConnectionInfoID,
			Target:                 *target,
			Status:                 "ready",
		})
	}

	_, err = dao.ExecutionStatusCreate(&model.ExecutionStatus{
		ExecutionID:    executionID,
		TargetMappings: targetMappings,
		StartedAt:      time.Now(),
		FinishedAt:     time.Time{},
	})
	if err != nil {
		logger.Println(logger.ERROR, true, "Failed to create execution status ("+
			"ExecutionID: "+executionID+")")
		return nil, nil, err
	}

	return exList, targetMappings, nil
}

func MigrateSoftware(execution *Execution) {
	exStatus := "finished"
	var idx int

	executionStatus, err := dao.ExecutionStatusGet(execution.ExecutionID)
	if err != nil {
		logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
			", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
			", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
			", Error=failed to initialize executionStatus: "+err.Error())
		return
	}

	for i, target := range executionStatus.TargetMappings {
		if target.Target.NamespaceID == execution.Target.NamespaceID &&
			target.Target.MCIID == execution.Target.MCIID &&
			target.Target.VMID == execution.Target.VMID &&
			target.SourceConnectionInfoID == execution.SourceConnectionInfoID {
			idx = i

			executionStatus.TargetMappings[idx].Status = "running"

			err := dao.ExecutionStatusUpdate(executionStatus)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
					", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
					", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
					", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
					", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
					", Error=failed to update execution status: "+err.Error())
				return
			}

			break
		}
	}

	defer func() {
		_ = execution.SourceClient.Close()
		_ = execution.TargetClient.Close()
		executionStatus.TargetMappings[idx].Status = exStatus
		executionStatus.FinishedAt = time.Now()
		err := dao.ExecutionStatusUpdate(executionStatus)
		if err != nil {
			logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
				", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
				", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
				", Error=failed to update execution status: "+err.Error())
		}
	}()

	// Cache system types once at the beginning
	var sourceSystemType, targetSystemType SystemType
	migrationLogger, loggerErr := initLoggerWithUUID(execution.ExecutionID)
	if loggerErr != nil {
		logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
			", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
			", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
			", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
			", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
			", Error=failed to initialize logger: "+loggerErr.Error())
		exStatus = "failed"
		return
	}
	defer migrationLogger.Close()

	sourceSystemType, err = getSystemType(execution.SourceClient, migrationLogger)
	if err != nil {
		logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
			", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
			", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
			", Error=failed to detect source system type: "+err.Error())
		exStatus = "failed"
		return
	}

	targetSystemType, err = getSystemType(execution.TargetClient, migrationLogger)
	if err != nil {
		logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
			", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
			", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
			", Error=failed to detect target system type: "+err.Error())
		exStatus = "failed"
		return
	}

	if sourceSystemType != targetSystemType {
		logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
			", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
			", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
			", Error=system type mismatch")
		exStatus = "failed"
		return
	}

	var updateSoftwareInstallStatus = func(order int, status string, errMsg string, updateStartedTime bool) {
		if status == "failed" {
			exStatus = "finished with error"
		}

		var softwareMigrationStatus model.SoftwareMigrationStatus

		for _, sms := range execution.MigrationStatusList {
			if sms.Order == order {
				softwareMigrationStatus = sms
				break
			}
		}

		softwareMigrationStatus.Status = status
		softwareMigrationStatus.ErrorMessage = errMsg
		now := time.Now()
		if updateStartedTime {
			softwareMigrationStatus.StartedAt = now
		}
		softwareMigrationStatus.UpdatedAt = now

		err := dao.SoftwareMigrationStatusUpdate(&softwareMigrationStatus)
		if err != nil {
			logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
				", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
				", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
				", Error="+err.Error())
		}
	}

	var dockerRuntimeChecked bool
	var podmanRuntimeChecked bool

	for _, ex := range execution.MigrationList.Containers {
		if ex.Runtime == string(softwaremodel.SoftwareContainerRuntimeTypeDocker) && !dockerRuntimeChecked {
			err = runtimeInstaller(execution.TargetClient, softwaremodel.SoftwareContainerRuntimeTypeDocker, migrationLogger)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
					", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
					", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
					", InstallType=container, Error="+err.Error())
			}
			dockerRuntimeChecked = true
		} else if ex.Runtime == string(softwaremodel.SoftwareContainerRuntimeTypePodman) && !podmanRuntimeChecked {
			err = runtimeInstaller(execution.TargetClient, softwaremodel.SoftwareContainerRuntimeTypePodman, migrationLogger)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
					", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
					", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
					", InstallType=container, Error="+err.Error())
			}
			podmanRuntimeChecked = true
		}
	}

	for _, ex := range execution.MigrationList.Containers {
		updateSoftwareInstallStatus(ex.Order, "installing", "", true)

		logger.Println(logger.DEBUG, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
			", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
			", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
			", InstallType=container, ContainerName="+ex.Name)

		// 1. image pull
		imageRef := fmt.Sprintf("%s:%s", ex.ContainerImage.ImageName, ex.ContainerImage.ImageVersion)
		pulCmd := fmt.Sprintf("docker pull %s", imageRef)
		pullCmd := sudoWrapper(pulCmd, execution.TargetClient.SSHTarget.Password)
		if _, err := execution.TargetClient.Run(pullCmd); err != nil {
			updateSoftwareInstallStatus(ex.Order, "failed", err.Error(), false)
			continue
		}

		// 2. port validation
		if err := listenPortsValidator(execution.SourceClient, execution.TargetClient, ex.Name, migrationLogger); err != nil {
			updateSoftwareInstallStatus(ex.Order, "failed", err.Error(), false)
			continue
		}

		// 3. copy docker compose
		if ex.DockerComposePath != "" {
			baseDir := filepath.Dir(ex.DockerComposePath) // => /home/ubuntu
			dockerComposeFile := filepath.Base(ex.DockerComposePath)

			logger.Println(logger.INFO, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
				", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
				", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
				"ExecutionID="+execution.ExecutionID+", Info=Copying full directory: "+baseDir)

			if err := copyDirectoryWithChunks(execution.SourceClient, execution.TargetClient, baseDir, execution.ExecutionID, migrationLogger); err != nil {
				updateSoftwareInstallStatus(ex.Order, "failed", err.Error(), false)
				continue
			}

			// 3-1 volume
			for _, mount := range ex.MountPaths {
				hostPath := strings.Split(mount, ":")[0]
				if err := copyDirectoryWithChunks(execution.SourceClient, execution.TargetClient, hostPath, execution.ExecutionID, migrationLogger); err != nil {
					updateSoftwareInstallStatus(ex.Order, "failed", "failed to copy volume "+hostPath+": "+err.Error(), false)
					continue
				}
			}

			// 3-2 create compose network
			if ex.NetworkMode != "" && ex.NetworkMode != "bridge" {
				rmCmd := fmt.Sprintf("docker network rm %s || true", ex.NetworkMode)
				rmNetCmd := sudoWrapper(rmCmd, execution.TargetClient.SSHTarget.Password)
				_, _ = execution.TargetClient.Run(rmNetCmd)
			}

			copCmd := fmt.Sprintf("cd %s && docker compose -f "+dockerComposeFile+" up -d", baseDir)
			composeCmd := sudoWrapper(copCmd, execution.TargetClient.SSHTarget.Password)

			if _, err := execution.TargetClient.Run(composeCmd); err != nil {
				updateSoftwareInstallStatus(ex.Order, "failed", err.Error(), false)
			} else {
				updateSoftwareInstallStatus(ex.Order, "finished", "", false)
			}

			continue
		}

		// 4. generate network
		if ex.DockerComposePath == "" {
			if ex.NetworkMode != "" && ex.NetworkMode != "bridge" {
				checkCmd := fmt.Sprintf("docker network ls --format '{{.Name}}' | grep -w %s", ex.NetworkMode)
				checkNetCmd := sudoWrapper(checkCmd, execution.TargetClient.SSHTarget.Password)

				if _, err := execution.TargetClient.Run(checkNetCmd); err != nil {
					createCmd := fmt.Sprintf("docker network create %s", ex.NetworkMode)
					createNetCmd := sudoWrapper(createCmd, execution.TargetClient.SSHTarget.Password)

					if _, cerr := execution.TargetClient.Run(createNetCmd); cerr != nil {
						updateSoftwareInstallStatus(ex.Order, "failed", cerr.Error(), false)
						continue
					}
				}
			}
		}

		// 5. docker run
		dockerRunCmd := fmt.Sprintf("docker run -d --name %s", ex.Name)
		runCmd := sudoWrapper(dockerRunCmd, execution.TargetClient.SSHTarget.Password)

		validatedPorts := make(map[int]bool)
		for _, port := range ex.ContainerPorts {
			if port.HostPort > 0 && !validatedPorts[port.HostPort] {
				runCmd += fmt.Sprintf(" -p %d:%d/%s", port.HostPort, port.ContainerPort, port.Protocol)
				validatedPorts[port.HostPort] = true
			}
		}

		for _, mount := range ex.MountPaths {
			hostPath := strings.Split(mount, ":")[0]

			if err := copyDirectoryWithChunks(execution.SourceClient, execution.TargetClient, hostPath, execution.ExecutionID, migrationLogger); err != nil {
				updateSoftwareInstallStatus(ex.Order, "failed", "failed to copy mount path "+hostPath+": "+err.Error(), false)
				continue
			}

			runCmd += fmt.Sprintf(" -v %s", mount)
		}

		for _, env := range ex.Envs {
			runCmd += fmt.Sprintf(" -e %s=%s", env.Name, env.Value)
		}
		if ex.NetworkMode != "" {
			runCmd += fmt.Sprintf(" --network %s", ex.NetworkMode)
		}
		if ex.RestartPolicy != "" {
			runCmd += fmt.Sprintf(" --restart %s", ex.RestartPolicy)
		}
		runCmd += " " + imageRef

		if _, err := execution.TargetClient.Run(runCmd); err != nil {
			updateSoftwareInstallStatus(ex.Order, "failed", err.Error(), false)
			continue
		}

		updateSoftwareInstallStatus(ex.Order, "finished", "", false)
	}

	if len(execution.MigrationList.Packages) > 0 {
		logger.Println(logger.INFO, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
			", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
			", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
			"migrateSoftware: Starting repository and GPG keys migration")

		// Migrate repository configuration
		if err := MigrateRepositoryConfiguration(execution.SourceClient, execution.TargetClient, execution.ExecutionID, migrationLogger); err != nil {
			logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
				", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
				", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
				"migrateSoftware: Repository migration failed: "+err.Error())
			// Continue with package migration even if repo migration fails
		} else {
			logger.Println(logger.INFO, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
				", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
				", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
				"migrateSoftware: Repository migration completed successfully")
		}
	}

	for _, ex := range execution.MigrationList.Packages {
		updateSoftwareInstallStatus(ex.Order, "installing", "", true)

		err := runPlaybook(execution.ExecutionID, "package", ex.Name, execution.TargetClient.SSHTarget)
		if err != nil {
			logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
				", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
				", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
				", InstallType=package, Error="+err.Error())
			updateSoftwareInstallStatus(ex.Order, "failed", err.Error(), false)
			continue
		}

		err = configCopier(execution.SourceClient, execution.TargetClient, ex.Name, execution.ExecutionID, migrationLogger)
		if err != nil {
			logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
				", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
				", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
				", InstallType=package, Error="+err.Error())
			updateSoftwareInstallStatus(ex.Order, "failed", err.Error(), false)
			continue
		}

		err = serviceMigrator(execution.SourceClient, execution.TargetClient, ex.Name, execution.ExecutionID, migrationLogger)
		if err != nil {
			logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+execution.ExecutionID+
				", NS_ID="+execution.Target.NamespaceID+", MCI_ID="+execution.Target.MCIID+", VM_ID="+execution.Target.VMID+
				", SourceConnectionInfoID="+execution.SourceConnectionInfoID+
				", InstallType=package, Error="+err.Error())
			updateSoftwareInstallStatus(ex.Order, "failed", err.Error(), false)
			continue
		}

		updateSoftwareInstallStatus(ex.Order, "finished", "", false)
	}
}
