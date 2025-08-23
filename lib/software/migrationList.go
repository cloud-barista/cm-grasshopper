package software

import (
	"regexp"
	"strings"

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

func processSoftwarePackages(packages []softwaremodel.Package) []softwaremodel.PackageMigrationInfo {
	migrationPackages := make([]softwaremodel.PackageMigrationInfo, 0)

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

	return migrationPackages
}

func MakeMigrationListRes(sourceSoftwareList *softwaremodel.SoftwareList) *softwaremodel.MigrationList {
	var migrationList softwaremodel.MigrationList

	migrationList.Packages = processSoftwarePackages(sourceSoftwareList.Packages)

	return &migrationList
}
