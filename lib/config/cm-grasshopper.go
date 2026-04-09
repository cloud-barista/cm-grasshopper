package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/jollaman999/utils/fileutil"
	"gopkg.in/yaml.v3"
)

type cmGrasshopperConfig struct {
	CMGrasshopper struct {
		Listen struct {
			Port string `yaml:"port"`
		} `yaml:"listen"`
		Software struct {
			TempFolder string `yaml:"temp_folder"`
			LogFolder  string `yaml:"log_folder"`
		} `yaml:"software"`
		Ansible struct {
			PlaybookRootPath string `yaml:"playbook_root_path"`
		} `yaml:"ansible"`
		Honeybee struct {
			ServerAddress string `yaml:"server_address"`
			ServerPort    string `yaml:"server_port"`
		} `yaml:"honeybee"`
		Tumblebug struct {
			ServerAddress string `yaml:"server_address"`
			ServerPort    string `yaml:"server_port"`
			Username      string `yaml:"username"`
			Password      string `yaml:"password"`
		} `yaml:"tumblebug"`
		K8s struct {
			JobWorkerCount    int    `yaml:"job_worker_count"`
			JobLogFolder      string `yaml:"job_log_folder"`
			InstallTimeoutMin int    `yaml:"install_timeout_minute"`
			BackupTimeoutMin  int    `yaml:"backup_timeout_minute"`
			RestoreTimeoutMin int    `yaml:"restore_timeout_minute"`
		} `yaml:"k8s"`
	} `yaml:"cm-grasshopper"`
}

var CMGrasshopperConfig cmGrasshopperConfig
var cmGrasshopperConfigFile = "cm-grasshopper.yaml"

func checkCMGrasshopperConfigFile() error {
	if CMGrasshopperConfig.CMGrasshopper.Listen.Port == "" {
		return errors.New("config error: cm-grasshopper.listen.port is empty")
	}
	port, err := strconv.Atoi(CMGrasshopperConfig.CMGrasshopper.Listen.Port)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("config error: cm-grasshopper.listen.port has invalid value")
	}

	if CMGrasshopperConfig.CMGrasshopper.Software.TempFolder == "" {
		return errors.New("config error: cm-grasshopper.software.temp_folder is empty")
	}
	if !fileutil.IsExist(CMGrasshopperConfig.CMGrasshopper.Software.TempFolder) {
		return errors.New("config error: cm-grasshopper.software.temp_folder (" +
			CMGrasshopperConfig.CMGrasshopper.Software.TempFolder + ") is not exist")
	}

	if CMGrasshopperConfig.CMGrasshopper.Software.LogFolder == "" {
		return errors.New("config error: cm-grasshopper.software.log_folder is empty")
	}
	if !fileutil.IsExist(CMGrasshopperConfig.CMGrasshopper.Software.LogFolder) {
		return errors.New("config error: cm-grasshopper.software.log_folder(" +
			CMGrasshopperConfig.CMGrasshopper.Software.LogFolder + ") is not exist")
	}

	if CMGrasshopperConfig.CMGrasshopper.Ansible.PlaybookRootPath == "" {
		return errors.New("config error: cm-grasshopper.ansible.playbook_root_path is empty")
	}
	if !fileutil.IsExist(CMGrasshopperConfig.CMGrasshopper.Ansible.PlaybookRootPath) {
		return errors.New("config error: cm-grasshopper.ansible.playbook_root_path (" +
			CMGrasshopperConfig.CMGrasshopper.Ansible.PlaybookRootPath + ") is not exist")
	}

	if CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort == "" {
		return errors.New("config error: cm-grasshopper.honeybee.ServerPort is empty")
	}
	port, err = strconv.Atoi(CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("config error: cm-grasshopper.honeybee.ServerPort has invalid value")
	}

	if CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerPort == "" {
		return errors.New("config error: cm-grasshopper.tumblebug.ServerPort is empty")
	}
	port, err = strconv.Atoi(CMGrasshopperConfig.CMGrasshopper.Tumblebug.ServerPort)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("config error: cm-grasshopper.tumblebug.ServerPort has invalid value")
	}

	if CMGrasshopperConfig.CMGrasshopper.K8s.JobWorkerCount < 1 {
		return errors.New("config error: cm-grasshopper.k8s.job_worker_count must be greater than 0")
	}
	if CMGrasshopperConfig.CMGrasshopper.K8s.JobLogFolder == "" {
		return errors.New("config error: cm-grasshopper.k8s.job_log_folder is empty")
	}
	if !fileutil.IsExist(CMGrasshopperConfig.CMGrasshopper.K8s.JobLogFolder) {
		return errors.New("config error: cm-grasshopper.k8s.job_log_folder (" +
			CMGrasshopperConfig.CMGrasshopper.K8s.JobLogFolder + ") is not exist")
	}

	return nil
}

func readCMGrasshopperConfigFile() error {
	common.RootPath = os.Getenv(common.ModuleROOT)
	if len(common.RootPath) == 0 {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		common.RootPath = homeDir + "/." + strings.ToLower(common.ModuleName)
	}

	err := fileutil.CreateDirIfNotExist(common.RootPath)
	if err != nil {
		return err
	}

	ex, err := os.Executable()
	if err != nil {
		return err
	}

	exPath := filepath.Dir(ex)
	configDir := exPath + "/conf"
	if !fileutil.IsExist(configDir) {
		configDir = common.RootPath + "/conf"
	}

	data, err := os.ReadFile(configDir + "/" + cmGrasshopperConfigFile)
	if err != nil {
		return errors.New("can't find the config file (" + cmGrasshopperConfigFile + ")" + fmt.Sprintln() +
			"Must be placed in '." + strings.ToLower(common.ModuleName) + "/conf' directory " +
			"under user's home directory or 'conf' directory where running the binary " +
			"or 'conf' directory where placed in the path of '" + common.ModuleROOT + "' environment variable")
	}

	err = yaml.Unmarshal(data, &CMGrasshopperConfig)
	if err != nil {
		return err
	}

	err = checkCMGrasshopperConfigFile()
	if err != nil {
		return err
	}

	return nil
}

func prepareCMGrasshopperConfig() error {
	err := readCMGrasshopperConfigFile()
	if err != nil {
		return err
	}

	return nil
}

func GetK8sInstallTimeout() time.Duration {
	minutes := CMGrasshopperConfig.CMGrasshopper.K8s.InstallTimeoutMin
	if minutes < 1 {
		minutes = 30
	}
	return time.Duration(minutes) * time.Minute
}

func GetK8sBackupTimeout() time.Duration {
	minutes := CMGrasshopperConfig.CMGrasshopper.K8s.BackupTimeoutMin
	if minutes < 1 {
		minutes = 30
	}
	return time.Duration(minutes) * time.Minute
}

func GetK8sRestoreTimeout() time.Duration {
	minutes := CMGrasshopperConfig.CMGrasshopper.K8s.RestoreTimeoutMin
	if minutes < 1 {
		minutes = 30
	}
	return time.Duration(minutes) * time.Minute
}
