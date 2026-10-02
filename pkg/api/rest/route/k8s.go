package route

import (
	"strings"

	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func K8s(e *echo.Echo) {
	e.POST("/"+strings.ToLower(common.ShortModuleName)+"/k8s/migrate", controller.K8sMigrate)
}
