package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/lib/software"
	"github.com/cloud-barista/cm-grasshopper/lib/ssh"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	softwaremodel "github.com/cloud-barista/cm-model/sw"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// GetSoftwareMigrationList godoc
//
//	@ID				get-migration-list
//	@Summary		Get Migration List
//	@Description	Get software migration list.
//	@Tags			[Migration] Software migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			softwareMigrateReq body softwaremodel.SourceGroupSoftwareProperty true "Refined software list."
//	@Success		200	{object}	softwaremodel.SourceGroupSoftwareProperty	"Successfully get software migration list."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to get software migration list."
//	@Router			/software/migration_list [post]
func GetSoftwareMigrationList(c echo.Context) error {
	sourceGroupSoftwareProperty := new(softwaremodel.SourceGroupSoftwareProperty)
	err := c.Bind(sourceGroupSoftwareProperty)
	if err != nil {
		return err
	}

	migrationListRes, err := software.MakeMigrationListRes(sourceGroupSoftwareProperty)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, *migrationListRes, " ")
}

// MigrateSoftware godoc
//
//	@ID				migrate-software
//	@Summary		Migrate Software
//	@Description	Migrate pieces of software to target.
//	@Tags			[Migration] Software migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			nsId query string false "ID of target namespace."
//	@Param			mciId query string false "ID of target MCI."
//	@Param			softwareMigrateReq body model.SoftwareMigrateReq true "Software migrate request."
//	@Success		200	{object}	model.SoftwareMigrateRes	"Successfully migrated pieces of software."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to migrate pieces of software."
//	@Router			/software/migrate [post]
func MigrateSoftware(c echo.Context) error {
	softwareMigrateReq := new(softwaremodel.TargetGroupSoftwareProperty)
	err := c.Bind(softwareMigrateReq)
	if err != nil {
		return err
	}

	nsIdStr := c.QueryParam("nsId")
	mciIdStr := c.QueryParam("mciId")

	executionID := uuid.New().String()

	type ex struct {
		ExID         string
		ExList       *softwaremodel.MigrationList
		ExStatusList []model.ExecutionStatus
		SourceClient *ssh.Client
		TargetClient *ssh.Client
	}

	var exList = make([]ex, 0)
	var targetMappings []model.TargetMapping

	for _, server := range softwareMigrateReq.Servers {
		executionStatusList, sourceClient, targetClient, target, err :=
			software.PrepareSoftwareMigration(executionID, &server.MigrationList, server.SourceConnectionInfoID,
				nsIdStr, mciIdStr)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}

		exList = append(exList, ex{
			ExID:         executionID,
			ExList:       &server.MigrationList,
			ExStatusList: executionStatusList,
			SourceClient: sourceClient,
			TargetClient: targetClient,
		})

		targetMappings = append(targetMappings, model.TargetMapping{
			SourceConnectionInfoID: server.SourceConnectionInfoID,
			Target:                 *target,
		})
	}

	for _, e := range exList {
		go software.MigrateSoftware(e.ExID, e.ExList, e.ExStatusList, e.SourceClient, e.TargetClient)
	}

	return c.JSONPretty(http.StatusOK, model.SoftwareMigrateRes{
		ExecutionID:    executionID,
		TargetMappings: targetMappings,
	}, " ")
}

// GetSoftwareMigrationLog godoc
//
//	@ID				get-software-migration-log
//	@Summary		Get Software Migration Log
//	@Description	Get the software migration log.
//	@Tags			[Migration] Software migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			executionId path string true "ID of the software migration execution."
//	@Success		200	{object}	model.MigrationLogRes	"Successfully get the software migration log"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the software migration log"
//	@Router			/software/migrate/log/{executionId} [get]
func GetSoftwareMigrationLog(c echo.Context) error {
	executionID := c.Param("executionId")
	if executionID == "" {
		return common.ReturnErrorMsg(c, "Please provide the executionId.")
	}

	path, err := filepath.Abs(filepath.Join(config.CMGrasshopperConfig.CMGrasshopper.Software.LogFolder, executionID))
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return common.ReturnErrorMsg(c, fmt.Sprintf("Log path for executionID %s not found", executionID))
	}

	response := model.MigrationLogRes{
		UUID: executionID,
	}

	if content, err := os.ReadFile(filepath.Join(path, "install.log")); err == nil {
		response.InstallLog = string(content)
	}

	if content, err := os.ReadFile(filepath.Join(path, "migration.log")); err == nil {
		response.MigrationLog = string(content)
	}

	if response.InstallLog == "" && response.MigrationLog == "" {
		return common.ReturnErrorMsg(c, fmt.Sprintf("No log files found for executionID %s", executionID))
	}

	return c.JSONPretty(http.StatusOK, response, " ")
}
