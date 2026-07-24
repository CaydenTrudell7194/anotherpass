package handler

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const siteSettingsKey = "site_settings"

type SiteSettings struct {
	SiteName                  string `json:"site_name"`
	SiteSubtitle              string `json:"site_subtitle"`
	SiteNotice                string `json:"site_notice"`
	AllowRegister             bool   `json:"allow_register"`
	RegisterUserGroupID       uint   `json:"register_user_group_id"`
	RegisterRuleLimit         int    `json:"register_rule_limit"`
	RegisterExpireDays        int    `json:"register_expire_days"`
	ThemePolicy               string `json:"theme_policy"`
	BackgroundURL             string `json:"background_url"`
	MobileBackgroundURL       string `json:"mobile_background_url"`
	OfflineNodeSeconds        int    `json:"offline_node_seconds"`
	OfflineNodeRetentionHours int    `json:"offline_node_retention_hours"`
	TelegramEnabled           bool   `json:"telegram_enabled"`
	TelegramChatID            string `json:"telegram_chat_id"`
	TelegramBotToken          string `json:"telegram_bot_token"`
	TelegramBotConfigured     bool   `json:"telegram_bot_configured" gorm:"-"`
	EpayGateway               string `json:"epay_gateway"`
	EpayPid                   string `json:"epay_pid"`
	CodepayCreateUrl          string `json:"codepay_create_url"`
	CodepayMerchantId         string `json:"codepay_merchant_id"`
}

type PublicSiteInfo struct {
	SiteName            string `json:"site_name"`
	SiteSubtitle        string `json:"site_subtitle"`
	SiteNotice          string `json:"site_notice"`
	AllowRegister       bool   `json:"allow_register"`
	ThemePolicy         string `json:"theme_policy"`
	BackgroundURL       string `json:"background_url"`
	MobileBackgroundURL string `json:"mobile_background_url"`
}

var registerAttempts = struct {
	sync.Mutex
	byIP map[string][]time.Time
}{byIP: make(map[string][]time.Time)}

var registerHashSlots = make(chan struct{}, 4)

func DefaultSiteSettings() SiteSettings {
	return SiteSettings{
		SiteName: "转发面板", SiteSubtitle: "入口直出转发管理平台",
		RegisterUserGroupID: 1, RegisterRuleLimit: 100, RegisterExpireDays: 365,
		ThemePolicy: "classic", OfflineNodeSeconds: 90, OfflineNodeRetentionHours: 24,
	}
}

func LoadSiteSettings() SiteSettings {
	settings := DefaultSiteSettings()
	var row model.SystemConfig
	if err := model.DB.Where("key = ?", siteSettingsKey).First(&row).Error; err == nil {
		_ = json.Unmarshal([]byte(row.Value), &settings)
	}
	normalizeSiteSettings(&settings)
	return settings
}

func normalizeSiteSettings(settings *SiteSettings) {
	settings.SiteName = strings.TrimSpace(settings.SiteName)
	settings.SiteSubtitle = strings.TrimSpace(settings.SiteSubtitle)
	settings.BackgroundURL = strings.TrimSpace(settings.BackgroundURL)
	settings.MobileBackgroundURL = strings.TrimSpace(settings.MobileBackgroundURL)
	settings.TelegramChatID = strings.TrimSpace(settings.TelegramChatID)
	settings.TelegramBotConfigured = telegramBotToken() != "" || settings.TelegramBotToken != ""
	settings.EpayGateway = strings.TrimSpace(settings.EpayGateway)
	settings.EpayPid = strings.TrimSpace(settings.EpayPid)
	settings.CodepayCreateUrl = strings.TrimSpace(settings.CodepayCreateUrl)
	settings.CodepayMerchantId = strings.TrimSpace(settings.CodepayMerchantId)
	if settings.SiteName == "" {
		settings.SiteName = "转发面板"
	}
	if settings.SiteSubtitle == "" {
		settings.SiteSubtitle = "入口直出转发管理平台"
	}
	if settings.RegisterUserGroupID == 0 {
		settings.RegisterUserGroupID = 1
	}
	if settings.RegisterRuleLimit < 0 {
		settings.RegisterRuleLimit = 0
	}
	if settings.RegisterExpireDays < 1 {
		settings.RegisterExpireDays = 365
	}
	if settings.ThemePolicy != "classic" && settings.ThemePolicy != "transparent" {
		settings.ThemePolicy = "classic"
	}
	if settings.OfflineNodeSeconds < 20 {
		settings.OfflineNodeSeconds = 20
	}
	if settings.OfflineNodeSeconds > 3600 {
		settings.OfflineNodeSeconds = 3600
	}
	if settings.OfflineNodeRetentionHours < 1 {
		settings.OfflineNodeRetentionHours = 24
	}
	if settings.OfflineNodeRetentionHours > 8760 {
		settings.OfflineNodeRetentionHours = 8760
	}
}

