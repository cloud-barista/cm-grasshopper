package echo

import (
	"errors"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/model"
	"github.com/labstack/echo/v4"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
)

func checkHoneybeeAddress(honeybeeAddress string) error {
	addrSplit := strings.Split(honeybeeAddress, ":")
	if len(addrSplit) < 2 {
		return errors.New("honeybee_address must be {IP or IPv6 or Domain}:{Port} form")
	}
	port, err := strconv.Atoi(addrSplit[len(addrSplit)-1])
	if err != nil || port < 1 || port > 65535 {
		return errors.New("honeybee_address has invalid port value")
	}
	addr, _ := strings.CutSuffix(honeybeeAddress, ":"+strconv.Itoa(port))
	_, err = netip.ParseAddr(addr)
	if err != nil {
		_, err = net.LookupIP(addr)
		if err != nil {
			return errors.New("honeybee_address has invalid address value " +
				"or can't find the domain (" + addr + ")")
		}
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
		return returnErrorMsg(c, err.Error())
	}

	target, err := dao.TargetRegister(honeybeeAddress)
	if err != nil {
		return returnInternalError(c, err, "Error occurred while registering the target.")
	}

	return c.JSONPretty(http.StatusOK, target, " ")
}

func TargetGet(c echo.Context) error {
	uuid := c.QueryParam("uuid")
	if uuid == "" {
		return returnErrorMsg(c, "uuid is empty")
	}

	target, err := dao.TargetGet(uuid)
	if err != nil {
		return returnInternalError(c, err, "Error occurred while getting the target.")
	}

	return c.JSONPretty(http.StatusOK, target, " ")
}

func TargetGetList(c echo.Context) error {
	page, row, err := checkPageRow(c)
	if err != nil {
		return returnErrorMsg(c, err.Error())
	}

	uuid := c.QueryParam("uuid")
	honeybeeAddress := c.QueryParam("honeybee_address")
	if honeybeeAddress != "" {
		err = checkHoneybeeAddress(honeybeeAddress)
		if err != nil {
			return returnErrorMsg(c, err.Error())
		}
	}

	target := &model.Target{
		UUID:            uuid,
		HoneybeeAddress: honeybeeAddress,
	}

	targets, err := dao.TargetGetList(target, page, row)
	if err != nil {
		return returnInternalError(c, err, "Error occurred while getting the target list.")
	}

	return c.JSONPretty(http.StatusOK, targets, " ")
}

func TargetUpdate(c echo.Context) error {
	uuid := c.QueryParam("uuid")
	if uuid == "" {
		return returnErrorMsg(c, "uuid is empty")
	}

	honeybeeAddress := c.QueryParam("honeybee_address")
	err := checkHoneybeeAddress(honeybeeAddress)
	if err != nil {
		return returnErrorMsg(c, err.Error())
	}

	target, err := dao.TargetGet(uuid)
	if err != nil {
		return returnInternalError(c, err, "Error occurred while getting the target.")
	}

	if honeybeeAddress != "" {
		target.HoneybeeAddress = honeybeeAddress
	}

	err = dao.TargetUpdate(target)
	if err != nil {
		return returnInternalError(c, err, "Error occurred while updating the target.")
	}

	return c.JSONPretty(http.StatusOK, target, " ")
}

func TargetDelete(c echo.Context) error {
	uuid := c.QueryParam("uuid")
	if uuid == "" {
		return returnErrorMsg(c, "uuid is empty")
	}

	target, err := dao.TargetGet(uuid)
	if err != nil {
		return returnInternalError(c, err, "Error occurred while getting the target.")
	}

	err = dao.TargetDelete(target)
	if err != nil {
		return returnInternalError(c, err, "Error occurred while deleting the target.")
	}

	return c.JSONPretty(http.StatusOK, "TODO", " ")
}

func Target() {
	e.POST("/target/register", TargetRegister)
	e.GET("/target/get", TargetGet)
	e.GET("/target/list", TargetGetList)
	e.POST("/target/update", TargetUpdate)
	e.POST("/target/delete", TargetDelete)
}
