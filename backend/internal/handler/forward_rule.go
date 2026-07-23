package handler

import (
	"net/http"
	"strconv"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
)

func ListForwardRules(c *gin.Context) {
	userID := c.GetUint("user_id")
	isAdmin := c.GetBool("is_admin")
	var rules []model.ForwardRule
	q := model.DB.Order("id desc")
	if !isAdmin {
		q = q.Where("user_id = ?", userID)
	}
	q.Find(&rules)
	c.JSON(http.StatusOK, rules)
}

func CreateForwardRule(c *gin.Context) {
	userID := c.GetUint("user_id")
	var rule model.ForwardRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	rule.UserID = userID
	rule.Status = "pending"
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()
	if err := model.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, rule)
}

func UpdateForwardRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint("user_id")
	isAdmin := c.GetBool("is_admin")

	var rule model.ForwardRule
	q := model.DB.Where("id = ?", id)
	if !isAdmin {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.First(&rule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	var input struct {
		Name          *string  `json:"name"`
		DeviceGroupID *uint    `json:"device_group_id"`
		ListenPort    *int     `json:"listen_port"`
		TargetAddr    *string  `json:"target_addr"`
		TargetPort    *int     `json:"target_port"`
		Protocol      *string  `json:"protocol"`
		Rate          *float64 `json:"rate"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if input.Name != nil {
		rule.Name = *input.Name
	}
	if input.DeviceGroupID != nil {
		rule.DeviceGroupID = *input.DeviceGroupID
	}
	if input.ListenPort != nil {
		rule.ListenPort = *input.ListenPort
	}
	if input.TargetAddr != nil {
		rule.TargetAddr = *input.TargetAddr
	}
	if input.TargetPort != nil {
		rule.TargetPort = *input.TargetPort
	}
	if input.Protocol != nil {
		rule.Protocol = *input.Protocol
	}
	if input.Rate != nil {
		rule.Rate = *input.Rate
	}
	rule.UpdatedAt = time.Now()
	model.DB.Save(&rule)
	c.JSON(http.StatusOK, rule)
}

func DeleteForwardRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint("user_id")
	isAdmin := c.GetBool("is_admin")

	q := model.DB.Where("id = ?", id)
	if !isAdmin {
		q = q.Where("user_id = ?", userID)
	}
	q.Delete(&model.ForwardRule{})
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func ToggleForwardRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var rule model.ForwardRule
	if err := model.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	rule.Enabled = !rule.Enabled
	model.DB.Save(&rule)
	c.JSON(http.StatusOK, rule)
}

func BatchCreateForwardRules(c *gin.Context) {
	userID := c.GetUint("user_id")
	var rules []model.ForwardRule
	if err := c.ShouldBindJSON(&rules); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	now := time.Now()
	for i := range rules {
		rules[i].UserID = userID
		rules[i].Status = "pending"
		rules[i].CreatedAt = now
		rules[i].UpdatedAt = now
	}
	if len(rules) > 0 {
		model.DB.Create(&rules)
	}
	c.JSON(http.StatusOK, rules)
}
