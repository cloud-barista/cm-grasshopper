package main

import (
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/db"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	grasshopperjob "github.com/cloud-barista/cm-grasshopper/lib/job"
	"github.com/cloud-barista/cm-grasshopper/lib/rsautil"
	"github.com/cloud-barista/cm-grasshopper/lib/software"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/controller"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/server"
	"github.com/jollaman999/utils/cmd"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
)

func initSoftwareMigrationDependencies() error {
	_, err := cmd.RunCMD("ansible-playbook -h")
	if err != nil {
		return errors.New("'ansible-playbook' command not found please install Ansible")
	}

	err = software.CheckAnsibleVersion()
	if err != nil {
		return err
	}

	privateKeyPath := common.RootPath + "/" + common.HoneybeePrivateKeyFileName
	if !fileutil.IsExist(privateKeyPath) {
		return errors.New("Honeybee's private key not found (" + privateKeyPath + ")")
	}

	common.HoneybeePrivateKey, err = rsautil.ReadPrivateKey(privateKeyPath)
	if err != nil {
		return errors.New("error occurred while reading Honeybee's private key")
	}

	return nil
}

func initK8sMigrationDependencies() error {
	return grasshopperjob.InitDefaultManager(
		config.CMGrasshopperConfig.CMGrasshopper.K8s.JobWorkerCount,
		config.CMGrasshopperConfig.CMGrasshopper.K8s.JobLogFolder,
	)
}

func init() {
	err := config.PrepareConfigs()
	if err != nil {
		log.Fatalln(err)
	}

	err = logger.InitLogFile(common.RootPath+"/log", strings.ToLower(common.ModuleName))
	if err != nil {
		log.Panicln(err)
	}

	controller.OkMessage.Message = "API server is not ready"

	controller.OkMessage.Message = "Package Migration Config Database is not ready"
	err = db.Open()
	if err != nil {
		logger.Panicln(logger.ERROR, true, err.Error())
	}

	controller.OkMessage.Message = "Software migration dependencies are not ready"
	err = initSoftwareMigrationDependencies()
	if err != nil {
		logger.Panicln(logger.ERROR, true, err.Error())
	}

	controller.OkMessage.Message = "K8s migration dependencies are not ready"
	err = initK8sMigrationDependencies()
	if err != nil {
		logger.Panicln(logger.ERROR, true, err.Error())
	}

	controller.OkMessage.Message = "CM-Grasshopper API server is ready"
	controller.IsReady = true

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			wg.Done()
		}()
		server.Init()
	}()

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
