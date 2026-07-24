package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net"
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

var userNodeMonitorCache = struct {
	sync.RWMutex
	items map[uint]monitorCacheEntry
}{items: make(map[uint]monitorCacheEntry)}

type monitorTicket struct {
	GroupIDs  map[uint]bool
	UserID    uint
	IsAdmin   bool
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

func updateUserNodeMonitor(node *model.UserNode, metrics *nodeMetrics) {
	if metrics == nil {
		return
	}
	userNodeMonitorCache.Lock()
	userNodeMonitorCache.items[node.ID] = monitorCacheEntry{Node: model.Node{ID: node.ID, Name: node.Name, IP: node.IP, DeviceGroupID: 0, Status: node.Status, LastHeartbeat: node.LastHeartbeat}, Metrics: *metrics, UpdatedAt: time.Now()}
	userNodeMonitorCache.Unlock()
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
	monitorTickets.items[token] = monitorTicket{GroupIDs: allowed, UserID: c.GetUint("user_id"), IsAdmin: isAdmin, ExpiresAt: now.Add(30 * time.Second)}
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
	groups, nodes, userNodes := loadMonitorScope(ticket.GroupIDs, ticket.UserID, ticket.IsAdmin)
	ticks := 0
	for {
		if ticks > 0 && ticks%10 == 0 {
			groups, nodes, userNodes = loadMonitorScope(ticket.GroupIDs, ticket.UserID, ticket.IsAdmin)
		}
		if err := conn.WriteJSON(buildMonitorSnapshot(groups, nodes, userNodes, ticket.IsAdmin)); err != nil {
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
	IP4Geo        string      `json:"ip4_geo"`
	IP6Geo        string      `json:"ip6_geo"`
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

func loadMonitorScope(allowed map[uint]bool, userID uint, isAdmin bool) ([]model.DeviceGroup, []model.Node, []model.UserNode) {
	var groups []model.DeviceGroup
	var nodes []model.Node
	var userNodes []model.UserNode
	if len(allowed) > 0 {
		ids := mapKeys(allowed)
		model.DB.Where("id IN ?", ids).Order("sort_order asc, id asc").Find(&groups)
		model.DB.Where("device_group_id IN ?", ids).Find(&nodes)
	}
	if isAdmin {
		model.DB.Order("id asc").Find(&userNodes)
	}
	return groups, nodes, userNodes
}

func buildMonitorSnapshot(groups []model.DeviceGroup, nodes []model.Node, userNodes []model.UserNode, isAdmin bool) gin.H {
	nodeMonitorCache.RLock()
	defer nodeMonitorCache.RUnlock()
	userNodeMonitorCache.RLock()
	defer userNodeMonitorCache.RUnlock()
	byGroup := make(map[uint][]monitorNodeView)
	now := time.Now()
	seenIPs := make(map[string]bool)
	for _, node := range nodes {
		entry := nodeMonitorCache.items[node.ID]
		lastUpdate := entry.UpdatedAt
		online := node.Status == "online" && (!lastUpdate.IsZero() && now.Sub(lastUpdate) < 5*time.Second || lastUpdate.IsZero() && now.Sub(node.LastHeartbeat) < 20*time.Second)
		ip4Geo, ip6Geo := lookupNodeGeo(node.IP, seenIPs)
		view := monitorNodeView{
			ID: node.ID, DeviceGroupID: node.DeviceGroupID, Name: node.Name, IP: node.IP,
			IP4Geo: ip4Geo, IP6Geo: ip6Geo, Online: online,
			LastHeartbeat: node.LastHeartbeat, LastUpdate: lastUpdate, Metrics: entry.Metrics,
		}
		if !isAdmin {
			view.IP = ""
			view.Metrics.Hostname = ""
			view.Metrics.Platform = ""
			view.Metrics.PlatformVersion = ""
			view.Metrics.Arch = ""
			view.Metrics.Version = ""
			view.Metrics.CPUModel = ""
		}
		byGroup[node.DeviceGroupID] = append(byGroup[node.DeviceGroupID], view)
	}
	result := make([]monitorGroupView, 0, len(groups))
	for _, group := range groups {
		rows := byGroup[group.ID]
		sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
		result = append(result, monitorGroupView{ID: group.ID, Name: group.Name, Nodes: rows})
	}
	var userNodeRows []monitorNodeView
	for _, un := range userNodes {
		entry := userNodeMonitorCache.items[un.ID]
		lastUpdate := entry.UpdatedAt
		online := un.Status == "online" && (!lastUpdate.IsZero() && now.Sub(lastUpdate) < 5*time.Second || lastUpdate.IsZero() && now.Sub(un.LastHeartbeat) < 20*time.Second)
		ip4Geo, ip6Geo := lookupNodeGeo(un.IP, seenIPs)
		uview := monitorNodeView{
			ID: un.ID, Name: un.Name, IP: un.IP,
			IP4Geo: ip4Geo, IP6Geo: ip6Geo, Online: online,
			LastHeartbeat: un.LastHeartbeat, LastUpdate: lastUpdate, Metrics: entry.Metrics,
		}
		if !isAdmin {
			uview.IP = ""
			uview.Metrics.Hostname = ""
			uview.Metrics.Platform = ""
			uview.Metrics.PlatformVersion = ""
			uview.Metrics.Arch = ""
			uview.Metrics.Version = ""
			uview.Metrics.CPUModel = ""
		}
		userNodeRows = append(userNodeRows, uview)
	}
	if len(userNodeRows) > 0 {
		sort.Slice(userNodeRows, func(i, j int) bool { return userNodeRows[i].ID < userNodeRows[j].ID })
		result = append(result, monitorGroupView{ID: 0, Name: "用户节点", Nodes: userNodeRows})
	}
	return gin.H{"server_time": now.Unix(), "groups": result}
}

func lookupNodeGeo(ip string, seenIPs map[string]bool) (ip4Geo, ip6Geo string) {
	if ip == "" || seenIPs[ip] {
		return "", ""
	}
	seenIPs[ip] = true
	geoCache.RLock()
	entry, ok := geoCache.items[ip]
	geoCache.RUnlock()
	if !ok {
		go resolveGeo(ip)
		return "", ""
	}
	if net.ParseIP(ip).To4() != nil {
		return entry.Result.CountryCode, ""
	}
	return "", entry.Result.CountryCode
}

func mapKeys(values map[uint]bool) []uint {
	keys := make([]uint, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
