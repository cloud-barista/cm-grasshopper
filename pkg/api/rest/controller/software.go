package controller

import (
	"encoding/json"
	"errors"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/lib/software"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func uploadHandler(c echo.Context) (string, error) {
	file, err := c.FormFile("archive")
	if err != nil {
		return "", errors.New("failed to get file")
	}

	src, err := file.Open()
	if err != nil {
		return "", errors.New("failed to open file")
	}
	defer func() {
		_ = src.Close()
	}()

	id := uuid.New().String()
	destDir := filepath.Join("uploads", id)
	err = os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return "", errors.New("failed to create directory")
	}

	tempFileAbs := filepath.Join(destDir, file.Filename)
	tempFile, err := os.Create(tempFileAbs)
	if err != nil {
		return "", errors.New("failed to create temp file")
	}
	defer func() {
		_ = tempFile.Close()
	}()

	_, err = io.Copy(tempFile, src)
	if err != nil {
		return "", errors.New("failed to save file")
	}

	return tempFileAbs, nil
}

// RegisterSoftware godoc
//
// @Summary	Register Software
// @Description	Register the software.<br><br>[JSON Body Example]<br>{"architecture":"x86_64","install_type":"ansible","match_names":["telegraf"],"name":"telegraf","os":"Ubuntu","os_version":"22.04","version":"1.0"}
// @Tags		[Software]
// @Accept		mpfd
// @Produce		json
// @Param		json formData string true "Software register request JSON body string."
// @Param 		archive formData file true "Archive file to upload for ansible."
// @Success		200	{object}	model.SoftwareRegisterReq	"Successfully registered the software."
// @Failure		400	{object}	common.ErrorResponse		"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse		"Failed to sent SSH command."
// @Router		/grasshopper/software/register [post]
func RegisterSoftware(c echo.Context) error {
	err := c.Request().ParseMultipartForm(10 << 30) // 10GB
	if err != nil {
		return common.ReturnErrorMsg(c, "failed to parse multipart form")
	}

	jsonPart := c.FormValue("json")
	var softwareRegisterReq model.SoftwareRegisterReq
	err = json.Unmarshal([]byte(jsonPart), &softwareRegisterReq)
	if err != nil {
		return common.ReturnErrorMsg(c, "failed to parse json data")
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

	if softwareRegisterReq.MatchNames == nil || len(softwareRegisterReq.MatchNames) == 0 {
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

	var id = uuid.New().String()
	var sizeString = "0B"

	if softwareRegisterReq.InstallType == "ansible" {
		tempFilePath, err := uploadHandler(c)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
		defer func() {
			_ = os.RemoveAll(filepath.Join(tempFilePath, ".."))
		}()

		sizeString, err = software.SavePlaybook(id, tempFilePath)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
	} else {
		file, _ := c.FormFile("archive")
		if file != nil {
			return common.ReturnErrorMsg(c, "archive should not uploaded with the provided install type")
		}
	}

	sw := model.Software{
		ID:           id,
		InstallType:  softwareRegisterReq.InstallType,
		Name:         softwareRegisterReq.Name,
		Version:      softwareRegisterReq.Version,
		OS:           softwareRegisterReq.OS,
		OSVersion:    softwareRegisterReq.OSVersion,
		Architecture: softwareRegisterReq.Architecture,
		MatchNames:   matchNames,
		Size:         sizeString,
	}

	dbSW, err := dao.SoftwareCreate(&sw)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, dbSW, " ")
}

// GetExecutionList godoc
//
// @Summary	Get Execution List
// @Description	Get software migration execution list.
// @Tags		[Software]
// @Accept		json
// @Produce		json
// @Param		getExecutionListReq body model.GetExecutionListReq true "Software info list."
// @Success		200	{object}	model.GetExecutionListRes	"Successfully get migration execution list."
// @Failure		400	{object}	common.ErrorResponse		"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse		"Failed to get migration execution list."
// @Router		/grasshopper/software/execution_list [post]
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
// @Summary	Install Software
// @Description	Install pieces of software to target.
// @Tags		[Software]
// @Accept		json
// @Produce		json
// @Param		softwareInstallReq body model.SoftwareInstallReq true "Software install request."
// @Success		200	{object}	model.SoftwareInstallRes	"Successfully sent SSH command."
// @Failure		400	{object}	common.ErrorResponse		"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse		"Failed to sent SSH command."
// @Router		/grasshopper/software/install [post]
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
// @Summary		Delete Software
// @Description	Delete the software.
// @Tags		[Software]
// @Accept		json
// @Produce		json
// @Param		softwareId path string true "ID of the software."
// @Success		200	{object}	model.SimpleMsg			"Successfully update the software"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to delete the software"
// @Router		/grasshopper/software/{softwareId} [delete]
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
