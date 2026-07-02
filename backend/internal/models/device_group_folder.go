package models

import "time"

// DeviceGroupFolder là model cho thư mục phân loại device group
type DeviceGroupFolder struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `gorm:"size:255;not null" json:"name"`
}

// DeviceGroupFolderRel là quan hệ folder ↔ device group
type DeviceGroupFolderRel struct {
	FolderID  uint64 `gorm:"primaryKey" json:"folder_id"`
	DgID      uint64 `gorm:"primaryKey" json:"dg_id"`
	ShowOrder int    `gorm:"default:0" json:"show_order,omitempty"`
}

// DgFolderRsp là response cho folder API (giống web mẫu)
type DgFolderRsp struct {
	Folders          []DeviceGroupFolder `json:"folders"`
	UnclassifiedCount int                `json:"unclassified_count"`
}