func validateSiteSettings(settings *SiteSettings) string {
	if settings.RegisterRuleLimit < 0 || settings.RegisterExpireDays < 1 || settings.RegisterExpireDays > 3650 || settings.OfflineNodeSeconds < 20 || settings.OfflineNodeSeconds > 3600 || settings.OfflineNodeRetentionHours < 1 || settings.OfflineNodeRetentionHours > 8760 {
		return "注册配额、有效期或节点时间参数无效"
	}
	normalizeSiteSettings(settings)
	if len(settings.SiteName) > 64 || len(settings.SiteSubtitle) > 128 {
		return "站点名称或副标题过长"
	}
	if len(settings.SiteNotice) > 4096 {
		return "站点公告不能超过4096个字符"
	}
	if len(settings.BackgroundURL) > 1024 || len(settings.MobileBackgroundURL) > 1024 {
		return "背景图URL过长"
	}
	if settings.TelegramEnabled && (settings.TelegramChatID == "" || len(settings.TelegramChatID) > 64 || (telegramBotToken() == "" && settings.TelegramBotToken == "")) {
		return "Telegram Bot Token 未配置或 Chat ID 无效"
	}
	var group model.UserGroup
	if err := model.DB.First(&group, settings.RegisterUserGroupID).Error; err != nil {
		return "注册默认用户组不存在"
	}
	return ""
}

func PublicSiteSettings(c *gin.Context) {
	settings := LoadSiteSettings()
	c.JSON(http.StatusOK, PublicSiteInfo{SiteName: settings.SiteName, SiteSubtitle: settings.SiteSubtitle,
		SiteNotice: settings.SiteNotice, AllowRegister: settings.AllowRegister, ThemePolicy: settings.ThemePolicy,
		BackgroundURL: settings.BackgroundURL, MobileBackgroundURL: settings.MobileBackgroundURL})
}

func GetSiteSettings(c *gin.Context) {
	settings := LoadSiteSettings()
	if settings.TelegramBotToken != "" {
		settings.TelegramBotToken = "****"
	}
	c.JSON(http.StatusOK, settings)
}

