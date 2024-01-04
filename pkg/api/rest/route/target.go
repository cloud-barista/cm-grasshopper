package route

import (
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func Target(e *echo.Echo) {
	e.POST("/target/register", controller.TargetRegister)
	e.GET("/target/get", controller.TargetGet)
	e.GET("/target/list", controller.TargetGetList)
	e.POST("/target/update", controller.TargetUpdate)
	e.POST("/target/delete", controller.TargetDelete)
}
