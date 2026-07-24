package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

const clientVersion = "v1.5.0"

type MonitorMetrics struct {
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
	ProcessCount    uint64  `json:"process_count"`
	MemTotal        uint64  `json:"mem_total"`
	MemUsed         uint64  `json:"mem_used"`
	SwapTotal       uint64  `json:"swap_total"`
	SwapUsed        uint64  `json:"swap_used"`
	DiskTotal       uint64  `json:"disk_total"`
	DiskUsed        uint64  `json:"disk_used"`
	NetInSpeed      uint64  `json:"net_in_speed"`
	NetOutSpeed     uint64  `json:"net_out_speed"`
	NetInTransfer   uint64  `json:"net_in_transfer"`
	NetOutTransfer  uint64  `json:"net_out_transfer"`
	TCPConnCount    uint64  `json:"tcp_conn_count"`
	UDPConnCount    uint64  `json:"udp_conn_count"`
	UptimeSeconds   uint64  `json:"uptime_seconds"`
	BootTime        uint64  `json:"boot_time"`
}

type ForwardRule struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	DeviceGroupID int    `json:"device_group_id"`
	ListenPort    int    `json:"listen_port"`
	TargetAddr    string `json:"target_addr"`
	TargetPort    int    `json:"target_port"`
	Protocol      string `json:"protocol"`
	Enabled       bool   `json:"enabled"`
	Traffic       int64  `json:"traffic"`
}

type Config struct {
	ServerURL  string `json:"server_url"`
	Token      string `json:"token,omitempty"`
	GroupToken string `json:"group_token,omitempty"`
	InstanceID string `json:"instance_id,omitempty"`
	Name       string `json:"name,omitempty"`
	NodeID     int    `json:"node_id,omitempty"`
	DeviceID   int    `json:"device_id,omitempty"`
}

