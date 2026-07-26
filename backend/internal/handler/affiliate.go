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

var errRedeemInvalid = errors.New("redeem code invalid or unavailable")

func GetAffiliateInfo(c *gin.Context) {
	userID := c.GetUint("user_id")
	var aff model.Affiliate
	if err := model.DB.Where("user_id = ?", userID).First(&aff).Error; err != nil {
		code, codeErr := uniqueAffCode(8)
		if codeErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建推广信息失败"})
			return
		}
		// 使用站点默认佣金比例
		settings := LoadSiteSettings()
		aff = model.Affiliate{UserID: userID, Code: code, CommissionRate: settings.DefaultCommissionRate}
		if err := model.DB.Create(&aff).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建推广信息失败"})
			return
		}
	}
	var referralCount int64
	if err := model.DB.Model(&model.AffLog{}).Where("referrer_id = ?", userID).Count(&referralCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询推广统计失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": aff.Code, "commission_rate": aff.CommissionRate, "total_earned_cents": aff.TotalEarnedCents, "referral_count": referralCount, "invite_link": ""})
}

func AdminListAffiliates(c *gin.Context) {
	var affs []model.Affiliate
	model.DB.Order("id desc").Limit(100).Find(&affs)
	if len(affs) == 0 {
		c.JSON(http.StatusOK, []interface{}{})
		return
	}
	userIDs := make([]uint, len(affs))
	for i, a := range affs {
		userIDs[i] = a.UserID
	}
	var users []model.User
	model.DB.Where("id IN ?", userIDs).Find(&users)
	userMap := make(map[uint]model.User, len(users))
	for _, u := range users {
		userMap[u.ID] = u
	}
	type AffiliateWithUser struct {
		model.Affiliate
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	result := make([]AffiliateWithUser, len(affs))
	for i, a := range affs {
		u := userMap[a.UserID]
		result[i] = AffiliateWithUser{Affiliate: a, Username: u.Username, DisplayName: u.DisplayName}
	}
	c.JSON(http.StatusOK, result)
}

func AdminUpdateAffiliate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	var input struct {
		CommissionRate          *float64 `json:"commission_rate"`
		CommissionRatePercent   *float64 `json:"commission_rate_percent"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	updates := map[string]interface{}{}
	if input.CommissionRate != nil {
		if *input.CommissionRate < 0 || *input.CommissionRate > 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "佣金比例必须在 0-1 之间"})
			return
		}
		updates["commission_rate"] = *input.CommissionRate
	} else if input.CommissionRatePercent != nil {
		if *input.CommissionRatePercent < 0 || *input.CommissionRatePercent > 100 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "佣金比例必须在 0-100 之间"})
			return
		}
		updates["commission_rate"] = *input.CommissionRatePercent / 100.0
	}
	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有需要更新的字段"})
		return
	}
	result := model.DB.Model(&model.Affiliate{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil || result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "推广记录不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
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
			return errRedeemInvalid
		}
		if code.AmountCents < 100 || code.AmountCents > maxBalanceAmount {
			return errRedeemInvalid
		}
		if code.ExpiresAt != nil && code.ExpiresAt.Before(time.Now()) {
			return errRedeemInvalid
		}
		if code.MaxUses > 0 {
			if code.UsedCount >= code.MaxUses {
				return errRedeemInvalid
			}
			result := tx.Model(&model.RedeemCode{}).Where("id = ? AND used_count = ?", code.ID, code.UsedCount).Update("used_count", code.UsedCount+1)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return errRedeemInvalid
			}
		}
		var existing int64
		if err := tx.Model(&model.BalanceLedger{}).Where("operation_key = ?", "redeem:"+input.Code+":"+strconv.FormatUint(uint64(userID), 10)).Count(&existing).Error; err != nil {
			return err
		}
		if existing > 0 {
			return errRedeemInvalid
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
		if errors.Is(err, errRedeemInvalid) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "兑换码无效或不可用"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "兑换失败"})
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
	if err := c.ShouldBindJSON(&input); err != nil || input.Count < 1 || input.Count > 1000 || input.AmountCents < 100 || input.AmountCents > maxBalanceAmount || input.MaxUses < 1 || input.MaxUses > 100000 {
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

func uniqueAffCode(length int) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		code := randomAffCode(length)
		var existing int64
		if err := model.DB.Model(&model.Affiliate{}).Where("code = ?", code).Count(&existing).Error; err != nil {
			return "", err
		}
		if existing == 0 {
			return code, nil
		}
	}
	return "", errors.New("could not allocate unique affiliate code")
}
