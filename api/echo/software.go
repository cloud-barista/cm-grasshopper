package echo

import (
	"encoding/json"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-honeybee/model/software"
	"github.com/labstack/echo/v4"
	"net/http"
)

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
