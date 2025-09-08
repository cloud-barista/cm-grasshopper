package route

import (
	"net/http"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/labstack/echo/v4"
	echoSwagger "github.com/swaggo/echo-swagger"
)

func RegisterSwagger(e *echo.Echo) {
	swaggerRedirect := func(c echo.Context) error {
		return c.Redirect(http.StatusMovedPermanently, "/"+strings.ToLower(common.ShortModuleName)+"/api/index.html")
	}
	e.GET("", swaggerRedirect)
	e.GET("/", swaggerRedirect)
	e.GET("/"+strings.ToLower(common.ShortModuleName), swaggerRedirect)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/", swaggerRedirect)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/api", swaggerRedirect)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/api/", swaggerRedirect)
	e.GET("/"+strings.ToLower(common.ShortModuleName)+"/api/*", echoSwagger.WrapHandler)
}
