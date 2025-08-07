package software

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"strconv"
	"strings"
)

func findMatchedPackageMigrationConfig(os string, osVersion string,
	architecture model.SoftwareArchitecture, softwareName string, softwareVersion string) (*model.PackageMigrationConfig, error) {
	list, err := dao.PackageMigrationConfigGetList(&model.PackageMigrationConfig{
		MatchNames: softwareName,
	}, false, 0, 0)
	if err != nil {
		return nil, err
	}

	var matchedConfigs []model.PackageMigrationConfig
	for _, sw := range *list {
		if sw.Architecture != model.SoftwareArchitectureCommon &&
			(sw.OS != os || sw.OSVersion != osVersion || sw.Architecture != architecture) {
			continue
		}

		swMatchNames := strings.Split(sw.MatchNames, ",")
		for _, matchName := range swMatchNames {
			if softwareName == matchName {
				if softwareVersion == sw.Version {
					return &sw, nil
				}
				matchedConfigs = append(matchedConfigs, sw)
			}
		}
	}

	var latestSoftware model.PackageMigrationConfig
	if len(matchedConfigs) == 1 {
		if matchedConfigs[0].Version == softwareVersion {
			// Return if same version found
			return &matchedConfigs[0], nil
		}
		latestSoftware = matchedConfigs[0]
	} else if len(matchedConfigs) > 1 {
		for i := 1; i < len(matchedConfigs); i++ {
			if matchedConfigs[i].Version == softwareVersion {
				// Return if same version found
				return &matchedConfigs[i], nil
			}

			latestSoftwareVersionSplit := strings.Split(latestSoftware.Version, ".")
			matchedSoftwareVersionSplit := strings.Split(matchedConfigs[i].Version, ".")

			// Check only up to the 2nd digit
			// 13 vs blank			--> Pick blank as latest
			// 13 vs 13.X			--> Pick 13 as latest
			// 13.2 vs 13.1			--> Pick 13.2 as latest
			// 13.2.X vs 13.2		--> Pick 13.2 as latest
			// 13.2.X... vs 13.2	--> Pick 13.2 as latest
			// 13.2.X... vs 13		--> Pick 13 as latest

			// Pick X.~~~ higher (Pick higher major version)
			if len(latestSoftwareVersionSplit) > 0 && len(matchedSoftwareVersionSplit) > 0 {
				latestMajor, _ := strconv.Atoi(latestSoftwareVersionSplit[0])
				matchedSoftwareMajor, _ := strconv.Atoi(matchedSoftwareVersionSplit[0])

				if latestMajor < matchedSoftwareMajor {
					latestSoftware = matchedConfigs[i]
					latestSoftwareVersionSplit = strings.Split(latestSoftware.Version, ".")

					// Pick 0.X.~~~ higher (Pick higher minor version)
					if len(latestSoftwareVersionSplit) > 1 && len(matchedSoftwareVersionSplit) > 1 {
						latestMinor, _ := strconv.Atoi(latestSoftwareVersionSplit[1])
						matchedSoftwareMinor, _ := strconv.Atoi(matchedSoftwareVersionSplit[1])

						if latestMinor < matchedSoftwareMinor {
							latestSoftware = matchedConfigs[i]
						}
					}
				}
			}
		}
	}

	if latestSoftware.Version != "" {
		return &latestSoftware, nil
	}

	return nil, nil
}

// compareVersions compares two version strings
// returns: 1 if v1 > v2, 0 if v1 == v2, -1 if v1 < v2
func compareVersions(v1, v2 string) int {
	v1Parts := strings.Split(v1, "-")
	v2Parts := strings.Split(v2, "-")

	mainVersion1 := strings.Split(v1Parts[0], ".")
	mainVersion2 := strings.Split(v2Parts[0], ".")

	for i := 0; i < len(mainVersion1) && i < len(mainVersion2); i++ {
		n1, err1 := strconv.Atoi(mainVersion1[i])
		n2, err2 := strconv.Atoi(mainVersion2[i])

		if err1 != nil || err2 != nil {
			if mainVersion1[i] > mainVersion2[i] {
				return 1
			}
			if mainVersion1[i] < mainVersion2[i] {
				return -1
			}
			continue
		}

		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}

	if len(v1Parts) == 1 && len(v2Parts) == 1 {
		return 0
	}
	if len(v1Parts) == 1 {
		return 1
	}
	if len(v2Parts) == 1 {
		return -1
	}

	prerelease1 := v1Parts[1]
	prerelease2 := v2Parts[1]

	weights := map[string]int{
		"alpha": 1,
		"beta":  2,
		"rc":    3,
	}

	type1 := strings.Split(prerelease1, ".")
	type2 := strings.Split(prerelease2, ".")

	weight1 := weights[type1[0]]
	weight2 := weights[type2[0]]

	if weight1 != weight2 {
		if weight1 > weight2 {
			return 1
		}
		return -1
	}

	if len(type1) > 1 && len(type2) > 1 {
		num1, err1 := strconv.Atoi(type1[1])
		num2, err2 := strconv.Atoi(type2[1])
		if err1 == nil && err2 == nil {
			if num1 > num2 {
				return 1
			}
			if num1 < num2 {
				return -1
			}
		}
	}

	return 0
}

