package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
)

func ListNodeStatus(c *gin.Context) {
	var nodes []model.Node
	retention := time.Duration(LoadSiteSettings().OfflineNodeRetentionHours) * time.Hour
	cutoff := time.Now().Add(-retention)
	model.DB.Where("status = ? OR last_heartbeat >= ? OR (last_heartbeat = ? AND created_at >= ?)", "online", cutoff, time.Time{}, cutoff).Order("id desc").Find(&nodes)
	c.JSON(http.StatusOK, nodes)
}

func ListMyNodeStatus(c *gin.Context) {
	var user model.User
	if err := model.DB.First(&user, c.GetUint("user_id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	var groups []model.DeviceGroup
	if err := model.DB.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	groupIDs := make([]uint, 0)
	for _, group := range groups {
		if authorizeDeviceGroup(user.ID, group.ID, false) == nil {
			groupIDs = append(groupIDs, group.ID)
		}
	}
	var nodes []model.Node
	if len(groupIDs) > 0 {
		retention := time.Duration(LoadSiteSettings().OfflineNodeRetentionHours) * time.Hour
		cutoff := time.Now().Add(-retention)
		model.DB.Where("device_group_id IN ? AND (status = ? OR last_heartbeat >= ? OR (last_heartbeat = ? AND created_at >= ?))", groupIDs, "online", cutoff, time.Time{}, cutoff).Order("id desc").Find(&nodes)
	}
	c.JSON(http.StatusOK, nodes)
}

func NodeHeartbeat(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
		IP    string `json:"ip"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	var node model.Node
	if err := model.DB.Where("token = ?", req.Token).First(&node).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "节点认证失败"})
		return
	}
	now := time.Now()
	node.LastHeartbeat = now
	node.Status = "online"
	node.IP = req.IP
	model.DB.Save(&node)

	// Update device group online count
	var count int64
	model.DB.Model(&model.Node{}).Where("device_group_id = ? AND status = ?", node.DeviceGroupID, "online").Count(&count)
	model.DB.Model(&model.DeviceGroup{}).Where("id = ?", node.DeviceGroupID).Update("online_devices", count)

	c.JSON(http.StatusOK, gin.H{"status": "ok", "server_time": now.Unix()})
}

func RegisterNode(c *gin.Context) {
	var req struct {
		DeviceGroupID uint   `json:"device_group_id"`
		Name          string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成节点令牌失败"})
		return
	}
	token := hex.EncodeToString(b)
	node := model.Node{
		DeviceGroupID: req.DeviceGroupID,
		Name:          req.Name,
		Token:         token,
		Status:        "offline",
	}
	if err := model.DB.Create(&node).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"node_id": node.ID, "token": token})
}

func CheckOfflineNodes() {
	var nodes []model.Node
	model.DB.Where("status = ?", "online").Find(&nodes)
	now := time.Now()
	offlineAfter := time.Duration(LoadSiteSettings().OfflineNodeSeconds) * time.Second
	for _, n := range nodes {
		if now.Sub(n.LastHeartbeat) > offlineAfter {
			model.DB.Model(&n).Update("status", "offline")
			var count int64
			model.DB.Model(&model.Node{}).Where("device_group_id = ? AND status = ?", n.DeviceGroupID, "online").Count(&count)
			model.DB.Model(&model.DeviceGroup{}).Where("id = ?", n.DeviceGroupID).Update("online_devices", count)
		}
	}
}

func DeleteNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	result := model.DB.Delete(&model.Node{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "节点不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func GetNodeRules(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	var node model.Node
	if err := model.DB.Where("token = ?", req.Token).First(&node).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}
	var rules []model.ForwardRule
	model.DB.Where("device_group_id = ? AND enabled = ? AND status = ?", node.DeviceGroupID, true, "active").Find(&rules)
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}
