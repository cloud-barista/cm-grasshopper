package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	softwaremodel "github.com/cloud-barista/cm-model/sw"
)

type Source struct {
	ConnectionID string `json:"connection_id" yaml:"connection_uuid" validate:"required"`
}

type Target struct {
	NamespaceID string `json:"namespace_id" validate:"required"`
	MCIID       string `json:"mci_id" validate:"required"`
	VMID        string `json:"vm_id" validate:"required"`
}

type MigrationLogRes struct {
	UUID         string `json:"uuid"`
	InstallLog   string `json:"install_log"`
	MigrationLog string `json:"migration_log"`
}

type SoftwareMigrateReq struct {
	SourceConnectionInfoID string                      `json:"source_connection_info_id" validate:"required"`
	Target                 Target                      `json:"target" validate:"required"`
	MigrationList          softwaremodel.MigrationList `json:"migration_list"`
}

type TargetMapping struct {
	SourceConnectionInfoID string `json:"source_connection_info_id"`
	Target                 Target `json:"target" validate:"required"`
}

type SoftwareMigrateRes struct {
	ExecutionID    string          `json:"execution_id"`
	TargetMappings []TargetMapping `json:"target_mappings"`
}

type SoftwareMigrationStatus struct {
	ExecutionID            string                     `json:"execution_id" gorm:"primaryKey"`
	SourceConnectionInfoID string                     `json:"source_connection_info_id" gorm:"primaryKey"`
	Target                 Target                     `json:"target"`
	Order                  int                        `json:"order" gorm:"primaryKey"`
	SoftwareName           string                     `json:"software_name"`
	SoftwareVersion        string                     `json:"software_version"`
	SoftwareInstallType    softwaremodel.SoftwareType `json:"software_install_type" gorm:"primaryKey"`
	Status                 string                     `json:"status"`
	StartedAt              time.Time                  `json:"started_at"`
	UpdatedAt              time.Time                  `json:"updated_at"`
	ErrorMessage           string                     `json:"error_message"`
}

type SoftwareMigrationStatusList []SoftwareMigrationStatus

type SoftwareInstallStatusReq struct {
	ExecutionID string `json:"execution_id"`
}

func (t Target) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *Target) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for Target")
	}
	return json.Unmarshal(bytes, t)
}

func (esl SoftwareMigrationStatusList) Value() (driver.Value, error) {
	return json.Marshal(esl)
}

func (esl *SoftwareMigrationStatusList) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for SoftwareMigrationStatusList")
	}
	return json.Unmarshal(bytes, esl)
}
