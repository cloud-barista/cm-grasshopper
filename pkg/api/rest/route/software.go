package route

import (
	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func Software(e *echo.Echo) {
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/software/register", controller.RegisterSoftware)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/software/migration_list/:sgId", controller.GetMigrationList)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/software/migrate", controller.MigrateSoftware)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/software/migrate/log/:executionId", controller.GetSoftwareMigrationLog)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/software", controller.ListSoftware)
	e.DELETE("/"+strings.ToLower(common.ShortModuleName)+"/software/:softwareId", controller.DeleteSoftware)
}
