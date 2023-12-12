package echo

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func TargetRegister(c echo.Context) error {
	// TODO

	return c.JSONPretty(http.StatusOK, "TODO", " ")
}

func TargetGet(c echo.Context) error {
	// TODO

	return c.JSONPretty(http.StatusOK, "TODO", " ")
}

func TargetUpdate(c echo.Context) error {
	// TODO

	return c.JSONPretty(http.StatusOK, "TODO", " ")
}

func TargetDelete(c echo.Context) error {
	// TODO

	return c.JSONPretty(http.StatusOK, "TODO", " ")
}

func DAG() {
	e.POST("/target/register", TargetRegister)
	e.GET("/target/get", TargetGet)
	e.POST("/target/update", TargetUpdate)
	e.POST("/target/delete", TargetDelete)
}
