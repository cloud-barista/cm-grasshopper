package software

import (
	"encoding/json"
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/infra"
	"github.com/cloud-barista/cm-honeybee/agent/pkg/api/rest/model/onprem/software"
	honeybee "github.com/cloud-barista/cm-honeybee/server/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"strconv"
	"strings"
)

func findMatchedSoftware(os string, osVersion string,
	architecture string, softwareName string, softwareVersion string) (*model.Software, error) {
	list, err := dao.SoftwareGetList(&model.Software{
		MatchNames: softwareName,
	}, false, 0, 0)
	if err != nil {
		return nil, err
	}

	var matchedSoftwares []model.Software
	for _, sw := range *list {
		if sw.Architecture != "common" &&
			(sw.OS != os || sw.OSVersion != osVersion || sw.Architecture != architecture) {
			continue
		}

		swMatchNames := strings.Split(sw.MatchNames, ",")
		for _, matchName := range swMatchNames {
			if softwareName == matchName {
				if softwareVersion == sw.Version {
					return &sw, nil
				}
				matchedSoftwares = append(matchedSoftwares, sw)
			}
		}
	}

	var latestSoftware model.Software
	if len(matchedSoftwares) == 1 {
		if matchedSoftwares[0].Version == softwareVersion {
			// Return if same version found
			return &matchedSoftwares[0], nil
		}
		latestSoftware = matchedSoftwares[0]
	} else if len(matchedSoftwares) > 1 {
		for i := 1; i < len(matchedSoftwares); i++ {
			if matchedSoftwares[i].Version == softwareVersion {
				// Return if same version found
				return &matchedSoftwares[i], nil
			}

			latestSoftwareVersionSplit := strings.Split(latestSoftware.Version, ".")
			matchedSoftwareVersionSplit := strings.Split(matchedSoftwares[i].Version, ".")

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
					latestSoftware = matchedSoftwares[i]
					latestSoftwareVersionSplit = strings.Split(latestSoftware.Version, ".")

					// Pick 0.X.~~~ higher (Pick higher minor version)
					if len(latestSoftwareVersionSplit) > 1 && len(matchedSoftwareVersionSplit) > 1 {
						latestMinor, _ := strconv.Atoi(latestSoftwareVersionSplit[1])
						matchedSoftwareMinor, _ := strconv.Atoi(matchedSoftwareVersionSplit[1])

						if latestMinor < matchedSoftwareMinor {
							latestSoftware = matchedSoftwares[i]
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

func getConnectionInfoSoftware(sgID string, connectionID string) (*software.Software, error) {
	data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress+
		":"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort+
		"/honeybee/source_group/"+sgID+"/connection_info/"+connectionID+"/software", "", "")
	if err != nil {
		return nil, err
	}

	var softwareInfo software.Software
	err = json.Unmarshal(data, &softwareInfo)
	if err != nil {
		return nil, err
	}

	return &softwareInfo, nil
}

type softwarePackage struct {
	Name    string
	Version string
}

func convertToPackages(packages interface{}) []softwarePackage {
	var result []softwarePackage

	switch p := packages.(type) {
	case []software.DEB:
		for _, pkg := range p {
			result = append(result, softwarePackage{
				Name:    pkg.Package,
				Version: pkg.Version,
			})
		}
	case []software.RPM:
		for _, pkg := range p {
			result = append(result, softwarePackage{
				Name:    pkg.Name,
				Version: pkg.Version,
			})
		}
	}

	return result
}

func getSoftwarePackages(idLike string, softwareInfo *software.Software) ([]softwarePackage, error) {
	switch {
	case strings.Contains(idLike, "debian"), strings.Contains(idLike, "ubuntu"):
		return convertToPackages(softwareInfo.DEB), nil
	case strings.Contains(idLike, "rhel"), strings.Contains(idLike, "fedora"):
		return convertToPackages(softwareInfo.RPM), nil
	default:
		return nil, fmt.Errorf("unsupported OS type: %s", idLike)
	}
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

func processSoftwarePackages(infraInfo *infra.Infra, packages []softwarePackage) ([]model.MigrationSoftwareInfo, []string) {
	migrationList := make([]model.MigrationSoftwareInfo, 0)
	errMsgs := make([]string, 0)

	existingByID := make(map[string]bool)
	existingByName := make(map[string]int)

	var i int
	for _, pkg := range packages {
		sw, err := findMatchedSoftware(
			infraInfo.Compute.OS.OS.Name,
			infraInfo.Compute.OS.OS.VersionID,
			infraInfo.Compute.OS.Kernel.Architecture,
			pkg.Name,
			pkg.Version,
		)
		if err != nil {
			errMsg := fmt.Sprintf(
				"Software: Name=%s, Version=%s, Error=%s",
				pkg.Name,
				pkg.Version,
				err.Error(),
			)
			logger.Println(logger.DEBUG, true, errMsg)
			errMsgs = append(errMsgs, errMsg)
			continue
		}

		if sw == nil {
			continue
		}

		if existingByID[sw.ID] {
			continue
		}

		if idx, exists := existingByName[sw.Name]; exists {
			existingVersion := migrationList[idx].SoftwareVersion
			// Skip if existingVersion is greater than or equal to sw.Version
			// e.g: existingVersion="1.2.0", sw.Version="1.1.0" => compareVersions returns 1  => skip
			// e.g: existingVersion="1.2.0", sw.Version="1.2.0" => compareVersions returns 0  => skip
			// e.g: existingVersion="1.2.0-rc.1", sw.Version="1.2.0-beta.2" => compareVersions returns 1 => skip
			if compareVersions(existingVersion, sw.Version) >= 0 {
				continue
			}
			migrationList = append(migrationList[:idx], migrationList[idx+1:]...)
			delete(existingByID, migrationList[idx].SoftwareID)
		}

		i++

		newSoftware := model.MigrationSoftwareInfo{
			Order:               i,
			SoftwareID:          sw.ID,
			SoftwareName:        sw.Name,
			SoftwareVersion:     sw.Version,
			SoftwareInstallType: sw.InstallType,
		}

		migrationList = append(migrationList, newSoftware)
		existingByID[sw.ID] = true
		existingByName[sw.Name] = len(migrationList) - 1
	}

	return migrationList, errMsgs
}

func MakeMigrationListRes(sgID string) (*model.MigrationListRes, error) {
	data, err := common.GetHTTPRequest("http://"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerAddress+
		":"+config.CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort+
		"/honeybee/source_group/"+sgID+"/connection_info", "", "")
	if err != nil {
		return nil, err
	}

	var listConnectionInfoRes honeybee.ListConnectionInfoRes
	err = json.Unmarshal(data, &listConnectionInfoRes)
	if err != nil {
		return nil, err
	}

	var servers []model.MigrationServer

	for _, encryptedConnectionInfo := range listConnectionInfoRes.ConnectionInfo {
		var server model.MigrationServer

		infraInfo, err := getConnectionInfoInfra(sgID, encryptedConnectionInfo.ID)
		if err != nil {
			return nil, err
		}

		softwareInfo, err := getConnectionInfoSoftware(sgID, encryptedConnectionInfo.ID)
		if err != nil {
			return nil, err
		}

		packages, err := getSoftwarePackages(infraInfo.Compute.OS.OS.IDLike, softwareInfo)
		if err != nil {
			return nil, err
		}

		server.ConnectionInfoID = encryptedConnectionInfo.ID
		server.MigrationList, server.Errors = processSoftwarePackages(infraInfo, packages)
		servers = append(servers, server)
	}

	return &model.MigrationListRes{
		Server: servers,
	}, nil
}
