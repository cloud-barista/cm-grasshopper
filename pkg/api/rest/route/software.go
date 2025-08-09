package route

import (
	"strings"

	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func Software(e *echo.Echo) {
	// Package
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/software/package/migration_list", controller.GetPackageMigrationList)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/software/package/migrate", controller.MigratePackageSoftware)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/software/package/migrate/log/:executionId", controller.GetPackageSoftwareMigrationLog)
}