var (
	config      Config
	proxies     = make(map[int]*ProxyServer)
	mu          sync.Mutex
	trafficData = make(map[int]int64)
	httpClient  = &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > 0 && (req.URL.Host != via[0].URL.Host || req.URL.Scheme != via[0].URL.Scheme) {
				return errors.New("拒绝跨来源重定向")
			}
			return nil
		},
	}
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: nodeclient --config <配置文件> 或 --server <面板地址> --group-token <设备组Token> --output-config <配置文件>")
		os.Exit(1)
	}

	groupToken := ""
	outputConfig := ""
	for i := 1; i < len(os.Args); i++ {
		if i+1 >= len(os.Args) {
			fmt.Printf("参数 %s 缺少值\n", os.Args[i])
			os.Exit(1)
		}
		switch os.Args[i] {
		case "--config":
			if err := loadConfig(os.Args[i+1]); err != nil {
				fmt.Printf("读取配置失败: %v\n", err)
				os.Exit(1)
			}
			i++
		case "--server":
			config.ServerURL = os.Args[i+1]
			i++
		case "--token":
			config.Token = os.Args[i+1]
			i++
		case "--group-token":
			groupToken = os.Args[i+1]
			i++
		case "--output-config":
			outputConfig = os.Args[i+1]
			i++
		case "--device":
			var err error
			config.DeviceID, err = strconv.Atoi(os.Args[i+1])
			if err != nil || config.DeviceID <= 0 {
				fmt.Println("设备组ID无效")
				os.Exit(1)
			}
			i++
		default:
			fmt.Printf("未知参数: %s\n", os.Args[i])
			os.Exit(1)
		}
	}
	parsedURL, err := url.Parse(config.ServerURL)
	if err != nil || (parsedURL.Scheme != "https" && parsedURL.Scheme != "http") || parsedURL.Host == "" {
		fmt.Println("面板地址无效")
		os.Exit(1)
	}
	config.ServerURL = strings.TrimRight(config.ServerURL, "/")
	if groupToken != "" {
		if outputConfig == "" {
			fmt.Println("设备组Token模式需要 --output-config")
			os.Exit(1)
		}
		if err := writeGroupConfig(groupToken, outputConfig); err != nil {
			fmt.Printf("写入节点配置失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("节点凭据已写入安全配置文件")
		return
	}

	if config.ServerURL == "" || (config.Token == "" && config.GroupToken == "") {
		fmt.Println("参数不完整")
		os.Exit(1)
	}

	fmt.Printf("节点客户端启动，面板: %s\n", config.ServerURL)

	go websocketControlLoop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("\n正在关闭代理...")
	stopAllProxies()
}

func writeGroupConfig(groupToken, outputPath string) error {
	instanceID := ""
	if existing, err := os.ReadFile(outputPath); err == nil {
		var saved Config
		if json.Unmarshal(existing, &saved) == nil {
			instanceID = saved.InstanceID
		}
	}
	if instanceID == "" {
		b := make([]byte, 16)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		instanceID = hex.EncodeToString(b)
	}
	name, _ := os.Hostname()
	data, _ := json.Marshal(Config{ServerURL: config.ServerURL, GroupToken: groupToken, InstanceID: instanceID, Name: name})
	if err := os.WriteFile(outputPath, append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Chmod(outputPath, 0600)
}

func loadConfig(path string) error {
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &config)
}

type controlMessage struct {
	Type             string          `json:"type"`
	IP               string          `json:"ip,omitempty"`
	NodeID           int             `json:"node_id,omitempty"`
	DeviceGroupID    int             `json:"device_group_id,omitempty"`
	HeartbeatSeconds int             `json:"heartbeat_seconds,omitempty"`
	Rules            []ForwardRule   `json:"rules,omitempty"`
	Metrics          *MonitorMetrics `json:"metrics,omitempty"`
}

func websocketControlLoop() {
	backoff := time.Second
	for {
		status, connected, err := runWebSocketSession()
		if err != nil {
			fmt.Printf("WebSocket 连接断开: %v\n", err)
		}
		if status == http.StatusUnauthorized || status == http.StatusForbidden {
			stopAllProxies()
		}
		if connected {
			stopAllProxies()
			backoff = time.Second
		}
		time.Sleep(backoff)
		if backoff < 30*time.Second {
			backoff *= 2
		}
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
	}
}

func runWebSocketSession() (int, bool, error) {
	base, _ := url.Parse(config.ServerURL)
	if base.Scheme == "https" {
		base.Scheme = "wss"
	} else {
		base.Scheme = "ws"
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/api/node/ws"
	header := http.Header{}
	token := config.GroupToken
	if token == "" {
		token = config.Token
	}
	header.Set("Authorization", "Bearer "+token)
	if config.InstanceID != "" {
		header.Set("X-Node-Instance", config.InstanceID)
	}
	if config.Name != "" {
		header.Set("X-Node-Name", config.Name)
	}
	conn, resp, err := websocket.DefaultDialer.Dial(base.String(), header)
	if err != nil {
		if resp != nil {
			return resp.StatusCode, false, err
		}
		return 0, false, err
	}
	defer conn.Close()
	backoffMessage := make(chan controlMessage, 8)
	errCh := make(chan error, 1)
	go func() {
		for {
			var msg controlMessage
			if err := conn.ReadJSON(&msg); err != nil {
				errCh <- err
				return
			}
			select {
			case backoffMessage <- msg:
			default:
				errCh <- errors.New("control message backlog")
				return
			}
		}
	}()
	heartbeatSeconds := 15
	ticker := time.NewTicker(time.Duration(heartbeatSeconds) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case err := <-errCh:
			return 0, true, err
		case msg := <-backoffMessage:
			switch msg.Type {
			case "hello":
				config.NodeID = msg.NodeID
				config.DeviceID = msg.DeviceGroupID
				if msg.HeartbeatSeconds > 0 && msg.HeartbeatSeconds != heartbeatSeconds {
					heartbeatSeconds = msg.HeartbeatSeconds
					ticker.Reset(time.Duration(heartbeatSeconds) * time.Second)
				}
				fmt.Printf("WebSocket 已连接，节点ID: %d, 设备组: %d\n", config.NodeID, config.DeviceID)
			case "rules_snapshot":
				applyRules(msg.Rules)
			case "revoked":
				stopAllProxies()
				return http.StatusForbidden, true, errors.New("节点凭据已撤销")
			}
		case <-ticker.C:
			metrics := collectMonitorMetrics()
			if err := conn.WriteJSON(controlMessage{Type: "heartbeat", IP: getOutboundIP(), Metrics: &metrics}); err != nil {
				return 0, true, err
			}
		}
	}
}

func applyRules(rules []ForwardRule) {
	mu.Lock()
	defer mu.Unlock()

	active := make(map[int]ForwardRule)
	for _, rule := range rules {
		if !rule.Enabled || rule.Protocol != "tcp" || rule.ListenPort < 1 || rule.ListenPort > 65535 || rule.TargetPort < 1 || rule.TargetPort > 65535 {
			continue
		}
		if _, duplicate := active[rule.ListenPort]; duplicate {
			fmt.Printf("监听端口 %d 存在重复规则，跳过冲突规则 %d\n", rule.ListenPort, rule.ID)
			continue
		}
		active[rule.ListenPort] = rule
		existing := proxies[rule.ListenPort]
		if existing != nil && existing.rule.ListenPort == rule.ListenPort && existing.rule.TargetAddr == rule.TargetAddr &&
			existing.rule.TargetPort == rule.TargetPort && existing.rule.Protocol == rule.Protocol {
			continue
		}
		if existing != nil {
			existing.Stop()
			delete(proxies, rule.ListenPort)
		}
		proxy := NewProxyServer(rule)
		if err := proxy.Start(); err != nil {
			fmt.Printf("监听端口 %d 失败: %v\n", rule.ListenPort, err)
			continue
		}
		proxies[rule.ListenPort] = proxy
		fmt.Printf("启动代理: :%d -> %s:%d\n", rule.ListenPort, rule.TargetAddr, rule.TargetPort)
	}

	for port, proxy := range proxies {
		if _, ok := active[port]; !ok {
			proxy.Stop()
			delete(proxies, port)
			fmt.Printf("停止代理: :%d\n", port)
		}
	}
}

func stopAllProxies() {
	mu.Lock()
	defer mu.Unlock()
	for port, proxy := range proxies {
		proxy.Stop()
		delete(proxies, port)
	}
}

type ProxyServer struct {
	rule        ForwardRule
	listener    net.Listener
	connections map[net.Conn]struct{}
	connLimit   chan struct{}
	mu          sync.Mutex
	stopped     bool
}

func NewProxyServer(rule ForwardRule) *ProxyServer {
	return &ProxyServer{rule: rule, connections: make(map[net.Conn]struct{}), connLimit: make(chan struct{}, 256)}
}

func (p *ProxyServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", p.rule.ListenPort))
	if err != nil {
		return err
	}
	p.mu.Lock()
	p.listener = listener
	p.mu.Unlock()
	go p.acceptLoop()
	return nil
}

func (p *ProxyServer) acceptLoop() {
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			p.mu.Lock()
			stopped := p.stopped
			p.mu.Unlock()
			if !stopped {
				fmt.Printf("接受连接错误: %v\n", err)
			}
			return
		}
		select {
		case p.connLimit <- struct{}{}:
		default:
			conn.Close()
			continue
		}
		go p.handleConnection(conn)
	}
}

