package handler

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"forward-panel/internal/middleware"
	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type nodeControlMessage struct {
	Type             string              `json:"type"`
	IP               string              `json:"ip,omitempty"`
	IP4              string              `json:"ip4,omitempty"`
	IP6              string              `json:"ip6,omitempty"`
	NodeID           uint                `json:"node_id,omitempty"`
	DeviceGroupID    uint                `json:"device_group_id,omitempty"`
	HeartbeatSeconds int                 `json:"heartbeat_seconds,omitempty"`
	Rules            []model.ForwardRule `json:"rules,omitempty"`
	Metrics          *nodeMetrics        `json:"metrics,omitempty"`
}

type nodeMetrics struct {
	Hostname        string  `json:"hostname"`
	Platform        string  `json:"platform"`
	PlatformVersion string  `json:"platform_version"`
	Arch            string  `json:"arch"`
	Version         string  `json:"version"`
	CPUModel        string  `json:"cpu_model"`
	CPUPercent      float64 `json:"cpu_percent"`
	Load1           float64 `json:"load1"`
	Load5           float64 `json:"load5"`
	Load15          float64 `json:"load15"`
	ProcessCount    int     `json:"process_count"`
	MemTotal        int64   `json:"mem_total"`
	MemUsed         int64   `json:"mem_used"`
	SwapTotal       int64   `json:"swap_total"`
	SwapUsed        int64   `json:"swap_used"`
	DiskTotal       int64   `json:"disk_total"`
	DiskUsed        int64   `json:"disk_used"`
	NetInSpeed      int64   `json:"net_in_speed"`
	NetOutSpeed     int64   `json:"net_out_speed"`
	NetInTransfer   int64   `json:"net_in_transfer"`
	NetOutTransfer  int64   `json:"net_out_transfer"`
	TCPConnCount    int     `json:"tcp_conn_count"`
	UDPConnCount    int     `json:"udp_conn_count"`
	UptimeSeconds   int64   `json:"uptime_seconds"`
	BootTime        int64   `json:"boot_time"`
}

var nodeUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     checkWebSocketOrigin,
}

// allowedWSOrigins 返回允许建立 WebSocket 的浏览器 origin 集合。
// 默认从 PUBLIC_URL 推导同源,可以通过 WS_ALLOWED_ORIGINS 环境变量额外追加以逗号分隔的 origin。
// 节点客户端(nodeclient)通常不发送 Origin header,一律放行,避免误伤真实节点流量。
func allowedWSOrigins() map[string]bool {
	origins := make(map[string]bool)
	for _, raw := range strings.Split(os.Getenv("WS_ALLOWED_ORIGINS"), ",") {
		if v := strings.TrimSpace(raw); v != "" {
			origins[v] = true
		}
	}
	if pub := strings.TrimSpace(os.Getenv("PUBLIC_URL")); pub != "" {
		origins[deriveOrigin(pub)] = true
	}
	return origins
}

func deriveOrigin(rawURL string) string {
	if i := strings.Index(rawURL, "://"); i > 0 {
		rest := rawURL[i+3:]
		if j := strings.IndexAny(rest, "/?#"); j >= 0 {
			rest = rest[:j]
		}
		return rawURL[:i+3] + rest
	}
	return rawURL
}

// checkWebSocketOrigin 仅允许同源浏览器与节点客户端(空 Origin)连接。
func checkWebSocketOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true // 节点客户端通常不发送 Origin，直接放行
	}
	// 同源检查：允许同源 Origin，以及配置的额外 Origin
	for allowed := range allowedWSOrigins() {
		if origin == allowed {
			return true
		}
	}
	// 兜底：同源请求也放行（同协议+同域名+同端口）
	if u, err := url.Parse(origin); err == nil {
		if u.Host == r.Host {
			return true
		}
	}
	return false
}

type nodeSession struct {
	conn    *websocket.Conn
	writeMu sync.Mutex
}

func (s *nodeSession) writeJSON(value interface{}) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.conn.WriteJSON(value)
}

var activeNodeSessions = struct {
	sync.Mutex
	items map[uint]*nodeSession
}{items: make(map[uint]*nodeSession)}

func registerNodeSession(nodeID uint, session *nodeSession) bool {
	activeNodeSessions.Lock()
	defer activeNodeSessions.Unlock()
	if activeNodeSessions.items[nodeID] != nil {
		return false
	}
	activeNodeSessions.items[nodeID] = session
	return true
}

