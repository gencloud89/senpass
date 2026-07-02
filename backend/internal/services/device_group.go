package services

import (
	"fmt"

	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/models"
)

// DeviceGroupService xử lý business logic cho device group
type DeviceGroupService struct{}

// ListAll trả về tất cả device groups
func (s *DeviceGroupService) ListAll() ([]models.DeviceGroup, error) {
	var groups []models.DeviceGroup
	if err := database.DB.Order("show_order ASC, id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}

	// Đếm số server đang kết nối (từ node_clients)
	for i := range groups {
		var count int64
		database.DB.Model(&models.NodeClient{}).
			Where("group_id = ?", groups[i].ID).Count(&count)
		groups[i].DisplayNum = int(count)
	}

	return groups, nil
}

// GetByID trả về một device group theo ID
func (s *DeviceGroupService) GetByID(id uint64) (*models.DeviceGroup, error) {
	var group models.DeviceGroup
	if err := database.DB.First(&group, id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

// Create tạo device group mới
func (s *DeviceGroupService) Create(group *models.DeviceGroup) error {
	// Tự động sinh token nếu chưa có
	if group.Token == "" {
		group.Token = generateToken()
	}
	if group.EnableForGID == "" {
		group.EnableForGID = "1,2"
	}
	return database.DB.Create(group).Error
}

// Update cập nhật device group
func (s *DeviceGroupService) Update(id uint64, updates map[string]interface{}) error {
	return database.DB.Model(&models.DeviceGroup{}).Where("id = ?", id).Updates(updates).Error
}

// Delete xóa nhiều device group theo IDs
func (s *DeviceGroupService) Delete(ids []uint64) error {
	return database.DB.Where("id IN ?", ids).Delete(&models.DeviceGroup{}).Error
}

// ResetToken tạo token mới cho device group
func (s *DeviceGroupService) ResetToken(id uint64) (string, error) {
	newToken := generateToken()
	err := database.DB.Model(&models.DeviceGroup{}).
		Where("id = ?", id).
		Update("token", newToken).Error
	return newToken, err
}

// ResetTraffic đặt traffic_used về 0 cho nhiều group
func (s *DeviceGroupService) ResetTraffic(ids []uint64) error {
	return database.DB.Model(&models.DeviceGroup{}).
		Where("id IN ?", ids).
		Update("traffic_used", 0).Error
}

// Reorder cập nhật show_order cho danh sách group
func (s *DeviceGroupService) Reorder(ids []uint64, orders []int) error {
	if len(ids) != len(orders) {
		return fmt.Errorf("ids và orders phải cùng độ dài")
	}
	for i := range ids {
		if err := database.DB.Model(&models.DeviceGroup{}).
			Where("id = ?", ids[i]).
			Update("show_order", orders[i]).Error; err != nil {
			return err
		}
	}
	return nil
}
