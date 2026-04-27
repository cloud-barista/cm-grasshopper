package softwaremodel

import (
	"errors"
)

type SoftwareArchitecture string

const (
	SoftwareArchitectureCommon  SoftwareArchitecture = "common"
	SoftwareArchitectureX8664   SoftwareArchitecture = "x86_64"
	SoftwareArchitectureX86     SoftwareArchitecture = "x86"
	SoftwareArchitectureARMv5   SoftwareArchitecture = "armv5"
	SoftwareArchitectureARMv6   SoftwareArchitecture = "armv6"
	SoftwareArchitectureARMv7   SoftwareArchitecture = "armv7"
	SoftwareArchitectureARM64v8 SoftwareArchitecture = "arm64v8"
)

func CheckArchitecture(softwareArchitecture string) error {
	switch softwareArchitecture {
	case string(SoftwareArchitectureCommon):
		fallthrough
	case string(SoftwareArchitectureX8664):
		fallthrough
	case string(SoftwareArchitectureX86):
		fallthrough
	case string(SoftwareArchitectureARMv5):
		fallthrough
	case string(SoftwareArchitectureARMv6):
		fallthrough
	case string(SoftwareArchitectureARMv7):
		fallthrough
	case string(SoftwareArchitectureARM64v8):
		return nil
	default:
		return errors.New("invalid architecture")
	}
}

type SoftwareType string

const (
	SoftwareTypePackage    SoftwareType = "package"    // Installing via OS package manager.
	SoftwareTypeContainer  SoftwareType = "container"  // Installing as a container package.
	SoftwareTypeKubernetes SoftwareType = "kubernetes" // Installing as a Kubernetes package.
	SoftwareTypeBinary     SoftwareType = "binary"     // Moving the software as a binary executable.
)

type SoftwarePackageType string

const (
	SoftwarePackageTypeDEB SoftwarePackageType = "deb" // Debian based package type
	SoftwarePackageTypeRPM SoftwarePackageType = "rpm" // RHEL based package type
)

type SoftwareContainerRuntimeType string

const (
	SoftwareContainerRuntimeTypeDocker SoftwareContainerRuntimeType = "docker"
	SoftwareContainerRuntimeTypePodman SoftwareContainerRuntimeType = "podman"
)

type ContainerImage struct {
	ImageName         string               `json:"image_name" validate:"required"`
	ImageVersion      string               `json:"image_version" validate:"required"`
	ImageArchitecture SoftwareArchitecture `json:"image_architecture" validate:"required"`
	ImageHash         string               `json:"image_hash" validate:"required"`
}

type ContainerPort struct {
	ContainerPort int    `json:"container_port" validate:"required"` // NetworkSettings.Ports.{Port}/{Protocol} -> {Port}
	Protocol      string `json:"protocol" validate:"required"`       // NetworkSettings.Ports.{Port}/{Protocol} -> {Protocol}
	HostIP        string `json:"host_ip" validate:"required"`        // NetworkSettings.Ports.{Port}/{Protocol}.HostIp
	HostPort      int    `json:"host_port" validate:"required"`      // NetworkSettings.Ports.{Port}/{Protocol}.HostPort
}

type Env struct {
	Name  string `json:"name,omitempty" validate:"required"`
	Value string `json:"value,omitempty"`
}

type Binary struct {
	Name            string   `json:"name" validate:"required"`
	Version         string   `gorm:"version" json:"version" validate:"required"`
	UIDs            []int32  `json:"uids" validate:"required"`
	GIDs            []int32  `json:"gids" validate:"required"`
	CmdlineSlice    []string `json:"cmdline_slice"`
	Envs            []string `json:"envs" validate:"required"`
	NeededLibraries []string `json:"needed_libraries"`
	BinaryPath      string   `json:"binary_path,omitempty"`
	CustomDataPaths []string `json:"custom_data_paths"`
	CustomConfigs   []string `json:"custom_configs"`
	IsWine          bool     `json:"is_wine"`
}

type Package struct {
	Name                 string              `json:"name" validate:"required"`
	Type                 SoftwarePackageType `json:"type" validate:"required"`
	Version              string              `gorm:"version" json:"version" validate:"required"`
	NeededPackages       string              `json:"needed_packages,omitempty" validate:"required"`
	NeedToDeletePackages string              `json:"need_to_delete_packages,omitempty"`
	CustomDataPaths      []string            `json:"custom_data_paths,omitempty"`
	CustomConfigs        []string            `json:"custom_configs,omitempty"`
	RepoURL              string              `json:"repo_url,omitempty"`
	GPGKeyURL            string              `json:"gpg_key_url,omitempty"`
	RepoUseOSVersionCode bool                `json:"repo_use_os_version_code,omitempty" default:"false"`
}

type Container struct {
	Name              string                       `json:"name,omitempty" validate:"required"`
	Runtime           SoftwareContainerRuntimeType `json:"runtime,omitempty" validate:"required"` // Which runtime uses for the container (Docker, Podman)
	ContainerId       string                       `json:"container_id" validate:"required"`
	ContainerImage    ContainerImage               `json:"container_image,omitempty" validate:"required"`
	ContainerPorts    []ContainerPort              `json:"container_ports"`
	ContainerStatus   string                       `json:"container_status" validate:"required"`
	DockerComposePath string                       `json:"docker_compose_path"`
	MountPaths        []string                     `json:"mount_paths"`
	Envs              []Env                        `json:"envs"`
	NetworkMode       string                       `json:"network_mode,omitempty" validate:"required"`
	RestartPolicy     string                       `json:"restart_policy,omitempty" validate:"required"`
}

