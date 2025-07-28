package route

import (
	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
	"strings"
)

func Software(e *echo.Echo) {
	// Package
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/software/package/migration_config/register", controller.RegisterPackageMigrationConfig)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/software/package/migration_config", controller.ListPackageMigrationConfig)
	e.DELETE("/"+strings.ToLower(common.ShortModuleName)+"/software/package/migration_config/:migrationConfigId", controller.DeletePackageMigrationConfig)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/software/package/migration_list/:sgId", controller.GetPackageMigrationList)
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/software/package/migrate", controller.MigratePackageSoftware)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/software/package/migrate/log/:executionId", controller.GetPackageSoftwareMigrationLog)
}
