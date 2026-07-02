package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"nyanpass-backend/internal/models"
	"nyanpass-backend/internal/services"
)

type DeviceGroupHandler struct {
	svc *services.DeviceGroupService
}

func NewDeviceGroupHandler() *DeviceGroupHandler {
	return &DeviceGroupHandler{svc: &services.DeviceGroupService{}}
}

// ListAll — GET /api/v1/admin/devicegroup
func (h *DeviceGroupHandler) ListAll(c *gin.Context) {
	groups, err := h.svc.ListAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	if groups == nil {
		groups = []models.DeviceGroup{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": groups, "msg": ""})
}

// Create — PUT /api/v1/admin/devicegroup
func (h *DeviceGroupHandler) Create(c *gin.Context) {
	var group models.DeviceGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if err := h.svc.Create(&group); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": group, "msg": "创建成功"})
}

// Update — POST /api/v1/admin/devicegroup/:id
func (h *DeviceGroupHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "ID không hợp lệ"})
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}

	// Không cho phép cập nhật một số field
	delete(updates, "id")
	delete(updates, "created_at")
	delete(updates, "updated_at")
	delete(updates, "token")

	if err := h.svc.Update(id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "更新成功"})
}

// Delete — DELETE /api/v1/admin/devicegroup
func (h *DeviceGroupHandler) Delete(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Vui lòng chọn ít nhất một mục"})
		return
	}
	if err := h.svc.Delete(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// ResetToken — POST /api/v1/admin/devicegroup/:id/reset_token
func (h *DeviceGroupHandler) ResetToken(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	token, err := h.svc.ResetToken(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": token, "msg": "重置成功"})
}

// ResetTraffic — POST /api/v1/admin/devicegroup/reset_traffic
func (h *DeviceGroupHandler) ResetTraffic(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if err := h.svc.ResetTraffic(req.IDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "重置成功"})
}

// Reorder — POST /api/v1/admin/devicegroup/reorder
func (h *DeviceGroupHandler) Reorder(c *gin.Context) {
	var req struct {
		IDs       []uint64 `json:"ids"`
		ShowOrder []int    `json:"show_order"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if err := h.svc.Reorder(req.IDs, req.ShowOrder); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "排序成功"})
}
