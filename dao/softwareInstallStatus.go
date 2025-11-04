package dao

import (
	"errors"

	"github.com/cloud-barista/cm-grasshopper/db"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	softwaremodel "github.com/cloud-barista/cm-model/sw"
	"gorm.io/gorm"
)

func SoftwareMigrationStatusCreate(softwareInstallStatus *model.SoftwareMigrationStatus) (*model.SoftwareMigrationStatus, error) {
	result := db.DB.Create(softwareInstallStatus)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return softwareInstallStatus, nil
}

func SoftwareMigrationStatusGet(executionID string, sourceConnectionInfoID string, softwareInstallType softwaremodel.SoftwareType, order int) (*model.SoftwareMigrationStatus, error) {
	software := &model.SoftwareMigrationStatus{}

	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.DB.Where("execution_id = ? AND source_connection_info_id = ? AND software_install_type = ? AND `order` = ?", executionID, sourceConnectionInfoID, softwareInstallType, order).First(software)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("SoftwareMigrationStatus not found with the provided execution_id, source_connection_info_id, software_install_type, and order")
		}
		return nil, err
	}

	return software, nil
}

func SoftwareMigrationStatusGetList(executionID string, target model.Target, sourceConnectionInfoID string, page int, row int) (*[]model.SoftwareMigrationStatus, error) {
	softwareInstallStatusList := &[]model.SoftwareMigrationStatus{}

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
	}).Where("execution_id = ? AND source_connection_info_id = ? AND namespace_id = ? AND mci_id = ? AND vm_id = ?",
		executionID, sourceConnectionInfoID, target.NamespaceID, target.MCIID, target.VMID).Find(softwareInstallStatusList)

	err := result.Error
	if err != nil {
		return nil, err
	}

	return softwareInstallStatusList, nil
}

func SoftwareMigrationStatusUpdate(softwareInstallStatus *model.SoftwareMigrationStatus) error {
	result := db.DB.Model(&model.SoftwareMigrationStatus{}).Where("execution_id = ? AND source_connection_info_id = ? AND software_install_type = ? AND `order` = ?", softwareInstallStatus.ExecutionID, softwareInstallStatus.SourceConnectionInfoID, softwareInstallStatus.SoftwareInstallType, softwareInstallStatus.Order).Updates(softwareInstallStatus)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SoftwareMigrationStatusDelete(softwareInstallStatus *model.SoftwareMigrationStatus) error {
	result := db.DB.Delete(softwareInstallStatus)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
