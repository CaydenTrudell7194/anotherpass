//go:build linux

package main

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var monitorState struct {
	sync.Mutex
	cpuTotal uint64
	cpuIdle  uint64
	netIn    uint64
	netOut   uint64
	netTime  time.Time
}

func collectMonitorMetrics() MonitorMetrics {
	hostname, _ := os.Hostname()
	metrics := MonitorMetrics{
		Hostname: hostname,
		Platform: "linux",
		Arch:     runtime.GOARCH,
		Version:  clientVersion,
	}

	metrics.Platform, metrics.PlatformVersion = readOSRelease()
	metrics.CPUModel = readCPUModel()
	metrics.Load1, metrics.Load5, metrics.Load15 = readLoadAverage()
	metrics.ProcessCount = countProcesses()
	metrics.MemTotal, metrics.MemUsed, metrics.SwapTotal, metrics.SwapUsed = readMemory()
	metrics.DiskTotal, metrics.DiskUsed = readDisk()
	metrics.TCPConnCount = countSocketEntries("/proc/net/tcp") + countSocketEntries("/proc/net/tcp6")
	metrics.UDPConnCount = countSocketEntries("/proc/net/udp") + countSocketEntries("/proc/net/udp6")
	metrics.UptimeSeconds = readUptime()
	metrics.BootTime = readBootTime()

	cpuTotal, cpuIdle, cpuOK := readCPUStat()
	netIn, netOut, netOK := readNetwork()
	now := time.Now()
	monitorState.Lock()
	if cpuOK && monitorState.cpuTotal > 0 && cpuTotal > monitorState.cpuTotal {
		totalDelta := cpuTotal - monitorState.cpuTotal
		idleDelta := uint64(0)
		if cpuIdle >= monitorState.cpuIdle {
			idleDelta = cpuIdle - monitorState.cpuIdle
		}
		if idleDelta <= totalDelta {
			metrics.CPUPercent = float64(totalDelta-idleDelta) * 100 / float64(totalDelta)
		}
	}
	if cpuOK {
		monitorState.cpuTotal = cpuTotal
		monitorState.cpuIdle = cpuIdle
	}
	if netOK {
		metrics.NetInTransfer = netIn
		metrics.NetOutTransfer = netOut
		if !monitorState.netTime.IsZero() {
			seconds := now.Sub(monitorState.netTime).Seconds()
			if seconds > 0 && netIn >= monitorState.netIn && netOut >= monitorState.netOut {
				metrics.NetInSpeed = uint64(float64(netIn-monitorState.netIn) / seconds)
				metrics.NetOutSpeed = uint64(float64(netOut-monitorState.netOut) / seconds)
			}
		}
		monitorState.netIn = netIn
		monitorState.netOut = netOut
		monitorState.netTime = now
	}
	monitorState.Unlock()

	return metrics
}

func readOSRelease() (string, string) {
	values := make(map[string]string)
	file, err := os.Open("/etc/os-release")
	if err != nil {
		return "linux", ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), "=")
		if ok {
			values[key] = strings.Trim(strings.TrimSpace(value), "\"")
		}
	}
	platform := values["ID"]
	if platform == "" {
		platform = "linux"
	}
	return platform, values["VERSION_ID"]
}

func readCPUModel() string {
	file, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), ":")
		if ok && (strings.TrimSpace(key) == "model name" || strings.TrimSpace(key) == "Hardware" || strings.TrimSpace(key) == "Processor") {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func readCPUStat() (uint64, uint64, bool) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, 0, false
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, false
	}
	var total uint64
	values := make([]uint64, 0, 8)
	for index, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		if index < 8 {
			total += value
		}
		values = append(values, value)
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return total, idle, true
}

func readLoadAverage() (float64, float64, float64) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0
	}
	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)
	return load1, load5, load15
}

func readMemory() (uint64, uint64, uint64, uint64) {
	values := make(map[string]uint64)
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0, 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err == nil {
			values[strings.TrimSuffix(fields[0], ":")] = value * 1024
		}
	}
	memTotal := values["MemTotal"]
	memAvailable := values["MemAvailable"]
	swapTotal := values["SwapTotal"]
	swapFree := values["SwapFree"]
	return memTotal, subtractFloor(memTotal, memAvailable), swapTotal, subtractFloor(swapTotal, swapFree)
}

func readDisk() (uint64, uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0, 0
	}
	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	return total, subtractFloor(total, available)
}

func readNetwork() (uint64, uint64, bool) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0, false
	}
	defer file.Close()
	var netIn, netOut uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		name, counters, ok := strings.Cut(scanner.Text(), ":")
		if !ok || strings.TrimSpace(name) == "lo" {
			continue
		}
		fields := strings.Fields(counters)
		if len(fields) < 9 {
			continue
		}
		received, errIn := strconv.ParseUint(fields[0], 10, 64)
		sent, errOut := strconv.ParseUint(fields[8], 10, 64)
		if errIn == nil && errOut == nil {
			netIn += received
			netOut += sent
		}
	}
	return netIn, netOut, scanner.Err() == nil
}

func countProcesses() uint64 {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	var count uint64
	for _, entry := range entries {
		if entry.IsDir() {
			if _, err := strconv.ParseUint(entry.Name(), 10, 64); err == nil {
				count++
			}
		}
	}
	return count
}

func countSocketEntries(path string) uint64 {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	var lines uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines++
	}
	if lines > 0 {
		lines--
	}
	return lines
}

func readUptime() uint64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	uptime, _ := strconv.ParseFloat(fields[0], 64)
	if uptime < 0 {
		return 0
	}
	return uint64(uptime)
}

func readBootTime() uint64 {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[0] == "btime" {
			value, _ := strconv.ParseUint(fields[1], 10, 64)
			return value
		}
	}
	return 0
}

func subtractFloor(total, free uint64) uint64 {
	if free > total {
		return 0
	}
	return total - free
}
