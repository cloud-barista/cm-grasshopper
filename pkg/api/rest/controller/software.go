package controller

import (
	"embed"
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/dao"
	"github.com/cloud-barista/cm-grasshopper/lib/config"
	"github.com/cloud-barista/cm-grasshopper/lib/software"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/google/uuid"
	"github.com/jollaman999/utils/fileutil"
	"github.com/labstack/echo/v4"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//go:embed playbook_base
var playbookBase embed.FS

func writePlaybookFiles(softwareName string, destDir string, neededPackages []string, needToDeletePackages []string,
	repoURL string, gpgKeyURL string, repoUseOSVersionCode bool) error {
	err := fs.WalkDir(playbookBase, "playbook_base", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel("playbook_base", path)
		destPath := filepath.Join(destDir, relPath)

		err = fileutil.CreateDirIfNotExist(filepath.Dir(destPath))
		if err != nil {
			return err
		}

		data, err := playbookBase.ReadFile(path)
		if err != nil {
			return err
		}

		if strings.HasSuffix(destPath, "playbook.yml") {
			dataStr := string(data)
			dataStr = strings.ReplaceAll(dataStr, "SOFTWARE_NAME", softwareName)
			data = []byte(dataStr)
		} else if strings.HasSuffix(destPath, filepath.Join("tasks", "add_repo.yml")) {
			softwareName = strings.ReplaceAll(softwareName, "/", "_")
			softwareName = strings.ReplaceAll(softwareName, "\\", "_")
			softwareName = strings.ReplaceAll(softwareName, "\n", "_")
			softwareName = strings.ReplaceAll(softwareName, " ", "_")

			dataStr := string(data)
			dataStr = strings.ReplaceAll(dataStr, "SOFTWARE_NAME", softwareName)
			data = []byte(dataStr)
		} else if strings.HasSuffix(destPath, filepath.Join("tasks", "main.yml")) {
			if repoURL == "" {
				dataStr := string(data)
				dataStr = strings.ReplaceAll(dataStr, "- import_tasks: add_repo.yml\n", "")
				data = []byte(dataStr)
			}
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return err
		}

		if strings.HasSuffix(destPath, filepath.Join("vars", "main.yml")) {
			var content string

			if len(neededPackages) > 0 {
				content += "packages_to_install:\n"
				for _, packageName := range neededPackages {
					content += "  - " + packageName + "\n"
				}
			} else {
				content += "packages_to_install: []\n"
			}

			if len(needToDeletePackages) > 0 {
				content += "packages_to_delete:\n"
				for _, packageName := range needToDeletePackages {
					content += "  - " + packageName + "\n"
				}
			} else {
				content += "packages_to_delete: []\n"
			}

			if repoURL != "" {
				content += "repo_url: " + repoURL + "\n"
				if gpgKeyURL != "" {
					content += "gpg_key_url: " + gpgKeyURL + "\n"
				}
			}

			if repoUseOSVersionCode {
				content += "repo_use_os_version_code: true\n"
			} else {
				content += "repo_use_os_version_code: false\n"
			}

			return fileutil.WriteFileAppend(destPath, content)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// RegisterSoftware godoc
//
//	@ID				register-software
//	@Summary		Register Software
//	@Description	Register the software.
//	@Tags			[Software]
//	@Accept			json
//	@Produce		json
//	@Param			softwareRegisterReq body model.SoftwareRegisterReq true "Software info"
//	@Success		200	{object}	model.SoftwareRegisterReq	"Successfully registered the software."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to sent SSH command."
//	@Router			/software/register [post]
func RegisterSoftware(c echo.Context) error {
	var err error

	softwareRegisterReq := new(model.SoftwareRegisterReq)
	err = c.Bind(softwareRegisterReq)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	err = model.CheckInstallType(softwareRegisterReq.InstallType)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}
	err = model.CheckArchitecture(softwareRegisterReq.Architecture)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if softwareRegisterReq.Name == "" {
		return common.ReturnErrorMsg(c, "Please provide the name")
	}

	if softwareRegisterReq.Version == "" {
		return common.ReturnErrorMsg(c, "Please provide the version")
	}

	if softwareRegisterReq.InstallType == "package" {
		if softwareRegisterReq.OS == "" {
			return common.ReturnErrorMsg(c, "Please provide the os")
		}

		if softwareRegisterReq.OSVersion == "" {
			return common.ReturnErrorMsg(c, "Please provide the os version")
		}
	}

	if softwareRegisterReq.InstallType == "helm" {
		if softwareRegisterReq.RepoURL == "" {
			return common.ReturnErrorMsg(c, "Please provide the repo url")
		}
	}

	if len(softwareRegisterReq.MatchNames) == 0 {
		return common.ReturnErrorMsg(c, "Please provide the match names")
	}
	var matchNames string
	for _, matchName := range softwareRegisterReq.MatchNames {
		if strings.Contains(matchName, ",") {
			return common.ReturnErrorMsg(c, "Match name should not contain ','")
		}
		matchNames += matchName + ","
	}
	matchNames = matchNames[:len(matchNames)-1]

	if len(softwareRegisterReq.NeededPackages) == 0 {
		return common.ReturnErrorMsg(c, "Please provide the needed packages")
	}
	var neededPackages string
	if len(softwareRegisterReq.NeededPackages) > 0 {
		for _, neededPackage := range softwareRegisterReq.NeededPackages {
			if strings.Contains(neededPackage, ",") {
				return common.ReturnErrorMsg(c, "Each name of needed_packages should not contain ','")
			}
			neededPackages += neededPackage + ","
		}
		neededPackages = neededPackages[:len(neededPackages)-1]
	}

	var needToDeletePackages string
	if len(needToDeletePackages) > 0 {
		for _, needToDeletePackage := range softwareRegisterReq.NeedToDeletePackages {
			if strings.Contains(needToDeletePackage, ",") {
				return common.ReturnErrorMsg(c, "Each name of need_to_delete_packages should not contain ','")
			}
			needToDeletePackages += needToDeletePackage + ","
		}
		needToDeletePackages = needToDeletePackages[:len(needToDeletePackages)-1]
	}

	var customConfigs string
	if len(softwareRegisterReq.CustomConfigs) > 0 {
		for _, customConfig := range softwareRegisterReq.CustomConfigs {
			if strings.Contains(customConfig, ",") {
				return common.ReturnErrorMsg(c, "Each name of custom_configs should not contain ','")
			}
			customConfigs += customConfig + ","
		}
		customConfigs = customConfigs[:len(customConfigs)-1]
	}

	var id = uuid.New().String()

	sw := model.Software{
		ID:                   id,
		InstallType:          softwareRegisterReq.InstallType,
		Name:                 softwareRegisterReq.Name,
		Version:              softwareRegisterReq.Version,
		OS:                   softwareRegisterReq.OS,
		OSVersion:            softwareRegisterReq.OSVersion,
		Architecture:         softwareRegisterReq.Architecture,
		MatchNames:           matchNames,
		NeededPackages:       neededPackages,
		NeedToDeletePackages: needToDeletePackages,
		CustomConfigs:        customConfigs,
		RepoURL:              softwareRegisterReq.RepoURL,
		GPGKeyURL:            softwareRegisterReq.GPGKeyURL,
		RepoUseOSVersionCode: softwareRegisterReq.RepoUseOSVersionCode,
	}

	dbSW, err := dao.SoftwareCreate(&sw)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	destDir := filepath.Join(config.CMGrasshopperConfig.CMGrasshopper.Ansible.PlaybookRootPath, sw.ID)
	err = writePlaybookFiles(softwareRegisterReq.Name, destDir,
		softwareRegisterReq.NeededPackages, softwareRegisterReq.NeedToDeletePackages,
		softwareRegisterReq.RepoURL, softwareRegisterReq.GPGKeyURL, softwareRegisterReq.RepoUseOSVersionCode)
	if err != nil {
		_ = fileutil.DeleteDir(destDir)
		_ = dao.SoftwareDelete(&sw)

		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, dbSW, " ")
}

// GetMigrationList godoc
//
//	@ID				get-migration-list
//	@Summary		Get Migration List
//	@Description	Get software migration list.
//	@Tags			[Software]
//	@Accept			json
//	@Produce		json
//	@Param			sgId path string true "ID of the SourceGroup"
//	@Success		200	{object}	model.MigrationListRes	"Successfully get software migration list."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to get software migration list."
//	@Router			/software/migration_list/{sgId} [get]
func GetMigrationList(c echo.Context) error {
	sgID := c.Param("sgId")
	if sgID == "" {
		return common.ReturnErrorMsg(c, "Please provide the sgId.")
	}
	fmt.Println(sgID)

	migrationListRes, err := software.MakeMigrationListRes(sgID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, *migrationListRes, " ")
}

// MigrateSoftware godoc
//
//	@ID				migrate-software
//	@Summary		Migrate Software
//	@Description	Migrate pieces of software to target.
//	@Tags			[Software]
//	@Accept			json
//	@Produce		json
//	@Param			softwareMigrateReq body model.SoftwareMigrateReq true "Software migrate request."
//	@Success		200	{object}	model.SoftwareMigrateRes	"Successfully migrated pieces of software."
//	@Failure		400	{object}	common.ErrorResponse		"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse		"Failed to migrate pieces of software."
//	@Router			/software/migrate [post]
func MigrateSoftware(c echo.Context) error {
	softwareInstallReq := new(model.SoftwareMigrateReq)
	err := c.Bind(softwareInstallReq)
	if err != nil {
		return err
	}

	var migrationList []model.MigrationSoftwareInfo

	for i, id := range softwareInstallReq.SoftwareIDs {
		sw, err := dao.SoftwareGet(id)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}

		migrationList = append(migrationList, model.MigrationSoftwareInfo{
			Order:               i + 1,
			SoftwareID:          sw.ID,
			SoftwareName:        sw.Name,
			SoftwareVersion:     sw.Version,
			SoftwareInstallType: sw.InstallType,
		})
	}

	executionID := uuid.New().String()

	err = software.MigrateSoftware(executionID, &migrationList,
		softwareInstallReq.SourceConnectionInfoID, &softwareInstallReq.Target)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SoftwareMigrateRes{
		ExecutionID:   executionID,
		MigrationList: migrationList,
	}, " ")
}

// GetSoftwareMigrationLog godoc
//
//	@ID				get-software-migration-log
//	@Summary		Get Software Migration Log
//	@Description	Get the software migration log.
//	@Tags			[Software]
//	@Accept			json
//	@Produce		json
//	@Param			executionId path string true "ID of the software migration execution."
//	@Success		200	{object}	model.MigrationLogRes	"Successfully get the software migration log"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to get the software migration log"
//	@Router			/software/migrate/log/{executionId} [get]
func GetSoftwareMigrationLog(c echo.Context) error {
	executionID := c.Param("executionId")
	if executionID == "" {
		return common.ReturnErrorMsg(c, "Please provide the executionId.")
	}

	path, err := filepath.Abs(filepath.Join(config.CMGrasshopperConfig.CMGrasshopper.Software.LogFolder, executionID))
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return common.ReturnErrorMsg(c, fmt.Sprintf("Log path for executionID %s not found", executionID))
	}

	response := model.MigrationLogRes{
		UUID: executionID,
	}

	if content, err := os.ReadFile(filepath.Join(path, "install.log")); err == nil {
		response.InstallLog = string(content)
	}

	if content, err := os.ReadFile(filepath.Join(path, "migration.log")); err == nil {
		response.MigrationLog = string(content)
	}

	if response.InstallLog == "" && response.MigrationLog == "" {
		return common.ReturnErrorMsg(c, fmt.Sprintf("No log files found for executionID %s", executionID))
	}

	return c.JSONPretty(http.StatusOK, response, " ")
}

