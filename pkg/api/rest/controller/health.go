package controller

import (
	_ "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common" // Need for swag
	"github.com/labstack/echo/v4"
	"net/http"
)

type SimpleMsg struct {
	Message string `json:"message"`
}

// GetHealth func is for checking Grasshopper server health.
// @Summary Check Grasshopper is alive
// @Description Check Grasshopper is alive
// @Tags [Admin] System management
// @Accept  json
// @Produce  json
// @Success		200 {object}	SimpleMsg	"Successfully get heath state."
// @Failure		500	{object}	common.ErrorResponse	"Failed to check health."
//
// @Router /grasshopper/health [get]
func GetHealth(c echo.Context) error {
	okMessage := SimpleMsg{}
	okMessage.Message = "CM-Grasshopper API server is running"
	return c.JSONPretty(http.StatusOK, &okMessage, " ")
}
