package model

import "time"

type AuditLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ActorID   uint      `gorm:"index;not null" json:"actor_id"`
	Action    string    `gorm:"size:64;not null;index" json:"action"`
	TargetID  *uint     `json:"target_id,omitempty"`
	Detail    string    `gorm:"size:1024" json:"detail"`
	CreatedAt time.Time `json:"created_at"`
}
