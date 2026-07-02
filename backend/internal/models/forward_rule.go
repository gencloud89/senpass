package models

import "time"

// ForwardRule là model cho bảng forward_rules (luật chuyển tiếp)
type ForwardRule struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Name           string    `gorm:"size:255" json:"name"`
	UID            uint64    `gorm:"index" json:"uid"`
	Paused         bool      `gorm:"default:false" json:"paused"`
	ListenPort     int       `gorm:"index" json:"listen_port"`
	DeviceGroupIn  uint64    `gorm:"index" json:"device_group_in"`
	DeviceGroupOut uint64    `gorm:"index" json:"device_group_out"`
	TrafficUsed    int64     `gorm:"default:0" json:"traffic_used"`
	Config         string    `gorm:"type:text" json:"config"`
	Status           string    `gorm:"size:50;default:ForwardRuleStatus_Normal" json:"status"`
	DisplayUpdatedAt string    `gorm:"-" json:"display_updated_at"` // Format giống web mẫu: "2026-06-30 12:01:56 CST"
}

// ForwardRuleFolder là model cho thư mục phân loại forward rules
type ForwardRuleFolder struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UID       uint64    `gorm:"index" json:"uid"`
	Name      string    `gorm:"size:255;not null" json:"name"`
}

// ForwardRuleFolderRel là quan hệ folder ↔ rule
type ForwardRuleFolderRel struct {
	FolderID uint64 `gorm:"primaryKey" json:"folder_id"`
	RuleID   uint64 `gorm:"primaryKey" json:"rule_id"`
}

// FrFolderRsp là response cho folder API
type FrFolderRsp struct {
	Folders           []ForwardRuleFolder `json:"folders"`
	UnclassifiedCount int                 `json:"unclassified_count"`
}
