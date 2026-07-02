package database

import (
	"log"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"nyanpass-backend/internal/models"
)

var DB *gorm.DB

// Init khởi tạo kết nối database và auto-migrate
func Init(dsn string) error {
	// Hỗ trợ format: sqlite3://path hoặc path trực tiếp
	path := strings.TrimPrefix(dsn, "sqlite3://")

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	// Auto-migrate tất cả models
	err = db.AutoMigrate(
		&models.DeviceGroup{},
		&models.ChainOutbound{},
		&models.DeviceGroupFolder{},
		&models.DeviceGroupFolderRel{},
		&models.User{},
		&models.UserGroup{},
		&models.UserLogin{},
		&models.NodeClient{},
		&models.ForwardRule{},
		&models.ForwardRuleFolder{},
		&models.ForwardRuleFolderRel{},
		&models.DeviceGroupReplica{},
	)
	if err != nil {
		return err
	}

	DB = db
	log.Println("[DB] Database initialized successfully")
	return nil
}