func UpdateSiteSettings(c *gin.Context) {
	oldSettings := LoadSiteSettings()
	oldToken := oldSettings.TelegramBotToken

	var incoming SiteSettings
	if err := c.ShouldBindJSON(&incoming); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	settings := oldSettings
	settings.SiteName = incoming.SiteName
	settings.SiteSubtitle = incoming.SiteSubtitle
	settings.SiteNotice = incoming.SiteNotice
	settings.AllowRegister = incoming.AllowRegister
	settings.RegisterUserGroupID = incoming.RegisterUserGroupID
	settings.RegisterRuleLimit = incoming.RegisterRuleLimit
	settings.RegisterExpireDays = incoming.RegisterExpireDays
	settings.ThemePolicy = incoming.ThemePolicy
	settings.BackgroundURL = incoming.BackgroundURL
	settings.MobileBackgroundURL = incoming.MobileBackgroundURL
	settings.OfflineNodeSeconds = incoming.OfflineNodeSeconds
	settings.OfflineNodeRetentionHours = incoming.OfflineNodeRetentionHours
	settings.TelegramEnabled = incoming.TelegramEnabled
	settings.TelegramChatID = incoming.TelegramChatID
	settings.EpayGateway = incoming.EpayGateway
	settings.EpayPid = incoming.EpayPid
	settings.CodepayCreateUrl = incoming.CodepayCreateUrl
	settings.CodepayMerchantId = incoming.CodepayMerchantId

	if incoming.TelegramBotToken == "" || incoming.TelegramBotToken == "****" {
		settings.TelegramBotToken = oldToken
	} else {
		settings.TelegramBotToken = incoming.TelegramBotToken
	}

	if msg := validateSiteSettings(&settings); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}
	settings.TelegramBotConfigured = false
	value, err := json.Marshal(settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	row := model.SystemConfig{Key: siteSettingsKey, Value: string(value)}
	if err := model.DB.Where("key = ?", siteSettingsKey).Assign(model.SystemConfig{Value: row.Value}).FirstOrCreate(&row).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存失败"})
		return
	}
	if settings.TelegramBotToken != "" {
		settings.TelegramBotToken = "****"
	}
	settings.TelegramBotConfigured = telegramBotToken() != "" || oldToken != ""
	c.JSON(http.StatusOK, settings)
}

type RegisterReq struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	DisplayName string `json:"display_name"`
}

func Register(c *gin.Context) {
	settings := LoadSiteSettings()
	if !settings.AllowRegister {
		c.JSON(http.StatusForbidden, gin.H{"error": "站点未开放注册"})
		return
	}
	if !allowRegistrationAttempt(registrationClientIP(c)) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "注册请求过于频繁，请稍后再试"})
		return
	}
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.Username == "" || len(req.Username) > 64 || len(req.Password) < 8 || len(req.DisplayName) > 64 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名、显示名或密码无效"})
		return
	}
	select {
	case registerHashSlots <- struct{}{}:
		defer func() { <-registerHashSlots }()
	default:
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "注册服务繁忙，请稍后再试"})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码无效"})
		return
	}
	now := time.Now()
	user := model.User{Username: req.Username, Password: string(hash), DisplayName: req.DisplayName,
		UserGroupID: settings.RegisterUserGroupID, Status: "active",
		RuleLimit: settings.RegisterRuleLimit, ExpireAt: now.AddDate(0, 0, settings.RegisterExpireDays), CreatedAt: now, UpdatedAt: now}
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		var group model.UserGroup
		if err := tx.First(&group, settings.RegisterUserGroupID).Error; err != nil {
			return err
		}
		return tx.Create(&user).Error
	}); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "注册成功"})
}

func allowRegistrationAttempt(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	registerAttempts.Lock()
	defer registerAttempts.Unlock()
	previous := registerAttempts.byIP[ip]
	recent := previous[:0]
	for _, attempt := range previous {
		if attempt.After(cutoff) {
			recent = append(recent, attempt)
		}
	}
	if len(recent) >= 5 {
		registerAttempts.byIP[ip] = recent
		return false
	}
	registerAttempts.byIP[ip] = append(recent, now)
	if len(registerAttempts.byIP) > 4096 {
		for key, attempts := range registerAttempts.byIP {
			if len(attempts) == 0 || attempts[len(attempts)-1].Before(cutoff) {
				delete(registerAttempts.byIP, key)
			}
		}
	}
	return true
}

func registrationClientIP(c *gin.Context) string {
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		host = c.Request.RemoteAddr
	}
	if host == "127.0.0.1" || host == "::1" {
		if forwarded := strings.TrimSpace(c.GetHeader("X-Real-IP")); net.ParseIP(forwarded) != nil {
			return forwarded
		}
	}
	return host
}
