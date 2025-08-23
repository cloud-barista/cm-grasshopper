package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/lib/software"
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
//	@Param			softwareMigrateReq body softwaremodel.SoftwareList true "Refined software list."
//	@Success		200	{object}	softwaremodel.MigrationList	"Successfully get software migration list."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to get software migration list."
//	@Router			/software/migration_list [post]
func GetSoftwareMigrationList(c echo.Context) error {
	sourceSoftwareList := new(softwaremodel.SoftwareList)
	err := c.Bind(sourceSoftwareList)
	if err != nil {
		return err
	}

	migrationListRes := software.MakeMigrationListRes(sourceSoftwareList)

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
//	@Param			softwareMigrateReq body model.SoftwareMigrateReq true "Software migrate request."
//	@Success		200	{object}	model.SoftwareMigrateRes	"Successfully migrated pieces of software."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to migrate pieces of software."
//	@Router			/software/migrate [post]
func MigrateSoftware(c echo.Context) error {
	softwareMigrateReq := new(model.SoftwareMigrateReq)
	err := c.Bind(softwareMigrateReq)
	if err != nil {
		return err
	}

	executionID := uuid.New().String()

	err = software.MigrateSoftware(executionID, &softwareMigrateReq.MigrationList,
		softwareMigrateReq.SourceConnectionInfoID, &softwareMigrateReq.Target)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SoftwareMigrateRes{
		ExecutionID:   executionID,
		MigrationList: softwareMigrateReq.MigrationList,
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
