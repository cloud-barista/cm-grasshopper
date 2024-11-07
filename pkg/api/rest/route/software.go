package route

import (
	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func Software(e *echo.Echo) {
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/software/register", controller.RegisterSoftware)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/software/execution_list", controller.GetExecutionList)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/software/install", controller.InstallSoftware)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/software", controller.ListSoftware)
	e.DELETE("/"+strings.ToLower(common.ShortModuleName)+"/software/:softwareId", controller.DeleteSoftware)
}
