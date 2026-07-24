package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var JWTSecret []byte

func ConfigureJWTSecret() error {
	s := os.Getenv("JWT_SECRET")
	if len(s) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	JWTSecret = []byte(s)
	return nil
}

type Claims struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	IsAdmin      bool   `json:"is_admin"`
	TokenVersion uint   `json:"token_version"`
	jwt.RegisteredClaims
}

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		claims := &Claims{}
		t, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil || !t.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "令牌无效"})
			return
		}
		var user model.User
		if err := model.DB.First(&user, claims.UserID).Error; err != nil || user.Status != "active" || user.TokenVersion != claims.TokenVersion || (!user.ExpireAt.IsZero() && time.Now().After(user.ExpireAt)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "账户无效或已过期"})
			return
		}
		c.Set("user_id", user.ID)
		c.Set("username", user.Username)
		c.Set("is_admin", user.IsAdmin)
		c.Next()
	}
}

func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		isAdmin, _ := c.Get("is_admin")
		if isAdmin != true {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "需要管理员权限"})
			return
		}
		c.Next()
	}
}
