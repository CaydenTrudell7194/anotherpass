package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"forward-panel/internal/config"
	"forward-panel/internal/handler"
	"forward-panel/internal/middleware"
	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()
	if err := cfg.ValidatePayments(); err != nil {
		log.Fatalf("支付配置错误: %v", err)
	}
	handler.ConfigurePayments(cfg)
	if err := middleware.ConfigureJWTSecret(); err != nil {
		log.Fatalf("安全配置错误: %v", err)
	}

	dsn := cfg.Database
	if len(dsn) > 9 && dsn[:9] == "sqlite3://" {
		dsn = dsn[9:]
	}
	os.MkdirAll(filepath.Dir(dsn), 0755)
	if err := model.InitDatabase(dsn); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	if err := initDefaults(); err != nil {
		log.Fatalf("初始化默认数据失败: %v", err)
	}

	// 节点离线检测
	go func() {
		for {
			handler.CheckOfflineNodes()
			time.Sleep(10 * time.Second)
		}
	}()

	r := gin.Default()
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("代理配置失败: %v", err)
	}
	r.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4<<20)
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "same-origin")
		c.Next()
	})

	r.Static("/assets", cfg.FrontendDir+"/assets")
	r.StaticFile("/", cfg.FrontendDir+"/index.html")
	r.StaticFile("/favicon.ico", cfg.FrontendDir+"/favicon.ico")

	api := r.Group("/api")
	{
		api.POST("/login", handler.Login)
		api.POST("/register", handler.Register)
		api.GET("/site", handler.PublicSiteSettings)
		api.GET("/payment/epay/notify", handler.EpayNotify)
		api.POST("/payment/epay/notify", handler.EpayNotify)
		api.GET("/payment/codepay/notify", handler.CodepayNotify)
		api.POST("/payment/codepay/notify", handler.CodepayNotify)
		api.GET("/payment/epay/return", handler.PaymentReturn)
		api.POST("/payment/epay/return", handler.PaymentReturn)
		api.GET("/payment/codepay/return", handler.PaymentReturn)
		api.POST("/payment/codepay/return", handler.PaymentReturn)

		auth := api.Group("")
		auth.Use(middleware.AuthRequired())
		{
			auth.GET("/profile", handler.GetProfile)
			auth.PUT("/password", handler.ChangePassword)

			auth.GET("/device_groups", handler.ListMyDeviceGroups)
			auth.GET("/nodes", handler.ListMyNodeStatus)
			auth.POST("/node-monitor/ticket", handler.CreateNodeMonitorTicket)
			auth.GET("/plans", handler.ListServicePlans)
			auth.GET("/orders", handler.ListOrders)
			auth.POST("/orders", handler.CreateOrder)
			auth.POST("/orders/balance", handler.CreateBalanceOrder)
			auth.GET("/balance/ledger", handler.ListBalanceLedger)
			auth.GET("/recharge/providers", handler.ListRechargeProviders)
			auth.POST("/recharge", handler.CreateRecharge)
			auth.GET("/recharge/orders", handler.ListRechargeOrders)
			auth.GET("/forward_rules", handler.ListForwardRules)
			auth.POST("/forward_rules", handler.CreateForwardRule)
			auth.PUT("/forward_rules/:id", handler.UpdateForwardRule)
			auth.DELETE("/forward_rules/:id", handler.DeleteForwardRule)
			auth.PUT("/forward_rules/:id/toggle", handler.ToggleForwardRule)
			auth.POST("/forward_rules/batch", handler.BatchCreateForwardRules)

			auth.POST("/user-nodes", handler.CreateUserNode)
			auth.GET("/user-nodes", handler.ListUserNodes)
			auth.GET("/user-nodes/:id/setup", handler.GetUserNodeSetup)
			auth.DELETE("/user-nodes/:id", handler.DeleteUserNode)

			auth.GET("/affiliate/info", handler.GetAffiliateInfo)
			auth.POST("/redeem", handler.RedeemCodeHandler)
		}

		admin := api.Group("/admin")
		admin.Use(middleware.AuthRequired())
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/dashboard", handler.AdminDashboard)
			admin.GET("/settings", handler.GetSiteSettings)
			admin.PUT("/settings", handler.UpdateSiteSettings)

			admin.GET("/users", handler.ListUsers)
			admin.POST("/users", handler.CreateUser)
			admin.PUT("/users/:id", handler.UpdateUser)
			admin.DELETE("/users/:id", handler.DeleteUser)
			admin.POST("/users/:id/balance-adjustments", handler.AdminAdjustBalance)
			admin.GET("/users/:id/balance-ledger", handler.AdminListBalanceLedger)

			admin.GET("/user_groups", handler.ListUserGroups)
			admin.POST("/user_groups", handler.CreateUserGroup)
			admin.PUT("/user_groups/:id", handler.UpdateUserGroup)
			admin.DELETE("/user_groups/:id", handler.DeleteUserGroup)

			admin.GET("/device_groups", handler.ListDeviceGroups)
			admin.POST("/device_groups", handler.CreateDeviceGroup)
			admin.PUT("/device_groups/:id", handler.UpdateDeviceGroup)
			admin.DELETE("/device_groups/:id", handler.DeleteDeviceGroup)
			admin.GET("/device_groups/:id/node-token", handler.GetDeviceGroupNodeToken)
			admin.POST("/device_groups/:id/reset-node-token", handler.ResetDeviceGroupNodeToken)

			admin.GET("/nodes", handler.ListNodeStatus)
			admin.POST("/nodes/register", handler.RegisterNode)
			admin.POST("/nodes/:id/setup", handler.GetNodeSetup)
			admin.POST("/nodes/:id/rotate-token", handler.RotateNodeToken)
			admin.DELETE("/nodes/:id", handler.DeleteNode)
			admin.GET("/plans", handler.AdminListServicePlans)
			admin.POST("/plans", handler.AdminCreateServicePlan)
			admin.PUT("/plans/:id", handler.AdminUpdateServicePlan)
			admin.DELETE("/plans/:id", handler.AdminDeleteServicePlan)
			admin.GET("/orders", handler.AdminListOrders)
			admin.POST("/orders/:id/approve", handler.AdminApproveOrder)
			admin.POST("/orders/:id/reject", handler.AdminRejectOrder)

			admin.GET("/user-nodes", handler.AdminListAllUserNodes)
			admin.DELETE("/user-nodes/:id", handler.AdminDeleteUserNode)

			admin.GET("/affiliates", handler.AdminListAffiliates)
			admin.POST("/redeem-codes", handler.AdminCreateRedeemCodes)
			admin.GET("/redeem-codes", handler.AdminListRedeemCodes)
			admin.DELETE("/redeem-codes/:id", handler.AdminDeleteRedeemCode)
		}

		api.POST("/node/heartbeat", handler.NodeHeartbeat)
		api.POST("/node/rules", handler.GetNodeRules)
		api.POST("/node/enroll", handler.EnrollNode)
		api.GET("/node/ws", handler.NodeWebSocket)
		api.GET("/node-monitor/ws", handler.NodeMonitorWebSocket)
	}

	log.Printf("面板启动，监听 %s", cfg.Listen)
	server := &http.Server{
		Addr: cfg.Listen, Handler: r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("启动失败: %v", err)
	}
}

func initDefaults() error {
	var count int64
	model.DB.Model(&model.UserGroup{}).Count(&count)
	if count == 0 {
		if err := model.DB.Create(&model.UserGroup{Name: "默认用户组", Description: "系统默认用户组"}).Error; err != nil {
			return err
		}
	}

	model.DB.Model(&model.User{}).Count(&count)
	if count == 0 {
		adminPwd := os.Getenv("ADMIN_PASSWORD")
		if adminPwd == "" {
			return fmt.Errorf("ADMIN_PASSWORD is required on first startup")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(adminPwd), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := model.DB.Create(&model.User{
			Username:    "admin",
			Password:    string(hash),
			DisplayName: "管理员",
			UserGroupID: 1,
			Status:      "active",
			IsAdmin:     true,
			ExpireAt:    time.Now().AddDate(10, 0, 0),
			RuleLimit:   9999,
		}).Error; err != nil {
			return err
		}
		fmt.Println("已创建默认管理员 admin，请妥善保管安装时设置的密码")
	}
	return nil
}
