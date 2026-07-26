package handler

import (
	"time"

	"forward-panel/internal/model"

	"gorm.io/gorm"
)

func EnforceUserQuotas() {
	now := time.Now()
	var users []model.User
	model.DB.Where("status = ?", "active").Find(&users)
	for _, u := range users {
		quotaExhausted := false
		reason := ""
		if !u.ExpireAt.IsZero() && now.After(u.ExpireAt) {
			quotaExhausted = true
			reason = "套餐已到期"
		}
		if u.TrafficLimit > 0 && u.TrafficUsed >= u.TrafficLimit {
			quotaExhausted = true
			reason = "流量已用尽"
		}
		if quotaExhausted {
			// 停用该用户所有已启用规则
			model.DB.Model(&model.ForwardRule{}).Where("user_id = ? AND enabled = ?", u.ID, true).
				Update("enabled", false)
			continue
		}
		// 自动续费检查
		if u.AutoRenew && u.BalanceCents > 0 {
			tryAutoRenew(&u)
		}
	}
}

func tryAutoRenew(user *model.User) {
	now := time.Now()
	// 仅在到期前 1 天内尝试续费
	if !user.ExpireAt.IsZero() && user.ExpireAt.After(now.Add(-24*time.Hour)) {
		return
	}
	var plan model.ServicePlan
	if err := model.DB.Where("enabled = ?", true).Order("price_cents asc").First(&plan).Error; err != nil {
		return
	}
	if user.BalanceCents < plan.PriceCents {
		return
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.User{}).Where("id = ? AND balance_cents >= ?", user.ID, plan.PriceCents).
			UpdateColumn("balance_cents", gorm.Expr("balance_cents - ?", plan.PriceCents))
		if result.RowsAffected != 1 {
			return nil
		}
		base := now
		if user.ExpireAt.After(base) {
			base = user.ExpireAt
		}
		tx.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
			"rule_limit":    plan.RuleLimit,
			"traffic_limit": plan.TrafficLimit,
			"traffic_used":  0,
			"expire_at":     base.AddDate(0, 0, plan.DurationDays),
			"updated_at":    now,
		})
		return nil
	})
	if err == nil {
		user.BalanceCents -= plan.PriceCents
	}
}

func CheckUserRuleQuota(userID uint) (bool, string) {
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		return false, "用户不存在"
	}
	if !user.ExpireAt.IsZero() && time.Now().After(user.ExpireAt) {
		return false, "套餐已到期，请续费"
	}
	if user.TrafficLimit > 0 && user.TrafficUsed >= user.TrafficLimit {
		return false, "流量已用尽，请续费"
	}
	return true, ""
}
