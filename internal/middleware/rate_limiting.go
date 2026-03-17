package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/time/rate"

	"videostreamgo/internal/types"
)

// RateLimiter tracks rate limits per key (tenant ID or IP address)
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(r rate.Limit, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst,
	}
}

// getLimiter returns the limiter for a key, creating one if needed
func (i *RateLimiter) getLimiter(key string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(i.rate, i.burst)
		i.limiters[key] = limiter
	}

	return limiter
}

// getRateLimitKey extracts the rate limit key from the gin context.
// Priority: tenant ID from context > IP address
// This prevents noisy neighbor problem by limiting per-tenant instead of globally
func getRateLimitKey(c *gin.Context) string {
	// First, try to get tenant ID from context
	if tenantID, exists := c.Get(string(types.ContextKeyTenantID)); exists {
		if id, ok := tenantID.(uuid.UUID); ok && id != uuid.Nil {
			return "tenant:" + id.String()
		}
	}

	// Fall back to IP address if tenant context is not available
	return "ip:" + c.ClientIP()
}

// RateLimitMiddleware creates a Gin middleware for rate limiting
func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := getRateLimitKey(c)
		limiter := rl.getLimiter(key)

		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, types.ErrorResponse(
				"RATE_LIMIT_EXCEEDED",
				"Too many requests. Please try again later.",
				"",
			))
			return
		}

		c.Next()
	}
}

// GlobalRateLimiter provides a package-level rate limiter (per-tenant)
var GlobalRateLimiter = NewRateLimiter(rate.Limit(100), 200)

// RateLimit provides a simple rate limiting middleware using the global limiter
func RateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(GlobalRateLimiter)
}

// RateLimitByEndpoint creates a rate limiter for specific endpoints
func RateLimitByEndpoint(ratePerSecond rate.Limit, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(ratePerSecond, burst)
	return RateLimitMiddleware(limiter)
}

// slidingWindowLimiter implements a sliding window rate limiter
type slidingWindowLimiter struct {
	windowSize time.Duration
	limit      int
	requests   map[string][]time.Time
	mu         sync.RWMutex
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(windowSize time.Duration, limit int) *slidingWindowLimiter {
	return &slidingWindowLimiter{
		windowSize: windowSize,
		limit:      limit,
		requests:   make(map[string][]time.Time),
	}
}

// Allow checks if a request is allowed
func (s *slidingWindowLimiter) Allow(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-s.windowSize)

	// Clean old requests
	requests := s.requests[key]
	var valid []time.Time
	for _, t := range requests {
		if t.After(windowStart) {
			valid = append(valid, t)
		}
	}
	s.requests[key] = valid

	// Check limit
	if len(valid) >= s.limit {
		return false
	}

	// Add new request
	s.requests[key] = append(valid, now)
	return true
}

// SlidingWindowRateLimit creates a middleware using sliding window algorithm
func SlidingWindowRateLimit(windowSize time.Duration, limit int) gin.HandlerFunc {
	limiter := NewSlidingWindowLimiter(windowSize, limit)
	return func(c *gin.Context) {
		key := getRateLimitKey(c)
		if !limiter.Allow(key) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, types.ErrorResponse(
				"RATE_LIMIT_EXCEEDED",
				"Too many requests. Please try again later.",
				"",
			))
			return
		}
		c.Next()
	}
}
