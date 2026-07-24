//go:build cgo

package handler

import (
	"path/filepath"
	"testing"
	"time"

	"forward-panel/internal/model"
)

func TestMonitorSnapshotUsesFreshMetricsForOnlineState(t *testing.T) {
	if err := model.InitDatabase(filepath.Join(t.TempDir(), "monitor.db")); err != nil {
		t.Fatal(err)
	}
	group := model.DeviceGroup{Name: "entry", Type: model.DeviceGroupEntryForceDirect}
	if err := model.DB.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	node := model.Node{Name: "node", DeviceGroupID: group.ID, Token: "monitor-token", Status: "online", LastHeartbeat: time.Now()}
	if err := model.DB.Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	updateNodeMonitor(node, &nodeMetrics{Hostname: "host", CPUPercent: 42})
	snapshot := buildMonitorSnapshot([]model.DeviceGroup{group}, []model.Node{node})
	groups := snapshot["groups"].([]monitorGroupView)
	if len(groups) != 1 || len(groups[0].Nodes) != 1 || !groups[0].Nodes[0].Online || groups[0].Nodes[0].Metrics.CPUPercent != 42 {
		t.Fatalf("unexpected snapshot: %#v", groups)
	}
	nodeMonitorCache.Lock()
	entry := nodeMonitorCache.items[node.ID]
	entry.UpdatedAt = time.Now().Add(-6 * time.Second)
	nodeMonitorCache.items[node.ID] = entry
	nodeMonitorCache.Unlock()
	snapshot = buildMonitorSnapshot([]model.DeviceGroup{group}, []model.Node{node})
	groups = snapshot["groups"].([]monitorGroupView)
	if groups[0].Nodes[0].Online {
		t.Fatal("stale metrics must mark node offline")
	}
}
