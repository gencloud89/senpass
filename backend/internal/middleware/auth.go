package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"nyanpass-backend/internal/database"
	"nyanpass-backend/internal/models"
)

// CORS middleware cho phép cross-origin requests từ frontend
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// AuthRequired kiểm tra token trong bảng user_logins
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "未登录或登录已过期",
			})
			c.Abort()
			return
		}

		var login models.UserLogin
		if err := database.DB.Where("token = ?", token).First(&login).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "Token không hợp lệ hoặc đã hết hạn",
			})
			c.Abort()
			return
		}

		// Lưu UID vào context để handlers dùng
		c.Set("uid", login.UID)
		c.Next()
	}
}

// AdminRequired kiểm tra quyền admin từ DB
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid, exists := c.Get("uid")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "Không có quyền truy cập",
			})
			c.Abort()
			return
		}

		var user models.User
		if err := database.DB.Where("id = ?", uid).First(&user).Error; err != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "Người dùng không tồn tại",
			})
			c.Abort()
			return
		}

		if !user.Admin {
			c.JSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "Yêu cầu quyền admin",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
