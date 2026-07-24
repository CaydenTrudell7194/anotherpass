package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetAffiliateInfo(c *gin.Context) {
	userID := c.GetUint("user_id")
	var aff model.Affiliate
	if err := model.DB.Where("user_id = ?", userID).First(&aff).Error; err != nil {
		code := randomAffCode(8)
		aff = model.Affiliate{UserID: userID, Code: code}
		if err := model.DB.Create(&aff).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建推广信息失败"})
			return
		}
	}
	var count int64
	model.DB.Model(&model.User{}).Where("id IN (SELECT user_id FROM affiliates WHERE code != '' AND ? LIKE CONCAT('%', code, '%'))", "").Count(&count)
	c.JSON(http.StatusOK, gin.H{"code": aff.Code, "commission_rate": aff.CommissionRate, "total_earned_cents": aff.TotalEarnedCents, "referral_count": 0})
}

func AdminListAffiliates(c *gin.Context) {
	var affs []model.Affiliate
	model.DB.Order("id desc").Limit(100).Find(&affs)
	c.JSON(http.StatusOK, affs)
}

func RedeemCodeHandler(c *gin.Context) {
	var input struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	input.Code = strings.TrimSpace(strings.ToUpper(input.Code))
	userID := c.GetUint("user_id")

	var code model.RedeemCode
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("code = ?", input.Code).First(&code).Error; err != nil {
			return err
		}
		if code.ExpiresAt != nil && code.ExpiresAt.Before(time.Now()) {
			return errors.New("兑换码已过期")
		}
		if code.MaxUses > 0 && code.UsedCount >= code.MaxUses {
			return errors.New("兑换码已达使用上限")
		}
		var existing int64
		if err := tx.Model(&model.BalanceLedger{}).Where("operation_key = ?", "redeem:"+input.Code+":"+strconv.FormatUint(uint64(userID), 10)).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return errors.New("该兑换码已使用")
		}
		if err := tx.Model(&code).Update("used_count", code.UsedCount+1).Error; err != nil {
			return err
		}
		result := tx.Model(&model.User{}).Where("id = ? AND balance_cents <= ?", userID, int64(^uint64(0)>>1)-code.AmountCents).UpdateColumn("balance_cents", gorm.Expr("balance_cents + ?", code.AmountCents))
		if result.Error != nil || result.RowsAffected != 1 {
			return errors.New("余额更新失败")
		}
		var user model.User
		if err := tx.Select("balance_cents").First(&user, userID).Error; err != nil {
			return err
		}
		ledger := model.BalanceLedger{
			UserID: userID, DeltaCents: code.AmountCents, BalanceAfterCents: user.BalanceCents,
			Kind: model.LedgerKindRecharge, OperationKey: "redeem:" + input.Code + ":" + strconv.FormatUint(uint64(userID), 10),
			RequestHash: hex.EncodeToString([]byte(input.Code)), Note: "兑换码充值", CreatedAt: time.Now(),
		}
		return tx.Create(&ledger).Error
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "兑换码不存在"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "兑换成功", "amount_cents": code.AmountCents})
}

func AdminCreateRedeemCodes(c *gin.Context) {
	var input struct {
		Count       int   `json:"count"`
		AmountCents int64 `json:"amount_cents"`
		MaxUses     int   `json:"max_uses"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Count < 1 || input.Count > 1000 || input.AmountCents < 100 || input.MaxUses < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	codes := make([]model.RedeemCode, input.Count)
	for i := 0; i < input.Count; i++ {
		b := make([]byte, 12)
		rand.Read(b)
		codes[i] = model.RedeemCode{Code: strings.ToUpper(hex.EncodeToString(b)), AmountCents: input.AmountCents, MaxUses: input.MaxUses}
	}
	if err := model.DB.Create(&codes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"count": input.Count, "amount_cents": input.AmountCents})
}

func AdminListRedeemCodes(c *gin.Context) {
	var codes []model.RedeemCode
	model.DB.Order("id desc").Limit(100).Find(&codes)
	c.JSON(http.StatusOK, codes)
}

func AdminDeleteRedeemCode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	result := model.DB.Delete(&model.RedeemCode{}, id)
	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "兑换码不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func randomAffCode(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}
