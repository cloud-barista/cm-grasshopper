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
//	@Summary		Delete the computing target
//	@Description	Delete the target.
//	@Tags			[Sample] Delete the target
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	TargetDeleteResponse	"Successfully delete the target"
//	@Failure		404	{object}	TargetDeleteResponse	"Failed to delete the target"
//	@Router			/target/delete [post]
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
