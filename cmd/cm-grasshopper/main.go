package main

import (
	"errors"
	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/db"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/lib/rsautil"
	"github.com/cloud-barista/cm-grasshopper/lib/software"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/server"
	"github.com/jollaman999/utils/cmd"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

func init() {
	err := config.PrepareConfigs()
	if err != nil {
		log.Fatalln(err)
	}

	err = logger.InitLogFile(common.RootPath+"/log", strings.ToLower(common.ModuleName))
	if err != nil {
		log.Panicln(err)
	}

	_, err = cmd.RunCMD("ansible-playbook -h")
	if err != nil {
		logger.Panicln(logger.ERROR, false,
			"'ansible-playbook' command not found please install Ansible!")
	}

	err = software.CheckAnsibleVersion()
	if err != nil {
		log.Panicln(err)
	}

	controller.OkMessage.Message = "API server is not ready"

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()
		server.Init()
	}()

	controller.OkMessage.Message = "Package Migration Config Database is not ready"
	err = db.Open()
	if err != nil {
		logger.Panicln(logger.ERROR, true, err.Error())
	}

	controller.OkMessage.Message = "Honeybee's RSA private key is not ready"
	privateKeyPath := common.RootPath + "/" + common.HoneybeePrivateKeyFileName
	if !fileutil.IsExist(privateKeyPath) {
		logger.Panicln(logger.ERROR, true, errors.New("Honeybee's private key not found ("+privateKeyPath+")"))
	}

	common.HoneybeePrivateKey, err = rsautil.ReadPrivateKey(privateKeyPath)
	if err != nil {
		logger.Panicln(logger.ERROR, true, "error occurred while reading Honeybee's private key")
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
