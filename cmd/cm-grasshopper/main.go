package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

const shutdownTimeout = 30 * time.Second

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
}

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	controller.SetOkMessage("API server is not ready")

	controller.SetOkMessage("Package Migration Config Database is not ready")
	if err := db.Open(); err != nil {
		logger.Panicln(logger.ERROR, true, err.Error())
	}

	controller.SetOkMessage("Software migration dependencies are not ready")
	if err := initSoftwareMigrationDependencies(); err != nil {
		logger.Panicln(logger.ERROR, true, err.Error())
	}

	controller.SetOkMessage("K8s migration dependencies are not ready")
	if err := initK8sMigrationDependencies(); err != nil {
		logger.Panicln(logger.ERROR, true, err.Error())
	}

	controller.SetOkMessage("CM-Grasshopper API server is ready")
	controller.SetReady(true)

	e := server.New()
	server.PrintBanner()

	serverErr := make(chan error, 1)
	go func() {
		err := e.Start(":" + config.CMGrasshopperConfig.CMGrasshopper.Listen.Port)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case sig := <-sigChan:
		logger.Println(logger.INFO, false, "Received signal "+sig.String()+", exiting "+common.ModuleName+" module...")
	case err := <-serverErr:
		logger.Println(logger.ERROR, true, "API server error: "+err.Error())
	}

	controller.SetReady(false)
	controller.SetOkMessage("CM-Grasshopper API server is shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		logger.Println(logger.ERROR, false, "echo shutdown error: "+err.Error())
	}

	grasshopperjob.StopDefaultManager(shutdownCtx)

	db.Close()
	logger.CloseLogFile()
}
