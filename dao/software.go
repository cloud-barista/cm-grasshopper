package dao

import (
	"errors"
	"github.com/cloud-barista/cm-grasshopper/db"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"gorm.io/gorm"
)

func SoftwareInstallStatusCreate(softwareInstallStatus *model.SoftwareInstallStatus) (*model.SoftwareInstallStatus, error) {
	result := db.DB.Create(softwareInstallStatus)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return softwareInstallStatus, nil
}

func SoftwareInstallStatusGet(executionID string) (*model.SoftwareInstallStatus, error) {
	software := &model.SoftwareInstallStatus{}

	// Ensure db.DB is not nil to avoid runtime panics
	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.DB.Where("execution_id = ?", executionID).First(software)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SoftwareInstallStatus not found with the provided execution_id")
		}
		return nil, err
	}

	return software, nil
}

func SoftwareInstallStatusGetList(page int, row int) (*[]model.SoftwareInstallStatus, error) {
	softwareInstallStatusList := &[]model.SoftwareInstallStatus{}
	// Ensure db.DB is not nil to avoid runtime panics
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
	}).Find(softwareInstallStatusList)

	err := result.Error
	if err != nil {
		return nil, err
	}

	return softwareInstallStatusList, nil
}

func SoftwareInstallStatusUpdate(softwareInstallStatus *model.SoftwareInstallStatus) error {
	result := db.DB.Model(&model.SoftwareInstallStatus{}).Where("execution_id = ?", softwareInstallStatus.ExecutionID).Updates(softwareInstallStatus)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SoftwareInstallStatusDelete(softwareInstallStatus *model.SoftwareInstallStatus) error {
	result := db.DB.Delete(softwareInstallStatus)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
