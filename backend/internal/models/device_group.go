package models

import "time"

// DeviceGroupType định nghĩa loại device group
type DeviceGroupType string

const (
	DeviceGroupTypeInbound        DeviceGroupType = "DeviceGroupType_Inbound"
	DeviceGroupTypeOutboundBySite DeviceGroupType = "DeviceGroupType_OutboundBySite"
	DeviceGroupTypeOutboundByUser DeviceGroupType = "DeviceGroupType_OutboundByUser"
	DeviceGroupTypeAgentOnly      DeviceGroupType = "DeviceGroupType_AgentOnly"
	DeviceGroupTypeChainOutbound  DeviceGroupType = "DeviceGroupType_ChainOutbound"
)

// DeviceGroup là model cho bảng device_groups
type DeviceGroup struct {
	ID            uint64          `gorm:"primaryKey" json:"id"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Name          string          `gorm:"size:255" json:"name"`
	Type          DeviceGroupType `gorm:"size:50" json:"type"`
	Token         string          `gorm:"size:255" json:"token"`
	UID           uint64          `gorm:"index" json:"uid"`
	Ratio         string          `gorm:"size:50;default:0" json:"ratio"`
	EnableForGID  string          `gorm:"size:500" json:"enable_for_gid"`
	TrafficUsed   int64           `gorm:"default:0" json:"traffic_used"`
	ConnectHost   string          `gorm:"size:500" json:"connect_host,omitempty"`
	PortRange     string          `gorm:"size:50" json:"port_range,omitempty"`
	AllowedOut    string          `gorm:"type:text" json:"allowed_out,omitempty"`
	AllowedIn     string          `gorm:"type:text" json:"allowed_in,omitempty"`
	Config        string          `gorm:"type:text" json:"config,omitempty"`
	DownSec       int64           `gorm:"default:0" json:"down_sec,omitempty"`
	FallbackGroup uint64          `gorm:"default:0" json:"fallback_group,omitempty"`
	Note          string          `gorm:"type:text" json:"note,omitempty"`
	ShowOrder     int             `gorm:"default:0" json:"show_order,omitempty"`
	HideStatus    int             `gorm:"default:0" json:"hide_status,omitempty"`

	// Không lưu trong DB — tính từ bảng chain_outbounds
	DisplayNum int `gorm:"-" json:"display_num,omitempty"`
}

// DeviceGroupReplica — liên kết replica giữa các inbound group
// Dùng để tự động đồng bộ forward rules giữa group gốc và group replica
type DeviceGroupReplica struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	PrimaryGroupID uint64    `gorm:"index" json:"primary_group_id"`
	ReplicaGroupID uint64    `gorm:"index" json:"replica_group_id"`
}
