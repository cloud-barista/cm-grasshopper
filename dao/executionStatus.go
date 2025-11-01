package dao

import (
	"errors"

	"github.com/cloud-barista/cm-grasshopper/db"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"gorm.io/gorm"
)

func ExecutionStatusCreate(executionStatus *model.ExecutionStatus) (*model.ExecutionStatus, error) {
	result := db.DB.Create(executionStatus)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return executionStatus, nil
}

func ExecutionStatusGet(executionID string) (*model.ExecutionStatus, error) {
	status := &model.ExecutionStatus{}

	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.DB.Where("execution_id = ?", executionID).First(status)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("ExecutionStatus not found with the provided execution_id")
		}
		return nil, err
	}

	return status, nil
}

func ExecutionStatusGetList(page int, row int) (*[]model.ExecutionStatus, error) {
	executionStatusList := &[]model.ExecutionStatus{}

	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.DB.Scopes(func(d *gorm.DB) *gorm.DB {
		var filtered = d

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
	}).Find(executionStatusList)

	err := result.Error
	if err != nil {
		return nil, err
	}

	return executionStatusList, nil
}

func ExecutionStatusUpdate(executionStatus *model.ExecutionStatus) error {
	result := db.DB.Model(&model.ExecutionStatus{}).Where("execution_id = ?", executionStatus.ExecutionID).Updates(executionStatus)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func ExecutionStatusDelete(executionStatus *model.ExecutionStatus) error {
	result := db.DB.Delete(executionStatus)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
