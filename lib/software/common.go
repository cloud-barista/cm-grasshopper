package software

import (
	"fmt"
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

func getFolderSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}

func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
