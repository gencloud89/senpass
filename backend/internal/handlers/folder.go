package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/models"
)

type FolderHandler struct{}

func NewFolderHandler() *FolderHandler {
	return &FolderHandler{}
}

// ListFolders — GET /api/v1/admin/devicegroup/folder
func (h *FolderHandler) ListFolders(c *gin.Context) {
	var folders []models.DeviceGroupFolder
	database.DB.Order("id ASC").Find(&folders)
	if folders == nil {
		folders = []models.DeviceGroupFolder{}
	}

	// Đếm device group chưa phân loại
	var unclassified int64
	database.DB.Model(&models.DeviceGroup{}).Count(&unclassified)
	// Trừ đi các group đã được bind vào folder
	var classified int64
	database.DB.Model(&models.DeviceGroupFolderRel{}).
		Select("DISTINCT dg_id").Count(&classified)
	unclassified = unclassified - classified

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": models.DgFolderRsp{
			Folders:           folders,
			UnclassifiedCount: int(unclassified),
		},
		"msg": "",
	})
}

// CreateFolder — PUT /api/v1/admin/devicegroup/folder
func (h *FolderHandler) CreateFolder(c *gin.Context) {
	var folder models.DeviceGroupFolder
	if err := c.ShouldBindJSON(&folder); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if err := database.DB.Create(&folder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": folder, "msg": "创建成功"})
}

// DeleteFolder — DELETE /api/v1/admin/devicegroup/folder
func (h *FolderHandler) DeleteFolder(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if err := database.DB.Where("id IN ?", req.IDs).Delete(&models.DeviceGroupFolder{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	// Xóa quan hệ folder_rel
	database.DB.Where("folder_id IN ?", req.IDs).Delete(&models.DeviceGroupFolderRel{})
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// BindFolder — POST /api/v1/admin/devicegroup/folder/bind
func (h *FolderHandler) BindFolder(c *gin.Context) {
	var req struct {
		FolderID uint64   `json:"folder_id"`
		ItemIDs  []uint64 `json:"item_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// Xóa quan hệ cũ của các item
	database.DB.Where("dg_id IN ?", req.ItemIDs).Delete(&models.DeviceGroupFolderRel{})

	// Tạo quan hệ mới
	for _, dgID := range req.ItemIDs {
		rel := models.DeviceGroupFolderRel{
			FolderID: req.FolderID,
			DgID:     dgID,
		}
		database.DB.Create(&rel)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "绑定成功"})
}
