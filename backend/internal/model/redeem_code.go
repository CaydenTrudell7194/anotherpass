package model

import "time"

type RedeemCode struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Code        string     `gorm:"size:32;uniqueIndex;not null" json:"code"`
	AmountCents int64      `gorm:"not null" json:"amount_cents"`
	MaxUses     int        `gorm:"default:1" json:"max_uses"`
	UsedCount   int        `gorm:"default:0" json:"used_count"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}
