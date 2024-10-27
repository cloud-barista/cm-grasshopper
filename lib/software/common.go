package software

import (
	"github.com/jollaman999/utils/logger"
	"os"
	"path/filepath"
)

func getFileExtension(file string) string {
	ext := filepath.Ext(file)
	return ext
}

func getFiles(pwd string) ([]os.DirEntry, error) {
	files, err := os.ReadDir(pwd)
	if err != nil {
		errMsg := err.Error()
		logger.Logger.Println(logger.ERROR, true, errMsg)
		return nil, err
	}

	return files, nil
}

func deleteDir(path string) {
	err := os.RemoveAll(path)
	if err != nil {
		logger.Println(logger.ERROR, true, err)
	}
}

func moveDir(src string, dst string) error {
	err := os.Rename(src, dst)
	if err != nil {
		logger.Println(logger.ERROR, true, err)
		return err
	}

	return nil
}
