package model

type Target struct {
	UUID            string `gorm:"primaryKey"`
	HoneybeeAddress string `gorm:"column:honeybee_address"`
}
