package model

import (
	"errors"
	"time"
)

type InstallType string

const (
	PACKAGE InstallType = "package"
	ANSIBLE InstallType = "ansible"
	SCRIPT  InstallType = "script"
)

type Architecture string

const (
	COMMON Architecture = "common"
	X8664  Architecture = "x86_64"
	X86    Architecture = "x86"
	ARM64  Architecture = "arm64"
	ARM    Architecture = "arm"
)

func ToInstallType(input string) (InstallType, error) {
	switch input {
	case "package":
		return PACKAGE, nil
	case "ansible":
		return ANSIBLE, nil
	case "script":
		return SCRIPT, nil
	default:
		return "", errors.New("invalid install type")
	}
}

func ToArchitecture(input string) (Architecture, error) {
	switch input {
	case "common":
		return COMMON, nil
	case "x86_64":
		return X8664, nil
	case "x86":
		return X86, nil
	case "arm":
		return ARM, nil
	case "arm64":
		return ARM64, nil
	default:
		return "", errors.New("invalid architecture")
	}
}

type Software struct {
	ID           string       `gorm:"primaryKey" json:"uuid" validate:"required"`
	InstallType  InstallType  `gorm:"install_type" json:"install_type" validate:"required"`
	Name         string       `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" validate:"required"`
	Version      string       `gorm:"version" json:"version" validate:"required"`
	OS           string       `gorm:"os" json:"os" validate:"required"`
	OSVersion    string       `gorm:"os_version" json:"os_version" validate:"required"`
	Architecture Architecture `gorm:"architecture" json:"architecture" validate:"required"`
	MatchNames   []string     `gorm:"match_names" json:"match_names" validate:"required"`
	Size         string       `gorm:"size" json:"size" validate:"required"`
	CreatedAt    time.Time    `gorm:"column:created_at" json:"created_at" validate:"required"`
	UpdatedAt    time.Time    `gorm:"column:updated_at" json:"updated_at" validate:"required"`
}

type SoftwareInfo struct {
	Name    string `json:"name" validate:"required"`
	Version string `json:"version" validate:"required"`
}

type Source struct {
	ConnectionID string `json:"connection_id" yaml:"connection_uuid" validate:"required"`
}

type Target struct {
	NamespaceID string `json:"namespace_id" validate:"required"`
	MCISID      string `json:"mcis_id" validate:"required"`
	VMID        string `json:"vm_id" validate:"required"`
}

type SoftwareRegisterReq struct {
	InstallType  string   `gorm:"install_type" json:"install_type" validate:"required"`
	Name         string   `gorm:"index:,column:name,unique;type:text collate nocase" json:"name" validate:"required"`
	Version      string   `gorm:"version" json:"version" validate:"required"`
	OS           string   `gorm:"os" json:"os" validate:"required"`
	OSVersion    string   `gorm:"os_version" json:"os_version" validate:"required"`
	Architecture string   `gorm:"architecture" json:"architecture" validate:"required"`
	MatchNames   []string `gorm:"match_names" json:"match_names" validate:"required"`
}

type Execution struct {
	Order    int      `json:"order"`
	Software Software `json:"software"`
}

type SoftwareInstallReq struct {
	Source       Source         `json:"source" validate:"required"`
	Target       Target         `json:"target" validate:"required"`
	SoftwareList []SoftwareInfo `json:"software_list" validate:"required"`
}

type SoftwareInstallRes struct {
	ExecutionList []Execution `json:"execution_list"`
}
