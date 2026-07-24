package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
)

func CreateUserNode(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "节点名称无效"})
		return
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}
	token := hex.EncodeToString(b)
	userNode := model.UserNode{
		UserID: c.GetUint("user_id"),
		Name:   req.Name,
		Token:  token,
		Status: "offline",
	}
	if err := model.DB.Create(&userNode).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"node_id": userNode.ID, "token": token})
}

func ListUserNodes(c *gin.Context) {
	var nodes []model.UserNode
	model.DB.Where("user_id = ?", c.GetUint("user_id")).Order("id desc").Find(&nodes)
	c.JSON(http.StatusOK, nodes)
}

func DeleteUserNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	result := model.DB.Where("id = ? AND user_id = ?", id, c.GetUint("user_id")).Delete(&model.UserNode{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "节点不存在"})
		return
	}
	revokeUserNodeSession(uint(id))
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func GetUserNodeSetup(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	var node model.UserNode
	if err := model.DB.Where("id = ? AND user_id = ?", id, c.GetUint("user_id")).First(&node).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "节点不存在"})
		return
	}
	c.Header("Cache-Control", "no-store, private")
	c.Header("Pragma", "no-cache")
	c.JSON(http.StatusOK, gin.H{"node_id": node.ID, "token": node.Token})
}

func AdminListAllUserNodes(c *gin.Context) {
	var nodes []model.UserNode
	model.DB.Order("id desc").Find(&nodes)
	c.JSON(http.StatusOK, nodes)
}

func AdminDeleteUserNode(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID无效"})
		return
	}
	result := model.DB.Delete(&model.UserNode{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "节点不存在"})
		return
	}
	revokeUserNodeSession(uint(id))
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
