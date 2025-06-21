package model

import (
	"errors"
)

type SoftwareArchitecture string

const (
	SoftwareArchitectureCommon SoftwareArchitecture = "common"
	SoftwareArchitectureX8664  SoftwareArchitecture = "x86_64"
	SoftwareArchitectureX86    SoftwareArchitecture = "x86"
	SoftwareArchitectureARM    SoftwareArchitecture = "arm"
	SoftwareArchitectureARM64  SoftwareArchitecture = "arm64"
)

func CheckArchitecture(softwareArchitecture SoftwareArchitecture) error {
	switch softwareArchitecture {
	case SoftwareArchitectureCommon:
		fallthrough
	case SoftwareArchitectureX8664:
		fallthrough
	case SoftwareArchitectureX86:
		fallthrough
	case SoftwareArchitectureARM:
		fallthrough
	case SoftwareArchitectureARM64:
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
	NeededLibraries []string `json:"needed_libraries"`
	BinaryPath      string   `json:"binary_path,omitempty"`
	CustomDataPaths []string `json:"custom_data_paths"`
	CustomConfigs   string   `json:"custom_configs"`
}

type Package struct {
	Name                 string   `json:"name" validate:"required"`
	Version              string   `gorm:"version" json:"version" validate:"required"`
	NeededPackages       string   `json:"needed_packages" validate:"required"`
	NeedToDeletePackages string   `json:"need_to_delete_packages"`
	CustomDataPaths      []string `json:"custom_data_paths"`
	CustomConfigs        string   `json:"custom_configs"`
	RepoURL              string   `json:"repo_url"`
	GPGKeyURL            string   `json:"gpg_key_url"`
	RepoUseOSVersionCode bool     `json:"repo_use_os_version_code" default:"false"`
}

type Container struct {
	Name              string          `json:"name,omitempty" validate:"required"`
	Runtime           string          `json:"runtime,omitempty" validate:"required"` // Which runtime uses for the container (Docker, Podman, ...)
	ContainerImage    ContainerImage  `json:"container_image,omitempty" validate:"required"`
	ContainerPorts    []ContainerPort `json:"container_ports"`
	ContainerStatus   string          `json:"container_status" validate:"required"`
	DockerComposePath string          `json:"docker_compose_path"`
	MountPaths        []string        `json:"mount_paths"`
	Envs              []Env           `json:"envs"`
	NetworkMode       string          `json:"network_mode,omitempty" validate:"required"`
	RestartPolicy     string          `json:"restart_policy,omitempty" validate:"required"`
}

type Kubernetes struct {
	Version    string                 `json:"version,omitempty" validate:"required"` // Same as release
	KubeConfig string                 `json:"kube_config" validate:"required"`
	Resources  map[string]interface{} `json:"resources,omitempty"  validate:"required"`
}
