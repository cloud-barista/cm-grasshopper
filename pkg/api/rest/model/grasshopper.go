package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

type Source struct {
	ConnectionID string `json:"connection_id" yaml:"connection_uuid" validate:"required"`
}

type Target struct {
	NamespaceID string `json:"namespace_id" validate:"required"`
	MCIID       string `json:"mci_id" validate:"required"`
	VMID        string `json:"vm_id" validate:"required"`
}

type PackageMigrationConfig struct {
	ID                   string               `gorm:"primaryKey" json:"uuid" validate:"required"`
	Name                 string               `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" validate:"required"`
	Version              string               `gorm:"version" json:"version" validate:"required"`
	OS                   string               `gorm:"os" json:"os" validate:"required"`
	OSVersion            string               `gorm:"os_version" json:"os_version" validate:"required"`
	Architecture         SoftwareArchitecture `gorm:"architecture" json:"architecture" validate:"required"`
	MatchNames           string               `gorm:"match_names" json:"match_names" validate:"required"`
	NeededPackages       string               `json:"needed_packages" validate:"required"`
	NeedToDeletePackages string               `json:"need_to_delete_packages"`
	CustomConfigs        string               `json:"custom_configs"`
	RepoURL              string               `json:"repo_url"`
	GPGKeyURL            string               `json:"gpg_key_url"`
	RepoUseOSVersionCode bool                 `json:"repo_use_os_version_code" default:"false"`
	CreatedAt            time.Time            `gorm:"column:created_at" json:"created_at" validate:"required"`
	UpdatedAt            time.Time            `gorm:"column:updated_at" json:"updated_at" validate:"required"`
}

type PackageMigrationConfigReq struct {
	ID                   string               `gorm:"primaryKey" json:"uuid" validate:"required"`
	Name                 string               `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" validate:"required"`
	Version              string               `gorm:"version" json:"version" validate:"required"`
	OS                   string               `gorm:"os" json:"os" validate:"required"`
	OSVersion            string               `gorm:"os_version" json:"os_version" validate:"required"`
	Architecture         SoftwareArchitecture `gorm:"architecture" json:"architecture" validate:"required"`
	MatchNames           []string             `gorm:"match_names" json:"match_names" validate:"required"`
	NeededPackages       []string             `json:"needed_packages" validate:"required"`
	NeedToDeletePackages []string             `json:"need_to_delete_packages"`
	CustomConfigs        []string             `json:"custom_configs"`
	RepoURL              string               `json:"repo_url"`
	GPGKeyURL            string               `json:"gpg_key_url"`
	RepoUseOSVersionCode bool                 `json:"repo_use_os_version_code" default:"false"`
}

type BinaryMigrationInfo struct {
	Order           int      `json:"order"`
	Name            string   `json:"name" validate:"required"`
	Version         string   `gorm:"version" json:"version" validate:"required"`
	NeededLibraries []string `json:"needed_libraries"`
	BinaryPath      string   `json:"binary_path,omitempty"`
	CustomDataPaths []string `json:"custom_data_paths"`
	CustomConfigs   string   `json:"custom_configs"`
}

type MigrationServer struct {
	ConnectionInfoID string        `json:"connection_info_id"`
	MigrationList    MigrationList `json:"migration_list"`
	Errors           []string      `json:"errors"`
}

type MigrationListRes struct {
	Servers []MigrationServer `json:"servers"`
}

type MigrationLogRes struct {
	UUID         string `json:"uuid"`
	InstallLog   string `json:"install_log"`
	MigrationLog string `json:"migration_log"`
}

type SoftwareMigrateReq struct {
	SourceConnectionInfoID string        `json:"source_connection_info_id" validate:"required"`
	Target                 Target        `json:"target" validate:"required"`
	MigrationList          MigrationList `json:"migration_list"`
}

type SoftwareMigrateRes struct {
	ExecutionID   string        `json:"execution_id"`
	MigrationList MigrationList `json:"migration_list"`
}

type ExecutionStatus struct {
	Order               int          `json:"order"`
	SoftwareName        string       `json:"software_name"`
	SoftwareVersion     string       `json:"software_version"`
	SoftwareInstallType SoftwareType `json:"software_install_type"`
	Status              string       `json:"status"`
	StartedAt           time.Time    `json:"started_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
	ErrorMessage        string       `json:"error_message"`
}

type ExecutionStatusList []ExecutionStatus

type SoftwareInstallStatusReq struct {
	ExecutionID string `json:"execution_id"`
}

type SoftwareInstallStatus struct {
	ExecutionID     string              `gorm:"primaryKey:,column:execution_id" json:"execution_id"`
	Target          Target              `gorm:"target" json:"target"`
	ExecutionStatus ExecutionStatusList `gorm:"execution_status" json:"execution_status"`
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
