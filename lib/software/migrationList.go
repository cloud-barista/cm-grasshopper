package software

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	softwaremodel "github.com/cloud-barista/cm-grasshopper/smdl"
	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
)

var libraryPackagePatterns = []string{
	"^lib.*-dev",
	"^lib.*[0-9]+$",
	".*-devel",
	".*-headers",
	".*-doc",
	".*-man",
	".*-common",
	".*-locale",
	".*-dbg",
	".*-data$",
}

func isLibraryPackage(packageName string) bool {
	for _, pattern := range libraryPackagePatterns {
		matched, _ := regexp.MatchString(pattern, packageName)
		if matched {
			return true
		}
	}
	return false
}

var containerRuntimeRelatedPackagePatterns = []string{
	".*docker.*",
	".*podman.*",
	".*crio.*",
	".*cri-o.*",
	".*containerd.*",
	".*runc.*",
	".*buildah.*",
	".*skopeo.*",
}

func isContainerRuntimeRelatedPackage(packageName string) bool {
	for _, pattern := range containerRuntimeRelatedPackagePatterns {
		matched, _ := regexp.MatchString(pattern, packageName)
		if matched {
			return true
		}
	}
	return false
}

var genericKernelPackagePatterns = []string{
	"^linux-generic.*",
	"^linux-image.*generic.*",
	"^linux-headers.*generic.*",
	"^linux-modules.*generic.*",
	"^linux-tools.*generic.*",
	"^kernel.*",
	"^kernel-core.*",
	"^kernel-modules.*",
	"^kernel-headers.*",
	"^kernel-devel.*",
	"^kernel-tools.*",
}

func isGenericKernelPackage(packageName string) bool {
	for _, pattern := range genericKernelPackagePatterns {
		matched, _ := regexp.MatchString(pattern, packageName)
		if matched {
			return true
		}
	}
	return false
}

var packageManagerPackagePatterns = []string{
	"apt",
	"yum",
	"dnf",
	"dnf-data",
	".*libdnf.*",
}

func isPackageManagerPackage(packageName string) bool {
	for _, pattern := range packageManagerPackagePatterns {
		matched, _ := regexp.MatchString(pattern, packageName)
		if matched {
			return true
		}
	}
	return false
}

// systemBasePackages are packages that belong to the base OS image, the cloud
// provider's guest tooling, the kernel/boot chain or host-level daemons. A freshly
// provisioned target VM already ships these, so migrating them is pure overhead:
// it multiplies the number of SSH round trips (slowing the whole run and starving
// the target's sshd) and produces spurious failures when host services such as
// chrony or irqbalance cannot start in the new environment. Application packages
// the user actually deployed (mariadb, php, nginx, nfs-kernel-server, rpcbind, ...)
// are intentionally NOT listed here and still migrate.
var systemBasePackages = map[string]bool{
	"apport": true, "apport-symptoms": true, "friendly-recovery": true,
	"command-not-found": true, "ubuntu-minimal": true, "ubuntu-standard": true,
	"ubuntu-server": true, "ubuntu-advantage-tools": true, "ubuntu-pro-client": true,
	"ubuntu-release-upgrader-core": true, "landscape-common": true, "motd-news-config": true,
	"chrony": true, "rsyslog": true, "sysstat": true, "irqbalance": true, "ufw": true,
	"apparmor": true, "fwupd": true, "unattended-upgrades": true, "update-notifier-common": true,
	"update-manager-core": true, "packagekit": true, "packagekit-tools": true, "pastebinit": true,
	"run-one": true, "powermgmt-base": true, "bash-completion": true, "build-essential": true,
	"fakeroot": true, "nano": true, "telnet": true, "tcpdump": true, "mtr-tiny": true,
	"ncurses-term": true, "publicsuffix": true, "shared-mime-info": true, "xauth": true,
	"xdg-user-dirs": true, "lsscsi": true, "nvme-cli": true, "ntfs-3g": true, "cifs-utils": true,
	"pollinate": true, "sosreport": true, "netplan.io": true, "mdadm": true, "multipath-tools": true,
	"open-iscsi": true, "thermald": true, "bolt": true, "byobu": true, "distro-info-data": true,
	"walinuxagent": true, "azure-vm-utils": true, "open-vm-tools": true, "cloud-init": true,
}

