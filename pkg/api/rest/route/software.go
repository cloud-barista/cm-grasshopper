package route

import (
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func Software(e *echo.Echo) {
	e.POST("/software/register", controller.RegisterSoftware)
	e.POST("/software/execution_list", controller.GetExecutionList)
	e.POST("/software/install", controller.InstallSoftware)
	e.DELETE("/software/:softwareId", controller.DeleteSoftware)
}
