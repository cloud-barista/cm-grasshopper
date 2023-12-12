package echo

import (
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

var e *echo.Echo

func Init() {
	e = echo.New()

	DAG()

	err := e.Start(":" + config.CMGrasshopperConfig.CMGrasshopper.Listen.Port)
	logger.Panicln(logger.ERROR, true, err)
}
