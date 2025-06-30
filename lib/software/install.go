package software

import (
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"strings"
	"time"
)

func MigrateSoftware(executionID string, executionList *model.MigrationList,
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
			SoftwareInstallType: model.SoftwareTypePackage,
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
			SoftwareInstallType: model.SoftwareTypeContainer,
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

	go func(id string, exList *model.MigrationList, exStatusList []model.ExecutionStatus,
		s *ssh.Client, t *ssh.Client) {
		defer func() {
			_ = s.Close()
			_ = t.Close()
		}()

		var updateStatus = func(i int, status string, errMsg string, updateStartedTime bool) {
			exStatusList[i].Status = status
			exStatusList[i].ErrorMessage = errMsg
			now := time.Now()
			if updateStartedTime {
				exStatusList[i].StartedAt = now
			}
			exStatusList[i].UpdatedAt = now

			err := dao.SoftwareInstallStatusUpdate(&model.SoftwareInstallStatus{
				ExecutionID:     executionID,
				ExecutionStatus: exStatusList,
			})
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", Error="+err.Error())
			}
		}

		for i, execution := range exList.Packages {
			updateStatus(i, "installing", "", true)

			err := runPlaybook(id, execution.PackageMigrationConfigID, execution.Name, t.SSHTarget)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", InstallType=package, Error="+err.Error())
				updateStatus(i, "failed", err.Error(), false)

				return
			}

			migrationLogger, err := initLoggerWithUUID(executionID)
			if err != nil {
				errMsg := fmt.Sprintf("failed to initialize logger: %v", err)
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", InstallType=package, Error="+errMsg)
				updateStatus(i, "failed", errMsg, false)
			}

			err = configCopier(s, t, execution.Name, executionID, migrationLogger)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", InstallType=package, Error="+err.Error())
				updateStatus(i, "failed", err.Error(), false)

				migrationLogger.Close()
				return
			}

			if execution.PackageMigrationConfigID != "" {
				sw, err := dao.PackageMigrationConfigGet(execution.PackageMigrationConfigID)
				if err != nil {
					msg := "migrateSoftware: ExecutionID=" + executionID +
						", Error=" + err.Error()
					logger.Println(logger.WARN, true, msg)
					migrationLogger.Printf(WARN, msg)
				}
				if sw != nil {
					customConfigsSplit := strings.Split(sw.CustomConfigs, ",")
					if len(customConfigsSplit) > 0 && customConfigsSplit[0] != "" {
						migrationLogger.Printf(INFO, "Starting to copy custom configs")
						var customConfigs []ConfigFile
						for _, customConfig := range customConfigsSplit {
							customConfigs = append(customConfigs, ConfigFile{
								Path:   customConfig,
								Status: "Custom",
							})
						}
						err = copyConfigFiles(s, t, customConfigs, migrationLogger)
						if err != nil {
							logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
								", Error="+err.Error())
							updateStatus(i, "failed", err.Error(), false)

							migrationLogger.Close()
							return
						}
					}
				}
			}

			err = serviceMigrator(s, t, execution.Name, executionID, migrationLogger)
			if err != nil {
				logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
					", InstallType=package, Error="+err.Error())
				//updateStatus(i, "failed", err.Error(), false)
				//
				//migrationLogger.Close()
				//return
			}

			updateStatus(i, "finished", "", true)
		}
	}(executionID, executionList, executionStatusList, sourceClient, targetClient)

	return nil
}
