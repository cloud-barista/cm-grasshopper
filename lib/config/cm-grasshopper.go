package config

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/jollaman999/utils/fileutil"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type cmGrasshopperConfig struct {
	CMGrasshopper struct {
		Listen struct {
			Port string `yaml:"port"`
		} `yaml:"listen"`
		Honeybee struct {
			ServerAddress string `yaml:"server_address"`
			ServerPort    string `yaml:"server_port"`
		} `yaml:"honeybee"`
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

	if CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort == "" {
		return errors.New("config error: cm-grasshopper.honeybee.ServerPort is empty")
	}
	port, err = strconv.Atoi(CMGrasshopperConfig.CMGrasshopper.Honeybee.ServerPort)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("config error: cm-grasshopper.honeybee.ServerPort has invalid value")
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
