package model

import "time"

type User struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Username      string    `gorm:"uniqueIndex;size:64" json:"username"`
	Password      string    `gorm:"size:256" json:"-"`
	DisplayName   string    `gorm:"size:64" json:"display_name"`
	UserGroupID   uint      `gorm:"default:1" json:"user_group_id"`
	Status        string    `gorm:"default:active;size:16" json:"status"`
	TrafficLimit  int64     `gorm:"default:0" json:"traffic_limit"`
	TrafficUsed   int64     `gorm:"default:0" json:"traffic_used"`
	RuleLimit     int       `gorm:"default:100" json:"rule_limit"`
	ExpireAt      time.Time `json:"expire_at"`
	IsAdmin       bool      `gorm:"default:false" json:"is_admin"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type UserGroup struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;size:64" json:"name"`
	Description string    `gorm:"size:256" json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
