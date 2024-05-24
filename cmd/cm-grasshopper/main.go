package main

import (
	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/db"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/server"
	"github.com/jollaman999/utils/logger"
	"github.com/jollaman999/utils/syscheck"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func init() {
	err := syscheck.CheckRoot()
	if err != nil {
		log.Fatalln(err)
	}

	err = config.PrepareConfigs()
	if err != nil {
		log.Fatalln(err)
	}

	err = logger.InitLogFile(common.RootPath+"/log", strings.ToLower(common.ModuleName))
	if err != nil {
		log.Panicln(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()
		controller.OkMessage.Message = "API server is not ready"
		server.Init()
	}()

	controller.OkMessage.Message = "Database is not ready"
	err = db.Open()
	if err != nil {
		logger.Panicln(logger.ERROR, true, err.Error())
	}

	controller.OkMessage.Message = "CM-Grasshopper API server is ready"
	controller.IsReady = true

	wg.Wait()
}

func end() {
	db.Close()

	logger.CloseLogFile()
}

func main() {
	// Catch the exit signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Println(logger.INFO, false, "Exiting "+common.ModuleName+" module...")
		end()
		os.Exit(0)
	}()
}
