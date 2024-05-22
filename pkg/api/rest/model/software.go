package model

type SoftwareInstallReq struct {
	ConnectionUUID string   `json:"connection_uuid" yaml:"connection_uuid" validate:"required"`
	PackageType    string   `json:"package_type" yaml:"package_type" validate:"required"`
	PackageNames   []string `json:"package_names" yaml:"package_names" validate:"required"`
}

type SoftwareInstallRes struct {
	Output string `json:"output" yaml:"output"`
}
