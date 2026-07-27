package license

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// activateRequest 激活请求
type activateRequest struct {
	Key         string `json:"key"`
	Fingerprint string `json:"fingerprint"`
	Domain      string `json:"domain"`
	Timestamp   int64  `json:"timestamp"`
	Signature   string `json:"signature"`
}

// licenseResponse 授权服务器响应
type licenseResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
	License     *struct {
		Key            string `json:"key"`
		ExpiresAt      string `json:"expires_at,omitempty"`
		DeactivateCount int   `json:"deactivate_count,omitempty"`
	} `json:"license,omitempty"`
	ServerTime  int64  `json:"server_time"`
	NextInterval int   `json:"next_interval,omitempty"`
	Signature   string `json:"signature"`
}

// publicKeyPEM 内嵌 RSA 公钥（占位，编译时替换或通过环境变量注入）
// 建议：编译时通过 -X ldflags 注入 base64 编码的公钥，或通过环境变量 LICENSE_PUBLIC_KEY 传入
// 为防静态分析，此处的默认值不应是真实公钥，而是在 CI 中替换
var publicKeyPEM = ""

func init() {
	// 优先从环境变量读取公钥
	if pk := getEnvOrDefault("LICENSE_PUBLIC_KEY", ""); pk != "" {
		publicKeyPEM = pk
	}
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// hmacSign 计算 HMAC-SHA256 签名
func hmacSign(data, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify 执行启动授权验证
func (l *License) Verify() error {
	// 1. 采集机器指纹
	fp := getMachineFingerprint()
	if fp == "" {
		l.State = StateError
		return fmt.Errorf("无法采集机器指纹")
	}
	l.Fingerprint = fp

	// 2. 检查必要配置
	if l.LicenseKey == "" {
		l.State = StateError
		return fmt.Errorf("缺少卡密，请设置 LICENSE_KEY 环境变量")
	}
	if l.Domain == "" {
		l.State = StateError
		return fmt.Errorf("缺少域名，请设置 LICENSE_DOMAIN 环境变量")
	}
	if l.ServerURL == "" {
		l.State = StateError
		return fmt.Errorf("缺少授权服务器地址，请设置 LICENSE_SERVER 环境变量")
	}
	if publicKeyPEM == "" && l.PublicKey != "" {
		publicKeyPEM = l.PublicKey
	}
	if publicKeyPEM == "" {
		l.State = StateError
		return fmt.Errorf("缺少 RSA 公钥，请设置 LICENSE_PUBLIC_KEY 环境变量")
	}

	// 3. 发送激活请求
	now := time.Now()
	req := activateRequest{
		Key:         l.LicenseKey,
		Fingerprint: fp,
		Domain:      l.Domain,
		Timestamp:   now.Unix(),
	}

	// 对请求内容做 HMAC 签名（使用卡密作为 HMAC secret）
	sigPayload := fmt.Sprintf("%s%s%s%d", req.Key, req.Fingerprint, req.Domain, req.Timestamp)
	req.Signature = hmacSign(sigPayload, l.LicenseKey)

	body, _ := json.Marshal(req)

	resp, err := http.Post(l.ServerURL+"/api/activate", "application/json", bytes.NewReader(body))
	if err != nil {
		l.State = StateError
		return fmt.Errorf("连接授权服务器失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var result licenseResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		l.State = StateError
		return fmt.Errorf("授权服务器响应格式错误: %w", err)
	}

	// 4. 验证服务器响应 RSA 签名
	// 验证签名的原始数据取响应 body 中除 signature 外的规范化 JSON 字符串
	// 实际使用时建议服务器返回时在 sign_data 字段指明签名原文
	verifyData := fmt.Sprintf(`{"status":"%s","server_time":%d}`, result.Status, result.ServerTime)
	if !verifySignature(verifyData, result.Signature, publicKeyPEM) {
		l.State = StateError
		return fmt.Errorf("授权服务器响应签名验证失败")
	}

	// 5. 处理结果
	switch result.Status {
	case "ok":
		l.State = StateActive
		if result.License != nil {
			if result.License.ExpiresAt != "" {
				if exp, err := time.Parse(time.RFC3339, result.License.ExpiresAt); err == nil {
					l.ExpiresAt = exp
				}
			}
			l.DeactivateCount = result.License.DeactivateCount
		}
		return nil

	case "deactivated":
		l.State = StateDeactivated
		return fmt.Errorf("授权已被管理员解绑")

	case "revoked":
		l.State = StateRevoked
		return fmt.Errorf("授权已被管理员吊销")

	default:
		msg := result.Message
		if msg == "" {
			msg = "授权验证失败"
		}
		l.State = StateError
		return fmt.Errorf(msg)
	}
}
