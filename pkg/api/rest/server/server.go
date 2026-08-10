package server

import (
	"fmt"
	"net"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/middlewares"

	"github.com/cloud-barista/cm-grasshopper/lib/config"
	_ "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/docs" // Grasshopper Documentation
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/route"
	"github.com/jollaman999/utils/logger"
	"github.com/labstack/echo/v4"
)

const (
	infoColor   = "\033[1;34m%s\033[0m"
	noticeColor = "\033[1;36m%s\033[0m"
)

const (
	website = " https://github.com/cloud-barista/cm-grasshopper"
)

func getLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		logger.Println(logger.ERROR, true, err)
		return ""
	}
	defer func() {
		_ = conn.Close()
	}()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	localIP := strings.Split(localAddr.String(), ":")
	if len(localIP) == 0 {
		logger.Println(logger.ERROR, true, "Failed to get local IP.")
		return ""
	}

	return localIP[0]
}

// @title CM-Grasshopper REST API
// @version latest
// @description Software migration management module

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath /grasshopper

// New builds the echo server with all routes registered. The caller is
// responsible for starting and gracefully shutting it down so the lifecycle
// can be coordinated with the rest of the process.
func New() *echo.Echo {
	e := echo.New()

	e.Use(middlewares.CustomLogger())

	// Hide Echo Banner
	e.HideBanner = true

	route.Software(e)
	route.Job(e)
	route.Velero(e)
	route.RegisterSwagger(e)
	route.RegisterUtility(e)

	return e
}

// PrintBanner prints the CM-Grasshopper repository link and the API docs URL.
// Called by the caller right before starting the server so the user sees the
// dashboard URL on stdout.
func PrintBanner() {
	endpoint := getLocalIP() + ":" + config.CMGrasshopperConfig.CMGrasshopper.Listen.Port
	apiDocsDashboard := " http://" + endpoint + "/" + strings.ToLower(common.ShortModuleName) + "/api/index.html"

	fmt.Println("\n ")
	fmt.Println(" CM-Grasshopper repository:")
	fmt.Printf(infoColor, website)
	fmt.Println("\n ")
	fmt.Println(" API Docs Dashboard:")
	fmt.Printf(noticeColor, apiDocsDashboard)
	fmt.Println("\n ")
}
