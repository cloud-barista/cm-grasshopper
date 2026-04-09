package controller

import (
	"net/http"

	joblib "github.com/cloud-barista/cm-grasshopper/lib/job"
	k8scommon "github.com/cloud-barista/cm-grasshopper/lib/k8s/common"
	k8svelero "github.com/cloud-barista/cm-grasshopper/lib/k8s/velero"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	commonmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/common"
	jobmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/job"
	veleromodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/velero"
	"github.com/labstack/echo/v4"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

var veleroService = k8svelero.NewService()

func getClusterFromRole(req *commonmodel.MultiClusterEnvelope, role string) *commonmodel.ClusterAccess {
	if role == "source" {
		return req.SourceCluster
	}
	return req.TargetCluster
}

// VeleroHealth godoc
//
//	@ID				velero-health
//	@Summary		Check Velero Health
//	@Description	Check Velero availability on source or target cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			role path string true "Cluster role. Use source or target."
//	@Param			request body commonmodel.MultiClusterEnvelope true "Cluster access request."
//	@Success		200	{object}	veleromodel.HealthResponse	"Successfully checked Velero health."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to check Velero health."
//	@Router			/velero/{role}/health [post]
func VeleroHealth(c echo.Context) error {
	role := c.Param("role")
	req := new(commonmodel.MultiClusterEnvelope)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	cluster := getClusterFromRole(req, role)
	if err := k8scommon.ValidateClusterAccess(cluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	result, err := veleroService.HealthCheck(c.Request().Context(), cluster)
	if err != nil {
		return common.ReturnInternalError(c, err, "velero health check failed")
	}

	return c.JSONPretty(http.StatusOK, result, " ")
}

// VeleroInstall godoc
//
//	@ID				velero-install
//	@Summary		Install Velero
//	@Description	Install or upgrade Velero on source or target cluster using MinIO.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			role path string true "Cluster role. Use source or target."
//	@Param			request body veleromodel.InstallRequest true "Velero install request."
//	@Success		200	{object}	jobmodel.StartResponse	"Successfully started Velero installation."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to install Velero."
//	@Router			/velero/{role}/install [post]
func VeleroInstall(c echo.Context) error {
	role := c.Param("role")
	req := new(veleromodel.InstallRequest)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	cluster := getClusterFromRole(&req.MultiClusterEnvelope, role)
	if err := k8scommon.ValidateClusterAccess(cluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if req.Storage == nil || req.Storage.MinIO == nil {
		return common.ReturnErrorMsg(c, "minio access is required")
	}
	if err := k8scommon.ValidateMinIOAccess(req.Storage.MinIO); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	job, err := veleroService.InstallAsync(role, cluster, req.Storage.MinIO, req.Install.Force, req.Install.VolumeBackupMode)
	if err != nil {
		return common.ReturnInternalError(c, err, "velero install failed")
	}

	return c.JSONPretty(http.StatusOK, toJobStartResponse(job), " ")
}

// ListBackups godoc
//
//	@ID				list-velero-backups
//	@Summary		List Backups
//	@Description	List Velero backups on source cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			request body commonmodel.MultiClusterEnvelope true "Source cluster access request."
//	@Success		200	{array}		veleromodel.BackupResponse	"Successfully listed backups."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to list backups."
//	@Router			/velero/source/backups/list [post]
func ListBackups(c echo.Context) error {
	req := new(commonmodel.MultiClusterEnvelope)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.SourceCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	result, err := veleroService.ListBackups(c.Request().Context(), req.SourceCluster)
	if err != nil {
		return common.ReturnInternalError(c, err, "list backups failed")
	}
	return c.JSONPretty(http.StatusOK, result, " ")
}

// GetBackup godoc
//
//	@ID				get-velero-backup
//	@Summary		Get Backup
//	@Description	Get Velero backup detail on source cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			name path string true "Backup name."
//	@Param			request body commonmodel.MultiClusterEnvelope true "Source cluster access request."
//	@Success		200	{object}	veleromodel.BackupResponse	"Successfully got backup detail."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get backup detail."
//	@Router			/velero/source/backups/{name} [post]
func GetBackup(c echo.Context) error {
	req := new(commonmodel.MultiClusterEnvelope)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.SourceCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	result, err := veleroService.GetBackup(c.Request().Context(), req.SourceCluster, c.Param("name"))
	if err != nil {
		return common.ReturnInternalError(c, err, "get backup failed")
	}
	return c.JSONPretty(http.StatusOK, result, " ")
}

// CreateBackup godoc
//
//	@ID				create-velero-backup
//	@Summary		Create Backup
//	@Description	Create Velero backup on source cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			request body veleromodel.BackupRequest true "Velero backup request."
//	@Success		200	{object}	veleromodel.BackupResponse	"Successfully created backup."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to create backup."
//	@Router			/velero/source/backups [post]
func CreateBackup(c echo.Context) error {
	req := new(veleromodel.BackupRequest)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.SourceCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	result, err := veleroService.CreateBackup(c.Request().Context(), req.SourceCluster, req.Backup)
	if err != nil {
		return common.ReturnInternalError(c, err, "create backup failed")
	}
	return c.JSONPretty(http.StatusOK, result, " ")
}

// DeleteBackup godoc
//
//	@ID				delete-velero-backup
//	@Summary		Delete Backup
//	@Description	Delete Velero backup on source cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			name path string true "Backup name."
//	@Param			request body commonmodel.MultiClusterEnvelope true "Source cluster access request."
//	@Success		200	{object}	model.SimpleMsg			"Successfully deleted backup."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to delete backup."
//	@Router			/velero/source/backups/{name}/delete [post]
func DeleteBackup(c echo.Context) error {
	req := new(commonmodel.MultiClusterEnvelope)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.SourceCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if err := veleroService.DeleteBackup(c.Request().Context(), req.SourceCluster, c.Param("name")); err != nil {
		return common.ReturnInternalError(c, err, "delete backup failed")
	}
	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "backup deleted"}, " ")
}

// ValidateBackup godoc
//
//	@ID				validate-velero-backup
//	@Summary		Validate Backup
//	@Description	Validate Velero backup status on source cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			name path string true "Backup name."
//	@Param			request body commonmodel.MultiClusterEnvelope true "Source cluster access request."
//	@Success		200	{object}	veleromodel.BackupResponse	"Successfully validated backup."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to validate backup."
//	@Router			/velero/source/backups/{name}/validate [post]
func ValidateBackup(c echo.Context) error {
	req := new(commonmodel.MultiClusterEnvelope)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.SourceCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	result, err := veleroService.ValidateBackup(c.Request().Context(), req.SourceCluster, c.Param("name"))
	if err != nil {
		return common.ReturnInternalError(c, err, "validate backup failed")
	}
	return c.JSONPretty(http.StatusOK, result, " ")
}

// ListRestores godoc
//
//	@ID				list-velero-restores
//	@Summary		List Restores
//	@Description	List Velero restores on target cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			request body commonmodel.MultiClusterEnvelope true "Target cluster access request."
//	@Success		200	{array}		veleromodel.RestoreResponse	"Successfully listed restores."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to list restores."
//	@Router			/velero/target/restores/list [post]
func ListRestores(c echo.Context) error {
	req := new(commonmodel.MultiClusterEnvelope)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.TargetCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	result, err := veleroService.ListRestores(c.Request().Context(), req.TargetCluster)
	if err != nil {
		return common.ReturnInternalError(c, err, "list restores failed")
	}
	return c.JSONPretty(http.StatusOK, result, " ")
}

// GetRestore godoc
//
//	@ID				get-velero-restore
//	@Summary		Get Restore
//	@Description	Get Velero restore detail on target cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			name path string true "Restore name."
//	@Param			request body commonmodel.MultiClusterEnvelope true "Target cluster access request."
//	@Success		200	{object}	veleromodel.RestoreResponse	"Successfully got restore detail."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get restore detail."
//	@Router			/velero/target/restores/{name} [post]
func GetRestore(c echo.Context) error {
	req := new(commonmodel.MultiClusterEnvelope)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.TargetCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	result, err := veleroService.GetRestore(c.Request().Context(), req.TargetCluster, c.Param("name"))
	if err != nil {
		return common.ReturnInternalError(c, err, "get restore failed")
	}
	return c.JSONPretty(http.StatusOK, result, " ")
}

// CreateRestore godoc
//
//	@ID				create-velero-restore
//	@Summary		Create Restore
//	@Description	Create Velero restore on target cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			request body veleromodel.RestoreRequest true "Velero restore request."
//	@Success		200	{object}	veleromodel.RestoreResponse	"Successfully created restore."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to create restore."
//	@Router			/velero/target/restores [post]
func CreateRestore(c echo.Context) error {
	req := new(veleromodel.RestoreRequest)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.TargetCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if req.Restore.BackupName == "" {
		return common.ReturnErrorMsg(c, "backupName is required")
	}

	result, err := veleroService.CreateRestore(c.Request().Context(), req.TargetCluster, req.Restore)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return common.ReturnErrorMsg(c, "backup is not available on target cluster yet; wait for backup sync and retry")
		}
		return common.ReturnInternalError(c, err, "create restore failed")
	}
	return c.JSONPretty(http.StatusOK, result, " ")
}

