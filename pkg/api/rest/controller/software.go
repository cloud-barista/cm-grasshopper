package controller

import (
	"encoding/json"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/software"
	"github.com/labstack/echo/v4"
	"net/http"
)

// SoftwareGetList godoc
//
// @Summary		Get a list of software information.
// @Description	Get software information.
// @Tags			[Software] Get software info
// @Accept			json
// @Produce		json
// @Param			uuid query string true "UUID of the target"
// @Success		200	{object}	software.Software	"Successfully get information of software."
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get information of software."
// @Router			/software/list [get]
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
