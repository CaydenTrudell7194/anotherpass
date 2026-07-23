package handler

import (
	"net/http"
	"strconv"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
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
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if user.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名不能为空"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	user.Password = string(hash)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	if user.ExpireAt.IsZero() {
		user.ExpireAt = time.Now().AddDate(1, 0, 0)
	}
	if err := model.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败，用户名可能已存在"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func UpdateUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var user model.User
	if err := model.DB.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	var updated model.User
	if err := c.ShouldBindJSON(&updated); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	updated.ID = user.ID
	updated.Password = user.Password
	updated.CreatedAt = user.CreatedAt
	updated.UpdatedAt = time.Now()
	if updated.Password != "" && updated.Password != user.Password {
		hash, _ := bcrypt.GenerateFromPassword([]byte(updated.Password), bcrypt.DefaultCost)
		updated.Password = string(hash)
	}
	model.DB.Save(&updated)
	c.JSON(http.StatusOK, updated)
}

func DeleteUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	model.DB.Delete(&model.User{}, id)
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
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	group.ID = uint(id)
	model.DB.Save(&group)
	c.JSON(http.StatusOK, group)
}

func DeleteUserGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	model.DB.Delete(&model.UserGroup{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