func (p *ProxyServer) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopped = true
	if p.listener != nil {
		p.listener.Close()
	}
	for conn := range p.connections {
		conn.Close()
	}
	p.connections = make(map[net.Conn]struct{})
}

func (p *ProxyServer) handleConnection(src net.Conn) {
	defer func() { <-p.connLimit }()
	defer src.Close()
	dst, err := net.DialTimeout("tcp", net.JoinHostPort(p.rule.TargetAddr, fmt.Sprintf("%d", p.rule.TargetPort)), 10*time.Second)
	if err != nil {
		fmt.Printf("连接目标 %s:%d 失败: %v\n", p.rule.TargetAddr, p.rule.TargetPort, err)
		return
	}
	defer dst.Close()
	p.mu.Lock()
	if p.stopped {
		p.mu.Unlock()
		return
	}
	p.connections[src] = struct{}{}
	p.connections[dst] = struct{}{}
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.connections, src)
		delete(p.connections, dst)
		p.mu.Unlock()
	}()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		n, _ := io.Copy(dst, src)
		if tcp, ok := dst.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
		mu.Lock()
		trafficData[p.rule.ID] += n
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		n, _ := io.Copy(src, dst)
		if tcp, ok := src.(*net.TCPConn); ok {
			tcp.CloseWrite()
		}
		mu.Lock()
		trafficData[p.rule.ID] += n
		mu.Unlock()
	}()
	wg.Wait()
}

func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "0.0.0.0"
	}
	defer conn.Close()
	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return ""
	}
	return host
}

func httpGet(url string) []byte {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body
}

func httpPostWithBody(url string, data []byte) ([]byte, int, error) {
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return body, resp.StatusCode, nil
}
