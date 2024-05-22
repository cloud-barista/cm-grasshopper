package controller

import (
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

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

	client, err := ssh.NewSSHClient(softwareInstallReq.ConnectionUUID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var softwareInstallRes model.SoftwareInstallRes

	var packageNames string
	var out string

	for _, name := range softwareInstallReq.PackageNames {
		packageNames += " " + name
	}

	packageType := strings.ToLower(softwareInstallReq.PackageType)
	if packageType == "apt" {
		out, err = client.RunBash("echo " + client.ConnectionInfo.Password + " | sudo -S -k apt-get install" + packageNames)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
	} else if packageType == "yum" {
		out, err = client.RunBash("echo " + client.ConnectionInfo.Password + " | sudo -S -k yum install" + packageNames)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
	} else {
		return common.ReturnErrorMsg(c, "Invalid package type: "+softwareInstallReq.PackageType)
	}

	softwareInstallRes.Output = out

	return c.JSONPretty(http.StatusOK, softwareInstallRes, " ")
}