// systemBasePackagePatterns catch versioned/variant base packages (kernel, boot
// chain, cloud agents) that cannot be enumerated exactly.
var systemBasePackagePatterns = []string{
	"^linux-", "^grub", "^plymouth", "^cryptsetup", "^initramfs",
	"^snap", "^cloud-init", "^cloud-guest", "^cloud-initramfs",
	"^walinuxagent", "^azure-", "^open-vm-tools",
}

func isSystemBasePackage(packageName string) bool {
	if systemBasePackages[packageName] {
		return true
	}
	for _, pattern := range systemBasePackagePatterns {
		matched, _ := regexp.MatchString(pattern, packageName)
		if matched {
			return true
		}
	}
	return false
}

// systemBaseBinaries are init-system and base-OS daemons that cm-honeybee may
// collect as running "binaries" (they are long-lived processes) but that must not
// be migrated: the target already runs its own init/base daemons, and re-homing
// e.g. systemd or its templated units (systemd-fsck@.service) onto a live target
// simply fails. Real application binaries (tomcat, httpd, ...) are never listed.
var systemBaseBinaries = map[string]bool{
	"systemd": true, "systemd-resolved": true, "systemd-networkd": true,
	"systemd-timesyncd": true, "systemd-logind": true, "systemd-journald": true,
	"systemd-udevd": true, "udevadm": true, "rpcbind": true, "rpc.statd": true,
	"rpc.mountd": true, "rpc.idmapd": true, "rpc.gssd": true, "multipathd": true,
	"snapd": true, "polkitd": true, "accounts-daemon": true, "irqbalance": true,
	"chronyd": true, "cron": true, "atd": true, "dbus-daemon": true,
	"networkd-dispatcher": true, "unattended-upgrade": true, "packagekitd": true,
	"walinuxagent": true,
}

func isSystemBaseBinary(binaryName string) bool {
	return systemBaseBinaries[binaryName]
}

//func getConnectionInfoInfra(sgID string, connectionID string) (*infra.Infra, error) {
//	data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress+
//		":"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort+
//		"/honeybee/source_group/"+sgID+"/connection_info/"+connectionID+"/infra", "", "")
//	if err != nil {
//		return nil, err
//	}
//
//	var infraInfo infra.Infra
//	err = json.Unmarshal(data, &infraInfo)
//	if err != nil {
//		return nil, err
//	}
//
//	return &infraInfo, nil
//}

func processSoftwareBinaries(prevOrder *int, binaries []softwaremodel.Binary) ([]softwaremodel.BinaryMigrationInfo, []string) {
	migrationBinaries := make([]softwaremodel.BinaryMigrationInfo, 0)
	errMsgs := make([]string, 0)

	for _, b := range binaries {
		if isSystemBaseBinary(b.Name) {
			continue
		}

		*prevOrder++

		newBinary := softwaremodel.BinaryMigrationInfo{
			Order:            *prevOrder,
			Name:             b.Name,
			Version:          b.Version,
			UIDs:             b.UIDs,
			GIDs:             b.GIDs,
			CmdlineSlice:     b.CmdlineSlice,
			Envs:             b.Envs,
			NeededLibraries:  b.NeededLibraries,
			RequiredPackages: b.RequiredPackages,
			BinaryPath:       b.BinaryPath,
			CustomDataPaths:  b.CustomDataPaths,
			CustomConfigs:    b.CustomConfigs,
			IsWine:           b.IsWine,
			WinePrefix:       b.WinePrefix,
			LaunchType:       b.LaunchType,
			SystemdUnitName:  b.SystemdUnitName,
			SystemdUnitPath:  b.SystemdUnitPath,
			SystemdEnabled:   b.SystemdEnabled,
			WorkingDirectory: b.WorkingDirectory,
			ServiceType:      b.ServiceType,
			PIDFile:          b.PIDFile,
		}

		migrationBinaries = append(migrationBinaries, newBinary)
	}

	return migrationBinaries, errMsgs
}

