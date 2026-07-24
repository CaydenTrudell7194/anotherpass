package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	maxBalanceAmount = int64(1_000_000_000_000)
	maxOperationKey  = 128
	maxLedgerNote    = 500
)

var (
	errInsufficientBalance = errors.New("insufficient balance")
	errIdempotencyConflict = errors.New("idempotency key conflict")
	errNoBalanceChange     = errors.New("balance unchanged")
	errInvalidPlanPrice    = errors.New("invalid plan price")
)

func CreateBalanceOrder(c *gin.Context) {
	key, ok := operationKey(c)
	if !ok {
		return
	}
	var input struct {
		PlanID uint `json:"plan_id"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || input.PlanID == 0 || decoder.Decode(&struct{}{}) != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	userID := c.GetUint("user_id")
	requestHash := hashRequest(fmt.Sprintf("balance-order:%d:%d", userID, input.PlanID))
	var order model.Order
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var existing model.Order
		err := tx.Where("user_id = ? AND idempotency_key = ?", userID, key).First(&existing).Error
		if err == nil {
			if existing.UserID != userID || existing.PlanID != input.PlanID || existing.PaymentMethod != model.PaymentMethodBalance {
				return errIdempotencyConflict
			}
			order = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var plan model.ServicePlan
		if err := tx.Where("id = ? AND enabled = ?", input.PlanID, true).First(&plan).Error; err != nil {
			return err
		}
		if plan.PriceCents <= 0 || plan.PriceCents > maxBalanceAmount {
			return errInvalidPlanPrice
		}
		now := time.Now()
		result := tx.Model(&model.User{}).Where("id = ? AND balance_cents >= ?", userID, plan.PriceCents).
			UpdateColumn("balance_cents", gorm.Expr("balance_cents - ?", plan.PriceCents))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errInsufficientBalance
		}
		order = model.Order{
			UserID: userID, PlanID: plan.ID, PlanName: plan.Name, PlanPriceCents: plan.PriceCents,
			PlanDurationDays: plan.DurationDays, PlanRuleLimit: plan.RuleLimit, PlanUserGroupID: plan.UserGroupID,
			Status: model.OrderStatusApproved, PaymentMethod: model.PaymentMethodBalance, PaidCents: plan.PriceCents,
			IdempotencyKey: key, ReviewedAt: &now, FulfilledAt: &now,
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		if err := fulfillOrder(tx, &order, now); err != nil {
			return err
		}
		var user model.User
		if err := tx.Select("balance_cents").First(&user, userID).Error; err != nil {
			return err
		}
		ledger := model.BalanceLedger{
			UserID: userID, DeltaCents: -plan.PriceCents, BalanceAfterCents: user.BalanceCents,
			Kind: model.LedgerKindOrderDebit, OrderID: &order.ID, OperationKey: hashRequest(fmt.Sprintf("balance-op:%d:%s", userID, key)),
			RequestHash: requestHash, Note: "Balance payment for order", CreatedAt: now,
		}
		return tx.Create(&ledger).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "套餐不存在或已停用"})
		return
	}
	if errors.Is(err, errInsufficientBalance) {
		c.JSON(http.StatusConflict, gin.H{"error": "余额不足"})
		return
	}
	if errors.Is(err, errInvalidPlanPrice) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "零元或价格异常套餐不能使用余额购买"})
		return
	}
	if errors.Is(err, errIdempotencyConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": "Idempotency-Key已用于其他请求"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "余额支付失败"})
		return
	}
	c.JSON(http.StatusCreated, order)
}

func AdminAdjustBalance(c *gin.Context) {
	key, ok := operationKey(c)
	if !ok {
		return
	}
	userID, ok := commerceID(c)
	if !ok {
		return
	}
	var input struct {
		TargetBalanceCents *int64 `json:"target_balance_cents"`
		Reason             string `json:"reason"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || input.TargetBalanceCents == nil || decoder.Decode(&struct{}{}) != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	input.Reason = strings.TrimSpace(input.Reason)
	targetBalance := *input.TargetBalanceCents
	if targetBalance < 0 || targetBalance > maxBalanceAmount || input.Reason == "" || utf8.RuneCountInString(input.Reason) > maxLedgerNote {
		c.JSON(http.StatusBadRequest, gin.H{"error": "目标余额或原因无效"})
		return
	}
	requestHash := hashRequest(fmt.Sprintf("adjustment:%d:%d:%s", userID, targetBalance, input.Reason))
	actorID := c.GetUint("user_id")
	ledgerKey := hashRequest(fmt.Sprintf("admin-op:%d:%s", actorID, key))
	var ledger model.BalanceLedger
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("operation_key = ?", ledgerKey).First(&ledger).Error
		if err == nil {
			if ledger.RequestHash != requestHash || ledger.Kind != model.LedgerKindAdminAdjustment || ledger.UserID != userID {
				return errIdempotencyConflict
			}
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var user model.User
		if err := tx.Select("id", "balance_cents").First(&user, userID).Error; err != nil {
			return err
		}
		if user.BalanceCents == targetBalance {
			return errNoBalanceChange
		}
		delta := targetBalance - user.BalanceCents
		result := tx.Model(&model.User{}).Where("id = ? AND balance_cents = ?", userID, user.BalanceCents).
			UpdateColumn("balance_cents", targetBalance)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("concurrent balance update")
		}
		ledger = model.BalanceLedger{
			UserID: userID, DeltaCents: delta, BalanceAfterCents: targetBalance,
			Kind: model.LedgerKindAdminAdjustment, ActorUserID: &actorID, OperationKey: ledgerKey,
			RequestHash: requestHash, Note: input.Reason, CreatedAt: time.Now(),
		}
		return tx.Create(&ledger).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	if errors.Is(err, errNoBalanceChange) {
		c.JSON(http.StatusConflict, gin.H{"error": "目标余额与当前余额相同"})
		return
	}
	if errors.Is(err, errIdempotencyConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": "Idempotency-Key已用于其他请求"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "余额调整失败"})
		return
	}
	c.JSON(http.StatusOK, ledger)
}

func ListBalanceLedger(c *gin.Context) {
	listBalanceLedger(c, c.GetUint("user_id"))
}

func AdminListBalanceLedger(c *gin.Context) {
	userID, ok := commerceID(c)
	if !ok {
		return
	}
	var count int64
	if err := model.DB.Model(&model.User{}).Where("id = ?", userID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}
	if count == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	listBalanceLedger(c, userID)
}

func listBalanceLedger(c *gin.Context, userID uint) {
	var entries []model.BalanceLedger
	if err := model.DB.Where("user_id = ?", userID).Order("id desc").Limit(commerceListLimit(c)).Find(&entries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询余额流水失败"})
		return
	}
	c.JSON(http.StatusOK, entries)
}

func operationKey(c *gin.Context) (string, bool) {
	key := c.GetHeader("Idempotency-Key")
	if len(key) == 0 || len(key) > maxOperationKey {
		c.JSON(http.StatusBadRequest, gin.H{"error": "需要有效的Idempotency-Key"})
		return "", false
	}
	for _, char := range []byte(key) {
		if char < 0x21 || char > 0x7e {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key必须为可打印ASCII字符"})
			return "", false
		}
	}
	return key, true
}

func hashRequest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