// DeleteRestore godoc
//
//	@ID				delete-velero-restore
//	@Summary		Delete Restore
//	@Description	Delete Velero restore on target cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			name path string true "Restore name."
//	@Param			request body commonmodel.MultiClusterEnvelope true "Target cluster access request."
//	@Success		200	{object}	model.SimpleMsg			"Successfully deleted restore."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to delete restore."
//	@Router			/velero/target/restores/{name}/delete [post]
func DeleteRestore(c echo.Context) error {
	req := new(commonmodel.MultiClusterEnvelope)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.TargetCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if err := veleroService.DeleteRestore(c.Request().Context(), req.TargetCluster, c.Param("name")); err != nil {
		return common.ReturnInternalError(c, err, "delete restore failed")
	}
	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "restore deleted"}, " ")
}

// ValidateRestore godoc
//
//	@ID				validate-velero-restore
//	@Summary		Validate Restore
//	@Description	Validate Velero restore status on target cluster.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			name path string true "Restore name."
//	@Param			request body commonmodel.MultiClusterEnvelope true "Target cluster access request."
//	@Success		200	{object}	veleromodel.RestoreResponse	"Successfully validated restore."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to validate restore."
//	@Router			/velero/target/restores/{name}/validate [post]
func ValidateRestore(c echo.Context) error {
	req := new(commonmodel.MultiClusterEnvelope)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.TargetCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	result, err := veleroService.ValidateRestore(c.Request().Context(), req.TargetCluster, c.Param("name"))
	if err != nil {
		return common.ReturnInternalError(c, err, "validate restore failed")
	}
	return c.JSONPretty(http.StatusOK, result, " ")
}

