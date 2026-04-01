package controller

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cloud-barista/cm-grasshopper/dao"
	joblib "github.com/cloud-barista/cm-grasshopper/lib/job"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
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
//	@Success		200	{object}	model.JobExecution		"Successfully got the job status."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the job status."
//	@Router			/job/status/{jobId} [get]
func GetJobStatus(c echo.Context) error {
	jobID := c.Param("jobId")
	if jobID == "" {
		return common.ReturnErrorMsg(c, "Please provide the jobId.")
	}

	job, err := dao.JobExecutionGet(jobID)
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

	return c.JSONPretty(http.StatusOK, job, " ")
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
//	@Success		200	{object}	model.JobLogRes			"Successfully got the job log."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the job log."
//	@Router			/job/log/{jobId} [get]
func GetJobLog(c echo.Context) error {
	jobID := c.Param("jobId")
	if jobID == "" {
		return common.ReturnErrorMsg(c, "Please provide the jobId.")
	}

	job, err := dao.JobExecutionGet(jobID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	res := model.JobLogRes{
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
//	@Success		200	{array}		model.JobExecution		"Successfully listed jobs."
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to list jobs."
//	@Router			/job/status [get]
func ListJobStatus(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	jobList, err := dao.JobExecutionGetList(page, row)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, jobList, " ")
}
