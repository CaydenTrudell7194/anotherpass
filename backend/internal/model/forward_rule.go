package model

import "time"

type ForwardRule struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index" json:"user_id"`
	Name          string    `gorm:"size:128" json:"name"`
	DeviceGroupID uint      `gorm:"index" json:"device_group_id"`
	ListenPort    int       `json:"listen_port"`
	TargetAddr    string    `gorm:"size:256" json:"target_addr"`
	TargetPort    int       `json:"target_port"`
	Protocol      string    `gorm:"size:16;default:tcp" json:"protocol"`
	Status        string    `gorm:"size:16;default:pending" json:"status"`
	Traffic       int64     `gorm:"default:0" json:"traffic"`
	Rate          float64   `gorm:"default:1" json:"rate"`
	Enabled       bool      `gorm:"default:true" json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
