package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func AdminDashboard(c *gin.Context) {
	var userCount, ruleCount, groupCount, nodeCount int64
	model.DB.Model(&model.User{}).Count(&userCount)
	model.DB.Model(&model.ForwardRule{}).Count(&ruleCount)
	model.DB.Model(&model.DeviceGroup{}).Count(&groupCount)
	model.DB.Model(&model.Node{}).Where("status = ?", "online").Count(&nodeCount)

	var totalTraffic int64
	model.DB.Model(&model.ForwardRule{}).Select("COALESCE(SUM(traffic), 0)").Scan(&totalTraffic)

	c.JSON(http.StatusOK, gin.H{
		"user_count":         userCount,
		"rule_count":         ruleCount,
		"device_group_count": groupCount,
		"online_node_count":  nodeCount,
		"total_traffic":      totalTraffic,
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
	user.UpdatedAt = time.Now()
	user.TokenVersion++
	if err := model.DB.Save(&user).Error; err != nil {
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
	model.DB.Delete(&model.UserGroup{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
