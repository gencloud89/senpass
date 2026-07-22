package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter — giới hạn số request theo IP trong khoảng thời gian
type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	limit    int
	window   time.Duration
}

type visitor struct {
	count    int
	lastSeen time.Time
}

// NewRateLimiter tạo rate limiter mới
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		window:   window,
	}
	// Cleanup goroutine mỗi 5 phút
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rl.mu.Lock()
			for ip, v := range rl.visitors {
				if time.Since(v.lastSeen) > rl.window {
					delete(rl.visitors, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

// Middleware trả về Gin handler
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rl.mu.Lock()
		defer rl.mu.Unlock()

		ip := c.ClientIP()
		v, exists := rl.visitors[ip]

		if !exists || time.Since(v.lastSeen) > rl.window {
			rl.visitors[ip] = &visitor{count: 1, lastSeen: time.Now()}
			c.Next()
			return
		}

		v.count++
		v.lastSeen = time.Now()

		if v.count > rl.limit {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code": 429,
				"msg":  "Quá nhiều yêu cầu. Vui lòng thử lại sau.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
