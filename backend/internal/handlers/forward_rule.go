package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/models"
)

type ForwardRuleHandler struct{}

func NewForwardRuleHandler() *ForwardRuleHandler { return &ForwardRuleHandler{} }

// List — GET /api/v1/admin/forward
func (h *ForwardRuleHandler) List(c *gin.Context) {
	var rules []models.ForwardRule
	tx := database.DB.Model(&models.ForwardRule{})
	if uid := c.Query("uid"); uid != "" {
		tx = tx.Where("uid = ?", uid)
	}
	tx.Order("id DESC").Find(&rules)
	if rules == nil {
		rules = []models.ForwardRule{}
	}
	for i := range rules {
		rules[i].DisplayUpdatedAt = rules[i].UpdatedAt.Format("2006-01-02 15:04:05") + " CST"
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rules, "msg": ""})
}

// Create — PUT /api/v1/admin/forward
func (h *ForwardRuleHandler) Create(c *gin.Context) {
	var rule models.ForwardRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if rule.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Tên rule không được để trống"})
		return
	}
	if rule.Config == "" { rule.Config = "{}" }
	if rule.Status == "" { rule.Status = "ForwardRuleStatus_Normal" }
	if err := database.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": rule, "msg": "创建成功"})
}

// Update — POST /api/v1/admin/forward/:id
func (h *ForwardRuleHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	delete(updates, "id"); delete(updates, "created_at"); delete(updates, "updated_at")
	if err := database.DB.Model(&models.ForwardRule{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "更新成功"})
}

// Delete — DELETE /api/v1/admin/forward
func (h *ForwardRuleHandler) Delete(c *gin.Context) {
	var req struct{ IDs []uint64 `json:"ids"` }
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Vui lòng chọn ít nhất một rule"})
		return
	}
	database.DB.Where("id IN ?", req.IDs).Delete(&models.ForwardRule{})
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// ResetTraffic — POST /api/v1/admin/forward/reset_traffic
func (h *ForwardRuleHandler) ResetTraffic(c *gin.Context) {
	var req struct{ IDs []uint64 `json:"ids"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	database.DB.Model(&models.ForwardRule{}).Where("id IN ?", req.IDs).Update("traffic_used", 0)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "重置成功"})
}

// Diagnose — POST /api/v1/admin/forward/:id/diagnose
func (h *ForwardRuleHandler) Diagnose(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var rule models.ForwardRule
	if err := database.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "Rule không tồn tại"})
		return
	}
	var inDG, outDG models.DeviceGroup
	inErr := database.DB.First(&inDG, rule.DeviceGroupIn).Error
	outErr := database.DB.First(&outDG, rule.DeviceGroupOut).Error
	ok := inErr == nil && outErr == nil
	status := "ForwardRuleStatus_Normal"
	if !ok { status = "ForwardRuleStatus_Failed" }
	database.DB.Model(&rule).Update("status", status)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"status": status, "ok": ok}, "msg": "诊断完成"})
}

// ========== FOLDER ==========

// ListFolders — GET /api/v1/admin/forward/folder
func (h *ForwardRuleHandler) ListFolders(c *gin.Context) {
	var folders []models.ForwardRuleFolder
	database.DB.Order("id ASC").Find(&folders)
	if folders == nil { folders = []models.ForwardRuleFolder{} }
	var unclassified int64
	database.DB.Model(&models.ForwardRule{}).Count(&unclassified)
	var classified int64
	database.DB.Model(&models.ForwardRuleFolderRel{}).Select("DISTINCT rule_id").Count(&classified)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": models.FrFolderRsp{
		Folders: folders, UnclassifiedCount: int(unclassified) - int(classified),
	}, "msg": ""})
}

// CreateFolder — PUT /api/v1/admin/forward/folder
func (h *ForwardRuleHandler) CreateFolder(c *gin.Context) {
	var folder models.ForwardRuleFolder
	if err := c.ShouldBindJSON(&folder); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	database.DB.Create(&folder)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": folder, "msg": "创建成功"})
}

// DeleteFolder — DELETE /api/v1/admin/forward/folder
func (h *ForwardRuleHandler) DeleteFolder(c *gin.Context) {
	var req struct{ IDs []uint64 `json:"ids"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	database.DB.Where("id IN ?", req.IDs).Delete(&models.ForwardRuleFolder{})
	database.DB.Where("folder_id IN ?", req.IDs).Delete(&models.ForwardRuleFolderRel{})
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// BindFolder — POST /api/v1/admin/forward/folder/bind
func (h *ForwardRuleHandler) BindFolder(c *gin.Context) {
	var req struct {
		FolderID uint64   `json:"folder_id"`
		ItemIDs  []uint64 `json:"item_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	database.DB.Where("rule_id IN ?", req.ItemIDs).Delete(&models.ForwardRuleFolderRel{})
	for _, rid := range req.ItemIDs {
		database.DB.Create(&models.ForwardRuleFolderRel{FolderID: req.FolderID, RuleID: rid})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "绑定成功"})
}
