package route

import (
	"strings"

	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func Velero(e *echo.Echo) {
	base := "/" + strings.ToLower(common.ShortModuleName) + "/velero"

	e.POST(base+"/:role/health", controller.VeleroHealth)
	e.POST(base+"/:role/install", controller.VeleroInstall)

	e.POST(base+"/source/backups/list", controller.ListBackups)
	e.POST(base+"/source/backups", controller.CreateBackup)
	e.POST(base+"/source/backups/:name", controller.GetBackup)
	e.POST(base+"/source/backups/:name/delete", controller.DeleteBackup)
	e.POST(base+"/source/backups/:name/validate", controller.ValidateBackup)

	e.POST(base+"/target/restores/list", controller.ListRestores)
	e.POST(base+"/target/restores", controller.CreateRestore)
	e.POST(base+"/target/restores/:name", controller.GetRestore)
	e.POST(base+"/target/restores/:name/delete", controller.DeleteRestore)
	e.POST(base+"/target/restores/:name/validate", controller.ValidateRestore)

	e.POST(base+"/migration/precheck", controller.VeleroMigrationPrecheck)
	e.POST(base+"/migration/execute", controller.VeleroMigrationExecute)
}
