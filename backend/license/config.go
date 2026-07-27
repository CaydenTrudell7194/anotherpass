package license

import "time"

// LicenseState 授权状态
type LicenseState int

const (
	StateUnverified LicenseState = iota
	StateActive
	StateDeactivated
	StateRevoked
	StateError
)

// Config 授权配置
type Config struct {
	ServerURL   string // 授权服务器地址 (如 https://license.example.com)
	LicenseKey  string // 卡密
	Domain      string // 面板域名
	PublicKey   string // RSA 公钥 (PEM 格式)
	AdminAPIKey string // 管理员 API Key (可选，用于后台操作)
}

// License 授权实例
type License struct {
	Config
	Fingerprint   string
	State         LicenseState
	ExpiresAt     time.Time
	LastHeartbeat time.Time
	DeactivateCount int
	stopCh        chan struct{}
}

// Status 返回给前端的授权状态
type Status struct {
	Activated   bool   `json:"activated"`
	LicenseKey  string `json:"license_key,omitempty"`
	Domain      string `json:"domain,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	State       string `json:"state"`
	LastHeartbeat string `json:"last_heartbeat,omitempty"`
	DeactivateCount int `json:"deactivate_count,omitempty"`
}

// New 创建授权实例
func New(cfg Config) *License {
	return &License{
		Config:    cfg,
		State:     StateUnverified,
		stopCh:    make(chan struct{}),
	}
}
