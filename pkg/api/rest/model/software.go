package model

type SoftwareInstallReq struct {
	ConnectionID string   `json:"connection_id" yaml:"connection_uuid" validate:"required"`
	PackageType  string   `json:"package_type" yaml:"package_type" validate:"required"`
	PackageNames []string `json:"package_names" yaml:"package_names" validate:"required"`
}

type SoftwareInstallRes struct {
	Output string `json:"output" yaml:"output"`
}
