package license

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// heartbeatRequest 心跳请求
type heartbeatRequest struct {
	Fingerprint string `json:"fingerprint"`
	Domain      string `json:"domain"`
	Timestamp   int64  `json:"timestamp"`
	Signature   string `json:"signature"`
}

// StartHeartbeat 启动后台心跳协程
func (l *License) StartHeartbeat() {
	if l.State != StateActive {
		return
	}

	interval := 300 // 默认 5 分钟
	failCount := 0
	maxFails := 3 // 连续失败次数上限，超过后仍放行（防服务器宕机误杀）

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				result, err := l.doHeartbeat()
				if err != nil {
					failCount++
					if failCount < maxFails {
						// 临时失败，继续
						continue
					}
					// 超过连续失败上限，放行但标记警告
					continue
				}

				failCount = 0
				l.LastHeartbeat = time.Now()

				switch result {
				case "deactivated":
					l.State = StateDeactivated
					return

				case "revoked":
					l.State = StateRevoked
					return
				}

			case <-l.stopCh:
				return
			}
		}
	}()
}

// StopHeartbeat 停止心跳
func (l *License) StopHeartbeat() {
	close(l.stopCh)
}

// doHeartbeat 执行单次心跳
func (l *License) doHeartbeat() (string, error) {
	now := time.Now()
	req := heartbeatRequest{
		Fingerprint: l.Fingerprint,
		Domain:      l.Domain,
		Timestamp:   now.Unix(),
	}

	// HMAC 签名
	sigPayload := fmt.Sprintf("%s%s%d", req.Fingerprint, req.Domain, req.Timestamp)
	req.Signature = hmacSign(sigPayload, l.LicenseKey)

	body, _ := json.Marshal(req)

	resp, err := http.Post(l.ServerURL+"/api/verify", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("心跳请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	var result licenseResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("心跳响应解析失败: %w", err)
	}

	// 验证 RSA 签名（与服务器约定的签名原文格式一致）
	verifyData := fmt.Sprintf(`{"status":"%s","server_time":%d}`, result.Status, result.ServerTime)
	if !verifySignature(verifyData, result.Signature, publicKeyPEM) {
		return "", fmt.Errorf("心跳响应签名验证失败")
	}

	return result.Status, nil
}

// Status 获取当前授权状态（供管理后台使用）
func (l *License) Status() Status {
	s := Status{
		LicenseKey:       maskString(l.LicenseKey, 4),
		Domain:           l.Domain,
		Fingerprint:      maskString(l.Fingerprint, 8),
		DeactivateCount:  l.DeactivateCount,
	}

	switch l.State {
	case StateActive:
		s.State = "active"
		s.Activated = true
		if !l.ExpiresAt.IsZero() {
			s.ExpiresAt = l.ExpiresAt.Format(time.RFC3339)
		}
	case StateDeactivated:
		s.State = "deactivated"
	case StateRevoked:
		s.State = "revoked"
	default:
		s.State = "unactivated"
	}

	if !l.LastHeartbeat.IsZero() {
		s.LastHeartbeat = l.LastHeartbeat.Format(time.RFC3339)
	}

	return s
}

// maskString 遮罩字符串中间部分
func maskString(s string, visible int) string {
	if len(s) <= visible*2 {
		return s
	}
	return s[:visible] + "..." + s[len(s)-visible:]
}
