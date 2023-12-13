package dao

import (
	"errors"
	"github.com/cloud-barista/cm-grasshopper/db"
	"github.com/cloud-barista/cm-grasshopper/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TargetRegister(honeybeeAddress string) (target *model.Target, err error) {
	UUID, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	target = &model.Target{
		UUID:            UUID.String(),
		HoneybeeAddress: honeybeeAddress,
	}

	result := db.DB.Create(target)
	err = result.Error
	if err != nil {
		return nil, err
	}

	return target, nil
}

func TargetGet(UUID string) (*model.Target, error) {
	target := &model.Target{}

	result := db.DB.Where("uuid = ?", UUID).Find(target)
	err := result.Error
	if err != nil {
		return nil, err
	}

	return target, nil
}

func TargetGetList(target *model.Target, page int, row int) (*[]model.Target, error) {
	targets := &[]model.Target{}

	result := db.DB.Scopes(func(d *gorm.DB) *gorm.DB {
		var filtered = d

		if len(target.UUID) != 0 {
			filtered = filtered.Where("uuid LIKE ?", "%"+target.UUID+"%")
		}

		if len(target.HoneybeeAddress) != 0 {
			filtered = filtered.Where("honeybee_address LIKE ?", "%"+target.HoneybeeAddress+"%")
		}

		if page != 0 && row != 0 {
			offset := (page - 1) * row

			return filtered.Offset(offset).Limit(row)
		} else if row != 0 && page == 0 {
			filtered.Error = errors.New("row is not 0 but page is 0")

			return nil
		} else if page != 0 && row == 0 {
			filtered.Error = errors.New("page is not 0 but row is 0")

			return nil
		}

		return filtered
	}).Find(targets)

	err := result.Error
	if err != nil {
		return nil, err
	}

	return targets, nil
}

func TargetDelete(target *model.Target) error {
	result := db.DB.Delete(target)
	err := result.Error
	if err != nil {
		return err
	}

	return nil
}
