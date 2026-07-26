package middleware

import (
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// trustedProxyNets 是面板允许的回源代理网段。仅有这些来源发来的
// X-Forwarded-For / X-Real-IP 才会被信任。默认仅信任本地回环,
// 与 install.sh 中 Caddy 反代到 127.0.0.1:18888 的部署模型一致。
var trustedProxyNets = func() []*net.IPNet {
	cidrs := []string{"127.0.0.0/8", "::1/128", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	if extra := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES")); extra != "" {
		cidrs = strings.Split(extra, ",")
	}
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(strings.TrimSpace(c))
		if err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}()

// isTrustedProxy 判断直连对端是否属于可信反代网段。
func isTrustedProxy(ip net.IP) bool {
	if ip == nil {
		return false
	}
	for _, n := range trustedProxyNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// TrustedClientIP 提取经过可信反代后的真实客户端 IP。
// 与 gin.TrustedProxies 行为一致,但避免各 handler 重复实现导致绕过。
// 不可信对端的 XFF/X-Real-IP 一律忽略,直接返回 RemoteAddr。
func TrustedClientIP(c *gin.Context) string {
	remoteIP := c.ClientIP()
	if !isTrustedProxy(net.ParseIP(remoteIP)) {
		return remoteIP
	}
	if ip := strings.TrimSpace(c.GetHeader("X-Real-IP")); net.ParseIP(ip) != nil {
		return ip
	}
	if fwd := strings.TrimSpace(c.GetHeader("X-Forwarded-For")); fwd != "" {
		first := strings.TrimSpace(strings.Split(fwd, ",")[0])
		if net.ParseIP(first) != nil {
			return first
		}
	}
	return remoteIP
}

// loginAttemptStore 登录失败计数器。按 (username + ip) 维度记录窗口内失败次数,
// 超过阈值后返回 false 拒绝尝试。同时按 ip 维度做总量上限,防止爆破小字典。
type loginAttemptStore struct {
	mu   sync.Mutex
	byKey map[string][]time.Time
	byIP  map[string][]time.Time
}

var loginAttempts = &loginAttemptStore{
	byKey: make(map[string][]time.Time),
	byIP:  make(map[string][]time.Time),
}

const (
	loginFailWindow        = 5 * time.Minute
	loginFailKeyThreshold  = 5
	loginFailIPThreshold   = 20
	loginFailIPMaxEntries  = 4096
)

// AllowLoginAttempt 判定是否能继续尝试登录。返回 true 表示允许,
// 返回 false 时应在 handler 中直接以 429 拒绝,且**不要计入失败**。
func AllowLoginAttempt(username, ip string) bool {
	now := time.Now()
	cutoff := now.Add(-loginFailWindow)
	loginAttempts.mu.Lock()
	defer loginAttempts.mu.Unlock()

	key := strings.ToLower(username)
	if !pruneAndCheck(loginAttempts.byKey, key, cutoff, loginFailKeyThreshold) {
		return false
	}
	if !pruneAndCheck(loginAttempts.byIP, ip, cutoff, loginFailIPThreshold) {
		return false
	}
	return true
}

// RecordLoginFailure 在登录失败后调用,更新两个维度的失败计数。
func RecordLoginFailure(username, ip string) {
	now := time.Now()
	loginAttempts.mu.Lock()
	defer loginAttempts.mu.Unlock()
	key := strings.ToLower(username)
	loginAttempts.byKey[key] = append(loginAttempts.byKey[key], now)
	loginAttempts.byIP[ip] = append(loginAttempts.byIP[ip], now)
	if len(loginAttempts.byIP) > loginFailIPMaxEntries {
		evictExpired(loginAttempts.byIP, now.Add(-loginFailWindow))
		evictExpired(loginAttempts.byKey, now.Add(-loginFailWindow))
	}
}

// RecordLoginSuccess 清空该用户在所有 IP 上的失败计数,避免历史失败累计误伤。
func RecordLoginSuccess(username string) {
	loginAttempts.mu.Lock()
	defer loginAttempts.mu.Unlock()
	delete(loginAttempts.byKey, strings.ToLower(username))
}

func pruneAndCheck(store map[string][]time.Time, key string, cutoff time.Time, threshold int) bool {
	previous := store[key]
	recent := make([]time.Time, 0, len(previous))
	for _, t := range previous {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	if len(recent) >= threshold {
		store[key] = recent
		return false
	}
	store[key] = recent
	return true
}

func evictExpired(store map[string][]time.Time, cutoff time.Time) {
	for k, v := range store {
		if len(v) == 0 || v[len(v)-1].Before(cutoff) {
			delete(store, k)
		}
	}
}
