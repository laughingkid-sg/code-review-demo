package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type clientBucket struct {
	tokens     float64
	lastRefill time.Time
}

// RateLimiter provides in-memory token bucket rate limiting per IP address.
type RateLimiter struct {
	mu         sync.Mutex
	clients    map[string]*clientBucket
	rate       float64 // tokens per second
	capacity   float64 // maximum tokens
	lastClean  time.Time
	cleanEvery time.Duration
}

// NewRateLimiter creates a RateLimiter with configured requests per minute (RPM).
func NewRateLimiter(rpm int) *RateLimiter {
	if rpm <= 0 {
		rpm = 100
	}
	rate := float64(rpm) / 60.0
	capacity := float64(rpm)

	return &RateLimiter{
		clients:    make(map[string]*clientBucket),
		rate:       rate,
		capacity:   capacity,
		lastClean:  time.Now(),
		cleanEvery: 5 * time.Minute,
	}
}

// Allow checks whether a client IP has available request tokens.
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Periodic cleanup of stale clients
	if now.Sub(rl.lastClean) > rl.cleanEvery {
		for ip, b := range rl.clients {
			if now.Sub(b.lastRefill) > 10*time.Minute {
				delete(rl.clients, ip)
			}
		}
		rl.lastClean = now
	}

	bucket, exists := rl.clients[clientIP]
	if !exists {
		rl.clients[clientIP] = &clientBucket{
			tokens:     rl.capacity - 1,
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on elapsed duration
	elapsed := now.Sub(bucket.lastRefill).Seconds()
	bucket.tokens = bucket.tokens + (elapsed * rl.rate)
	if bucket.tokens > rl.capacity {
		bucket.tokens = rl.capacity
	}
	bucket.lastRefill = now

	if bucket.tokens >= 1.0 {
		bucket.tokens -= 1.0
		return true
	}

	return false
}

// RateLimitMiddleware returns a Gin middleware that enforces rate limiting per client IP.
func RateLimitMiddleware(limiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		if !limiter.Allow(clientIP) {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded, please retry later",
			})
			return
		}
		c.Next()
	}
}
