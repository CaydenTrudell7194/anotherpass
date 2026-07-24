package model

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDatabase(dbPath string) error {
	var err error
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}
	if err := DB.AutoMigrate(&User{}, &UserGroup{}, &DeviceGroup{}, &Node{}, &ForwardRule{}, &SystemConfig{}, &ServicePlan{}, &Order{}); err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_group_instance_unique ON nodes(device_group_id, instance_id) WHERE instance_id <> ''").Error; err != nil {
		return err
	}
	// Older releases created immediately-deployed rules with a pending status.
	return DB.Model(&ForwardRule{}).Where("status = ?", "pending").Update("status", "active").Error
}
