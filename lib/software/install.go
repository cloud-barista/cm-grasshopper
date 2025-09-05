package software

import (
	"encoding/json"
	"errors"
	"fmt"
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

func PrepareSoftwareMigration(executionID string, executionList *softwaremodel.MigrationList,
	sourceConnectionInfoID string, nsId string, mciId string) (executionStatusList []model.ExecutionStatus,
	sourceClient *ssh.Client, targetClient *ssh.Client, target *model.Target, err error) {
	sourceClient, err = ssh.NewSSHClient(ssh.ConnectionTypeSource, sourceConnectionInfoID, "", "")
	if err != nil {
		return nil, nil, nil, nil,
			fmt.Errorf("failed to connect to source host: %v", err)
	}

	vmId, err := getVMId(sourceConnectionInfoID, nsId, mciId)
	if err != nil {
		return nil, nil, nil, nil,
			fmt.Errorf("failed to get VM ID: %v", err)
	}

	target = &model.Target{
		NamespaceID: nsId,
		MCIID:       mciId,
		VMID:        vmId,
	}

	targetClient, err = ssh.NewSSHClient(ssh.ConnectionTypeTarget, target.VMID, target.NamespaceID, target.MCIID)
	if err != nil {
		_ = sourceClient.Close()

		return nil, nil, nil, nil,
			fmt.Errorf("failed to connect to target host: %v", err)
	}

	for _, execution := range executionList.Packages {
		executionStatusList = append(executionStatusList, model.ExecutionStatus{
			Order:               execution.Order,
			SoftwareName:        execution.Name,
			SoftwareVersion:     execution.Version,
			SoftwareInstallType: softwaremodel.SoftwareTypePackage,
			Status:              "ready",
			StartedAt:           time.Time{},
			ErrorMessage:        "",
		})
	}

	for _, execution := range executionList.Containers {
		executionStatusList = append(executionStatusList, model.ExecutionStatus{
			Order:               execution.Order,
			SoftwareName:        execution.Name,
			SoftwareVersion:     execution.ContainerImage.ImageVersion,
			SoftwareInstallType: softwaremodel.SoftwareTypeContainer,
			Status:              "ready",
			StartedAt:           time.Time{},
			ErrorMessage:        "",
		})
	}

	_, err = dao.SoftwareInstallStatusCreate(&model.SoftwareInstallStatus{
		ExecutionID:     executionID,
		Target:          *target,
		ExecutionStatus: executionStatusList,
	})
	if err != nil {
		_ = sourceClient.Close()
		_ = targetClient.Close()

		return nil, nil, nil, nil, err
	}

	return executionStatusList, sourceClient, targetClient, target, nil
}

func MigrateSoftware(executionID string, executionList *softwaremodel.MigrationList,
	executionStatusList []model.ExecutionStatus, sourceClient *ssh.Client, targetClient *ssh.Client) {
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
		executionStatusList[i].Status = status
		executionStatusList[i].ErrorMessage = errMsg
		now := time.Now()
		if updateStartedTime {
			executionStatusList[i].StartedAt = now
		}
		executionStatusList[i].UpdatedAt = now

		err := dao.SoftwareInstallStatusUpdate(&model.SoftwareInstallStatus{
			SoftwareInstallType: installType,
			ExecutionID:         executionID,
			ExecutionStatus:     executionStatusList,
		})
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

		// TODO: Container Migration Process
		// 1. docker images pull
		imageRef := fmt.Sprintf("%s:%s",
			execution.ContainerImage.ImageName,
			execution.ContainerImage.ImageVersion)
		pullCmd := fmt.Sprintf("sudo docker pull %s", imageRef)
		if _, err := targetClient.Run(pullCmd); err != nil {
			logger.Println(logger.ERROR, true,
				"migrateSoftware: ExecutionID="+executionID+
					", InstallType=container, ContainerName="+execution.Name+
					", Error=failed to pull image: "+err.Error())
			updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", err.Error(), false)
			continue
		}

		// 2. generated network
		if execution.NetworkMode != "" && execution.NetworkMode != "bridge" {
			checkNetCmd := fmt.Sprintf("sudo docker network ls --format '{{.Name}}' | grep -w %s", execution.NetworkMode)
			if _, err := targetClient.Run(checkNetCmd); err != nil {
				createNetCmd := fmt.Sprintf("sudo docker network create %s", execution.NetworkMode)
				if _, cerr := targetClient.Run(createNetCmd); cerr != nil {
					logger.Println(logger.ERROR, true,
						"migrateSoftware: ExecutionID="+executionID+
							", Container="+execution.Name+
							", Error=failed to create network "+execution.NetworkMode+": "+cerr.Error())
					updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", cerr.Error(), false)
					continue
				}
				logger.Println(logger.INFO, true,
					"migrateSoftware: ExecutionID="+executionID+
						", Container="+execution.Name+
						", Info=Created missing network "+execution.NetworkMode+" ✅")
			} else {
				logger.Println(logger.INFO, true,
					"migrateSoftware: ExecutionID="+executionID+
						", Container="+execution.Name+
						", Info=Network "+execution.NetworkMode+" already exists, reusing")
			}
		}

		// 3. generated run cmd
		runCmd := fmt.Sprintf("sudo docker run -d --name %s", execution.Name)

		// TODO: Container Listen Ports Validation
		validatedPorts := make(map[int]bool)
		for _, port := range execution.ContainerPorts {
			if port.HostPort > 0 {
				validateCmd := fmt.Sprintf("nc -z 127.0.0.1 %d", port.HostPort)
				_, err = targetClient.Run(validateCmd)
				if err != nil {
					if !validatedPorts[port.HostPort] {
						runCmd += fmt.Sprintf(" -p %d:%d/%s",
							port.HostPort,
							port.ContainerPort,
							port.Protocol,
						)
						validatedPorts[port.HostPort] = true
						logger.Println(logger.INFO, true,
							"migrateSoftware: ExecutionID="+executionID+
								", Container="+execution.Name+
								", Port "+fmt.Sprint(port.HostPort)+" mapped ✅")
					}
				} else {
					logger.Println(logger.WARN, true,
						"migrateSoftware: ExecutionID="+executionID+
							", Container="+execution.Name+
							", Port "+fmt.Sprint(port.HostPort)+" already in use, skipping ❌")
				}
			}
		}

		// add volume mount
		for _, mount := range execution.MountPaths {
			runCmd += fmt.Sprintf(" -v %s:%s", mount, mount)
		}

		// add env
		for _, env := range execution.Envs {
			runCmd += fmt.Sprintf(" -e %s=%s", env.Name, env.Value)
		}

		// network mode
		if execution.NetworkMode != "" {
			runCmd += fmt.Sprintf(" --network %s", execution.NetworkMode)
		}

		// restart policy
		if execution.RestartPolicy != "" {
			runCmd += fmt.Sprintf(" --restart %s", execution.RestartPolicy)
		}

		// image ref
		runCmd += " " + imageRef

		logger.Println(logger.DEBUG, true,
			"migrateSoftware: ExecutionID="+executionID+
				", Container="+execution.Name+
				", FinalRunCmd="+runCmd)

		// start container
		if _, err := targetClient.Run(runCmd); err != nil {
			logger.Println(logger.ERROR, true,
				"migrateSoftware: ExecutionID="+executionID+
					", InstallType=container, ContainerName="+execution.Name+
					", Error=failed to start container: "+err.Error())
			updateStatus(softwaremodel.SoftwareTypeContainer, i, "failed", err.Error(), false)
			continue
		}

		updateStatus(softwaremodel.SoftwareTypeContainer, i, "finished", "", true)
	}

	// Migrate repository configuration and GPG keys before package migration
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
