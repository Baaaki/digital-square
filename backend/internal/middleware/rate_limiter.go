package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiterConfig defines rate limiting rules
type RateLimiterConfig struct {
	MaxRequests int           // Maximum requests allowed in the window
	Window      time.Duration // Time window (e.g., 1 minute)
	BlockTime   time.Duration // How long to block after exceeding limit
}

// RateLimiter provides IP-based rate limiting using Redis
type RateLimiter struct {
	redis  *redis.Client
	ctx    context.Context
	config RateLimiterConfig
}

// NewRateLimiter creates a new rate limiter instance
func NewRateLimiter(redisClient *redis.Client, config RateLimiterConfig) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		ctx:    context.Background(),
		config: config,
	}
}

// Middleware returns a Gin middleware function for rate limiting
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract client IP address
		clientIP := c.ClientIP()

		// Check if IP is banned first (Phase 2 feature)
		if banned, _ := rl.IsIPBanned(clientIP); banned {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Your IP address has been banned",
			})
			c.Abort()
			return
		}

		// Rate limit check
		allowed, retryAfter, err := rl.CheckLimit(clientIP)
		if err != nil {
			// Log error but don't block request (fail open strategy)
			// In production, you might want to fail closed instead
			c.Next()
			return
		}

		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Too many requests. Please try again later.",
				"retry_after": int(retryAfter.Seconds()),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// CheckLimit uses token bucket algorithm via Redis
// Returns: (allowed bool, retryAfter duration, error)
func (rl *RateLimiter) CheckLimit(ip string) (bool, time.Duration, error) {
	key := fmt.Sprintf("ratelimit:%s", ip)

	// Use Redis INCR with EXPIRE for atomic counter
	// This implements a simple sliding window counter
	count, err := rl.redis.Incr(rl.ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	// Set expiry on first request (count = 1)
	if count == 1 {
		if err := rl.redis.Expire(rl.ctx, key, rl.config.Window).Err(); err != nil {
			return false, 0, err
		}
	}

	// Check if limit exceeded
	if count > int64(rl.config.MaxRequests) {
		// Get TTL to calculate retry-after
		ttl, err := rl.redis.TTL(rl.ctx, key).Result()
		if err != nil {
			ttl = rl.config.Window // Fallback to window size
		}
		return false, ttl, nil
	}

	return true, 0, nil
}

// IsIPBanned checks if an IP address is in the ban list (Phase 2 feature)
func (rl *RateLimiter) IsIPBanned(ip string) (bool, error) {
	exists, err := rl.redis.SIsMember(rl.ctx, "banned_ips", ip).Result()
	return exists, err
}

// BanIP adds an IP to the ban list (Phase 2 feature)
func (rl *RateLimiter) BanIP(ip string) error {
	return rl.redis.SAdd(rl.ctx, "banned_ips", ip).Err()
}

// UnbanIP removes an IP from the ban list (Phase 2 feature)
func (rl *RateLimiter) UnbanIP(ip string) error {
	return rl.redis.SRem(rl.ctx, "banned_ips", ip).Err()
}
