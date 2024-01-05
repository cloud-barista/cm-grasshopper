package controller

import (
	"errors"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/cloud-barista/cm-honeybee/pkg/api/rest/model/software"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"
	"strings"
)

type TargetDeleteResponse struct {
	software.Software
}

func checkHoneybeeAddress(honeybeeAddress string) error {
	addrSplit := strings.Split(honeybeeAddress, ":")
	if len(addrSplit) < 2 {
		return errors.New("honeybee_address must be {IP or IPv6 or Domain}:{Port} form")
	}
	port, err := strconv.Atoi(addrSplit[len(addrSplit)-1])
	if err != nil || port < 1 || port > 65535 {
		return errors.New("honeybee_address has invalid port value")
	}

	return nil
}

// TargetRegister godoc
//
// @Summary		Register the computing target
// @Description	Register the target.
// @Tags			[Target] Register target
// @Accept			json
// @Produce		json
// @Param			honeybee_address query string true "Honeybee address installed in the target"
// @Success		200	{object}	common.ErrorResponse	"Successfully register the target"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to register the target"
// @Router			/target/update [post]
func TargetRegister(c echo.Context) error {
	honeybeeAddress := c.QueryParam("honeybee_address")
	if honeybeeAddress == "" {
		return errors.New("honeybee_address is empty")
	}
	err := checkHoneybeeAddress(honeybeeAddress)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	target, err := dao.TargetRegister(honeybeeAddress)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while registering the target.")
	}

	return c.JSONPretty(http.StatusOK, target, " ")
}

// TargetGet godoc
//
// @Summary		Get information of the target.
// @Description	Get target information.
// @Tags			[Target] Get target
// @Accept			json
// @Produce		json
// @Param			uuid query string true "UUID of the target"
// @Success		200	{object}	model.Target	"Successfully get the target."
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get the target."
// @Router			/target/get [get]
func TargetGet(c echo.Context) error {
	uuid := c.QueryParam("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	target, err := dao.TargetGet(uuid)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the target.")
	}

	return c.JSONPretty(http.StatusOK, target, " ")
}

// TargetGetList godoc
//
// @Summary		Get a list of targets.
// @Description	Get a list of targets.
// @Tags			[Target] Get target list
// @Accept			json
// @Produce		json
// @Success		200	{object}	[]model.Target	"Successfully get a list of targets."
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to get a list of targets."
// @Router			/target/list [get]
func TargetGetList(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	uuid := c.QueryParam("uuid")
	honeybeeAddress := c.QueryParam("honeybee_address")

	target := &model.Target{
		UUID:            uuid,
		HoneybeeAddress: honeybeeAddress,
	}

	targets, err := dao.TargetGetList(target, page, row)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the target list.")
	}

	return c.JSONPretty(http.StatusOK, targets, " ")
}

// TargetUpdate godoc
//
// @Summary		Update the computing target
// @Description	Update the target.
// @Tags			[Target] Update target
// @Accept			json
// @Produce		json
// @Param			uuid query string true "UUID of the target"
// @Success		200	{object}	common.ErrorResponse	"Successfully update the target"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to update the target"
// @Router			/target/update [post]
func TargetUpdate(c echo.Context) error {
	uuid := c.QueryParam("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	honeybeeAddress := c.QueryParam("honeybee_address")
	if honeybeeAddress == "" {
		return errors.New("honeybee_address is empty")
	}
	err := checkHoneybeeAddress(honeybeeAddress)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	target, err := dao.TargetGet(uuid)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the target.")
	}

	if honeybeeAddress != "" {
		target.HoneybeeAddress = honeybeeAddress
	}

	err = dao.TargetUpdate(target)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while updating the target.")
	}

	return c.JSONPretty(http.StatusOK, target, " ")
}

// TargetDelete godoc
//
// @Summary		Delete the computing target
// @Description	Delete the target.
// @Tags			[Target] Delete target
// @Accept			json
// @Produce		json
// @Param			uuid query string true "UUID of the target"
// @Success		200	{object}	common.ErrorResponse	"Successfully delete the target"
// @Failure		400	{object}	common.ErrorResponse	"Sent bad request."
// @Failure		500	{object}	common.ErrorResponse	"Failed to delete the target"
// @Router			/target/delete [post]
func TargetDelete(c echo.Context) error {
	uuid := c.QueryParam("uuid")
	if uuid == "" {
		return common.ReturnErrorMsg(c, "uuid is empty")
	}

	target, err := dao.TargetGet(uuid)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while getting the target.")
	}

	err = dao.TargetDelete(target)
	if err != nil {
		return common.ReturnInternalError(c, err, "Error occurred while deleting the target.")
	}

	return c.JSONPretty(http.StatusOK, target, " ")
}
