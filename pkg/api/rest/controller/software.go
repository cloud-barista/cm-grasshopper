package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cloud-barista/cm-grasshopper/dao"
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
//	@Param			sourceSoftwareModel body softwaremodel.SourceSoftwareModel true "Refined software list."
//	@Success		200	{object}	softwaremodel.SourceSoftwareModel	"Successfully get software migration list."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to get software migration list."
//	@Router			/software/migration_list [post]
func GetSoftwareMigrationList(c echo.Context) error {
	sourceSoftwareModel := new(softwaremodel.SourceSoftwareModel)
	err := c.Bind(sourceSoftwareModel)
	if err != nil {
		return err
	}

	migrationListRes, err := software.MakeMigrationListRes(sourceSoftwareModel)
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
//	@Param			nsId query string true "ID of target namespace."
//	@Param			mciId query string true "ID of target MCI."
//	@Param			targetSoftwareModel body softwaremodel.TargetSoftwareModel true "Software migrate request."
//	@Success		200	{object}	model.SoftwareMigrateRes	"Successfully migrated pieces of software."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to migrate pieces of software."
//	@Router			/software/migrate [post]
func MigrateSoftware(c echo.Context) error {
	targetSoftwareModel := new(softwaremodel.TargetSoftwareModel)
	err := c.Bind(targetSoftwareModel)
	if err != nil {
		return err
	}

	nsIdStr := c.QueryParam("nsId")
	if nsIdStr == "" {
		return common.ReturnErrorMsg(c, "Please provide the nsId.")
	}

	mciIdStr := c.QueryParam("mciId")
	if mciIdStr == "" {
		return common.ReturnErrorMsg(c, "Please provide the mciId.")
	}

	executionID := uuid.New().String()

	exList, targetMappings, err :=
		software.PrepareSoftwareMigration(executionID, targetSoftwareModel.TargetSoftwareModel.Servers,
			nsIdStr, mciIdStr)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	for _, ex := range exList {
		go software.MigrateSoftware(&ex)
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

// GetSoftwareMigrationStatus godoc
//
//	@ID				get-software-migration-status
//	@Summary		Get Software Migration Status
//	@Description	Get the software migration status.
//	@Tags			[Migration] Software migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			executionId path string true "ID of the software migration execution."
//	@Success		200	{object}	model.MigrationLogRes	"Successfully get the software migration status"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the software migration status"
//	@Router			/software/migrate/status/{executionId} [get]
func GetSoftwareMigrationStatus(c echo.Context) error {
	executionID := c.Param("executionId")
	if executionID == "" {
		return common.ReturnErrorMsg(c, "Please provide the executionId.")
	}

	ex, err := dao.ExecutionStatusGet(executionID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	var targetMappings []model.TargetMappingWithSoftwareMigrationList

	for _, target := range ex.TargetMappings {
		list, err := dao.SoftwareMigrationStatusGetList(executionID, target.Target, target.SourceConnectionInfoID, 0, 0)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}

		var softwareMigrationStatusList []model.SoftwareMigrationStatusSoftwareStatusOnly

		for _, sm := range *list {
			softwareMigrationStatusList = append(softwareMigrationStatusList, model.SoftwareMigrationStatusSoftwareStatusOnly{
				Order:               sm.Order,
				SoftwareName:        sm.SoftwareName,
				SoftwareVersion:     sm.SoftwareVersion,
				SoftwareInstallType: sm.SoftwareInstallType,
				Status:              sm.Status,
				StartedAt:           sm.StartedAt,
				UpdatedAt:           sm.UpdatedAt,
				ErrorMessage:        sm.ErrorMessage,
			})
		}

		targetMappings = append(targetMappings, model.TargetMappingWithSoftwareMigrationList{
			SourceConnectionInfoID:      target.SourceConnectionInfoID,
			Target:                      target.Target,
			Status:                      target.Status,
			SoftwareMigrationStatusList: softwareMigrationStatusList,
		})
	}

	executionStatus := model.ExecutionStatusWithSoftwareMigrationList{
		ExecutionID:    executionID,
		TargetMappings: targetMappings,
		StartedAt:      ex.StartedAt,
		FinishedAt:     ex.FinishedAt,
	}

	return c.JSONPretty(http.StatusOK, executionStatus, " ")
}

// ListSoftwareMigrationStatus godoc
//
//	@ID				list-software-migration-status
//	@Summary		List Software Migration Status
//	@Description	Get a list of software migration status.
//	@Tags			[Migration] Software migration APIs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	[]model.SoftwareMigrateRes	"Successfully get a list of software migration status"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the software migration status"
//	@Router			/software/migrate/status [get]
func ListSoftwareMigrationStatus(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	softwareMigrationStatusList, err := dao.ExecutionStatusGetList(page, row)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, softwareMigrationStatusList, " ")
}