// ListSoftware godoc
//
//	@ID				list-software
//	@Summary		List Software
//	@Description	Get a list of connection information.
//	@Tags			[Software]
//	@Accept			json
//	@Produce		json
//	@Param			install_type query string false "Installation type of the software"
//	@Param			name query string false "Name of the software"
//	@Param			version query string false "Version of the software"
//	@Param			os query string false "Operating system of the software"
//	@Param			os_version query string false "Operating system version"
//	@Param			architecture query string false "Architecture of the software"
//	@Param			match_names query string false "Matching names of the software"
//	@Param			needed_packages query string false "Packages needed to install for the software"
//	@Param			need_to_delete_packages query string false "Packages that need to be deleted for the software"
//	@Param			repo_url query string false "Repository URL for install the software"
//	@Param			gpg_key_url query string false "GPG key URL for install the software"
//	@Param			repo_use_os_version_code query bool false "If repository URL uses OS version code. (For debian based OSs.)"
//	@Success		200	{object}	[]model.Software	"Successfully get a list of software."
//	@Failure		400	{object}	common.ErrorResponse			"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse			"Failed to get a list of software."
//	@Router			/software [get]
func ListSoftware(c echo.Context) error {
	page, row, err := common.CheckPageRow(c)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	sw := &model.Software{
		InstallType:          c.QueryParam("install_type"),
		Name:                 c.QueryParam("name"),
		Version:              c.QueryParam("version"),
		OS:                   c.QueryParam("os"),
		OSVersion:            c.QueryParam("os_version"),
		Architecture:         c.QueryParam("architecture"),
		MatchNames:           c.QueryParam("match_names"),
		NeededPackages:       c.QueryParam("needed_packages"),
		NeedToDeletePackages: c.QueryParam("need_to_delete_packages"),
		RepoURL:              c.QueryParam("repo_url"),
		GPGKeyURL:            c.QueryParam("gpg_key_url"),
	}
	var isRepoUseOSVersionCodeSet bool
	sw.RepoUseOSVersionCode, err = strconv.ParseBool(c.QueryParam("repo_use_os_version_code"))
	if err == nil {
		isRepoUseOSVersionCodeSet = true
	}

	softwares, err := dao.SoftwareGetList(sw, isRepoUseOSVersionCodeSet, page, row)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, &softwares, " ")
}

// DeleteSoftware godoc
//
//	@ID				delete-software
//	@Summary		Delete Software
//	@Description	Delete the software.
//	@Tags			[Software]
//	@Accept			json
//	@Produce		json
//	@Param			softwareId path string true "ID of the software."
//	@Success		200	{object}	model.SimpleMsg			"Successfully update the software"
//	@Failure		400	{object}	common.ErrorResponse	"Sent bad request."
//	@Failure		500	{object}	common.ErrorResponse	"Failed to delete the software"
//	@Router			/software/{softwareId} [delete]
func DeleteSoftware(c echo.Context) error {
	swID := c.Param("softwareId")
	if swID == "" {
		return common.ReturnErrorMsg(c, "Please provide the softwareId.")
	}

	sw, err := dao.SoftwareGet(swID)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	if sw.InstallType == "ansible" {
		err = software.DeletePlaybook(swID)
		if err != nil {
			return common.ReturnErrorMsg(c, err.Error())
		}
	}

	err = dao.SoftwareDelete(sw)
	if err != nil {
		return common.ReturnErrorMsg(c, err.Error())
	}

	return c.JSONPretty(http.StatusOK, model.SimpleMsg{Message: "success"}, " ")
}
