package auth

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	rateLimitWindow  = time.Minute
	rateLimitMaxReqs = 10
)

type ipLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}

var loginLimiter = &ipLimiter{attempts: make(map[string][]time.Time)}

func (l *ipLimiter) allow(ip string) bool {
	now := time.Now()
	cutoff := now.Add(-rateLimitWindow)

	l.mu.Lock()
	defer l.mu.Unlock()

	times := l.attempts[ip]
	valid := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= rateLimitMaxReqs {
		l.attempts[ip] = valid
		return false
	}
	l.attempts[ip] = append(valid, now)
	return true
}

func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if !loginLimiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"status":  false,
				"message": "Too many login attempts. Please try again later.",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}
