package handler

import (
	"net/http"
	"strconv"
	"strings"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
)

func ListDeviceGroups(c *gin.Context) {
	var groups []model.DeviceGroup
	model.DB.Order("sort_order asc, id asc").Find(&groups)
	c.JSON(http.StatusOK, groups)
}

func ListMyDeviceGroups(c *gin.Context) {
	userID := c.GetUint("user_id")
	var user model.User
	if err := model.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	var groups []model.DeviceGroup
	model.DB.Where("type IN ?", []string{
		string(model.DeviceGroupEntryForceDirect),
		string(model.DeviceGroupEntryOptionalDirect),
	}).Order("sort_order asc, id asc").Find(&groups)

	var accessible []model.DeviceGroup
	uidStr := strconv.FormatUint(uint64(user.UserGroupID), 10)
	for _, g := range groups {
		if g.UserGroupIDs == "" {
			accessible = append(accessible, g)
			continue
		}
		ids := strings.Split(g.UserGroupIDs, ",")
		for _, id := range ids {
			if strings.TrimSpace(id) == uidStr {
				accessible = append(accessible, g)
				break
			}
		}
	}
	c.JSON(http.StatusOK, accessible)
}

func CreateDeviceGroup(c *gin.Context) {
	var group model.DeviceGroup
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

func UpdateDeviceGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var orig model.DeviceGroup
	if err := model.DB.First(&orig, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "设备组不存在"})
		return
	}
	var input struct {
		Name           *string  `json:"name"`
		Type           *string  `json:"type"`
		UserGroupIDs   *string  `json:"user_group_ids"`
		ConnectionAddr *string  `json:"connection_addr"`
		Rate           *float64 `json:"rate"`
		HideInProbe    *bool    `json:"hide_in_probe"`
		Notes          *string  `json:"notes"`
		SortOrder      *int     `json:"sort_order"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	if input.Name != nil {
		orig.Name = *input.Name
	}
	if input.Type != nil {
		orig.Type = model.DeviceGroupType(*input.Type)
	}
	if input.UserGroupIDs != nil {
		orig.UserGroupIDs = *input.UserGroupIDs
	}
	if input.ConnectionAddr != nil {
		orig.ConnectionAddr = *input.ConnectionAddr
	}
	if input.Rate != nil {
		orig.Rate = *input.Rate
	}
	if input.HideInProbe != nil {
		orig.HideInProbe = *input.HideInProbe
	}
	if input.Notes != nil {
		orig.Notes = *input.Notes
	}
	if input.SortOrder != nil {
		orig.SortOrder = *input.SortOrder
	}
	model.DB.Save(&orig)
	c.JSON(http.StatusOK, orig)
}

func DeleteDeviceGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	model.DB.Delete(&model.DeviceGroup{}, id)
	model.DB.Where("device_group_id = ?", id).Delete(&model.Node{})
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
