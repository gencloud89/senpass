package models

import "time"

// NodeClient lưu thông tin node client kết nối về panel
type NodeClient struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	GroupID         uint64    `gorm:"index" json:"group_id"`
	Name            string    `gorm:"size:50" json:"name"`
	Handle          string    `gorm:"size:64;uniqueIndex" json:"handle"`
	Token           string    `gorm:"size:255" json:"token"`
	IP4             string    `gorm:"size:50" json:"ip4"`
	IP4Geo          string    `gorm:"size:10" json:"ip4_geo"`
	IP6             string    `gorm:"size:50" json:"ip6"`
	IP6Geo          string    `gorm:"size:10" json:"ip6_geo"`
	CPU             float64   `json:"cpu"`
	CPUModel        string    `gorm:"size:255" json:"cpu_model"`
	MemUsed         uint64    `json:"mem_used"`
	MemTotal        uint64    `json:"mem_total"`
	DiskUsed        uint64    `json:"disk_used"`
	DiskTotal       uint64    `json:"disk_total"`
	NetInTransfer   uint64    `json:"net_in_transfer"`
	NetOutTransfer  uint64    `json:"net_out_transfer"`
	NetInSpeed      uint64    `json:"net_in_speed"`
	NetOutSpeed     uint64    `json:"net_out_speed"`
	Uptime          uint64    `json:"uptime"`
	Load1           float64   `json:"load1"`
	Load5           float64   `json:"load5"`
	Load15          float64   `json:"load15"`
	TCPConnCount    int       `json:"tcp_conn_count"`
	UDPConnCount    int       `json:"udp_conn_count"`
	ProcessCount    int       `json:"process_count"`
	Platform        string    `gorm:"size:50" json:"platform"`
	PlatformVersion string    `gorm:"size:50" json:"platform_version"`
	Arch            string    `gorm:"size:20" json:"arch"`
	BootTime        int64     `json:"boot_time"`
	Hostname        string    `gorm:"size:255" json:"hostname"`
	LastSeen        int64     `json:"last_seen"`
	LastPull        int64     `json:"last_pull"`
	Weight          int       `gorm:"default:1" json:"weight"`
}
