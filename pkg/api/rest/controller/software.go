package controller

import (
	"encoding/json"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/software"
	"github.com/labstack/echo/v4"
	"net/http"
)

type GetSoftwareResponse struct {
	software.Software
}

// SoftwareGetList godoc
//
//	@Summary		Get a list of integrated software information
//	@Description	Get information of all software.
//	@Tags			[Sample] Get software
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	GetSoftwareResponse	"Successfully get software list."
//	@Failure		404	{object}	GetSoftwareResponse	"Error occurred while getting software list."
//	@Router			/software/list [get]
func SoftwareGetList(c echo.Context) error {
	uuid := c.QueryParam("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	target, err := dao.TargetGet(uuid)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the target.")
	}

	var softwareList software.Software

	data, err := common.GetHTTPRequest("http://" + target.HoneybeeAddress + "/software")
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting software list.")
	}
	err = json.Unmarshal(data, &softwareList)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while parsing software list.")
	}

	return c.JSONPretty(http.StatusOK, softwareList, " ")
}
