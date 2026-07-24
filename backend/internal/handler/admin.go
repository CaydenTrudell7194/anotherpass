package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var errUserHasCommerceHistory = errors.New("user has commerce history")

func AdminDashboard(c *gin.Context) {
	var userCount, activeUserCount, ruleCount, groupCount, nodeCount, orderCount, approvedCount int64
	model.DB.Model(&model.User{}).Count(&userCount)
	model.DB.Model(&model.User{}).Where("status = ?", "active").Count(&activeUserCount)
	model.DB.Model(&model.ForwardRule{}).Count(&ruleCount)
	model.DB.Model(&model.DeviceGroup{}).Count(&groupCount)
	model.DB.Model(&model.Node{}).Where("status = ?", "online").Count(&nodeCount)
	model.DB.Model(&model.Order{}).Count(&orderCount)
	model.DB.Model(&model.Order{}).Where("status = ?", "approved").Count(&approvedCount)

	var totalTraffic, totalRechargeCents int64
	model.DB.Model(&model.ForwardRule{}).Select("COALESCE(SUM(traffic), 0)").Scan(&totalTraffic)
	model.DB.Model(&model.RechargeOrder{}).Where("status = ?", "paid").Select("COALESCE(SUM(amount_cents), 0)").Scan(&totalRechargeCents)

	c.JSON(http.StatusOK, gin.H{
		"user_count":           userCount,
		"active_user_count":    activeUserCount,
		"rule_count":           ruleCount,
		"device_group_count":   groupCount,
		"online_node_count":    nodeCount,
		"total_traffic":        totalTraffic,
		"total_orders":         orderCount,
		"approved_orders":      approvedCount,
		"total_recharge_cents": totalRechargeCents,
	})
}

func ListUsers(c *gin.Context) {
	var users []model.User
	model.DB.Omit("password").Order("id desc").Find(&users)
	c.JSON(http.StatusOK, users)
}

func CreateUser(c *gin.Context) {
	var input struct {
		Username     string    `json:"username"`
		Password     string    `json:"password"`
		DisplayName  string    `json:"display_name"`
		UserGroupID  uint      `json:"user_group_id"`
		Status       string    `json:"status"`
		TrafficLimit int64     `json:"traffic_limit"`
		RuleLimit    int       `json:"rule_limit"`
		ExpireAt     time.Time `json:"expire_at"`
		IsAdmin      bool      `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	input.Username = strings.TrimSpace(input.Username)
	if input.Username == "" || len(input.Username) > 64 || len(input.Password) < 8 || input.TrafficLimit < 0 || input.RuleLimit < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名、密码或限制参数无效"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码无效"})
		return
	}
	if input.UserGroupID == 0 {
		input.UserGroupID = 1
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Status != "active" && input.Status != "disabled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户状态无效"})
		return
	}
	if input.ExpireAt.IsZero() {
		input.ExpireAt = time.Now().AddDate(1, 0, 0)
	}
	now := time.Now()
	user := model.User{Username: input.Username, Password: string(hash), DisplayName: input.DisplayName,
		UserGroupID: input.UserGroupID, Status: input.Status, TrafficLimit: input.TrafficLimit,
		RuleLimit: input.RuleLimit, ExpireAt: input.ExpireAt, IsAdmin: input.IsAdmin, CreatedAt: now, UpdatedAt: now}
	if err := model.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败，用户名可能已存在"})
		return
	}
	user.Password = ""
	c.JSON(http.StatusOK, user)
}

func UpdateUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	var user model.User
	if err := model.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	var input struct {
		Username     *string    `json:"username"`
		Password     *string    `json:"password"`
		DisplayName  *string    `json:"display_name"`
		UserGroupID  *uint      `json:"user_group_id"`
		Status       *string    `json:"status"`
		TrafficLimit *int64     `json:"traffic_limit"`
		RuleLimit    *int       `json:"rule_limit"`
		IsAdmin      *bool      `json:"is_admin"`
		ExpireAt     *time.Time `json:"expire_at"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if input.Username != nil {
		user.Username = strings.TrimSpace(*input.Username)
		if user.Username == "" || len(user.Username) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户名无效"})
			return
		}
	}
	if input.Password != nil && *input.Password != "" {
		if len(*input.Password) < 8 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少需要8个字符"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "密码无效"})
			return
		}
		user.Password = string(hash)
	}
	if input.DisplayName != nil {
		user.DisplayName = *input.DisplayName
	}
	if input.UserGroupID != nil {
		user.UserGroupID = *input.UserGroupID
	}
	if input.Status != nil {
		if *input.Status != "active" && *input.Status != "disabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "用户状态无效"})
			return
		}
		user.Status = *input.Status
	}
	if input.TrafficLimit != nil {
		if *input.TrafficLimit < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "流量限制无效"})
			return
		}
		user.TrafficLimit = *input.TrafficLimit
	}
	if input.RuleLimit != nil {
		if *input.RuleLimit < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "规则限制无效"})
			return
		}
		user.RuleLimit = *input.RuleLimit
	}
	if input.IsAdmin != nil {
		user.IsAdmin = *input.IsAdmin
	}
	if input.ExpireAt != nil {
		user.ExpireAt = *input.ExpireAt
	}
	updates := map[string]interface{}{"updated_at": time.Now(), "token_version": gorm.Expr("token_version + 1")}
	if input.Username != nil {
		updates["username"] = user.Username
	}
	if input.Password != nil && *input.Password != "" {
		updates["password"] = user.Password
	}
	if input.DisplayName != nil {
		updates["display_name"] = user.DisplayName
	}
	if input.UserGroupID != nil {
		updates["user_group_id"] = user.UserGroupID
	}
	if input.Status != nil {
		updates["status"] = user.Status
	}
	if input.TrafficLimit != nil {
		updates["traffic_limit"] = user.TrafficLimit
	}
	if input.RuleLimit != nil {
		updates["rule_limit"] = user.RuleLimit
	}
	if input.IsAdmin != nil {
		updates["is_admin"] = user.IsAdmin
	}
	if input.ExpireAt != nil {
		updates["expire_at"] = user.ExpireAt
	}
	if err := model.DB.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	if err := model.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	user.Password = ""
	c.JSON(http.StatusOK, user)
}

