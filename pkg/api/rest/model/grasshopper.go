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

type ExecutionStatus struct {
	Order               int                        `json:"order"`
	SoftwareName        string                     `json:"software_name"`
	SoftwareVersion     string                     `json:"software_version"`
	SoftwareInstallType softwaremodel.SoftwareType `json:"software_install_type"`
	Status              string                     `json:"status"`
	StartedAt           time.Time                  `json:"started_at"`
	UpdatedAt           time.Time                  `json:"updated_at"`
	ErrorMessage        string                     `json:"error_message"`
}

type ExecutionStatusList []ExecutionStatus

type SoftwareInstallStatusReq struct {
	ExecutionID string `json:"execution_id"`
}

type SoftwareInstallStatus struct {
	SoftwareInstallType softwaremodel.SoftwareType `json:"software_install_type"`
	ExecutionID         string                     `gorm:"primaryKey:,column:execution_id" json:"execution_id"`
	Target              Target                     `gorm:"target" json:"target"`
	ExecutionStatus     ExecutionStatusList        `gorm:"execution_status" json:"execution_status"`
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

func (esl ExecutionStatusList) Value() (driver.Value, error) {
	return json.Marshal(esl)
}

func (esl *ExecutionStatusList) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for ExecutionStatusList")
	}
	return json.Unmarshal(bytes, esl)
}
