package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/models"
	"nyanpass-backend/internal/services"
)

type AdminUserHandler struct{}

func NewAdminUserHandler() *AdminUserHandler {
	return &AdminUserHandler{}
}

// ListUsers — GET /api/v1/admin/user
func (h *AdminUserHandler) ListUsers(c *gin.Context) {
	var users []models.User
	tx := database.DB.Model(&models.User{})
	if q := c.Query("username"); q != "" {
		tx = tx.Where("username LIKE ?", "%"+q+"%")
	}
	var count int64
	tx.Count(&count)
	tx.Order("id ASC").Find(&users)
	if users == nil {
		users = []models.User{}
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": users, "count": count})
}

// CreateUser — PUT /api/v1/admin/user
func (h *AdminUserHandler) CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "用户名和密码不能为空"})
		return
	}

	hash, _ := services.HashPassword(req.Password)
	user := models.User{
		Username: req.Username,
		Password: hash,
	}
	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": "用户名已存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": user, "msg": "创建成功"})
}

// UpdateUser — POST /api/v1/admin/user/:id
func (h *AdminUserHandler) UpdateUser(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	delete(updates, "id")
	delete(updates, "password")
	delete(updates, "created_at")
	delete(updates, "updated_at")

	if err := database.DB.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "更新成功"})
}

// DeleteUsers — DELETE /api/v1/admin/user
func (h *AdminUserHandler) DeleteUsers(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	if err := database.DB.Where("id IN ?", req.IDs).Delete(&models.User{}).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}

// ResetPassword — POST /api/v1/admin/user/:id/reset_password
func (h *AdminUserHandler) ResetPassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "请输入新密码"})
		return
	}
	hash, _ := services.HashPassword(req.Password)
	database.DB.Model(&models.User{}).Where("id = ?", id).Update("password", hash)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "密码重置成功"})
}

// ========== USER GROUP ==========

// ListUserGroups — GET /api/v1/admin/usergroup
func (h *AdminUserHandler) ListUserGroups(c *gin.Context) {
	var groups []models.UserGroup
	database.DB.Order("show_order ASC, id ASC").Find(&groups)
	if groups == nil {
		groups = []models.UserGroup{}
	}
	// Đếm số user trong mỗi group
	for i := range groups {
		var cnt int64
		database.DB.Model(&models.User{}).Where("group_id = ?", groups[i].ID).Count(&cnt)
		groups[i].UserCount = cnt
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": groups, "msg": ""})
}

// CreateUserGroup — PUT /api/v1/admin/usergroup
func (h *AdminUserHandler) CreateUserGroup(c *gin.Context) {
	var group models.UserGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	database.DB.Create(&group)
	c.JSON(http.StatusOK, gin.H{"code": 0, "data": group, "msg": "创建成功"})
}

// UpdateUserGroup — POST /api/v1/admin/usergroup/:id
func (h *AdminUserHandler) UpdateUserGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	database.DB.Model(&models.UserGroup{}).Where("id = ?", id).Updates(updates)
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "更新成功"})
}

// DeleteUserGroups — DELETE /api/v1/admin/usergroup
func (h *AdminUserHandler) DeleteUserGroups(c *gin.Context) {
	var req struct {
		IDs []uint64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	database.DB.Where("id IN ?", req.IDs).Delete(&models.UserGroup{})
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "删除成功"})
}
