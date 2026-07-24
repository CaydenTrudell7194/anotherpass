package model

import "time"

type DeviceGroupType string

const (
	DeviceGroupEntryForceDirect    DeviceGroupType = "entry_force_direct"
	DeviceGroupEntryOptionalDirect DeviceGroupType = "entry_optional_direct"
	DeviceGroupEntry               DeviceGroupType = "entry"
	DeviceGroupMonitor             DeviceGroupType = "monitor"
)

type DeviceGroup struct {
	ID             uint            `gorm:"primaryKey" json:"id"`
	Name           string          `gorm:"size:128" json:"name"`
	Type           DeviceGroupType `gorm:"size:32;default:entry" json:"type"`
	UserGroupIDs   string          `gorm:"size:256" json:"user_group_ids"`
	ConnectionAddr string          `gorm:"size:256" json:"connection_addr"`
	Rate           float64         `gorm:"default:1" json:"rate"`
	HideInProbe    bool            `gorm:"default:false" json:"hide_in_probe"`
	Notes          string          `gorm:"size:512" json:"notes"`
	SortOrder      int             `gorm:"default:0" json:"sort_order"`
	TrafficUsed    int64           `gorm:"default:0" json:"traffic_used"`
	OnlineDevices  int             `gorm:"default:0" json:"online_devices"`
	NodeToken      string          `gorm:"size:64;index" json:"-"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type Node struct {
	ID            uint        `gorm:"primaryKey" json:"id"`
	DeviceGroupID uint        `gorm:"index" json:"device_group_id"`
	DeviceGroup   DeviceGroup `gorm:"foreignKey:DeviceGroupID" json:"-"`
	Name          string      `gorm:"size:128" json:"name"`
	IP            string      `gorm:"size:64" json:"ip"`
	Token         string      `gorm:"size:256;uniqueIndex" json:"-"`
	InstanceID    string      `gorm:"size:64;index" json:"instance_id"`
	EnrollHash    string      `gorm:"size:64;index" json:"-"`
	EnrollExpires time.Time   `json:"-"`
	Status        string      `gorm:"size:16;default:offline" json:"status"`
	LastHeartbeat time.Time   `json:"last_heartbeat"`
	TrafficUp     int64       `gorm:"default:0" json:"traffic_up"`
	TrafficDown   int64       `gorm:"default:0" json:"traffic_down"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}
