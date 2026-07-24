package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxCommerceListLimit = 100
	maxOrderNoteLength   = 500
)

var errOrderStateConflict = errors.New("order state conflict")

type servicePlanInput struct {
	Name         *string `json:"name"`
	Description  *string `json:"description"`
	PriceCents   *int64  `json:"price_cents"`
	DurationDays *int    `json:"duration_days"`
	RuleLimit    *int    `json:"rule_limit"`
	UserGroupID  *uint   `json:"user_group_id"`
	Enabled      *bool   `json:"enabled"`
}

func ListServicePlans(c *gin.Context) {
	var plans []model.ServicePlan
	if err := model.DB.Where("enabled = ?", true).Order("id asc").Limit(maxCommerceListLimit).Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询套餐失败"})
		return
	}
	c.JSON(http.StatusOK, plans)
}

func CreateOrder(c *gin.Context) {
	var input struct {
		PlanID   uint   `json:"plan_id"`
		UserNote string `json:"user_note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.PlanID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	input.UserNote = strings.TrimSpace(input.UserNote)
	if utf8.RuneCountInString(input.UserNote) > maxOrderNoteLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备注不能超过500个字符"})
		return
	}

	userID := c.GetUint("user_id")
	var order model.Order
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.User{}).Where("id = ?", userID).UpdateColumn("id", gorm.Expr("id"))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		var plan model.ServicePlan
		if err := tx.Where("id = ? AND enabled = ?", input.PlanID, true).First(&plan).Error; err != nil {
			return err
		}
		var pending int64
		if err := tx.Model(&model.Order{}).Where("user_id = ? AND status = ?", userID, model.OrderStatusPending).Count(&pending).Error; err != nil {
			return err
		}
		if pending >= 3 {
			return errOrderStateConflict
		}
		order = model.Order{
			UserID: userID, PlanID: plan.ID, PlanName: plan.Name, PlanPriceCents: plan.PriceCents,
			PlanDurationDays: plan.DurationDays, PlanRuleLimit: plan.RuleLimit,
			PlanUserGroupID: plan.UserGroupID, Status: model.OrderStatusPending, UserNote: input.UserNote,
		}
		return tx.Create(&order).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "套餐不存在或已停用"})
		return
	}
	if errors.Is(err, errOrderStateConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": "待处理订单最多为3个"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建订单失败"})
		return
	}
	notifyTelegram("新订单 #" + strconv.FormatUint(uint64(order.ID), 10) + "，套餐：" + order.PlanName)
	c.JSON(http.StatusCreated, order)
}

func ListOrders(c *gin.Context) {
	var orders []model.Order
	if err := model.DB.Where("user_id = ?", c.GetUint("user_id")).Order("id desc").Limit(commerceListLimit(c)).Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询订单失败"})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func AdminListServicePlans(c *gin.Context) {
	var plans []model.ServicePlan
	if err := model.DB.Order("id desc").Limit(commerceListLimit(c)).Find(&plans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询套餐失败"})
		return
	}
	c.JSON(http.StatusOK, plans)
}

func AdminCreateServicePlan(c *gin.Context) {
	var input servicePlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	plan := model.ServicePlan{Enabled: true}
	if err := applyServicePlanInput(&plan, input, true); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validatePlanUserGroup(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户组不存在"})
		return
	}
	if err := model.DB.Create(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建套餐失败"})
		return
	}
	c.JSON(http.StatusCreated, plan)
}

func AdminUpdateServicePlan(c *gin.Context) {
	id, ok := commerceID(c)
	if !ok {
		return
	}
	var plan model.ServicePlan
	if err := model.DB.First(&plan, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "套餐不存在"})
		return
	}
	var input servicePlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := applyServicePlanInput(&plan, input, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validatePlanUserGroup(&plan); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户组不存在"})
		return
	}
	if err := model.DB.Save(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新套餐失败"})
		return
	}
	c.JSON(http.StatusOK, plan)
}

func AdminDeleteServicePlan(c *gin.Context) {
	id, ok := commerceID(c)
	if !ok {
		return
	}
	var count int64
	if err := model.DB.Model(&model.Order{}).Where("plan_id = ?", id).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查订单失败"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "套餐已有订单引用，不能删除"})
		return
	}
	result := model.DB.Delete(&model.ServicePlan{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除套餐失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "套餐不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func AdminListOrders(c *gin.Context) {
	db := model.DB.Order("id desc").Limit(commerceListLimit(c))
	if status := c.Query("status"); status != "" {
		if !validOrderStatus(status) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "订单状态无效"})
			return
		}
		db = db.Where("status = ?", status)
	}
	var orders []model.Order
	if err := db.Find(&orders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询订单失败"})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func AdminApproveOrder(c *gin.Context) {
	reviewOrder(c, model.OrderStatusApproved)
}

func AdminRejectOrder(c *gin.Context) {
	reviewOrder(c, model.OrderStatusRejected)
}

func reviewOrder(c *gin.Context, targetStatus string) {
	id, ok := commerceID(c)
	if !ok {
		return
	}
	var input struct {
		AdminNote string `json:"admin_note"`
	}
	if err := c.ShouldBindJSON(&input); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	input.AdminNote = strings.TrimSpace(input.AdminNote)
	if utf8.RuneCountInString(input.AdminNote) > maxOrderNoteLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "备注不能超过500个字符"})
		return
	}

	var order model.Order
	stateChanged := false
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&order, id).Error; err != nil {
			return err
		}
		if order.Status == targetStatus {
			return nil
		}
		if order.Status != model.OrderStatusPending {
			return errOrderStateConflict
		}
		now := time.Now()
		result := tx.Model(&model.Order{}).Where("id = ? AND status = ?", order.ID, model.OrderStatusPending).Updates(map[string]interface{}{
			"status": targetStatus, "admin_note": input.AdminNote, "reviewed_by": c.GetUint("user_id"),
			"reviewed_at": now, "updated_at": now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errOrderStateConflict
		}
		stateChanged = true
		if targetStatus == model.OrderStatusApproved {
			var user model.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, order.UserID).Error; err != nil {
				return err
			}
			base := now
			if user.ExpireAt.After(base) {
				base = user.ExpireAt
			}
			updates := map[string]interface{}{
				"rule_limit": order.PlanRuleLimit, "expire_at": base.AddDate(0, 0, order.PlanDurationDays), "updated_at": now,
			}
			if order.PlanUserGroupID != nil {
				var count int64
				if err := tx.Model(&model.UserGroup{}).Where("id = ?", *order.PlanUserGroupID).Count(&count).Error; err != nil || count != 1 {
					return gorm.ErrRecordNotFound
				}
				updates["user_group_id"] = *order.PlanUserGroupID
			}
			if err := tx.Model(&user).Updates(updates).Error; err != nil {
				return err
			}
		}
		return tx.First(&order, id).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "订单或用户不存在"})
		return
	}
	if errors.Is(err, errOrderStateConflict) {
		c.JSON(http.StatusConflict, gin.H{"error": "订单已处理"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理订单失败"})
		return
	}
	if stateChanged {
		notifyTelegram("订单 #" + strconv.FormatUint(uint64(order.ID), 10) + " 已" + map[string]string{model.OrderStatusApproved: "批准", model.OrderStatusRejected: "拒绝"}[targetStatus])
	}
	c.JSON(http.StatusOK, order)
}

func applyServicePlanInput(plan *model.ServicePlan, input servicePlanInput, creating bool) error {
	if input.Name != nil {
		plan.Name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		plan.Description = strings.TrimSpace(*input.Description)
	}
	if input.PriceCents != nil {
		plan.PriceCents = *input.PriceCents
	}
	if input.DurationDays != nil {
		plan.DurationDays = *input.DurationDays
	}
	if input.RuleLimit != nil {
		plan.RuleLimit = *input.RuleLimit
	}
	if input.UserGroupID != nil {
		if *input.UserGroupID == 0 {
			plan.UserGroupID = nil
		} else {
			groupID := *input.UserGroupID
			plan.UserGroupID = &groupID
		}
	}
	if input.Enabled != nil {
		plan.Enabled = *input.Enabled
	}
	if creating && (input.Name == nil || input.PriceCents == nil || input.DurationDays == nil || input.RuleLimit == nil) {
		return errors.New("缺少套餐必填字段")
	}
	if plan.Name == "" || utf8.RuneCountInString(plan.Name) > 100 || utf8.RuneCountInString(plan.Description) > 500 {
		return errors.New("套餐名称或描述无效")
	}
	if plan.PriceCents < 0 || plan.DurationDays <= 0 || plan.DurationDays > 3650 || plan.RuleLimit < 0 || plan.RuleLimit > 1000000 {
		return errors.New("套餐价格、时长或规则限制无效")
	}
	return nil
}

func validatePlanUserGroup(plan *model.ServicePlan) error {
	if plan.UserGroupID == nil {
		return nil
	}
	var count int64
	if err := model.DB.Model(&model.UserGroup{}).Where("id = ?", *plan.UserGroupID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func commerceID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return 0, false
	}
	return uint(id), true
}

func commerceListLimit(c *gin.Context) int {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit < 1 {
		return 50
	}
	if limit > maxCommerceListLimit {
		return maxCommerceListLimit
	}
	return limit
}

func validOrderStatus(status string) bool {
	return status == model.OrderStatusPending || status == model.OrderStatusApproved || status == model.OrderStatusRejected
}
