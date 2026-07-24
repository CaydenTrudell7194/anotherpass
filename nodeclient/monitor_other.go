//go:build !linux

package main

import (
	"os"
	"runtime"
)

func collectMonitorMetrics() MonitorMetrics {
	hostname, _ := os.Hostname()
	return MonitorMetrics{
		Hostname: hostname,
		Platform: runtime.GOOS,
		Arch:     runtime.GOARCH,
		Version:  clientVersion,
	}
}
