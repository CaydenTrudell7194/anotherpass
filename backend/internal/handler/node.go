package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NodeListItem struct {
	model.Node
	DeviceGroupName string                `json:"device_group_name"`
	DeviceGroupType model.DeviceGroupType `json:"device_group_type"`
}

func ListNodeStatus(c *gin.Context) {
	var nodes []NodeListItem
	result := model.DB.Table("nodes").Select("nodes.*, device_groups.name AS device_group_name, device_groups.type AS device_group_type").Joins("LEFT JOIN device_groups ON device_groups.id = nodes.device_group_id").Order("nodes.id desc").Scan(&nodes)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询节点失败"})
		return
	}
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
	updateNodeHeartbeat(&node, req.IP)

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
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 128 || req.DeviceGroupID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "节点名称或设备组无效"})
		return
	}
	if err := ensureDirectEntryGroup(req.DeviceGroupID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "设备组不存在或不是入口直出类型"})
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
	c.JSON(http.StatusOK, gin.H{"node_id": node.ID})
}

func GetNodeSetup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	var node model.Node
	if err := model.DB.First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "节点不存在"})
		return
	}
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成注册码失败"})
		return
	}
	code := hex.EncodeToString(b)
	hash := sha256.Sum256([]byte(code))
	if err := model.DB.Model(&node).Updates(map[string]interface{}{
		"enroll_hash":    hex.EncodeToString(hash[:]),
		"enroll_expires": time.Now().Add(10 * time.Minute),
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成注册码失败"})
		return
	}
	c.Header("Cache-Control", "no-store, private")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, gin.H{"node_id": node.ID, "device_group_id": node.DeviceGroupID, "enroll_code": code, "expires_in": 600})
}

func EnrollNode(c *gin.Context) {
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	hash := sha256.Sum256([]byte(strings.TrimSpace(req.Code)))
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成节点令牌失败"})
		return
	}
	token := hex.EncodeToString(b)
	var node model.Node
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("enroll_hash = ? AND enroll_expires > ?", hex.EncodeToString(hash[:]), time.Now()).First(&node).Error; err != nil {
			return err
		}
		result := tx.Model(&model.Node{}).Where("id = ? AND enroll_hash = ?", node.ID, hex.EncodeToString(hash[:])).Updates(map[string]interface{}{"enroll_hash": "", "enroll_expires": time.Time{}, "token": token})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "注册码无效或已过期"})
		return
	}
	revokeNodeSession(node.ID)
	c.Header("Cache-Control", "no-store, private")
	c.JSON(http.StatusOK, gin.H{"token": token})
}

func RotateNodeToken(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成节点令牌失败"})
		return
	}
	token := hex.EncodeToString(b)
	result := model.DB.Model(&model.Node{}).Where("id = ?", id).Update("token", token)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "令牌更新失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "节点不存在"})
		return
	}
	revokeNodeSession(uint(id))
	c.JSON(http.StatusOK, gin.H{"message": "节点令牌已轮换"})
}

func CheckOfflineNodes() {
	var nodes []model.Node
	model.DB.Where("status = ?", "online").Find(&nodes)
	now := time.Now()
	offlineAfter := time.Duration(LoadSiteSettings().OfflineNodeSeconds) * time.Second
	for _, n := range nodes {
		if now.Sub(n.LastHeartbeat) > offlineAfter {
			cutoff := now.Add(-offlineAfter)
			markNodeOfflineBefore(n.ID, n.DeviceGroupID, &cutoff)
		}
	}
}

func DeleteNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	var node model.Node
	if err := model.DB.First(&node, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "节点不存在"})
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
	revokeNodeSession(node.ID)
	recountOnlineNodes(node.DeviceGroupID)
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
	if err := ensureDirectEntryGroup(node.DeviceGroupID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "设备组不支持入口直出"})
		return
	}
	rules, err := loadRulesForGroup(node.DeviceGroupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询规则失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rules": rules})
}

func ensureDirectEntryGroup(groupID uint) error {
	var group model.DeviceGroup
	if err := model.DB.First(&group, groupID).Error; err != nil {
		return err
	}
	if group.Type != model.DeviceGroupEntryForceDirect && group.Type != model.DeviceGroupEntryOptionalDirect {
		return gorm.ErrInvalidData
	}
	return nil
}

func loadRulesForGroup(groupID uint) ([]model.ForwardRule, error) {
	var rules []model.ForwardRule
	err := model.DB.Where("device_group_id = ? AND enabled = ? AND status = ?", groupID, true, "active").Order("id asc").Find(&rules).Error
	return rules, err
}

func updateNodeHeartbeat(node *model.Node, ip string) {
	now := time.Now()
	values := map[string]interface{}{"last_heartbeat": now, "status": "online", "ip": strings.TrimSpace(ip)}
	transition := model.DB.Model(&model.Node{}).Where("id = ? AND status <> ?", node.ID, "online").Updates(values)
	if transition.RowsAffected == 0 {
		model.DB.Model(&model.Node{}).Where("id = ?", node.ID).Updates(values)
	}
	node.LastHeartbeat = now
	node.Status = "online"
	node.IP = strings.TrimSpace(ip)
	recountOnlineNodes(node.DeviceGroupID)
	if transition.Error == nil && transition.RowsAffected == 1 {
		notifyTelegram("节点上线：" + node.Name)
	}
}

func markNodeOffline(nodeID, groupID uint) {
	markNodeOfflineBefore(nodeID, groupID, nil)
}

func markNodeOfflineBefore(nodeID, groupID uint, cutoff *time.Time) {
	query := model.DB.Model(&model.Node{}).Where("id = ? AND status = ?", nodeID, "online")
	if cutoff != nil {
		query = query.Where("last_heartbeat < ?", *cutoff)
	}
	result := query.Update("status", "offline")
	recountOnlineNodes(groupID)
	if result.Error == nil && result.RowsAffected == 1 {
		var node model.Node
		if model.DB.Select("name").First(&node, nodeID).Error == nil {
			notifyTelegram("节点离线：" + node.Name)
		}
	}
}

func recountOnlineNodes(groupID uint) {
	var count int64
	model.DB.Model(&model.Node{}).Where("device_group_id = ? AND status = ?", groupID, "online").Count(&count)
	model.DB.Model(&model.DeviceGroup{}).Where("id = ?", groupID).Update("online_devices", count)
}