func unregisterNodeSession(nodeID, groupID uint, session *nodeSession) {
	activeNodeSessions.Lock()
	if activeNodeSessions.items[nodeID] == session {
		delete(activeNodeSessions.items, nodeID)
		activeNodeSessions.Unlock()
		markNodeOffline(nodeID, groupID)
		return
	}
	activeNodeSessions.Unlock()
}

func currentNodeSession(nodeID uint, session *nodeSession) bool {
	activeNodeSessions.Lock()
	defer activeNodeSessions.Unlock()
	return activeNodeSessions.items[nodeID] == session
}

func revokeNodeSession(nodeID uint) {
	activeNodeSessions.Lock()
	session := activeNodeSessions.items[nodeID]
	delete(activeNodeSessions.items, nodeID)
	activeNodeSessions.Unlock()
	if session != nil {
		_ = session.writeJSON(nodeControlMessage{Type: "revoked"})
		_ = session.conn.Close()
		var node model.Node
		if err := model.DB.Select("device_group_id").First(&node, nodeID).Error; err == nil {
			markNodeOffline(nodeID, node.DeviceGroupID)
		}
	}
}

var activeUserNodeSessions = struct {
	sync.Mutex
	items map[uint]*nodeSession
}{items: make(map[uint]*nodeSession)}

func registerUserNodeSession(nodeID uint, session *nodeSession) bool {
	activeUserNodeSessions.Lock()
	defer activeUserNodeSessions.Unlock()
	if activeUserNodeSessions.items[nodeID] != nil {
		return false
	}
	activeUserNodeSessions.items[nodeID] = session
	return true
}

func unregisterUserNodeSession(nodeID uint, session *nodeSession) {
	activeUserNodeSessions.Lock()
	if activeUserNodeSessions.items[nodeID] == session {
		delete(activeUserNodeSessions.items, nodeID)
		activeUserNodeSessions.Unlock()
		markUserNodeOffline(nodeID)
		return
	}
	activeUserNodeSessions.Unlock()
}

func currentUserNodeSession(nodeID uint, session *nodeSession) bool {
	activeUserNodeSessions.Lock()
	defer activeUserNodeSessions.Unlock()
	return activeUserNodeSessions.items[nodeID] == session
}

func revokeUserNodeSession(nodeID uint) {
	activeUserNodeSessions.Lock()
	session := activeUserNodeSessions.items[nodeID]
	delete(activeUserNodeSessions.items, nodeID)
	activeUserNodeSessions.Unlock()
	if session != nil {
		_ = session.writeJSON(nodeControlMessage{Type: "revoked"})
		_ = session.conn.Close()
		markUserNodeOffline(nodeID)
	}
}

