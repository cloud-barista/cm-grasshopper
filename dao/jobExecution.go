package dao

import (
	"errors"

	"github.com/cloud-barista/cm-grasshopper/db"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"gorm.io/gorm"
)

func JobExecutionCreate(job *model.JobExecution) (*model.JobExecution, error) {
	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.DB.Create(job)
	if result.Error != nil {
		return nil, result.Error
	}

	return job, nil
}

func JobExecutionGet(jobID string) (*model.JobExecution, error) {
	job := &model.JobExecution{}

	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.DB.Where("job_id = ?", jobID).First(job)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("JobExecution not found with the provided job_id")
		}
		return nil, result.Error
	}

	return job, nil
}

func JobExecutionGetList(page int, row int) (*[]model.JobExecution, error) {
	jobList := &[]model.JobExecution{}

	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.DB.Scopes(func(d *gorm.DB) *gorm.DB {
		filtered := d.Order("updated_at desc")

		if page != 0 && row != 0 {
			offset := (page - 1) * row
			return filtered.Offset(offset).Limit(row)
		} else if row != 0 && page == 0 {
			filtered.Error = errors.New("row is not 0 but page is 0")
			return filtered
		} else if page != 0 && row == 0 {
			filtered.Error = errors.New("page is not 0 but row is 0")
			return filtered
		}
		return filtered
	}).Find(jobList)

	if result.Error != nil {
		return nil, result.Error
	}

	return jobList, nil
}

func JobExecutionUpdate(job *model.JobExecution) error {
	if db.DB == nil {
		return errors.New("database connection is not initialized")
	}

	result := db.DB.Model(&model.JobExecution{}).Where("job_id = ?", job.JobID).Updates(job)
	return result.Error
}
