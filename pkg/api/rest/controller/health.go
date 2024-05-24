package controller

import (
	_ "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common" // Need for swag
	"github.com/labstack/echo/v4"
	"net/http"
)

type SimpleMsg struct {
	Message string `json:"message"`
}

var OkMessage = SimpleMsg{}
var IsReady = false

// CheckReady func is for checking Grasshopper server health.
// @Summary Check Ready
// @Description Check Grasshopper is ready
// @Tags [Admin] System management
// @Accept		json
// @Produce		json
// @Success		200 {object}	SimpleMsg				"Successfully get ready state."
// @Failure		500	{object}	common.ErrorResponse	"Failed to check ready state."
//
// @Router /grasshopper/readyz [get]
func CheckReady(c echo.Context) error {
	status := http.StatusOK

	if !IsReady {
		status = http.StatusServiceUnavailable
	}

	return c.JSONPretty(status, &OkMessage, " ")
}
