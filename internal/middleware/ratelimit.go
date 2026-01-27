package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"videostreamgo/internal/config"
	"videostreamgo/internal/types"
)

// TenantTier defines rate limit tiers
type TenantTier string

const (
	TierFree TenantTier = "free"
	TierPro  TenantTier = "pro"
	TierEnt  TenantTier = "enterprise"
)

// TierConfig holds rate limit configuration per tier
type TierConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
}

// DefaultTierConfigs defines rate limits per tenant tier
var DefaultTierConfigs = map[TenantTier]TierConfig{
	TierFree: {RequestsPerMinute: 100, RequestsPerHour: 1000, RequestsPerDay: 10000},
	TierPro:  {RequestsPerMinute: 1000, RequestsPerHour: 10000, RequestsPerDay: 100000},
	TierEnt:  {RequestsPerMinute: 10000, RequestsPerHour: 100000, RequestsPerDay: 1000000},
}

// RedisRateLimiter implements distributed rate limiting using Redis sorted sets
type RedisRateLimiter struct {
	client *redis.Client
	cfg    *config.Config
	tiers  map[TenantTier]TierConfig
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(client *redis.Client, cfg *config.Config) *RedisRateLimiter {
	return &RedisRateLimiter{
		client: client,
		cfg:    cfg,
		tiers:  DefaultTierConfigs,
	}
}

// rateLimitKey generates a Redis key for rate limiting
func (r *RedisRateLimiter) rateLimitKey(tenantID, endpoint, window string) string {
	return fmt.Sprintf("ratelimit:%s:%s:%s", tenantID, endpoint, window)
}

// Allow checks if a request should be allowed based on rate limits
func (r *RedisRateLimiter) Allow(ctx *gin.Context, tenantID string, tier TenantTier, endpoint string) (bool, time.Duration) {
	config := r.tiers[tier]
	now := time.Now()

	// Check per-minute limit using sliding window
	allowed, retryAfter := r.checkSlidingWindow(ctx, tenantID, endpoint, "minute", now, config.RequestsPerMinute, 60*time.Second)
	if !allowed {
		return false, retryAfter
	}

	return true, 0
}

// checkSlidingWindow implements sliding window rate limiting with Redis sorted sets
func (r *RedisRateLimiter) checkSlidingWindow(ctx *gin.Context, tenantID, endpoint, window string, now time.Time, limit int, windowDuration time.Duration) (bool, time.Duration) {
	windowStart := now.Add(-windowDuration)
	minScore := float64(windowStart.UnixMilli())

	key := r.rateLimitKey(tenantID, endpoint, window)

	// Remove expired entries older than window start
	r.client.ZRemRangeByScore(ctx.Request.Context(), key, "-inf", fmt.Sprintf("%f", minScore))

	// Count current requests in window using approximate count
	count, err := r.client.ZCard(ctx.Request.Context(), key).Result()
	if err != nil {
		// On error, allow the request (fail open)
		return true, 0
	}

	if count >= int64(limit) {
		// Get the oldest request to calculate retry-after
		oldest, _ := r.client.ZRange(ctx.Request.Context(), key, 0, 0).Result()
		if len(oldest) > 0 {
			oldestTime := now
			retryAfter := time.Until(oldestTime.Add(windowDuration))
			if retryAfter < 0 {
				retryAfter = windowDuration
			}
			return false, retryAfter
		}
		return false, windowDuration
	}

	// Add new request with current timestamp as score (member is unique to avoid duplicates)
	member := redis.Z{
		Score:  float64(now.UnixNano()),
		Member: fmt.Sprintf("%d", now.UnixNano()),
	}
	r.client.ZAdd(ctx.Request.Context(), key, member)

	// Set expiry on the key (slightly longer than window to allow cleanup)
	r.client.Expire(ctx.Request.Context(), key, windowDuration*2)

	return true, 0
}

// RedisRateLimitMiddleware creates a Gin middleware for distributed rate limiting
func RedisRateLimitMiddleware(limiter *RedisRateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get tenant ID from context (set by tenant middleware)
		tenantID, exists := c.Get(string(types.ContextKeyInstanceID))
		if !exists {
			// Fallback to IP-based limiting for unauthenticated requests
			tenantID = c.ClientIP()
		}

		// Get tenant tier from context or default to free
		tierStr, _ := c.Get("tenant_tier")
		tier := TierFree
		if tierStr != nil {
			if t, ok := tierStr.(TenantTier); ok {
				tier = t
			}
		}

		// Get endpoint identifier
		endpoint := c.Request.Method + ":" + c.FullPath()
		if endpoint == c.Request.Method+":" {
			endpoint = c.Request.Method + ":" + c.Request.URL.Path
		}

		allowed, retryAfter := limiter.Allow(c, tenantID.(string), tier, endpoint)

		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
			c.Header("X-RateLimit-Limit", strconv.Itoa(limiter.tiers[tier].RequestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.AbortWithStatusJSON(http.StatusTooManyRequests, types.ErrorResponse(
				"RATE_LIMIT_EXCEEDED",
				"Too many requests. Please try again later.",
				fmt.Sprintf("Rate limit exceeded. Retry after %v", retryAfter),
			))
			return
		}

		c.Next()
	}
}

// GetTierFromSubscription maps subscription tier to rate limit tier
func GetTierFromSubscription(subscription string) TenantTier {
	switch subscription {
	case "pro", "professional":
		return TierPro
	case "enterprise", "business":
		return TierEnt
	default:
		return TierFree
	}
}

// CustomRedisRateLimit creates a rate limiter with custom limits
func CustomRedisRateLimit(client *redis.Client, cfg *config.Config, limits map[TenantTier]TierConfig) *RedisRateLimiter {
	rl := NewRedisRateLimiter(client, cfg)
	if limits != nil {
		rl.tiers = limits
	}
	return rl
}
