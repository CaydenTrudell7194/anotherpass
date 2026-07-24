package handler

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxBatchRules = 100

var errListenPortConflict = errors.New("listen port conflict")

type forwardRuleInput struct {
	Name          string  `json:"name"`
	DeviceGroupID uint    `json:"device_group_id"`
	ListenPort    int     `json:"listen_port"`
	TargetAddr    string  `json:"target_addr"`
	TargetPort    int     `json:"target_port"`
	Dest          string  `json:"dest"`
	Protocol      string  `json:"protocol"`
	Rate          float64 `json:"rate"`
	Enabled       *bool   `json:"enabled"`
}

type forwardRuleUpdateInput struct {
	Name          *string  `json:"name"`
	DeviceGroupID *uint    `json:"device_group_id"`
	ListenPort    *int     `json:"listen_port"`
	TargetAddr    *string  `json:"target_addr"`
	TargetPort    *int     `json:"target_port"`
	Dest          *string  `json:"dest"`
	Protocol      *string  `json:"protocol"`
	Rate          *float64 `json:"rate"`
	Enabled       *bool    `json:"enabled"`
}

func parseID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return 0, false
	}
	return uint(id), true
}

func parseDest(dest string) (addr string, port int) {
	lines := strings.Split(strings.TrimSpace(dest), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			a := strings.TrimSpace(parts[0])
			p, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err == nil && p >= 1 && p <= 65535 && a != "" && len(a) <= 256 {
				return a, p
			}
		}
	}
	return "", 0
}

func validateForwardRule(input *forwardRuleInput) string {
	input.Name = strings.TrimSpace(input.Name)
	input.TargetAddr = strings.TrimSpace(input.TargetAddr)
	input.Protocol = strings.ToLower(strings.TrimSpace(input.Protocol))
	if input.Protocol == "" {
		input.Protocol = "tcp"
	}
	if input.Dest != "" {
		addr, port := parseDest(input.Dest)
		if addr != "" && port > 0 {
			input.TargetAddr = addr
			input.TargetPort = port
		}
	}
	if input.Name == "" || len(input.Name) > 128 {
		return "规则名称不能为空且不能超过128个字符"
	}
	if input.DeviceGroupID == 0 {
		return "设备组无效"
	}
	if input.ListenPort < 1 || input.ListenPort > 65535 || input.TargetPort < 1 || input.TargetPort > 65535 {
		return "端口必须在1到65535之间"
	}
	if input.TargetAddr == "" || len(input.TargetAddr) > 256 {
		return "目标地址不能为空且不能超过256个字符"
	}
	if input.Protocol != "tcp" {
		return "当前仅支持TCP协议"
	}
	if math.IsNaN(input.Rate) || math.IsInf(input.Rate, 0) || input.Rate < 0 {
		return "倍率无效"
	}
	return ""
}

