package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRateLimiter creates a rate limiter with miniredis for testing
func setupTestRateLimiter(maxRequests int, window time.Duration) (*RateLimiter, *miniredis.Miniredis) {
	// Create miniredis instance (in-memory Redis)
	mr := miniredis.RunT(&testing.T{})

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// Create rate limiter config
	config := RateLimiterConfig{
		MaxRequests: maxRequests,
		Window:      window,
		BlockTime:   5 * time.Minute,
	}

	// Create rate limiter
	rl := NewRateLimiter(client, config)

	return rl, mr
}

// TestRateLimiter_AllowsRequestsUnderLimit tests that requests under the limit are allowed
func TestRateLimiter_AllowsRequestsUnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl, mr := setupTestRateLimiter(5, 1*time.Minute)
	defer mr.Close()

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make 5 requests (under limit)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345" // Simulate same IP
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}
}

// TestRateLimiter_BlocksRequestsOverLimit tests that requests over the limit are blocked
func TestRateLimiter_BlocksRequestsOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl, mr := setupTestRateLimiter(5, 1*time.Minute)
	defer mr.Close()

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make 5 successful requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "Request %d should succeed", i+1)
	}

	// 6th request should be rate limited
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code, "6th request should be rate limited")
	assert.Contains(t, w.Header().Get("Retry-After"), "", "Should have Retry-After header")
}

// TestRateLimiter_DifferentIPsIndependent tests that different IPs have independent limits
func TestRateLimiter_DifferentIPsIndependent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl, mr := setupTestRateLimiter(3, 1*time.Minute)
	defer mr.Close()

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// IP 1: Make 3 requests (at limit)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "IP1 request %d should succeed", i+1)
	}

	// IP 2: Should still have full quota
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.2:12345"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code, "IP2 request %d should succeed", i+1)
	}

	// IP 1: 4th request should be blocked
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code, "IP1 4th request should be rate limited")
}

// TestRateLimiter_CheckLimit tests the CheckLimit method directly
func TestRateLimiter_CheckLimit(t *testing.T) {
	rl, mr := setupTestRateLimiter(3, 1*time.Minute)
	defer mr.Close()

	ip := "192.168.1.100"

	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		allowed, _, err := rl.CheckLimit(ip)
		require.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// 4th request should be denied
	allowed, retryAfter, err := rl.CheckLimit(ip)
	require.NoError(t, err)
	assert.False(t, allowed, "4th request should be denied")
	assert.Greater(t, retryAfter, time.Duration(0), "Should have retry-after duration")
}

// TestRateLimiter_BanIP tests IP banning functionality
func TestRateLimiter_BanIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl, mr := setupTestRateLimiter(100, 1*time.Minute)
	defer mr.Close()

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	bannedIP := "192.168.1.100"

	// Ban the IP
	err := rl.BanIP(bannedIP)
	require.NoError(t, err)

	// Request from banned IP should be blocked
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = bannedIP + ":12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code, "Banned IP should be forbidden")
	assert.Contains(t, w.Body.String(), "banned", "Response should mention ban")
}

// TestRateLimiter_UnbanIP tests IP unbanning functionality
func TestRateLimiter_UnbanIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl, mr := setupTestRateLimiter(100, 1*time.Minute)
	defer mr.Close()

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	ip := "192.168.1.100"

	// Ban the IP
	err := rl.BanIP(ip)
	require.NoError(t, err)

	// Verify banned
	banned, err := rl.IsIPBanned(ip)
	require.NoError(t, err)
	assert.True(t, banned, "IP should be banned")

	// Unban the IP
	err = rl.UnbanIP(ip)
	require.NoError(t, err)

	// Verify unbanned
	banned, err = rl.IsIPBanned(ip)
	require.NoError(t, err)
	assert.False(t, banned, "IP should be unbanned")

	// Request should now succeed
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = ip + ":12345"
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "Unbanned IP should succeed")
}

// TestRateLimiter_IsIPBanned tests the IsIPBanned method
func TestRateLimiter_IsIPBanned(t *testing.T) {
	rl, mr := setupTestRateLimiter(100, 1*time.Minute)
	defer mr.Close()

	ip := "192.168.1.100"

	// IP should not be banned initially
	banned, err := rl.IsIPBanned(ip)
	require.NoError(t, err)
	assert.False(t, banned, "IP should not be banned initially")

	// Ban the IP
	err = rl.BanIP(ip)
	require.NoError(t, err)

	// IP should now be banned
	banned, err = rl.IsIPBanned(ip)
	require.NoError(t, err)
	assert.True(t, banned, "IP should be banned")
}

// TestRateLimiter_WindowExpiry tests that rate limit resets after window expires
func TestRateLimiter_WindowExpiry(t *testing.T) {
	// Use a very short window for testing
	rl, mr := setupTestRateLimiter(2, 1*time.Second)
	defer mr.Close()

	ip := "192.168.1.100"

	// Make 2 requests (at limit)
	for i := 0; i < 2; i++ {
		allowed, _, err := rl.CheckLimit(ip)
		require.NoError(t, err)
		assert.True(t, allowed, "Request %d should be allowed", i+1)
	}

	// 3rd request should be denied
	allowed, _, err := rl.CheckLimit(ip)
	require.NoError(t, err)
	assert.False(t, allowed, "3rd request should be denied")

	// Fast-forward time in miniredis
	mr.FastForward(2 * time.Second)

	// After window expires, requests should be allowed again
	allowed, _, err = rl.CheckLimit(ip)
	require.NoError(t, err)
	assert.True(t, allowed, "Request should be allowed after window expires")
}

// TestRateLimiter_ConcurrentRequests tests rate limiting under concurrent load
func TestRateLimiter_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl, mr := setupTestRateLimiter(10, 1*time.Minute)
	defer mr.Close()

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Simulate 20 concurrent requests from same IP
	successCount := 0
	rateLimitedCount := 0

	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			successCount++
		} else if w.Code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Should allow exactly 10 requests and block 10
	assert.Equal(t, 10, successCount, "Should allow exactly 10 requests")
	assert.Equal(t, 10, rateLimitedCount, "Should block exactly 10 requests")
}

// TestRateLimiter_RetryAfterHeader tests that Retry-After header is set correctly
func TestRateLimiter_RetryAfterHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rl, mr := setupTestRateLimiter(1, 1*time.Minute)
	defer mr.Close()

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request succeeds
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request should be rate limited
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusTooManyRequests, w2.Code)

	// Check Retry-After header exists and is reasonable
	retryAfter := w2.Header().Get("Retry-After")
	assert.NotEmpty(t, retryAfter, "Retry-After header should be set")
}

// Benchmark tests

// BenchmarkRateLimiter_CheckLimit benchmarks the CheckLimit method
func BenchmarkRateLimiter_CheckLimit(b *testing.B) {
	rl, mr := setupTestRateLimiter(1000000, 1*time.Minute)
	defer mr.Close()

	ip := "192.168.1.100"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = rl.CheckLimit(ip)
	}
}

// BenchmarkRateLimiter_Middleware benchmarks the middleware
func BenchmarkRateLimiter_Middleware(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)

	rl, mr := setupTestRateLimiter(1000000, 1*time.Minute)
	defer mr.Close()

	router := gin.New()
	router.Use(rl.Middleware())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
