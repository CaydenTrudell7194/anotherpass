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
	if err := DB.AutoMigrate(&User{}, &UserGroup{}, &DeviceGroup{}, &Node{}, &ForwardRule{}, &SystemConfig{}, &ServicePlan{}, &Order{}, &BalanceLedger{}, &RechargeOrder{}, &UserNode{}, &Affiliate{}, &RedeemCode{}); err != nil {
		return err
	}
	for _, statement := range []string{
		`CREATE TRIGGER IF NOT EXISTS balance_ledgers_no_update BEFORE UPDATE ON balance_ledgers BEGIN SELECT RAISE(ABORT, 'balance ledgers are immutable'); END`,
		`CREATE TRIGGER IF NOT EXISTS balance_ledgers_no_delete BEFORE DELETE ON balance_ledgers BEGIN SELECT RAISE(ABORT, 'balance ledgers are immutable'); END`,
	} {
		if err := DB.Exec(statement).Error; err != nil {
			return err
		}
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_user_nodes_token ON user_nodes(token)").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_nodes_group_instance_unique ON nodes(device_group_id, instance_id) WHERE instance_id <> ''").Error; err != nil {
		return err
	}
	if err := DB.Exec("DROP INDEX IF EXISTS idx_orders_idempotency_key_unique").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_orders_user_idempotency_unique ON orders(user_id, idempotency_key) WHERE idempotency_key <> ''").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_recharge_provider_trade_unique ON recharge_orders(provider, provider_trade_no) WHERE provider_trade_no <> ''").Error; err != nil {
		return err
	}
	if err := DB.Exec("DROP INDEX IF EXISTS idx_recharge_idempotency_unique").Error; err != nil {
		return err
	}
	if err := DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_recharge_user_idempotency_unique ON recharge_orders(user_id, idempotency_key) WHERE idempotency_key <> ''").Error; err != nil {
		return err
	}
	if err := DB.Exec("UPDATE orders SET paid_cents = plan_price_cents, fulfilled_at = reviewed_at WHERE status = 'approved' AND payment_method = 'manual' AND fulfilled_at IS NULL").Error; err != nil {
		return err
	}
	// Older releases created immediately-deployed rules with a pending status.
	return DB.Model(&ForwardRule{}).Where("status = ?", "pending").Update("status", "active").Error
}
