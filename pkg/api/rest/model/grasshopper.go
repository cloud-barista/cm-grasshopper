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

type TargetMapping struct {
	SourceConnectionInfoID string `json:"source_connection_info_id"`
	Target                 Target `json:"target" validate:"required"`
	Status                 string `json:"status"`
}

type SoftwareMigrateRes struct {
	ExecutionID    string          `json:"execution_id"`
	TargetMappings []TargetMapping `json:"target_mappings"`
}

type SoftwareMigrationStatus struct {
	ExecutionID            string                     `json:"execution_id" gorm:"primaryKey"`
	SourceConnectionInfoID string                     `json:"source_connection_info_id" gorm:"primaryKey"`
	NamespaceID            string                     `json:"namespace_id" validate:"required"`
	MCIID                  string                     `json:"mci_id" validate:"required"`
	VMID                   string                     `json:"vm_id" validate:"required"`
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

type TargetMappingList []TargetMapping

type ExecutionStatus struct {
	ExecutionID    string            `json:"execution_id" gorm:"primaryKey"`
	TargetMappings TargetMappingList `json:"target_mappings"`
	StartedAt      time.Time         `json:"started_at"`
	FinishedAt     time.Time         `json:"finished_at"`
}

type SoftwareMigrationStatusReq struct {
	ExecutionID string `json:"execution_id"`
}

type SoftwareMigrationStatusRes struct {
	ExecutionStatusList []ExecutionStatus `json:"execution_status_list"`
}

type SoftwareMigrationStatusSoftwareStatusOnly struct {
	Order               int                        `json:"order" gorm:"primaryKey"`
	SoftwareName        string                     `json:"software_name"`
	SoftwareVersion     string                     `json:"software_version"`
	SoftwareInstallType softwaremodel.SoftwareType `json:"software_install_type" gorm:"primaryKey"`
	Status              string                     `json:"status"`
	StartedAt           time.Time                  `json:"started_at"`
	UpdatedAt           time.Time                  `json:"updated_at"`
	ErrorMessage        string                     `json:"error_message"`
}

type TargetMappingWithSoftwareMigrationList struct {
	SourceConnectionInfoID      string                                      `json:"source_connection_info_id"`
	Target                      Target                                      `json:"target" validate:"required"`
	Status                      string                                      `json:"status"`
	SoftwareMigrationStatusList []SoftwareMigrationStatusSoftwareStatusOnly `json:"software_migration_status_list"`
}

type ExecutionStatusWithSoftwareMigrationList struct {
	ExecutionID    string                                   `json:"execution_id" gorm:"primaryKey"`
	TargetMappings []TargetMappingWithSoftwareMigrationList `json:"target_mappings"`
	StartedAt      time.Time                                `json:"started_at"`
	FinishedAt     time.Time                                `json:"finished_at"`
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

func (tm TargetMapping) Value() (driver.Value, error) {
	return json.Marshal(tm)
}

func (tm *TargetMapping) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for TargetMapping")
	}
	return json.Unmarshal(bytes, tm)
}

func (tml TargetMappingList) Value() (driver.Value, error) {
	return json.Marshal(tml)
}

func (tml *TargetMappingList) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("invalid type for TargetMappingList")
	}
	return json.Unmarshal(bytes, tml)
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
