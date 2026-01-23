package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Test_RateLimiter_BasicFunctionality tests basic rate limiting functionality
func Test_RateLimiter_BasicFunctionality(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 60,
		RequestsPerHour:   1000,
		RequestsPerDay:    10000,
		BurstSize:         10,
	})

	r := gin.New()
	r.Use(limiter.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request should succeed
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Check rate limit headers are present
	assert.Contains(t, w.Header(), "X-RateLimit-Limit")
	assert.Contains(t, w.Header(), "X-RateLimit-Remaining")
	assert.Contains(t, w.Header(), "X-RateLimit-Reset")
}

// Test_RateLimiter_ExceedsLimit tests that requests exceeding limit are rejected
func Test_RateLimiter_ExceedsLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 5, // Very low limit for testing
		RequestsPerHour:   100,
		RequestsPerDay:    1000,
		BurstSize:         2,
	})

	r := gin.New()
	r.Use(limiter.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Make requests up to the limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if i < 5 {
			assert.Equal(t, http.StatusOK, w.Code)
		}
	}

	// Next request should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

// Test_RateLimiter_DifferentIPs tests that different IPs have separate limits
func Test_RateLimiter_DifferentIPs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 10,
		RequestsPerHour:   100,
		RequestsPerDay:    1000,
		BurstSize:         5,
	})

	r := gin.New()
	r.Use(limiter.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// IP 1 makes requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-Forwarded-For", "192.168.1.1")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	// IP 2 should have its own limit
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_RateLimiter_ConcurrentAccess tests rate limiting under concurrent access
func Test_RateLimiter_ConcurrentAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 100,
		RequestsPerHour:   1000,
		RequestsPerDay:    10000,
		BurstSize:         20,
	})

	r := gin.New()
	r.Use(limiter.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	var wg sync.WaitGroup
	results := make(chan int, 50)

	// Launch 50 concurrent requests
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			results <- w.Code
		}()
	}

	wg.Wait()
	close(results)

	// Count results
	successCount := 0
	rateLimitedCount := 0
	for code := range results {
		if code == http.StatusOK {
			successCount++
		} else if code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Most requests should succeed (within rate limit)
	assert.Greater(t, successCount, 40)
}

// Test_RateLimiter_ResetAfterWindow tests that limit resets after time window
func Test_RateLimiter_ResetAfterWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a limiter with 1 second window
	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 2, // Very low limit
		RequestsPerHour:   100,
		RequestsPerDay:    1000,
		BurstSize:         1,
	})

	r := gin.New()
	r.Use(limiter.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// Use up the limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	// Should be rate limited
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Wait for rate limit window to reset
	time.Sleep(time.Second)

	// Should succeed after reset
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_RateLimiter_Headers tests rate limit response headers
func Test_RateLimiter_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 60,
		RequestsPerHour:   1000,
		RequestsPerDay:    10000,
		BurstSize:         10,
	})

	r := gin.New()
	r.Use(limiter.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check headers are present
	assert.Equal(t, "60", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "59", w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
}

// Test_RateLimiter_RetryAfterHeader tests Retry-After header on rate limit
func Test_RateLimiter_RetryAfterHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewRateLimiter(RateLimiterConfig{
		RequestsPerMinute: 1,
		RequestsPerHour:   100,
		RequestsPerDay:    1000,
		BurstSize:         1,
	})

	r := gin.New()
	r.Use(limiter.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	// First request succeeds
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second request should be rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	// Check Retry-After header
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
}

// RateLimiterConfig for testing
type RateLimiterConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
	BurstSize         int
}

// NewRateLimiter creates a new rate limiter for testing
func NewRateLimiter(config RateLimiterConfig) *TestRateLimiter {
	return &TestRateLimiter{
		config:            config,
		requests:          make(map[string][]time.Time),
		mu:                sync.Mutex{},
		requestsPerMinute: config.RequestsPerMinute,
		requestsPerHour:   config.RequestsPerHour,
		requestsPerDay:    config.RequestsPerDay,
		burstSize:         config.BurstSize,
	}
}

// TestRateLimiter is a test version of the rate limiter
type TestRateLimiter struct {
	config            RateLimiterConfig
	requests          map[string][]time.Time
	mu                sync.Mutex
	requestsPerMinute int
	requestsPerHour   int
	requestsPerDay    int
	burstSize         int
}

// Middleware returns the rate limiting middleware
func (rl *TestRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		rl.mu.Lock()
		defer rl.mu.Unlock()

		// Clean old requests
		rl.cleanOldRequests(ip, now)

		// Count recent requests
		recentRequests := rl.requests[ip]

		// Check if within burst limit
		if len(recentRequests) >= rl.burstSize {
			c.Header("Retry-After", "1")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": map[string]string{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many requests",
				},
			})
			return
		}

		// Add request
		rl.requests[ip] = append(recentRequests, now)

		// Set headers
		remaining := rl.burstSize - len(rl.requests[ip])
		c.Header("X-RateLimit-Limit", string(rune('0'+rl.burstSize)))
		c.Header("X-RateLimit-Remaining", string(rune('0'+remaining)))
		c.Header("X-RateLimit-Reset", now.Format(time.RFC3339))

		c.Next()
	}
}

// cleanOldRequests removes requests outside the time window
func (rl *TestRateLimiter) cleanOldRequests(ip string, now time.Time) {
	cutoff := now.Add(-time.Minute)
	requests := rl.requests[ip]
	var validRequests []time.Time
	for _, reqTime := range requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	rl.requests[ip] = validRequests
}
