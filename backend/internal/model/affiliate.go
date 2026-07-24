package model

import "time"

type Affiliate struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	UserID           uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	Code             string    `gorm:"uniqueIndex;size:8;not null" json:"code"`
	CommissionRate   float64   `gorm:"default:0" json:"commission_rate"`
	TotalEarnedCents int64     `gorm:"default:0" json:"total_earned_cents"`
	CreatedAt        time.Time `json:"created_at"`
}

type AffLog struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	ReferrerID      uint      `gorm:"index;not null" json:"referrer_id"`
	ReferredID      uint      `gorm:"index;not null" json:"referred_id"`
	OrderID         *uint     `json:"order_id,omitempty"`
	AmountCents     int64     `gorm:"not null" json:"amount_cents"`
	CommissionCents int64     `gorm:"not null" json:"commission_cents"`
	CreatedAt       time.Time `json:"created_at"`
}