func processSoftwarePackages(prevOrder *int, packages []softwaremodel.Package) ([]softwaremodel.PackageMigrationInfo, []string) {
	migrationPackages := make([]softwaremodel.PackageMigrationInfo, 0)
	errMsgs := make([]string, 0)

	for _, pkg := range packages {
		if isLibraryPackage(pkg.Name) {
			continue
		}

		if isContainerRuntimeRelatedPackage(pkg.Name) {
			continue
		}

		if isGenericKernelPackage(pkg.Name) {
			continue
		}

		if isPackageManagerPackage(pkg.Name) {
			continue
		}

		if isSystemBasePackage(pkg.Name) {
			continue
		}

		*prevOrder++

		newSoftware := softwaremodel.PackageMigrationInfo{
			Order:                *prevOrder,
			Name:                 pkg.Name,
			Version:              pkg.Version,
			NeededPackages:       strings.Split(pkg.NeededPackages, ","),
			NeedToDeletePackages: strings.Split(pkg.NeedToDeletePackages, ","),
			CustomDataPaths:      []string{},
			CustomConfigs:        pkg.CustomConfigs,
			RepoURL:              pkg.RepoURL,
			GPGKeyURL:            pkg.GPGKeyURL,
			RepoUseOSVersionCode: pkg.RepoUseOSVersionCode,
		}

		migrationPackages = append(migrationPackages, newSoftware)
	}

	return migrationPackages, errMsgs
}

func processSoftwareContainers(prevOrder *int, containers []softwaremodel.Container) ([]softwaremodel.ContainerMigrationInfo, []string) {
	migrationContainers := make([]softwaremodel.ContainerMigrationInfo, 0)
	errMsgs := make([]string, 0)

	for _, c := range containers {
		*prevOrder++

		newContainer := softwaremodel.ContainerMigrationInfo{
			Order:       *prevOrder,
			Name:        c.Name,
			Runtime:     string(c.Runtime),
			ContainerId: c.ContainerId,
			ContainerImage: softwaremodel.ContainerImage{
				ImageName:         c.ContainerImage.ImageName,
				ImageVersion:      c.ContainerImage.ImageVersion,
				ImageArchitecture: c.ContainerImage.ImageArchitecture,
				ImageHash:         c.ContainerImage.ImageHash,
			},
			ContainerPorts:    c.ContainerPorts,
			ContainerStatus:   c.ContainerStatus,
			DockerComposePath: c.DockerComposePath,
			MountPaths:        c.MountPaths,
			Envs:              c.Envs,
			NetworkMode:       c.NetworkMode,
			RestartPolicy:     c.RestartPolicy,
		}

		migrationContainers = append(migrationContainers, newContainer)
	}

	return migrationContainers, errMsgs
}

func MakeMigrationListRes(sourceSoftwareModel *softwaremodel.SourceSoftwareModel) (*softwaremodel.TargetSoftwareModel, error) {
	data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress+
		":"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort+
		"/honeybee/source_group/"+sourceSoftwareModel.SourceSoftwareModel.SourceGroupId+"/connection_info", "", "")
	if err != nil {
		return nil, err
	}

	var listConnectionInfoRes honeybee.ListConnectionInfoRes
	err = json.Unmarshal(data, &listConnectionInfoRes)
	if err != nil {
		return nil, err
	}

	var servers []softwaremodel.MigrationServer

	for _, source := range sourceSoftwareModel.SourceSoftwareModel.ConnectionInfoList {
		var found bool

		for _, encryptedConnectionInfo := range listConnectionInfoRes.ConnectionInfo {
			if encryptedConnectionInfo.ID == source.ConnectionId {
				found = true
				break
			}
		}
		if !found {
			return nil, errors.New("connection info (ID=" + source.ConnectionId + ") not found")
		}

		var server softwaremodel.MigrationServer

		var prevOrder int

		server.MigrationList.Binaries, server.Errors = processSoftwareBinaries(&prevOrder, source.Softwares.Binaries)
		server.MigrationList.Packages, server.Errors = processSoftwarePackages(&prevOrder, source.Softwares.Packages)
		server.MigrationList.Containers, server.Errors = processSoftwareContainers(&prevOrder, source.Softwares.Containers)
		server.SourceConnectionInfoID = source.ConnectionId

		servers = append(servers, server)
	}

	return &softwaremodel.TargetSoftwareModel{
		TargetSoftwareModel: softwaremodel.TargetGroupSoftwareProperty{
			Servers: servers,
		},
	}, nil
}
