package db

import (
	"github.com/cloud-barista/cm-grasshopper/common"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"github.com/glebarez/sqlite"
	"github.com/jollaman999/utils/logger"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Open() error {
	var err error

	DB, err = gorm.Open(sqlite.Open(common.ModuleName+".db"), &gorm.Config{})
	if err != nil {
		logger.Panicln(logger.ERROR, true, err)
	}

	err = DB.AutoMigrate(&model.Software{})
	if err != nil {
		logger.Panicln(logger.ERROR, true, err)
	}

	return err
}

func Close() {
	if DB != nil {
		sqlDB, _ := DB.DB()
		_ = sqlDB.Close()
	}
}
