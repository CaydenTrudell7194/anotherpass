package model

import "time"

const (
	OrderStatusPending  = "pending"
	OrderStatusApproved = "approved"
	OrderStatusRejected = "rejected"

	PaymentMethodManual  = "manual"
	PaymentMethodBalance = "balance"

	LedgerKindOrderDebit      = "order_debit"
	LedgerKindAdminAdjustment = "admin_adjustment"
	LedgerKindRecharge        = "recharge"

	RechargeProviderEpay    = "epay"
	RechargeProviderCodepay = "codepay"
	RechargeStatusPending   = "pending"
	RechargeStatusPaid      = "paid"
	RechargeStatusFailed    = "failed"
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
	PaymentMethod    string     `gorm:"size:16;not null;default:manual" json:"payment_method"`
	PaidCents        int64      `gorm:"not null;default:0" json:"paid_cents"`
	IdempotencyKey   string     `gorm:"size:128;index" json:"-"`
	FulfilledAt      *time.Time `json:"fulfilled_at,omitempty"`
	CreatedAt        time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type BalanceLedger struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	UserID            uint      `gorm:"not null;index" json:"user_id"`
	DeltaCents        int64     `gorm:"not null" json:"delta_cents"`
	BalanceAfterCents int64     `gorm:"not null" json:"balance_after_cents"`
	Kind              string    `gorm:"size:32;not null;index" json:"kind"`
	OrderID           *uint     `gorm:"index" json:"order_id,omitempty"`
	ActorUserID       *uint     `gorm:"index" json:"actor_user_id,omitempty"`
	OperationKey      string    `gorm:"size:128;not null;uniqueIndex" json:"-"`
	RequestHash       string    `gorm:"size:64;not null" json:"-"`
	Note              string    `gorm:"size:500;not null" json:"note"`
	CreatedAt         time.Time `gorm:"not null;index" json:"created_at"`
}

type RechargeOrder struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	TradeNo         string     `gorm:"size:64;not null;uniqueIndex" json:"trade_no"`
	UserID          uint       `gorm:"not null;index" json:"user_id"`
	Provider        string     `gorm:"size:16;not null;index" json:"provider"`
	AmountCents     int64      `gorm:"not null" json:"amount_cents"`
	Status          string     `gorm:"size:16;not null;index" json:"status"`
	ProviderTradeNo string     `gorm:"size:128" json:"provider_trade_no,omitempty"`
	IdempotencyKey  string     `gorm:"size:128;index" json:"-"`
	RequestHash     string     `gorm:"size:64" json:"-"`
	PayURL          string     `gorm:"size:2048" json:"pay_url,omitempty"`
	CreatedAt       time.Time  `gorm:"not null;index" json:"created_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
}
