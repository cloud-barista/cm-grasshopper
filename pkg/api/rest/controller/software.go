package controller

import (
	"embed"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/lib/software"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/jollaman999/utils/fileutil"
	"github.com/labstack/echo/v4"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed playbook_base
var playbookBase embed.FS

func writePlaybookFiles(softwareName string, destDir string, neededPackages []string) error {
	err := fs.WalkDir(playbookBase, "playbook_base", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel("playbook_base", path)
		destPath := filepath.Join(destDir, relPath)

		err = fileutil.CreateDirIfNotExist(filepath.Dir(destPath))
		if err != nil {
			return err
		}

		data, err := playbookBase.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.HasSuffix(destPath, "playbook.yml") {
			dataStr := string(data)
			dataStr = strings.ReplaceAll(dataStr, "SOFTWARE_NAME", softwareName)
			data = []byte(dataStr)
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return err
		}

		if strings.HasSuffix(destPath, filepath.Join("vars", "main.yml")) {
			err := fileutil.WriteFileAppend(destPath, "packages:")
			if err != nil {
				return err
			}

			for _, packageName := range neededPackages {
				err := fileutil.WriteFileAppend(destPath, "\n  - "+packageName)
				if err != nil {
					return err
				}
			}

			return nil
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// RegisterSoftware godoc
//
//	@ID				register-software
//	@Summary		Register Software
//	@Description	Register the software.
//	@Tags			[Software]
//	@Accept			json
//	@Produce		json
//	@Param			softwareRegisterReq body model.SoftwareRegisterReq true "Software info"
//	@Success		200	{object}	model.SoftwareRegisterReq	"Successfully registered the software."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to sent SSH command."
//	@Router			/software/register [post]
func RegisterSoftware(c echo.Context) error {
	var err error

	softwareRegisterReq := new(model.SoftwareRegisterReq)
	err = c.Bind(softwareRegisterReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	err = model.CheckInstallType(softwareRegisterReq.InstallType)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	err = model.CheckArchitecture(softwareRegisterReq.Architecture)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if softwareRegisterReq.Name == "" {
		return common.ReturnErrorMsg(c, "Please provide the name")
	}

	if softwareRegisterReq.Version == "" {
		return common.ReturnErrorMsg(c, "Please provide the version")
	}

	if softwareRegisterReq.OS == "" {
		return common.ReturnErrorMsg(c, "Please provide the os")
	}

	if softwareRegisterReq.OSVersion == "" {
		return common.ReturnErrorMsg(c, "Please provide the os version")
	}

	if len(softwareRegisterReq.MatchNames) == 0 {
		return common.ReturnErrorMsg(c, "Please provide the match names")
	}
	var matchNames string
	for _, matchName := range softwareRegisterReq.MatchNames {
		if strings.Contains(matchName, ",") {
			return common.ReturnErrorMsg(c, "Match name should not contain ','")
		}
		matchNames = matchName + ","
	}
	matchNames = matchNames[:len(matchNames)-1]

	if len(softwareRegisterReq.NeededPackages) == 0 {
		return common.ReturnErrorMsg(c, "Please provide the needed packages")
	}
	var neededPackages string
	for _, neededPackage := range softwareRegisterReq.NeededPackages {
		if strings.Contains(neededPackage, ",") {
			return common.ReturnErrorMsg(c, "Needed package should not contain ','")
		}
		neededPackages = neededPackage + ","
	}
	neededPackages = neededPackages[:len(neededPackages)-1]

	var id = uuid.New().String()

	sw := model.Software{
		ID:             id,
		InstallType:    softwareRegisterReq.InstallType,
		Name:           softwareRegisterReq.Name,
		Version:        softwareRegisterReq.Version,
		OS:             softwareRegisterReq.OS,
		OSVersion:      softwareRegisterReq.OSVersion,
		Architecture:   softwareRegisterReq.Architecture,
		MatchNames:     matchNames,
		NeededPackages: neededPackages,
	}

	dbSW, err := dao.SoftwareCreate(&sw)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	destDir := filepath.Join(config.CMGrasshopperConfig.CMGrasshopper.Ansible.PlaybookRootPath, sw.ID)
	err = writePlaybookFiles(softwareRegisterReq.Name, destDir, softwareRegisterReq.NeededPackages)
	if err != nil {
		_ = fileutil.DeleteDir(destDir)
		_ = dao.SoftwareDelete(&sw)

		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, dbSW, " ")
}

// GetExecutionList godoc
//
//	@ID				get-execution-list
//	@Summary		Get Execution List
//	@Description	Get software migration execution list.
//	@Tags			[Software]
//	@Accept			json
//	@Produce		json
//	@Param			getExecutionListReq body model.GetExecutionListReq true "Software info list"
//	@Success		200	{object}	model.GetExecutionListRes	"Successfully get migration execution list."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to get migration execution list."
//	@Router			/software/execution_list [post]
func GetExecutionList(c echo.Context) error {
	var err error

	getExecutionListReq := new(model.GetExecutionListReq)
	err = c.Bind(getExecutionListReq)
	if err != nil {
		return err
	}

	executionListRes, err := software.MakeExecutionListRes(getExecutionListReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, *executionListRes, " ")
}

// InstallSoftware godoc
//
//	@ID				install-software
//	@Summary		Install Software
//	@Description	Install pieces of software to target.
//	@Tags			[Software]
//	@Accept			json
//	@Produce		json
//	@Param			softwareInstallReq body model.SoftwareInstallReq true "Software install request."
//	@Success		200	{object}	model.SoftwareInstallRes	"Successfully sent SSH command."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to sent SSH command."
//	@Router			/software/install [post]
func InstallSoftware(c echo.Context) error {
	softwareInstallReq := new(model.SoftwareInstallReq)
	err := c.Bind(softwareInstallReq)
	if err != nil {
		return err
	}

	var executionList []model.Execution

	for i, id := range softwareInstallReq.SoftwareIDs {
		sw, err := dao.SoftwareGet(id)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}

		executionList = append(executionList, model.Execution{
			Order:               i + 1,
			SoftwareID:          sw.ID,
			SoftwareName:        sw.Name,
			SoftwareVersion:     sw.Version,
			SoftwareInstallType: sw.InstallType,
		})
	}

	executionID := uuid.New().String()

	err = software.InstallSoftware(executionID, &executionList, &softwareInstallReq.Target)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SoftwareInstallRes{
		ExecutionID:   executionID,
		ExecutionList: executionList,
	}, " ")
}

// DeleteSoftware godoc
//
//	@ID				delete-software
//	@Summary		Delete Software
//	@Description	Delete the software.
//	@Tags			[Software]
//	@Accept			json
//	@Produce		json
//	@Param			softwareId path string true "ID of the software."
//	@Success		200	{object}	model.SimpleMsg			"Successfully update the software"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to delete the software"
//	@Router			/software/{softwareId} [delete]
func DeleteSoftware(c echo.Context) error {
	swID := c.Param("softwareId")
	if swID == "" {
		return common.ReturnErrorMsg(c, "Please provide the softwareId.")
	}

	sw, err := dao.SoftwareGet(swID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if sw.InstallType == "ansible" {
		err = software.DeletePlaybook(swID)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
	}

	err = dao.SoftwareDelete(sw)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}
