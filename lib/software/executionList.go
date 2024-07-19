package software

import (
	"errors"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/jollaman999/utils/logger"
	"strings"
)

func findMatchedSoftware(os string, osVersion string,
	architecture string, softwareInfo model.SoftwareInfo) (*model.Software, error) {
	list, err := dao.SoftwareGetList(&model.Software{
		MatchNames: softwareInfo.Name,
	}, 0, 0)
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
			if softwareInfo.Name == matchName {
				if softwareInfo.Version == sw.Version {
					return &sw, nil
				}
				matchedSoftwares = append(matchedSoftwares, sw)
			}
		}
	}

	var latestSoftware model.Software
	if len(matchedSoftwares) == 1 {
		if matchedSoftwares[0].Version == softwareInfo.Version {
			// Return if same version found
			return &matchedSoftwares[0], nil
		}
		latestSoftware = matchedSoftwares[0]
	} else if len(matchedSoftwares) > 1 {
		for i := 1; i < len(matchedSoftwares); i++ {
			if matchedSoftwares[i].Version == softwareInfo.Version {
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

			// Pick X.~~~ higher
			if len(latestSoftwareVersionSplit) > 0 && len(matchedSoftwareVersionSplit) > 0 {
				if latestSoftwareVersionSplit[0] < matchedSoftwareVersionSplit[0] {
					latestSoftware = matchedSoftwares[i]
					latestSoftwareVersionSplit = strings.Split(latestSoftware.Version, ".")

					// Pick 0.X.~~~ higher
					if len(latestSoftwareVersionSplit) > 1 && len(matchedSoftwareVersionSplit) > 1 {
						if latestSoftwareVersionSplit[1] < matchedSoftwareVersionSplit[1] {
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

	return nil, errors.New("no matched software found")
}

func MakeExecutionListRes(getExecutionListReq *model.GetExecutionListReq) (*model.GetExecutionListRes, error) {
	var executionList []model.Execution
	var i int
	var errMsgs []string

	for _, softwareInfo := range getExecutionListReq.SoftwareInfoList {
		software, err := findMatchedSoftware(getExecutionListReq.OS, getExecutionListReq.OSVersion,
			getExecutionListReq.Architecture, softwareInfo)
		if err != nil {
			errMsg := "Software: Name=" + softwareInfo.Name + ", Version=" + softwareInfo.Version + ", Error=" + err.Error()
			logger.Println(logger.DEBUG, true, errMsg)
			errMsgs = append(errMsgs, errMsg)
			continue
		}

		i++
		executionList = append(executionList, model.Execution{
			Order:               i,
			SoftwareID:          software.ID,
			SoftwareName:        software.Name,
			SoftwareVersion:     software.Version,
			SoftwareInstallType: software.InstallType,
		})
	}

	return &model.GetExecutionListRes{
		Executions: executionList,
		Errors:     errMsgs,
	}, nil
}
