package controller

import (
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/labstack/echo/v4"
)

func SoftwareGetList(c echo.Context) error {
	uuid := c.QueryParam("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	//data, err := common.GetHTTPRequest("http://XXX/software")
	//if err != nil {
	//	return common.ReturnInternalError(c, err, "Error occurred while getting software list.")
	//}
	//err = json.Unmarshal(data, &XXX)
	//if err != nil {
	//	return common.ReturnInternalError(c, err, "Error occurred while parsing software list.")
	//}
	//
	//return c.JSONPretty(http.StatusOK, softwareList, " ")

	return nil
}
