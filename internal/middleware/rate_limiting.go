package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"videostreamgo/internal/types"
)

// IPRateLimiter tracks rate limits per IP address
type IPRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

// NewIPRateLimiter creates a new rate limiter
func NewIPRateLimiter(r rate.Limit, burst int) *IPRateLimiter {
	return &IPRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    burst,
	}
}

// getLimiter returns the limiter for an IP, creating one if needed
func (i *IPRateLimiter) getLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.limiters[ip]
	if !exists {
		limiter = rate.NewLimiter(i.rate, i.burst)
		i.limiters[ip] = limiter
	}

	return limiter
}

// RateLimitMiddleware creates a Gin middleware for rate limiting
func RateLimitMiddleware(rl *IPRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiter := rl.getLimiter(ip)

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

// GlobalRateLimiter provides a package-level rate limiter
var GlobalRateLimiter = NewIPRateLimiter(rate.Limit(100), 200)

// RateLimit provides a simple rate limiting middleware using the global limiter
func RateLimit() gin.HandlerFunc {
	return RateLimitMiddleware(GlobalRateLimiter)
}

// RateLimitByEndpoint creates a rate limiter for specific endpoints
func RateLimitByEndpoint(ratePerSecond rate.Limit, burst int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(ratePerSecond, burst)
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
		ip := c.ClientIP()
		if !limiter.Allow(ip) {
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
