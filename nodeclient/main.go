package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

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
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
	NodeID    int    `json:"node_id"`
	DeviceID  int    `json:"device_id"`
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
	if len(os.Args) < 4 {
		fmt.Println("用法: nodeclient --server <面板地址> --token <节点令牌> --device <设备组ID>")
		fmt.Println("示例: nodeclient --server https://your.domain --token xxxxx --device 1")
		os.Exit(1)
	}

	for i := 1; i < len(os.Args); i++ {
		if i+1 >= len(os.Args) {
			fmt.Printf("参数 %s 缺少值\n", os.Args[i])
			os.Exit(1)
		}
		switch os.Args[i] {
		case "--server":
			config.ServerURL = os.Args[i+1]
			i++
		case "--token":
			config.Token = os.Args[i+1]
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

	if config.ServerURL == "" || config.Token == "" || config.DeviceID == 0 {
		fmt.Println("参数不完整")
		os.Exit(1)
	}

	fmt.Printf("节点客户端启动，面板: %s, 设备组: %d\n", config.ServerURL, config.DeviceID)

	go heartbeatLoop()
	go pollRulesLoop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("\n正在关闭代理...")
	stopAllProxies()
}

func heartbeatLoop() {
	for {
		ip := getOutboundIP()
		payload := map[string]interface{}{
			"token": config.Token,
			"ip":    ip,
		}
		data, _ := json.Marshal(payload)
		httpPost(config.ServerURL+"/api/node/heartbeat", data)
		time.Sleep(30 * time.Second)
	}
}

func pollRulesLoop() {
	for {
		rules, status, err := fetchRules()
		if err == nil {
			applyRules(rules)
		} else {
			fmt.Printf("获取规则失败: %v\n", err)
			if status == http.StatusUnauthorized || status == http.StatusForbidden {
				stopAllProxies()
			}
		}
		time.Sleep(20 * time.Second)
	}
}

func fetchRules() ([]ForwardRule, int, error) {
	payload := map[string]interface{}{
		"token":     config.Token,
		"device_id": config.DeviceID,
	}
	data, _ := json.Marshal(payload)
	resp, status, err := httpPostWithBody(config.ServerURL+"/api/node/rules", data)
	if err != nil {
		return nil, status, err
	}
	var result struct {
		Rules []ForwardRule `json:"rules"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, status, err
	}
	return result.Rules, status, nil
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

func httpPost(url string, data []byte) {
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("心跳失败: HTTP %d\n", resp.StatusCode)
	}
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
