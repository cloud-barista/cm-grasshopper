package software

import (
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"time"
)

func MigrateSoftware(executionID string, executionList *[]model.Execution,
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

	for _, execution := range *executionList {
		executionStatusList = append(executionStatusList, model.ExecutionStatus{
			Order:               execution.Order,
			SoftwareID:          execution.SoftwareID,
			SoftwareName:        execution.SoftwareName,
			SoftwareVersion:     execution.SoftwareVersion,
			SoftwareInstallType: execution.SoftwareInstallType,
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

	go func(id string, exList []model.Execution, exStatusList []model.ExecutionStatus,
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

		for i, execution := range exList {
			if execution.SoftwareInstallType == "package" {
				updateStatus(i, "installing", "", true)

				err := runPlaybook(id, execution.SoftwareID, s.SSHTarget)
				if err != nil {
					logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
						", InstallType=package, SoftwareID="+execution.SoftwareID+", Error="+err.Error())
					updateStatus(i, "failed", err.Error(), false)

					return
				}

				err = configCopier(s, t, execution.SoftwareName)
				if err != nil {
					logger.Println(logger.ERROR, true, "migrateSoftware: ExecutionID="+executionID+
						", InstallType=package, SoftwareID="+execution.SoftwareID+", Error="+err.Error())
					updateStatus(i, "failed", err.Error(), false)

					return
				}

				updateStatus(i, "finished", "", true)
			} else {
				updateStatus(i, "failed", "not supported install type", false)
			}
		}
	}(executionID, *executionList, executionStatusList, sourceClient, targetClient)

	return nil
}
