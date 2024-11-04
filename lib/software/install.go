package software

import (
	"encoding/json"
	"errors"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"strconv"
	"time"
)

func targetToSSHTarget(target *model.Target) (*model.SSHTarget, error) {
	data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerAddress+
		":"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerPort+
		"/tumblebug/ns/"+target.NamespaceID+"/mci/"+target.MCISID+"/vm/"+target.VMID,
		config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Username, config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Password)
	if err != nil {
		return nil, err
	}

	var vmInfo model.TBVMInfo
	err = json.Unmarshal(data, &vmInfo)
	if err != nil {
		return nil, err
	}

	if vmInfo.PublicIP == "" {
		return nil, errors.New("failed to get target VM's public IP")
	}

	if vmInfo.SSHPort == "" {
		return nil, errors.New("failed to get target VM's SSH port")
	}

	if vmInfo.SSHKeyID == "" {
		return nil, errors.New("failed to get target VM's SSH Key ID")
	}

	if vmInfo.VMUserName == "" {
		return nil, errors.New("failed to get target VM's user name")
	}

	sshPort, err := strconv.Atoi(vmInfo.SSHPort)
	if err != nil {
		return nil, errors.New("invalid ssh port")
	}

	data, err = common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerAddress+
		":"+config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerPort+
		"/tumblebug/ns/"+target.NamespaceID+"/resources/sshKey/"+vmInfo.SSHKeyID,
		config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Username, config.CMGrasshopperConfig.CMGrasshopper.Tumblebug.Password)
	if err != nil {
		return nil, err
	}

	var sshKeyInfo model.TBSSHKeyInfo
	err = json.Unmarshal(data, &sshKeyInfo)
	if err != nil {
		return nil, err
	}

	if sshKeyInfo.PrivateKey == "" {
		return nil, errors.New("failed to get target VM's private key")
	}

	return &model.SSHTarget{
		IP:         vmInfo.PublicIP,
		Port:       uint(sshPort),
		UseKeypair: true,
		Username:   vmInfo.VMUserName,
		Password:   "",
		PrivateKey: sshKeyInfo.PrivateKey,
	}, nil
}

func InstallSoftware(executionID string, executionList *[]model.Execution, target *model.Target) error {
	var executionStatusList []model.ExecutionStatus

	sshTarget, err := targetToSSHTarget(target)
	if err != nil {
		return err
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
		return err
	}

	go func(id string, exList []model.Execution, exStatusList []model.ExecutionStatus, t model.SSHTarget) {
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
				logger.Println(logger.ERROR, true, "installSoftware: ExecutionID="+executionID+
					", Error="+err.Error())
			}
		}

		for i, execution := range exList {
			if execution.SoftwareInstallType == "package" {
				updateStatus(i, "installing", "", true)
				err := runPlaybook(id, execution.SoftwareID, t)
				if err != nil {
					logger.Println(logger.ERROR, true, "installSoftware: ExecutionID="+executionID+
						", InstallType=ansible, SoftwareID="+execution.SoftwareID+", Error="+err.Error())
					updateStatus(i, "failed", err.Error(), false)
				}
				updateStatus(i, "finished", "", true)
			} else {
				updateStatus(i, "failed", "not supported install type", false)
			}
		}
	}(executionID, *executionList, executionStatusList, *sshTarget)

	return nil
}
