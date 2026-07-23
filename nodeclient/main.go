package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type ForwardRule struct {
	ID            int     `json:"id"`
	Name          string  `json:"name"`
	DeviceGroupID int     `json:"device_group_id"`
	ListenPort    int     `json:"listen_port"`
	TargetAddr    string  `json:"target_addr"`
	TargetPort    int     `json:"target_port"`
	Protocol      string  `json:"protocol"`
	Enabled       bool    `json:"enabled"`
	Traffic       int64   `json:"traffic"`
}

type Config struct {
	ServerURL string `json:"server_url"`
	Token     string `json:"token"`
	NodeID    int    `json:"node_id"`
	DeviceID  int    `json:"device_id"`
}

var (
	config     Config
	proxies    = make(map[int]*ProxyServer)
	mu         sync.Mutex
	trafficData = make(map[int]int64)
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("用法: nodeclient --server <面板地址> --token <节点令牌> --device <设备组ID>")
		fmt.Println("示例: nodeclient --server https://your.domain --token xxxxx --device 1")
		os.Exit(1)
	}

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--server":
			config.ServerURL = os.Args[i+1]
		case "--token":
			config.Token = os.Args[i+1]
		case "--device":
			config.DeviceID, _ = strconv.Atoi(os.Args[i+1])
		}
	}

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
		rules := fetchRules()
		if rules != nil {
			applyRules(rules)
		}
		time.Sleep(20 * time.Second)
	}
}

func fetchRules() []ForwardRule {
	payload := map[string]interface{}{
		"token":     config.Token,
		"device_id": config.DeviceID,
	}
	data, _ := json.Marshal(payload)
	resp := httpPostWithBody(config.ServerURL+"/api/node/rules", data)
	if resp == nil {
		return nil
	}
	var result struct {
		Rules []ForwardRule `json:"rules"`
	}
	json.Unmarshal(resp, &result)
	return result.Rules
}

func applyRules(rules []ForwardRule) {
	mu.Lock()
	defer mu.Unlock()

	active := make(map[int]bool)
	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		active[rule.ListenPort] = true
		if _, ok := proxies[rule.ListenPort]; !ok {
			proxy := NewProxyServer(rule)
			go proxy.Start()
			proxies[rule.ListenPort] = proxy
			fmt.Printf("启动代理: :%d -> %s:%d\n", rule.ListenPort, rule.TargetAddr, rule.TargetPort)
		}
	}

	for port, proxy := range proxies {
		if !active[port] {
			proxy.Stop()
			delete(proxies, port)
			fmt.Printf("停止代理: :%d\n", port)
		}
	}
}

func stopAllProxies() {
	mu.Lock()
	defer mu.Unlock()
	for _, proxy := range proxies {
		proxy.Stop()
	}
}

type ProxyServer struct {
	rule     ForwardRule
	listener net.Listener
	stopped  bool
}

func NewProxyServer(rule ForwardRule) *ProxyServer {
	return &ProxyServer{rule: rule}
}

func (p *ProxyServer) Start() {
	var err error
	p.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", p.rule.ListenPort))
	if err != nil {
		fmt.Printf("监听端口 %d 失败: %v\n", p.rule.ListenPort, err)
		return
	}
	for {
		conn, err := p.listener.Accept()
		if err != nil {
			if !p.stopped {
				fmt.Printf("接受连接错误: %v\n", err)
			}
			return
		}
		go p.handleConnection(conn)
	}
}

func (p *ProxyServer) Stop() {
	p.stopped = true
	if p.listener != nil {
		p.listener.Close()
	}
}

func (p *ProxyServer) handleConnection(src net.Conn) {
	defer src.Close()
	dst, err := net.DialTimeout("tcp", net.JoinHostPort(p.rule.TargetAddr, fmt.Sprintf("%d", p.rule.TargetPort)), 10*time.Second)
	if err != nil {
		fmt.Printf("连接目标 %s:%d 失败: %v\n", p.rule.TargetAddr, p.rule.TargetPort, err)
		return
	}
	defer dst.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		n, _ := io.Copy(dst, src)
		mu.Lock()
		trafficData[p.rule.ID] += n
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		n, _ := io.Copy(src, dst)
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
	return strings.Split(conn.LocalAddr().String(), ":")[0]
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
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func httpPostWithBody(url string, data []byte) []byte {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body
}
