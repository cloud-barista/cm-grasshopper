package dao

import (
	"errors"
	"github.com/cloud-barista/cm-grasshopper/db"
	"github.com/cloud-barista/cm-grasshopper/pkg/api/rest/model"
	"gorm.io/gorm"
	"time"
)

func SoftwareCreate(software *model.Software) (*model.Software, error) {
	software.CreatedAt = time.Now()
	software.UpdatedAt = time.Now()

	result := db.DB.Create(software)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return software, nil
}

func SoftwareGet(id string) (*model.Software, error) {
	software := &model.Software{}

	// Ensure db.DB is not nil to avoid runtime panics
	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.DB.Where("id = ?", id).First(software)
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

	// Ensure db.DB is not nil to avoid runtime panics
	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.DB.Where("name = ?", name).First(software)
	err := result.Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("software not found with the provided name")
		}
		return nil, err
	}

	return software, nil
}

func SoftwareGetList(software *model.Software, page int, row int) (*[]model.Software, error) {
	SoftwareList := &[]model.Software{}
	// Ensure db.DB is not nil to avoid runtime panics
	if db.DB == nil {
		return nil, errors.New("database connection is not initialized")
	}

	result := db.DB.Scopes(func(d *gorm.DB) *gorm.DB {
		var filtered = d

		if len(software.Name) != 0 {
			filtered = filtered.Where("name LIKE ?", "%"+software.Name+"%")
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

	result := db.DB.Model(&model.Software{}).Where("id = ?", software.ID).Updates(software)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}

func SoftwareDelete(software *model.Software) error {
	result := db.DB.Delete(software)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
