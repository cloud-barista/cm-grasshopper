package echo

import (
	"encoding/json"
	"net/http"

	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/model/software"
	"github.com/labstack/echo/v4"
)

type GetSoftwareResponse struct {
	software.Software
}

// GetSoftewareList godoc
//	@Summary		Get a list of Integrated Softeware information
//	@Description	Get information of all Softeware.
//	@Tags			[Sample] Get Softeware
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	GetSoftwareResponse	"(This is a sample description for success response in Swagger UI"
//	@Failure		404	{object}	GetSoftwareResponse	"Failed to get software"
//	@Router			/software/list [get]

func SoftwareGetList(c echo.Context) error {
	uuid := c.QueryParam("uuid")
	if uuid == "" {
		return returnErrorMsg(c, "uuid is empty")
	}

	target, err := dao.TargetGet(uuid)
	if err != nil {
		return returnInternalError(c, err, "Error occurred while getting the target.")
	}

	var softwareList software.Software

	data, err := getHTTPRequest("http://" + target.HoneybeeAddress + "/software")
	if err != nil {
		return returnInternalError(c, err, "Error occurred while getting software list.")
	}
	err = json.Unmarshal(data, &softwareList)
	if err != nil {
		return returnInternalError(c, err, "Error occurred while parsing software list.")
	}

	return c.JSONPretty(http.StatusOK, softwareList, " ")
}

func Software() {
	e.GET("/software/list", SoftwareGetList)
}