func authorizeDeviceGroup(userID, groupID uint, isAdmin bool) error {
	var group model.DeviceGroup
	if err := model.DB.First(&group, groupID).Error; err != nil {
		return err
	}
	if isAdmin {
		return nil
	}
	if group.Type != model.DeviceGroupEntryForceDirect && group.Type != model.DeviceGroupEntryOptionalDirect {
		return gorm.ErrRecordNotFound
	}
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		return err
	}
	if group.UserGroupIDs == "" {
		return nil
	}
	wanted := strconv.FormatUint(uint64(user.UserGroupID), 10)
	for _, id := range strings.Split(group.UserGroupIDs, ",") {
		if strings.TrimSpace(id) == wanted {
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func checkRuleLimit(db *gorm.DB, userID uint, additional int) error {
	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		return err
	}
	if user.RuleLimit <= 0 {
		return nil
	}
	var count int64
	if err := db.Model(&model.ForwardRule{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return err
	}
	if count+int64(additional) > int64(user.RuleLimit) {
		return gorm.ErrInvalidData
	}
	return nil
}

func checkListenPortConflict(db *gorm.DB, groupID uint, protocol string, listenPort int, excludeID uint) error {
	q := db.Model(&model.ForwardRule{}).Where("device_group_id = ? AND protocol = ? AND listen_port = ? AND enabled = ?", groupID, protocol, listenPort, true)
	if excludeID != 0 {
		q = q.Where("id <> ?", excludeID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errListenPortConflict
	}
	return nil
}

func inputToRule(input forwardRuleInput, userID uint, now time.Time) model.ForwardRule {
	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	return model.ForwardRule{
		UserID: userID, Name: input.Name, DeviceGroupID: input.DeviceGroupID,
		ListenPort: input.ListenPort, TargetAddr: input.TargetAddr, TargetPort: input.TargetPort,
		Protocol: input.Protocol, Rate: input.Rate, Enabled: enabled, Status: "active",
		CreatedAt: now, UpdatedAt: now,
	}
}

func ListForwardRules(c *gin.Context) {
	userID := c.GetUint("user_id")
	var rules []model.ForwardRule
	q := model.DB.Order("id desc")
	if !c.GetBool("is_admin") {
		q = q.Where("user_id = ?", userID)
	}
	if err := q.Find(&rules).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func CreateForwardRule(c *gin.Context) {
	userID := c.GetUint("user_id")
	var input forwardRuleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if msg := validateForwardRule(&input); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	if err := authorizeDeviceGroup(userID, input.DeviceGroupID, c.GetBool("is_admin")); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权使用该设备组"})
		return
	}
	rule := inputToRule(input, userID, time.Now())
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := checkRuleLimit(tx, userID, 1); err != nil {
			return err
		}
		if rule.Enabled {
			if err := checkListenPortConflict(tx, rule.DeviceGroupID, rule.Protocol, rule.ListenPort, 0); err != nil {
				return err
			}
		}
		return tx.Create(&rule).Error
	})
	if err != nil {
		if err == gorm.ErrInvalidData {
			c.JSON(http.StatusForbidden, gin.H{"error": "已达到规则数量限制"})
			return
		}
		if err == errListenPortConflict {
			c.JSON(http.StatusConflict, gin.H{"error": "该设备组的监听端口已被占用"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, rule)
}

func UpdateForwardRule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
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
	var patch forwardRuleUpdateInput
	if err := c.ShouldBindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	input := forwardRuleInput{Name: rule.Name, DeviceGroupID: rule.DeviceGroupID, ListenPort: rule.ListenPort,
		TargetAddr: rule.TargetAddr, TargetPort: rule.TargetPort, Protocol: rule.Protocol, Rate: rule.Rate, Enabled: &rule.Enabled}
	if patch.Name != nil {
		input.Name = *patch.Name
	}
	if patch.DeviceGroupID != nil {
		input.DeviceGroupID = *patch.DeviceGroupID
	}
	if patch.ListenPort != nil {
		input.ListenPort = *patch.ListenPort
	}
	if patch.TargetAddr != nil {
		input.TargetAddr = *patch.TargetAddr
	}
	if patch.TargetPort != nil {
		input.TargetPort = *patch.TargetPort
	}
	if patch.Dest != nil && *patch.Dest != "" {
		input.Dest = *patch.Dest
		addr, port := parseDest(input.Dest)
		if addr != "" && port > 0 {
			input.TargetAddr = addr
			input.TargetPort = port
		}
	}
	if patch.Protocol != nil {
		input.Protocol = *patch.Protocol
	}
	if patch.Rate != nil {
		input.Rate = *patch.Rate
	}
	if patch.Enabled != nil {
		input.Enabled = patch.Enabled
	}
	if msg := validateForwardRule(&input); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	if err := authorizeDeviceGroup(userID, input.DeviceGroupID, isAdmin); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权使用该设备组"})
		return
	}
	updated := inputToRule(input, rule.UserID, rule.CreatedAt)
	updated.ID = rule.ID
	updated.Status = rule.Status
	updated.Traffic = rule.Traffic
	updated.UpdatedAt = time.Now()
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if updated.Enabled {
			if err := checkListenPortConflict(tx, updated.DeviceGroupID, updated.Protocol, updated.ListenPort, updated.ID); err != nil {
				return err
			}
		}
		return tx.Save(&updated).Error
	})
	if err != nil {
		if err == errListenPortConflict {
			c.JSON(http.StatusConflict, gin.H{"error": "该设备组的监听端口已被占用"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func DeleteForwardRule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	q := model.DB.Where("id = ?", id)
	if !c.GetBool("is_admin") {
		q = q.Where("user_id = ?", c.GetUint("user_id"))
	}
	result := q.Delete(&model.ForwardRule{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func ToggleForwardRule(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var rule model.ForwardRule
	q := model.DB.Where("id = ?", id)
	if !c.GetBool("is_admin") {
		q = q.Where("user_id = ?", c.GetUint("user_id"))
	}
	if err := q.First(&rule).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
		return
	}
	rule.Enabled = !rule.Enabled
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if rule.Enabled {
			if err := checkListenPortConflict(tx, rule.DeviceGroupID, rule.Protocol, rule.ListenPort, rule.ID); err != nil {
				return err
			}
		}
		return tx.Save(&rule).Error
	})
	if err != nil {
		if err == errListenPortConflict {
			c.JSON(http.StatusConflict, gin.H{"error": "该设备组的监听端口已被占用"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, rule)
}

func BatchCreateForwardRules(c *gin.Context) {
	userID := c.GetUint("user_id")
	var inputs []forwardRuleInput
	if err := c.ShouldBindJSON(&inputs); err != nil || len(inputs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if len(inputs) > maxBatchRules {
		c.JSON(http.StatusBadRequest, gin.H{"error": "单次最多导入100条规则"})
		return
	}
	now := time.Now()
	rules := make([]model.ForwardRule, 0, len(inputs))
	for i := range inputs {
		if msg := validateForwardRule(&inputs[i]); msg != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "第" + strconv.Itoa(i+1) + "条规则: " + msg})
			return
		}
		if err := authorizeDeviceGroup(userID, inputs[i].DeviceGroupID, c.GetBool("is_admin")); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "第" + strconv.Itoa(i+1) + "条规则无权使用该设备组"})
			return
		}
		rules = append(rules, inputToRule(inputs[i], userID, now))
	}
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := checkRuleLimit(tx, userID, len(inputs)); err != nil {
			return err
		}
		seen := make(map[string]struct{}, len(rules))
		for _, rule := range rules {
			if !rule.Enabled {
				continue
			}
			key := strconv.FormatUint(uint64(rule.DeviceGroupID), 10) + ":" + rule.Protocol + ":" + strconv.Itoa(rule.ListenPort)
			if _, exists := seen[key]; exists {
				return errListenPortConflict
			}
			seen[key] = struct{}{}
			if err := checkListenPortConflict(tx, rule.DeviceGroupID, rule.Protocol, rule.ListenPort, 0); err != nil {
				return err
			}
		}
		return tx.Create(&rules).Error
	})
	if err != nil {
		if err == gorm.ErrInvalidData {
			c.JSON(http.StatusForbidden, gin.H{"error": "导入后将超过规则数量限制"})
			return
		}
		if err == errListenPortConflict {
			c.JSON(http.StatusConflict, gin.H{"error": "导入规则存在监听端口冲突"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "批量创建失败"})
		return
	}
	c.JSON(http.StatusOK, rules)
}
