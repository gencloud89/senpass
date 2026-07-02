package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/models"
	"nyanpass-backend/internal/services"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{svc: &services.AuthService{}}
}

// Login — POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Remember bool   `json:"remember"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "Dữ liệu không hợp lệ"})
		return
	}

	token, err := h.svc.Login(req.Username, req.Password, req.Remember)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": token, "msg": "登录成功"})
}

// Logout — POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token != "" {
		database.DB.Where("token = ?", token).Delete(&models.UserLogin{})
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "已登出"})
}

// GetUserInfo — GET /api/v1/user/info
func (h *AuthHandler) GetUserInfo(c *gin.Context) {
	token := c.GetHeader("Authorization")
	var login models.UserLogin
	if err := database.DB.Where("token = ?", token).First(&login).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"admin": false}})
		return
	}
	var user models.User
	if err := database.DB.Where("id = ?", login.UID).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"admin": false}})
		return
	}
	// Lấy group_name + plan_name + used_rules giống web mẫu
	groupName := ""
	if user.GroupID > 0 {
		var ug models.UserGroup
		if err := database.DB.Where("id = ?", user.GroupID).First(&ug).Error; err == nil {
			groupName = ug.Name
		}
	}
	var usedRules int64
	database.DB.Model(&models.ForwardRule{}).Where("uid = ?", user.ID).Count(&usedRules)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
		"id": user.ID, "username": user.Username, "admin": user.Admin,
		"balance": user.Balance, "group_id": user.GroupID, "group_name": groupName,
		"max_rules": user.MaxRules, "used_rules": usedRules,
		"traffic_enable": user.TrafficEnable, "traffic_used": user.TrafficUsed,
		"expire": user.Expire, "plan_name": "#0",
	}})
}
