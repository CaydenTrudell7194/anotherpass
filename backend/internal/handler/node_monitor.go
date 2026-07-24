package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sort"
	"sync"
	"time"

	"forward-panel/internal/model"

	"github.com/gin-gonic/gin"
)

type monitorCacheEntry struct {
	Node      model.Node
	Metrics   nodeMetrics
	UpdatedAt time.Time
}

var nodeMonitorCache = struct {
	sync.RWMutex
	items map[uint]monitorCacheEntry
}{items: make(map[uint]monitorCacheEntry)}

type monitorTicket struct {
	GroupIDs  map[uint]bool
	ExpiresAt time.Time
}

var monitorTickets = struct {
	sync.Mutex
	items map[string]monitorTicket
}{items: make(map[string]monitorTicket)}

func updateNodeMonitor(node model.Node, metrics *nodeMetrics) {
	if metrics == nil {
		return
	}
	nodeMonitorCache.Lock()
	nodeMonitorCache.items[node.ID] = monitorCacheEntry{Node: node, Metrics: *metrics, UpdatedAt: time.Now()}
	nodeMonitorCache.Unlock()
}

func CreateNodeMonitorTicket(c *gin.Context) {
	allowed := make(map[uint]bool)
	var groups []model.DeviceGroup
	if err := model.DB.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	isAdmin := c.GetBool("is_admin")
	for _, group := range groups {
		if isAdmin || authorizeDeviceGroup(c.GetUint("user_id"), group.ID, false) == nil {
			allowed[group.ID] = true
		}
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成连接票据失败"})
		return
	}
	token := hex.EncodeToString(b)
	monitorTickets.Lock()
	now := time.Now()
	for key, ticket := range monitorTickets.items {
		if ticket.ExpiresAt.Before(now) {
			delete(monitorTickets.items, key)
		}
	}
	monitorTickets.items[token] = monitorTicket{GroupIDs: allowed, ExpiresAt: now.Add(30 * time.Second)}
	monitorTickets.Unlock()
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"ticket": token})
}

func NodeMonitorWebSocket(c *gin.Context) {
	token := c.Query("ticket")
	monitorTickets.Lock()
	ticket, ok := monitorTickets.items[token]
	delete(monitorTickets.items, token)
	monitorTickets.Unlock()
	if !ok || ticket.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "连接票据无效"})
		return
	}
	conn, err := nodeUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	groups, nodes := loadMonitorScope(ticket.GroupIDs)
	ticks := 0
	for {
		if ticks > 0 && ticks%10 == 0 {
			groups, nodes = loadMonitorScope(ticket.GroupIDs)
		}
		if err := conn.WriteJSON(buildMonitorSnapshot(groups, nodes)); err != nil {
			return
		}
		ticks++
		<-ticker.C
	}
}

type monitorNodeView struct {
	ID            uint        `json:"id"`
	DeviceGroupID uint        `json:"device_group_id"`
	Name          string      `json:"name"`
	IP            string      `json:"ip"`
	Online        bool        `json:"online"`
	LastHeartbeat time.Time   `json:"last_heartbeat"`
	LastUpdate    time.Time   `json:"last_update"`
	Metrics       nodeMetrics `json:"metrics"`
}
type monitorGroupView struct {
	ID    uint              `json:"id"`
	Name  string            `json:"name"`
	Nodes []monitorNodeView `json:"nodes"`
}

func loadMonitorScope(allowed map[uint]bool) ([]model.DeviceGroup, []model.Node) {
	var groups []model.DeviceGroup
	var nodes []model.Node
	if len(allowed) > 0 {
		ids := mapKeys(allowed)
		model.DB.Where("id IN ?", ids).Order("sort_order asc, id asc").Find(&groups)
		model.DB.Where("device_group_id IN ?", ids).Find(&nodes)
	}
	return groups, nodes
}

func buildMonitorSnapshot(groups []model.DeviceGroup, nodes []model.Node) gin.H {
	nodeMonitorCache.RLock()
	defer nodeMonitorCache.RUnlock()
	byGroup := make(map[uint][]monitorNodeView)
	now := time.Now()
	for _, node := range nodes {
		entry := nodeMonitorCache.items[node.ID]
		lastUpdate := entry.UpdatedAt
		online := node.Status == "online" && (!lastUpdate.IsZero() && now.Sub(lastUpdate) < 5*time.Second || lastUpdate.IsZero() && now.Sub(node.LastHeartbeat) < 20*time.Second)
		byGroup[node.DeviceGroupID] = append(byGroup[node.DeviceGroupID], monitorNodeView{ID: node.ID, DeviceGroupID: node.DeviceGroupID, Name: node.Name, IP: node.IP, Online: online, LastHeartbeat: node.LastHeartbeat, LastUpdate: lastUpdate, Metrics: entry.Metrics})
	}
	result := make([]monitorGroupView, 0, len(groups))
	for _, group := range groups {
		rows := byGroup[group.ID]
		sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
		result = append(result, monitorGroupView{ID: group.ID, Name: group.Name, Nodes: rows})
	}
	return gin.H{"server_time": now.Unix(), "groups": result}
}

func mapKeys(values map[uint]bool) []uint {
	keys := make([]uint, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
