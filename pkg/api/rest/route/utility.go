package route

import (
	"strings"

	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"

	"github.com/labstack/echo/v4"
)

func RegisterUtility(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/readyz", controller.CheckReady)
}