func getConnectionInfoInfra(sgID string, connectionID string) (*infra.Infra, error) {
	data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress+
		":"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort+
		"/honeybee/source_group/"+sgID+"/connection_info/"+connectionID+"/infra", "", "")
	if err != nil {
		return nil, err
	}

	var infraInfo infra.Infra
	err = json.Unmarshal(data, &infraInfo)
	if err != nil {
		return nil, err
	}

	return &infraInfo, nil
}

func processSoftwarePackages(infraInfo *infra.Infra, packages []model.Package) ([]model.PackageMigrationInfo, []string) {
	migrationPackages := make([]model.PackageMigrationInfo, 0)
	errMsgs := make([]string, 0)

	existingByName := make(map[string]int)

	var i int
	for _, pkg := range packages {
		sw, err := findMatchedPackageMigrationConfig(
			infraInfo.Compute.OS.OS.Name,
			infraInfo.Compute.OS.OS.VersionID,
			model.SoftwareArchitecture(infraInfo.Compute.OS.Kernel.Architecture),
			pkg.Name,
			pkg.Version,
		)
		if err != nil {
			errMsg := fmt.Sprintf(
				"Package: Name=%s, Version=%s, Error=%s",
				pkg.Name,
				pkg.Version,
				err.Error(),
			)
			logger.Println(logger.DEBUG, true, errMsg)
			errMsgs = append(errMsgs, errMsg)
			continue
		}

		if sw == nil {
			i++

			migrationPackages = append(migrationPackages, model.PackageMigrationInfo{
				Order:                    i,
				Name:                     pkg.Name,
				Version:                  pkg.Version,
				NeededPackages:           strings.Split(pkg.NeededPackages, ","),
				NeedToDeletePackages:     strings.Split(pkg.NeedToDeletePackages, ","),
				CustomDataPaths:          []string{},
				CustomConfigs:            pkg.CustomConfigs,
				RepoURL:                  pkg.RepoURL,
				GPGKeyURL:                pkg.GPGKeyURL,
				RepoUseOSVersionCode:     pkg.RepoUseOSVersionCode,
				PackageMigrationConfigID: "",
			})

			continue
		}

		if idx, exists := existingByName[sw.Name]; exists {
			existingVersion := migrationPackages[idx].Version
			// Skip if existingVersion is greater than or equal to sw.Version
			// e.g: existingVersion="1.2.0", sw.Version="1.1.0" => compareVersions returns 1  => skip
			// e.g: existingVersion="1.2.0", sw.Version="1.2.0" => compareVersions returns 0  => skip
			// e.g: existingVersion="1.2.0-rc.1", sw.Version="1.2.0-beta.2" => compareVersions returns 1 => skip
			if compareVersions(existingVersion, sw.Version) >= 0 {
				continue
			}
			migrationPackages = append(migrationPackages[:idx], migrationPackages[idx+1:]...)
		}

		i++

		newSoftware := model.PackageMigrationInfo{
			Order:                    i,
			Name:                     sw.Name,
			Version:                  sw.Version,
			NeededPackages:           strings.Split(sw.NeededPackages, ","),
			NeedToDeletePackages:     strings.Split(sw.NeedToDeletePackages, ","),
			CustomDataPaths:          []string{},
			CustomConfigs:            sw.CustomConfigs,
			RepoURL:                  sw.RepoURL,
			GPGKeyURL:                sw.GPGKeyURL,
			RepoUseOSVersionCode:     sw.RepoUseOSVersionCode,
			PackageMigrationConfigID: sw.ID,
		}

		migrationPackages = append(migrationPackages, newSoftware)
		existingByName[sw.Name] = len(migrationPackages) - 1
	}

	return migrationPackages, errMsgs
}

func MakeMigrationListRes(sourceGroupSoftwareProperty *model.SourceGroupSoftwareProperty) (*model.MigrationListRes, error) {
	data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress+
		":"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort+
		"/honeybee/source_group/"+sourceGroupSoftwareProperty.SourceGroupId+"/connection_info", "", "")
	if err != nil {
		return nil, err
	}

	var listConnectionInfoRes honeybee.ListConnectionInfoRes
	err = json.Unmarshal(data, &listConnectionInfoRes)
	if err != nil {
		return nil, err
	}

	var servers []model.MigrationServer
	var sources model.SourceGroupSoftwareProperty

	for _, source := range sources.ConnectionInfoList {
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

		var server model.MigrationServer

		srcInfra, err := getConnectionInfoInfra(sourceGroupSoftwareProperty.SourceGroupId, source.ConnectionId)
		if err != nil {
			return nil, err
		}

		server.MigrationList.Packages, server.Errors = processSoftwarePackages(srcInfra, source.Softwares.Packages)
		server.ConnectionInfoID = source.ConnectionId

		servers = append(servers, server)
	}

	return &model.MigrationListRes{
		Servers: servers,
	}, nil
}
