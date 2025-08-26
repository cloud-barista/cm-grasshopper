package software

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"

	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	softwaremodel "github.com/cloud-barista/cm-model/sw"
)

var libraryPackagePatterns = []string{
	"lib.*-dev",
	"lib.*[0-9]+$",
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
	"linux-generic.*",
	"linux-image.*generic.*",
	"linux-headers.*generic.*",
	"linux-modules.*generic.*",
	"linux-tools.*generic.*",
	"kernel.*",
	"kernel-core.*",
	"kernel-modules.*",
	"kernel-headers.*",
	"kernel-devel.*",
	"kernel-tools.*",
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

func processSoftwarePackages(packages []softwaremodel.Package) ([]softwaremodel.PackageMigrationInfo, []string) {
	migrationPackages := make([]softwaremodel.PackageMigrationInfo, 0)
	errMsgs := make([]string, 0)

	var i int
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

		i++

		newSoftware := softwaremodel.PackageMigrationInfo{
			Order:                i,
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

		server.MigrationList.Packages, server.Errors = processSoftwarePackages(source.Softwares.Packages)
		server.SourceConnectionInfoID = source.ConnectionId

		servers = append(servers, server)
	}

	return &softwaremodel.TargetSoftwareModel{
		TargetSoftwareModel: softwaremodel.TargetGroupSoftwareProperty{
			Servers: servers,
		},
	}, nil
}
