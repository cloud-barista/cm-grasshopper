package software

import (
	"fmt"
	"time"

	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	softwaremodel "github.com/cloud-barista/cm-model/sw"
	"github.com/jollaman999/utils/logger"
)

func MigrateSoftware(executionID string, executionList *softwaremodel.MigrationList,
	sourceConnectionInfoID string, target *model.Target) error {
	var executionStatusList []model.ExecutionStatus

	sourceClient, err := ssh.NewSSHClient(ssh.ConnectionTypeSource, sourceConnectionInfoID, "", "")
	if err != nil {
		return fmt.Errorf("failed to connect to source host: %v", err)
	}

	targetClient, err := ssh.NewSSHClient(ssh.ConnectionTypeTarget, target.VMID, target.NamespaceID, target.MCIID)
	if err != nil {
		_ = sourceClient.Close()

		return fmt.Errorf("failed to connect to target host: %v", err)
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

		return err
	}

	go func(id string, exList *softwaremodel.MigrationList, exStatusList []model.ExecutionStatus,
		s *ssh.Client, t *ssh.Client) {
		defer func() {
			_ = s.Close()
			_ = t.Close()
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

		sourceSystemType, err := getSystemType(s, migrationLogger)
		if err != nil {
			logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
				", Error=failed to detect source system type: "+err.Error())
			return
		}

		targetSystemType, err = getSystemType(t, migrationLogger)
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
			exStatusList[i].Status = status
			exStatusList[i].ErrorMessage = errMsg
			now := time.Now()
			if updateStartedTime {
				exStatusList[i].StartedAt = now
			}
			exStatusList[i].UpdatedAt = now

			err := dao.SoftwareInstallStatusUpdate(&model.SoftwareInstallStatus{
				SoftwareInstallType: installType,
				ExecutionID:         executionID,
				ExecutionStatus:     exStatusList,
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

		for i, execution := range exList.Containers {
			if execution.Runtime == string(softwaremodel.SoftwareContainerRuntimeTypeDocker) {
				if dockerRuntimeChecked {
					if !dockerInstallOk {
						continue
					}
				} else {
					err = runtimeInstaller(t, softwaremodel.SoftwareContainerRuntimeTypeDocker, migrationLogger)
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
					err = runtimeInstaller(t, softwaremodel.SoftwareContainerRuntimeTypePodman, migrationLogger)
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
			// TODO: Container Listen Ports Validation

			updateStatus(softwaremodel.SoftwareTypePackage, i, "finished", "", true)
		}

		for i, execution := range exList.Packages {
			updateStatus(softwaremodel.SoftwareTypePackage, i, "installing", "", true)

			err := runPlaybook(id, "package", execution.Name, t.SSHTarget)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", InstallType=package, Error="+err.Error())
				updateStatus(softwaremodel.SoftwareTypePackage, i, "failed", err.Error(), false)

				return
			}

			err = configCopier(s, t, execution.Name, executionID, migrationLogger)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", InstallType=package, Error="+err.Error())
				updateStatus(softwaremodel.SoftwareTypePackage, i, "failed", err.Error(), false)
			}

			err = serviceMigrator(s, t, execution.Name, executionID, migrationLogger)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", InstallType=package, Error="+err.Error())
				updateStatus(softwaremodel.SoftwareTypePackage, i, "failed", err.Error(), false)
			}

			updateStatus(softwaremodel.SoftwareTypePackage, i, "finished", "", true)
		}
	}(executionID, executionList, executionStatusList, sourceClient, targetClient)

	return nil
}
