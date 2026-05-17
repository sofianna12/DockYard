package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

var (
	visitors   = make(map[string]*visitor)
	visitorsMu sync.Mutex
)

func init() {
	go func() {
		for {
			time.Sleep(time.Minute)
			visitorsMu.Lock()
			for ip, v := range visitors {
				if time.Since(v.lastSeen) > 3*time.Minute {
					delete(visitors, ip)
				}
			}
			visitorsMu.Unlock()
		}
	}()
}

func getVisitor(ip string) *rate.Limiter {
	visitorsMu.Lock()
	defer visitorsMu.Unlock()
	v, exists := visitors[ip]
	if !exists {
		// 10 requests per minute, burst of 15
		lim := rate.NewLimiter(rate.Every(6*time.Second), 15)
		visitors[ip] = &visitor{lim, time.Now()}
		return lim
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// RateLimit limits requests per client IP.
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !getVisitor(c.ClientIP()).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}
		c.Next()
	}
}
