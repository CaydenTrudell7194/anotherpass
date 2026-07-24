package model

import "time"

type UserNode struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"index" json:"user_id"`
	Name          string    `gorm:"size:128" json:"name"`
	Token         string    `gorm:"size:256;uniqueIndex" json:"-"`
	InstanceID    string    `gorm:"size:64" json:"instance_id"`
	IP            string    `gorm:"size:64" json:"ip"`
	Status        string    `gorm:"size:16;default:offline" json:"status"`
	LastHeartbeat time.Time `json:"last_heartbeat"`
	CreatedAt     time.Time `json:"created_at"`
}
