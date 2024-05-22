package route

import (
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func Software(e *echo.Echo) {
	e.POST("/software/install", controller.InstallSoftware)
}
