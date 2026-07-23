package main

import (
	"fmt"
	"log"
	"time"

	"forward-panel/internal/config"
	"forward-panel/internal/handler"
	"forward-panel/internal/middleware"
	"forward-panel/internal/model"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()

	dsn := cfg.Database
	if len(dsn) > 9 && dsn[:9] == "sqlite3://" {
		dsn = dsn[9:]
	}
	if err := model.InitDatabase(dsn); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	initDefaults()

	// 节点离线检测
	go func() {
		for {
			handler.CheckOfflineNodes()
			time.Sleep(60 * time.Second)
		}
	}()

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.Static("/assets", cfg.FrontendDir+"/assets")
	r.StaticFile("/", cfg.FrontendDir+"/index.html")
	r.StaticFile("/favicon.ico", cfg.FrontendDir+"/favicon.ico")

	api := r.Group("/api")
	{
		api.POST("/login", handler.Login)

		auth := api.Group("")
		auth.Use(middleware.AuthRequired())
		{
			auth.GET("/profile", handler.GetProfile)
			auth.PUT("/password", handler.ChangePassword)

			auth.GET("/device_groups", handler.ListMyDeviceGroups)
			auth.GET("/forward_rules", handler.ListForwardRules)
			auth.POST("/forward_rules", handler.CreateForwardRule)
			auth.PUT("/forward_rules/:id", handler.UpdateForwardRule)
			auth.DELETE("/forward_rules/:id", handler.DeleteForwardRule)
			auth.PUT("/forward_rules/:id/toggle", handler.ToggleForwardRule)
			auth.POST("/forward_rules/batch", handler.BatchCreateForwardRules)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired())
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/dashboard", handler.AdminDashboard)

			admin.GET("/users", handler.ListUsers)
			admin.POST("/users", handler.CreateUser)
			admin.PUT("/users/:id", handler.UpdateUser)
			admin.DELETE("/users/:id", handler.DeleteUser)

			admin.GET("/user_groups", handler.ListUserGroups)
			admin.POST("/user_groups", handler.CreateUserGroup)
			admin.PUT("/user_groups/:id", handler.UpdateUserGroup)
			admin.DELETE("/user_groups/:id", handler.DeleteUserGroup)

			admin.GET("/device_groups", handler.ListDeviceGroups)
			admin.POST("/device_groups", handler.CreateDeviceGroup)
			admin.PUT("/device_groups/:id", handler.UpdateDeviceGroup)
			admin.DELETE("/device_groups/:id", handler.DeleteDeviceGroup)

			admin.GET("/nodes", handler.ListNodeStatus)
			admin.POST("/nodes/register", handler.RegisterNode)
			admin.DELETE("/nodes/:id", handler.DeleteNode)
		}

		api.POST("/node/heartbeat", handler.NodeHeartbeat)
		api.POST("/node/rules", handler.GetNodeRules)
	}

	log.Printf("面板启动，监听 %s", cfg.Listen)
	if err := r.Run(cfg.Listen); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func initDefaults() {
	var count int64
	model.DB.Model(&model.UserGroup{}).Count(&count)
	if count == 0 {
		model.DB.Create(&model.UserGroup{Name: "默认用户组", Description: "系统默认用户组"})
	}

	model.DB.Model(&model.User{}).Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		model.DB.Create(&model.User{
			Username:    "admin",
			Password:    string(hash),
			DisplayName: "管理员",
			UserGroupID: 1,
			Status:      "active",
			IsAdmin:     true,
			ExpireAt:    time.Now().AddDate(10, 0, 0),
			RuleLimit:   9999,
		})
		fmt.Println("========================================")
		fmt.Println("  默认管理员: admin / admin")
		fmt.Println("========================================")
	}
}
