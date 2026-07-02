package models

// ChainOutbound là model cho bảng chain_outbounds (chuỗi chuyển tiếp)
type ChainOutbound struct {
	GroupID uint64 `gorm:"primaryKey" json:"group_id"`
	Seq     int    `gorm:"primaryKey" json:"seq"`
	ThisHop uint64 `gorm:"not null" json:"this_hop"`
	NextHop uint64 `gorm:"not null" json:"next_hop"`
	Mux     bool   `gorm:"default:false" json:"mux"`
}
