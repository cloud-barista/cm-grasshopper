package db

import (
	"embed"
	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/glebarez/sqlite"
	"github.com/jollaman999/utils/fileutil"
	"github.com/jollaman999/utils/logger"
	"gorm.io/gorm"
	"os"
)

//go:embed softwares.db
var embeddedDB embed.FS

var SoftwaresDB *gorm.DB
var DB *gorm.DB

func copyEmbeddedDB(dst string) error {
	dbBytes, err := embeddedDB.ReadFile("softwares.db")
	if err != nil {
		return err
	}

	return os.WriteFile(dst, dbBytes, 0644)
}

func Open() error {
	var err error

	targetPath := common.RootPath + "/softwares.db"
	if !fileutil.IsExist(targetPath) {
		err := copyEmbeddedDB(targetPath)
		if err != nil {
			return err
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
