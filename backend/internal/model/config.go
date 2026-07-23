package model

type SystemConfig struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Key   string `gorm:"uniqueIndex;size:128" json:"key"`
	Value string `gorm:"size:4096" json:"value"`
}
