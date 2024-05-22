package server

import (
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/middlewares"

	"github.com/cloud-barista/cm-grasshopper/lib/config"
	_ "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/docs" // Grasshopper Documentation
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/route"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

func Init() {
	e := echo.New()

	e.Use(middlewares.CustomLogger())

	route.Software(e)
	route.RegisterSwagger(e)
	route.RegisterUtility(e)

	err := e.Start(":" + config.CMGrasshopperConfig.CMGrasshopper.Listen.Port)
	logger.Panicln(logger.ERROR, true, err)
}
