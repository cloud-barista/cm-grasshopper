package dao

import (
	"errors"
	"github.com/cloud-barista/cm-grasshopper/db"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"gorm.io/gorm"
	"time"
)

func SoftwareCreate(software *model.Software) (*model.Software, error) {
	now := time.Now()
	software.CreatedAt = now
	software.UpdatedAt = now

	result := db.SoftwaresDB.Create(software)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return software, nil
}

func SoftwareGet(id string) (*model.Software, error) {
	software := &model.Software{}

	// Ensure db.SoftwaresDB is not nil to avoid runtime panics
	if db.SoftwaresDB == nil {
		return nil, errors.New("softwares database is not initialized")
	}

	result := db.SoftwaresDB.Where("id = ?", id).First(software)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("software not found with the provided id")
		}
		return nil, err
	}

	return software, nil
}

func SoftwareGetByName(name string) (*model.Software, error) {
	software := &model.Software{}

	// Ensure db.SoftwaresDB is not nil to avoid runtime panics
	if db.SoftwaresDB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.SoftwaresDB.Where("name = ?", name).First(software)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("software not found with the provided name")
		}
		return nil, err
	}

	return software, nil
}

func SoftwareGetList(software *model.Software, isRepoUseOSVersionCodeSet bool, page int, row int) (*[]model.Software, error) {
	SoftwareList := &[]model.Software{}
	// Ensure db.SoftwaresDB is not nil to avoid runtime panics
	if db.SoftwaresDB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.SoftwaresDB.Scopes(func(d *gorm.DB) *gorm.DB {
		var filtered = d

		if len(software.InstallType) != 0 {
			filtered = filtered.Where("install_type LIKE ?", "%"+software.InstallType+"%")
		}

		if len(software.Name) != 0 {
			filtered = filtered.Where("name LIKE ?", "%"+software.Name+"%")
		}

		if len(software.Version) != 0 {
			filtered = filtered.Where("version LIKE ?", "%"+software.Version+"%")
		}

		if len(software.OS) != 0 {
			filtered = filtered.Where("os LIKE ?", "%"+software.OS+"%")
		}

		if len(software.OSVersion) != 0 {
			filtered = filtered.Where("os_version LIKE ?", "%"+software.OSVersion+"%")
		}

		if len(software.Architecture) != 0 {
			filtered = filtered.Where("architecture LIKE ?", "%"+software.Architecture+"%")
		}

		if len(software.MatchNames) != 0 {
			filtered = filtered.Where("match_names LIKE ?", "%"+software.MatchNames+"%")
		}

		if len(software.NeededPackages) != 0 {
			filtered = filtered.Where("needed_packages LIKE ?", "%"+software.NeededPackages+"%")
		}

		if len(software.NeedToDeletePackages) != 0 {
			filtered = filtered.Where("need_to_delete_packages LIKE ?", "%"+software.NeedToDeletePackages+"%")
		}

		if len(software.RepoURL) != 0 {
			filtered = filtered.Where("repo_url LIKE ?", "%"+software.RepoURL+"%")
		}

		if len(software.GPGKeyURL) != 0 {
			filtered = filtered.Where("gpg_key_url LIKE ?", "%"+software.GPGKeyURL+"%")
		}

		if isRepoUseOSVersionCodeSet {
			filtered = filtered.Where("repo_use_os_version_code = ?", software.RepoUseOSVersionCode)
		}

		if page != 0 && row != 0 {
			offset := (page - 1) * row

			return filtered.Offset(offset).Limit(row)
		} else if row != 0 && page == 0 {
			filtered.Error = errors.New("row is not 0 but page is 0")
			return filtered
		} else if page != 0 && row == 0 {
			filtered.Error = errors.New("page is not 0 but row is 0")
			return filtered
		}
		return filtered
	}).Find(SoftwareList)

	err := result.Error
	if err != nil {
		return nil, err
	}

	return SoftwareList, nil
}

func SoftwareUpdate(software *model.Software) error {
	software.UpdatedAt = time.Now()

	result := db.SoftwaresDB.Model(&model.Software{}).Where("id = ?", software.ID).Updates(software)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SoftwareDelete(software *model.Software) error {
	result := db.SoftwaresDB.Delete(software)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
