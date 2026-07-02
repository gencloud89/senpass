package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

// AuthRequired kiểm tra token authorization
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

		// Token validation đơn giản — trong production cần kiểm tra DB
		if strings.TrimSpace(token) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "Token không hợp lệ",
			})
			c.Abort()
			return
		}

		// TODO: Validate token từ bảng user_logins
		// Tạm thời chấp nhận mọi token không rỗng

		c.Next()
	}
}

// AdminRequired kiểm tra quyền admin
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Kiểm tra user có admin=true từ DB
		c.Next()
	}
}
