package controller

import (
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"net/http"
)

// RegisterSoftware godoc
//
// @Summary	Register Software
// @Description	Register the software.
// @Tags		[Software]
// @Accept		json
// @Produce		json
// @Param		softwareRegisterReq body model.SoftwareRegisterReq true "Software register request."
// @Success		200	{object}	model.SoftwareRegisterReq	"Successfully registered the software."
// @Failure		400	{object}	common.ErrorResponse		"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse		"Failed to sent SSH command."
// @Router		/software/register [post]
func RegisterSoftware(c echo.Context) error {
	var err error

	softwareRegisterReq := new(model.SoftwareRegisterReq)
	err = c.Bind(softwareRegisterReq)
	if err != nil {
		return err
	}

	installType, err := model.ToInstallType(softwareRegisterReq.InstallType)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	architecture, err := model.ToArchitecture(softwareRegisterReq.Architecture)
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

	software := model.Software{
		ID:           uuid.New().String(),
		InstallType:  installType,
		Name:         softwareRegisterReq.Name,
		Version:      softwareRegisterReq.Version,
		OS:           softwareRegisterReq.OS,
		OSVersion:    softwareRegisterReq.OSVersion,
		Architecture: architecture,
		MatchNames:   softwareRegisterReq.MatchNames,
		Size:         "0B",
	}

	return c.JSONPretty(http.StatusOK, software, " ")
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
// @Router		/software/install [post]
func InstallSoftware(c echo.Context) error {
	var err error

	softwareInstallReq := new(model.SoftwareInstallReq)
	err = c.Bind(softwareInstallReq)
	if err != nil {
		return err
	}

	// TODO: Copy configuration files

	return c.JSONPretty(http.StatusOK, nil, " ")
}
