package controller

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/dao"
	joblib "github.com/cloud-barista/cm-grasshopper/lib/job"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	jobmodel "github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model/job"
	"github.com/labstack/echo/v4"
)

// GetJobStatus godoc
//
//	@ID				get-job-status
//	@Summary		Get Job Status
//	@Description	Get k8s migration job status by job ID.
//	@Tags			[Migration] K8s migration job APIs
//	@Accept			json
//	@Produce		json
//	@Param			jobId path string true "ID of the job."
//	@Success		200	{object}	jobmodel.ExecutionResponse	"Successfully got the job status."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the job status."
//	@Router			/job/status/{jobId} [get]
func GetJobStatus(c echo.Context) error {
	jobID := c.Param("jobId")
	if jobID == "" {
		return common.ReturnErrorMsg(c, "Please provide the jobId.")
	}

	job, err := dao.GetExecution(jobID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if joblib.DefaultManager != nil {
		if inMemory, exists := joblib.DefaultManager.GetJob(jobID); exists {
			job.Status = string(inMemory.Status)
			job.Progress = inMemory.Progress
			job.Message = inMemory.Message
			job.ErrorMessage = inMemory.ErrorMessage
			job.UpdatedAt = inMemory.UpdatedAt
			if !inMemory.FinishedAt.IsZero() {
				job.FinishedAt = inMemory.FinishedAt
			}
		}
	}
	return c.JSONPretty(http.StatusOK, toJobExecutionResponse(job), " ")
}

// GetJobLog godoc
//
//	@ID				get-job-log
//	@Summary		Get Job Log
//	@Description	Get k8s migration job log by job ID.
//	@Tags			[Migration] K8s migration job APIs
//	@Accept			json
//	@Produce		json
//	@Param			jobId path string true "ID of the job."
//	@Success		200	{object}	jobmodel.LogResponse	"Successfully got the job log."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the job log."
//	@Router			/job/log/{jobId} [get]
func GetJobLog(c echo.Context) error {
	jobID := c.Param("jobId")
	if jobID == "" {
		return common.ReturnErrorMsg(c, "Please provide the jobId.")
	}

	job, err := dao.GetExecution(jobID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	res := jobmodel.LogResponse{
		JobID:   job.JobID,
		Status:  job.Status,
		Message: job.Message,
	}

	if job.LogPath == "" {
		return common.ReturnErrorMsg(c, fmt.Sprintf("Log path for jobID %s not found", jobID))
	}

	content, err := os.ReadFile(job.LogPath)
	if err != nil {
		return common.ReturnErrorMsg(c, fmt.Sprintf("Failed to read log for jobID %s", jobID))
	}

	res.Log = string(content)
	return c.JSONPretty(http.StatusOK, res, " ")
}

// ListJobStatus godoc
//
//	@ID				list-job-status
//	@Summary		List Job Status
//	@Description	List k8s migration jobs.
//	@Tags			[Migration] K8s migration job APIs
//	@Accept			json
//	@Produce		json
//	@Param			page query int false "Page number."
//	@Param			row query int false "Rows per page."
//	@Success		200	{array}		jobmodel.ExecutionResponse	"Successfully listed jobs."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to list jobs."
//	@Router			/job/status [get]
func ListJobStatus(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	jobList, err := dao.ListExecutions(page, row)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	res := make([]jobmodel.ExecutionResponse, 0, len(*jobList))
	for i := range *jobList {
		res = append(res, toJobExecutionResponse(&(*jobList)[i]))
	}

	return c.JSONPretty(http.StatusOK, res, " ")
}

func deriveCurrentStage(message string) string {
	normalized := strings.ToLower(strings.TrimSpace(message))
	switch {
	case normalized == "":
		return ""
	case strings.Contains(normalized, "precheck"):
		return "precheck"
	case strings.Contains(normalized, "source backup") || strings.Contains(normalized, "waiting for backup"):
		return "backup"
	case strings.Contains(normalized, "sync into target cluster") || strings.Contains(normalized, "available on target cluster"):
		return "backup_sync"
	case strings.Contains(normalized, "target restore") || strings.Contains(normalized, "waiting for restore"):
		return "restore"
	case strings.Contains(normalized, "migration completed"):
		return "completed"
	default:
		return "processing"
	}
}

func toJobExecutionResponse(job *model.JobExecution) jobmodel.ExecutionResponse {
	if job == nil {
		return jobmodel.ExecutionResponse{}
	}

	return jobmodel.ExecutionResponse{
		JobID:             job.JobID,
		JobType:           job.JobType,
		ResourceType:      job.ResourceType,
		ResourceName:      job.ResourceName,
		SourceClusterName: job.SourceClusterName,
		TargetClusterName: job.TargetClusterName,
		SourceNamespace:   job.SourceNamespace,
		TargetNamespace:   job.TargetNamespace,
		Status:            job.Status,
		Progress:          job.Progress,
		Message:           job.Message,
		CurrentStage:      deriveCurrentStage(job.Message),
		Metadata:          job.Metadata,
		LogPath:           job.LogPath,
		ErrorMessage:      job.ErrorMessage,
		StartedAt:         job.StartedAt,
		UpdatedAt:         job.UpdatedAt,
		FinishedAt:        job.FinishedAt,
	}
}