// VeleroMigrationPrecheck godoc
//
//	@ID				velero-migration-precheck
//	@Summary		Precheck Migration
//	@Description	Check source cluster, target cluster, and MinIO before executing Velero migration.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			request body veleromodel.MigrationPrecheckRequest true "Velero migration precheck request."
//	@Success		200	{object}	veleromodel.PrecheckResponse	"Successfully completed precheck."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to run migration precheck."
//	@Router			/velero/migration/precheck [post]
func VeleroMigrationPrecheck(c echo.Context) error {
	req := new(veleromodel.MigrationPrecheckRequest)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.SourceCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.TargetCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if req.Storage == nil || req.Storage.MinIO == nil {
		return common.ReturnErrorMsg(c, "minio access is required")
	}
	if err := k8scommon.ValidateMinIOAccess(req.Storage.MinIO); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	result, err := veleroService.Precheck(c.Request().Context(), req.SourceCluster, req.TargetCluster, req.Storage.MinIO, req.Precheck)
	if err != nil {
		return common.ReturnInternalError(c, err, "velero migration precheck failed")
	}
	if c.QueryParam("verbose") == "true" {
		return c.JSONPretty(http.StatusOK, result, " ")
	}
	return c.JSONPretty(http.StatusOK, k8svelero.CompactPrecheckResponse(result), " ")
}

// VeleroMigrationExecute godoc
//
//	@ID				velero-migration-execute
//	@Summary		Execute Migration
//	@Description	Run source backup and target restore as a single asynchronous Velero migration job.
//	@Tags			[Migration] Velero migration APIs
//	@Accept			json
//	@Produce		json
//	@Param			request body veleromodel.MigrationExecuteRequest true "Velero migration execute request."
//	@Success		200	{object}	jobmodel.StartResponse	"Successfully started migration job."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to start migration job."
//	@Router			/velero/migration/execute [post]
func VeleroMigrationExecute(c echo.Context) error {
	req := new(veleromodel.MigrationExecuteRequest)
	if err := c.Bind(req); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.SourceCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if err := k8scommon.ValidateClusterAccess(req.TargetCluster); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	if req.Storage == nil || req.Storage.MinIO == nil {
		return common.ReturnErrorMsg(c, "minio access is required")
	}
	if err := k8scommon.ValidateMinIOAccess(req.Storage.MinIO); err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	job, err := veleroService.ExecuteMigrationAsync(req.SourceCluster, req.TargetCluster, req.Storage.MinIO, req.Migration)
	if err != nil {
		return common.ReturnInternalError(c, err, "velero migration execute failed")
	}
	return c.JSONPretty(http.StatusOK, toJobStartResponse(job), " ")
}

func toJobStartResponse(job *joblib.Info) jobmodel.StartResponse {
	if job == nil {
		return jobmodel.StartResponse{}
	}

	return jobmodel.StartResponse{
		JobID:        job.JobID,
		JobType:      job.JobType,
		ResourceType: job.ResourceType,
		ResourceName: job.ResourceName,
		Status:       string(job.Status),
		Progress:     job.Progress,
		Message:      job.Message,
		Metadata:     job.Metadata,
		LogPath:      job.LogPath,
		ErrorMessage: job.ErrorMessage,
		StartedAt:    job.StartedAt,
		UpdatedAt:    job.UpdatedAt,
		FinishedAt:   job.FinishedAt,
	}
}