type Kubernetes struct {
	Version    string                 `json:"version,omitempty" validate:"required"` // Same as release
	KubeConfig string                 `json:"kube_config" validate:"required"`
	Resources  map[string]interface{} `json:"resources,omitempty"  validate:"required"`
}

type SoftwareList struct {
	Binaries   []Binary     `json:"binaries"`
	Packages   []Package    `json:"packages"`
	Containers []Container  `json:"containers"`
	Kubernetes []Kubernetes `json:"kubernetes"`
}

type SourceConnectionInfoSoftwareProperty struct {
	ConnectionId string       `json:"connection_id" validate:"required"`
	Softwares    SoftwareList `json:"softwares"`
}

type SourceGroupSoftwareProperty struct {
	SourceGroupId      string                                 `json:"source_group_id" validate:"required"`
	ConnectionInfoList []SourceConnectionInfoSoftwareProperty `json:"connection_info_list"`
}

type SourceSoftwareModel struct {
	SourceSoftwareModel SourceGroupSoftwareProperty `json:"sourceSoftwareModel" validate:"required"`
}

type BinaryMigrationInfo struct {
	Order           int      `json:"order"`
	Name            string   `json:"name" validate:"required"`
	Version         string   `gorm:"version" json:"version" validate:"required"`
	UIDs            []int32  `json:"uids" validate:"required"`
	GIDs            []int32  `json:"gids" validate:"required"`
	CmdlineSlice    []string `json:"cmdline_slice"`
	Envs            []string `json:"envs" validate:"required"`
	NeededLibraries []string `json:"needed_libraries"`
	BinaryPath      string   `json:"binary_path,omitempty"`
	CustomDataPaths []string `json:"custom_data_paths"`
	CustomConfigs   []string `json:"custom_configs"`
	IsWine          bool     `json:"is_wine"`
}

type PackageMigrationInfo struct {
	Order                int      `json:"order"`
	Name                 string   `json:"name" validate:"required"`
	Version              string   `gorm:"version" json:"version" validate:"required"`
	NeededPackages       []string `json:"needed_packages" validate:"required"`
	NeedToDeletePackages []string `json:"need_to_delete_packages"`
	CustomDataPaths      []string `json:"custom_data_paths"`
	CustomConfigs        []string `json:"custom_configs"`
	RepoURL              string   `json:"repo_url"`
	GPGKeyURL            string   `json:"gpg_key_url"`
	RepoUseOSVersionCode bool     `json:"repo_use_os_version_code" default:"false"`
}

type ContainerMigrationInfo struct {
	Order             int             `json:"order"`
	Name              string          `json:"name,omitempty" validate:"required"`
	Runtime           string          `json:"runtime,omitempty" validate:"required"` // Which runtime uses for the container (Docker, Podman, ...)
	ContainerId       string          `json:"container_id" validate:"required"`
	ContainerImage    ContainerImage  `json:"container_image,omitempty" validate:"required"`
	ContainerPorts    []ContainerPort `json:"container_ports"`
	ContainerStatus   string          `json:"container_status" validate:"required"`
	DockerComposePath string          `json:"docker_compose_path"`
	MountPaths        []string        `json:"mount_paths"`
	Envs              []Env           `json:"envs"`
	NetworkMode       string          `json:"network_mode,omitempty" validate:"required"`
	RestartPolicy     string          `json:"restart_policy,omitempty" validate:"required"`
}

type KubernetesVelero struct {
	Provider             string `json:"provider" validate:"required"`
	Plugins              string `json:"plugins,omitempty"`
	Bucket               string `json:"bucket" validate:"required"`
	SecretFile           string `json:"secret_file"`
	BackupLocationConfig string `json:"backup_location_config" validate:"required"`
	Features             string `json:"features"`
}

type KubernetesMigrationInfo struct {
	Order      int                    `json:"order"`
	Version    string                 `json:"version,omitempty" validate:"required"` // Same as release
	KubeConfig string                 `json:"kube_config" validate:"required"`
	Resources  map[string]interface{} `json:"resources,omitempty"  validate:"required"`
	Velero     KubernetesVelero       `json:"velero" validate:"required"`
}

type MigrationList struct {
	Binaries   []BinaryMigrationInfo     `json:"binaries"`
	Packages   []PackageMigrationInfo    `json:"packages"`
	Containers []ContainerMigrationInfo  `json:"containers"`
	Kubernetes []KubernetesMigrationInfo `json:"kubernetes"`
}

type MigrationServer struct {
	SourceConnectionInfoID string        `json:"source_connection_info_id"`
	MigrationList          MigrationList `json:"migration_list"`
	Errors                 []string      `json:"errors"`
}

type TargetGroupSoftwareProperty struct {
	Servers []MigrationServer `json:"servers"`
}

type TargetSoftwareModel struct {
	TargetSoftwareModel TargetGroupSoftwareProperty `json:"targetSoftwareModel" validate:"required"`
}
