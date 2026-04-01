package route

import (
	"strings"

	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/labstack/echo/v4"
)

func Job(e *echo.Echo) {
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/job/status/:jobId", controller.GetJobStatus)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/job/log/:jobId", controller.GetJobLog)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/job/status", controller.ListJobStatus)
}
