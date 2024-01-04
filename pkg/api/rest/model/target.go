package model

type Target struct {
	UUID            string `gorm:"primaryKey" json:"uuid"`
	HoneybeeAddress string `gorm:"column:honeybee_address" json:"honeybee_address"`
}
