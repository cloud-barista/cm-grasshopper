package controller

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

type SimpleMsg struct {
	Message string `json:"message" example:"Any message"`
}

// GetHealth func is for checking Grasshopper server health.
// RestGetHealth godoc
// @Summary Check Grasshopper is alive
// @Description Check Grasshopper is alive
// @Tags [Admin] System management
// @Accept  json
// @Produce  json
// @Success 200 {object} SimpleMsg
// @Failure 404 {object} SimpleMsg
// @Failure 500 {object} SimpleMsg
// @Router /grasshopper/health [get]
func GetHealth(c echo.Context) error {
	okMessage := SimpleMsg{}
	okMessage.Message = "CM-Grasshopper API server is running"
	return c.JSONPretty(http.StatusOK, &okMessage, " ")
}
