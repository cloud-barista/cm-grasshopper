package db

import (
	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/glebarez/sqlite"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
	"gorm.io/gorm"
	"io"
	"os"
)

var SoftwaresDB *gorm.DB
var DB *gorm.DB

func copyFile(src string, dst string) (err error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = sourceFile.Close()
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = destFile.Close()
	}()

	_, err = io.Copy(destFile, sourceFile)

	return err
}

func Open() error {
	var err error

	sourceDB := "softwares.db"
	targetPath := common.RootPath + "/softwares.db"
	if !fileutil.IsExist(targetPath) {
		if fileutil.IsExist(sourceDB) {
			err := copyFile(sourceDB, targetPath)
			if err != nil {
				return err
			}
		}
	}

	SoftwaresDB, err = gorm.Open(sqlite.Open(common.RootPath+"/softwares.db"), &gorm.Config{})
	if err != nil {
		logger.Panicln(logger.ERROR, true, err)
	}

	err = SoftwaresDB.AutoMigrate(&model.Software{})
	if err != nil {
		logger.Panicln(logger.ERROR, true, err)
	}

	DB, err = gorm.Open(sqlite.Open(common.RootPath+"/"+common.ModuleName+".db"), &gorm.Config{})
	if err != nil {
		logger.Panicln(logger.ERROR, true, err)
	}

	err = DB.AutoMigrate(&model.SoftwareInstallStatus{})
	if err != nil {
		logger.Panicln(logger.ERROR, true, err)
	}

	return err
}

func Close() {
	if SoftwaresDB != nil {
		sqlDB, _ := SoftwaresDB.DB()
		_ = sqlDB.Close()
	}

	if DB != nil {
		sqlDB, _ := DB.DB()
		_ = sqlDB.Close()
	}
}
