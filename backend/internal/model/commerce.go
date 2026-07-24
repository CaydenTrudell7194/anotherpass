package model

import "time"

const (
	OrderStatusPending  = "pending"
	OrderStatusApproved = "approved"
	OrderStatusRejected = "rejected"
)

type ServicePlan struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	Description  string    `gorm:"size:500" json:"description"`
	PriceCents   int64     `gorm:"not null" json:"price_cents"`
	DurationDays int       `gorm:"not null" json:"duration_days"`
	RuleLimit    int       `gorm:"not null" json:"rule_limit"`
	UserGroupID  *uint     `json:"user_group_id,omitempty"`
	Enabled      bool      `gorm:"not null;index" json:"enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Order struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserID           uint       `gorm:"not null;index:idx_orders_user_status" json:"user_id"`
	PlanID           uint       `gorm:"not null;index" json:"plan_id"`
	PlanName         string     `gorm:"size:100;not null" json:"plan_name"`
	PlanPriceCents   int64      `gorm:"not null" json:"plan_price_cents"`
	PlanDurationDays int        `gorm:"not null" json:"plan_duration_days"`
	PlanRuleLimit    int        `gorm:"not null" json:"plan_rule_limit"`
	PlanUserGroupID  *uint      `json:"plan_user_group_id,omitempty"`
	Status           string     `gorm:"size:16;not null;index:idx_orders_user_status" json:"status"`
	UserNote         string     `gorm:"size:500" json:"user_note"`
	AdminNote        string     `gorm:"size:500" json:"admin_note"`
	ReviewedBy       *uint      `json:"reviewed_by,omitempty"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt        time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