func DeleteUser(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	if uint(id) == c.GetUint("user_id") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不能删除当前登录用户"})
		return
	}
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		var references int64
		if err := tx.Model(&model.Order{}).Where("user_id = ? OR reviewed_by = ?", id, id).Count(&references).Error; err != nil {
			return err
		}
		if references == 0 {
			if err := tx.Model(&model.BalanceLedger{}).Where("user_id = ? OR actor_user_id = ?", id, id).Count(&references).Error; err != nil {
				return err
			}
		}
		if references == 0 {
			if err := tx.Model(&model.RechargeOrder{}).Where("user_id = ?", id).Count(&references).Error; err != nil {
				return err
			}
		}
		if references > 0 {
			return errUserHasCommerceHistory
		}
		if err := tx.Where("user_id = ?", id).Delete(&model.ForwardRule{}).Error; err != nil {
			return err
		}
		result := tx.Delete(&model.User{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errUserHasCommerceHistory) {
			c.JSON(http.StatusConflict, gin.H{"error": "用户已有订单、充值记录或余额流水，不能删除"})
			return
		}
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func ListUserGroups(c *gin.Context) {
	var groups []model.UserGroup
	model.DB.Order("id asc").Find(&groups)
	c.JSON(http.StatusOK, groups)
}

func CreateUserGroup(c *gin.Context) {
	var group model.UserGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if err := model.DB.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, group)
}

func UpdateUserGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var group model.UserGroup
	if err := model.DB.First(&group, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户组不存在"})
		return
	}
	var input struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if input.Name != nil {
		group.Name = *input.Name
	}
	if input.Description != nil {
		group.Description = *input.Description
	}
	model.DB.Save(&group)
	c.JSON(http.StatusOK, group)
}

func DeleteUserGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	if LoadSiteSettings().RegisterUserGroupID == uint(id) {
		c.JSON(http.StatusConflict, gin.H{"error": "该用户组是公开注册默认组，不能删除"})
		return
	}
	var users int64
	if err := model.DB.Model(&model.User{}).Where("user_group_id = ?", id).Count(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查依赖失败"})
		return
	}
	if users > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该用户组仍有用户，不能删除"})
		return
	}
	var references int64
	if err := model.DB.Model(&model.ServicePlan{}).Where("user_group_id = ?", id).Count(&references).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查依赖失败"})
		return
	}
	if references == 0 {
		if err := model.DB.Model(&model.Order{}).Where("plan_user_group_id = ? AND status = ?", id, model.OrderStatusPending).Count(&references).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "检查依赖失败"})
			return
		}
	}
	if references > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "该用户组仍被套餐或待处理订单引用，不能删除"})
		return
	}
	result := model.DB.Delete(&model.UserGroup{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户组不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
