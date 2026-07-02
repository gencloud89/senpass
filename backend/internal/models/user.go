package models

import "time"

// User là model cho bảng users
type User struct {
	ID              uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Username        string    `gorm:"uniqueIndex;size:255" json:"username"`
	Password        string    `gorm:"size:255" json:"-"`
	Balance         string    `gorm:"size:50;default:0" json:"balance"`
	AffBalance      string    `gorm:"size:50;default:0" json:"aff_balance"`
	Inviter         uint64    `gorm:"default:0" json:"inviter"`
	InviteConfig    string    `gorm:"type:text" json:"invite_config"`
	InviteCode      string    `gorm:"size:100" json:"invite_code"`
	PlanID          uint64    `gorm:"default:0" json:"plan_id"`
	GroupID         uint64    `gorm:"index;default:0" json:"group_id"`
	MaxRules        int       `gorm:"default:0" json:"max_rules"`
	SpeedLimit      int       `gorm:"default:0" json:"speed_limit"`
	IPLimit         int       `gorm:"default:0" json:"ip_limit"`
	ConnectionLimit int       `gorm:"default:0" json:"connection_limit"`
	TrafficEnable   int64     `gorm:"default:0" json:"traffic_enable"`
	TrafficUsed     int64     `gorm:"default:0" json:"traffic_used"`
	Expire          int64     `gorm:"default:0" json:"expire"`
	AutoRenew       bool      `gorm:"default:false" json:"auto_renew"`
	Banned          bool      `gorm:"default:false" json:"banned"`
	Admin           bool      `gorm:"default:false" json:"admin"`
	AllowDevice     bool      `gorm:"default:false" json:"allow_device"`
	TelegramID      int64     `gorm:"default:0" json:"telegram_id"`
	Note            string    `gorm:"type:text" json:"note"`
}

// UserGroup là model cho bảng user_groups
type UserGroup struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `gorm:"size:255" json:"name"`
	ShowOrder int       `gorm:"default:0" json:"show_order"`
	UserCount int64     `gorm:"-" json:"user_count,omitempty"`
}

// UserLogin là model cho bảng user_logins (phiên đăng nhập)
type UserLogin struct {
	ID          uint64    `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	UID         uint64    `gorm:"index" json:"uid"`
	Token       string    `gorm:"index;size:255" json:"token"`
	TokenExpire int64     `gorm:"default:0" json:"token_expire"`
}