func userNodeWebSocket(c *gin.Context, clientIP string, token string, userNode *model.UserNode) {
	conn, err := nodeUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	session := &nodeSession{conn: conn}
	if !registerUserNodeSession(userNode.ID, session) {
		_ = session.writeJSON(nodeControlMessage{Type: "revoked"})
		return
	}
	defer unregisterUserNodeSession(userNode.ID, session)
	conn.SetReadLimit(64 << 10)
	_ = conn.SetReadDeadline(time.Now().Add(50 * time.Second))

	if err := session.writeJSON(nodeControlMessage{Type: "hello", NodeID: userNode.ID, HeartbeatSeconds: 1}); err != nil {
		return
	}
	updateUserNodeHeartbeat(userNode, clientIP, "", "")

	done := make(chan struct{})
	lastDBHeartbeat := time.Time{}
	go func() {
		defer close(done)
		for {
			var msg nodeControlMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			if msg.Type == "heartbeat" {
				if !currentUserNodeSession(userNode.ID, session) {
					return
				}
				_ = conn.SetReadDeadline(time.Now().Add(50 * time.Second))
				updateUserNodeMonitor(userNode, msg.Metrics)
				if time.Since(lastDBHeartbeat) >= 10*time.Second {
					updateUserNodeHeartbeat(userNode, clientIP, msg.IP4, msg.IP6)
					lastDBHeartbeat = time.Now()
				}
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			var current model.UserNode
			if err := model.DB.First(&current, userNode.ID).Error; err != nil || current.Token != token {
				_ = session.writeJSON(nodeControlMessage{Type: "revoked"})
				return
			}
		}
	}
}

func realClientIP(c *gin.Context) string {
	return middleware.TrustedClientIP(c)
}

func NodeWebSocket(c *gin.Context) {
	clientIP := realClientIP(c)
	token := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "节点认证失败"})
		return
	}
	var userNode model.UserNode
	if err := model.DB.Where("token = ?", token).First(&userNode).Error; err == nil {
		userNodeWebSocket(c, clientIP, token, &userNode)
		return
	}
	var node model.Node
	var group model.DeviceGroup
	groupAuth := model.DB.Where("node_token = ?", token).First(&group).Error == nil
	if groupAuth {
		instanceID := strings.TrimSpace(c.GetHeader("X-Node-Instance"))
		name := strings.TrimSpace(c.GetHeader("X-Node-Name"))
		if instanceID == "" || len(instanceID) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "节点实例ID无效"})
			return
		}
		if name == "" {
			name = "入口节点"
		}
		if len(name) > 128 {
			name = name[:128]
		}
		if err := model.DB.Where("device_group_id = ? AND instance_id = ?", group.ID, instanceID).First(&node).Error; err != nil {
			internalToken, tokenErr := newDeviceGroupToken()
			if tokenErr != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "节点登记失败"})
				return
			}
			node = model.Node{DeviceGroupID: group.ID, InstanceID: instanceID, Name: name, Token: internalToken, Status: "offline"}
			if err := model.DB.Create(&node).Error; err != nil {
				if err := model.DB.Where("device_group_id = ? AND instance_id = ?", group.ID, instanceID).First(&node).Error; err != nil {
					c.JSON(http.StatusConflict, gin.H{"error": "节点登记失败"})
					return
				}
			}
		}
	} else if err := model.DB.Where("token = ?", token).First(&node).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "节点认证失败"})
		return
	}
	if err := ensureDirectEntryGroup(node.DeviceGroupID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "节点设备组不支持入口直出"})
		return
	}
	conn, err := nodeUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	session := &nodeSession{conn: conn}
	if !registerNodeSession(node.ID, session) {
		_ = session.writeJSON(nodeControlMessage{Type: "revoked"})
		return
	}
	defer unregisterNodeSession(node.ID, node.DeviceGroupID, session)
	conn.SetReadLimit(64 << 10)
	_ = conn.SetReadDeadline(time.Now().Add(50 * time.Second))

	if err := session.writeJSON(nodeControlMessage{Type: "hello", NodeID: node.ID, DeviceGroupID: node.DeviceGroupID, HeartbeatSeconds: 1}); err != nil {
		return
	}
	rules, err := loadRulesForGroup(node.DeviceGroupID)
	if err != nil {
		return
	}
	encoded, _ := json.Marshal(rules)
	lastRules := string(encoded)
	lastSent := time.Now()
	if err := session.writeJSON(nodeControlMessage{Type: "rules_snapshot", Rules: rules}); err != nil {
		return
	}
	updateNodeHeartbeat(&node, clientIP, "", "")

	done := make(chan struct{})
	lastDBHeartbeat := time.Time{}
	go func() {
		defer close(done)
		for {
			var msg nodeControlMessage
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			if msg.Type == "heartbeat" {
				if !currentNodeSession(node.ID, session) {
					return
				}
				_ = conn.SetReadDeadline(time.Now().Add(50 * time.Second))
				updateNodeMonitor(node, msg.Metrics)
				if time.Since(lastDBHeartbeat) >= 10*time.Second {
					updateNodeHeartbeat(&node, clientIP, msg.IP4, msg.IP6)
					lastDBHeartbeat = time.Now()
				}
			}
		}
	}()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			var current model.Node
			if err := model.DB.First(&current, node.ID).Error; err != nil {
				_ = session.writeJSON(nodeControlMessage{Type: "revoked"})
				return
			}
			authValid := current.Token == token
			if groupAuth {
				var currentGroup model.DeviceGroup
				authValid = model.DB.First(&currentGroup, node.DeviceGroupID).Error == nil && currentGroup.NodeToken == token
			}
			if !authValid || ensureDirectEntryGroup(current.DeviceGroupID) != nil {
				_ = session.writeJSON(nodeControlMessage{Type: "revoked"})
				return
			}
			rules, err := loadRulesForGroup(current.DeviceGroupID)
			if err != nil {
				return
			}
			encoded, _ = json.Marshal(rules)
			fingerprint := string(encoded)
			if fingerprint != lastRules || time.Since(lastSent) >= 30*time.Second {
				if err := session.writeJSON(nodeControlMessage{Type: "rules_snapshot", Rules: rules}); err != nil {
					return
				}
				lastRules = fingerprint
				lastSent = time.Now()
			}
		}
	}
}

func revokeDeviceGroupSessions(groupID uint) {
	var nodes []model.Node
	if err := model.DB.Where("device_group_id = ?", groupID).Find(&nodes).Error; err != nil {
		return
	}
	for _, node := range nodes {
		revokeNodeSession(node.ID)
	}
}
