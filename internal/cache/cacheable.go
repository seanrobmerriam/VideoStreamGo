package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// TTL presets for different data types
const (
	TTLUser         = 5 * time.Minute
	TTLConfig       = 15 * time.Minute
	TTLStatic       = 1 * time.Hour
	TTLInstance     = 10 * time.Minute
	TTLTenantConfig = 15 * time.Minute
)

// CacheKeyPrefix holds versioned key prefixes
var CacheKeyPrefix = struct {
	V1 string
}{
	V1: "v1",
}

// Cacheable is a generic cache wrapper implementing cache-aside pattern
type Cacheable[T any] struct {
	client *RedisClient
	prefix string
	ttl    time.Duration
}

// NewCacheable creates a new cacheable wrapper
func NewCacheable[T any](client *RedisClient, prefix string, ttl time.Duration) *Cacheable[T] {
	return &Cacheable[T]{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// buildKey constructs a cache key with versioning
func (c *Cacheable[T]) buildKey(id string) string {
	return fmt.Sprintf("%s:%s:%s", c.prefix, c.prefix, id)
}

// Get retrieves data from cache, returns (data, cacheHit, error)
func (c *Cacheable[T]) Get(ctx context.Context, id string) (*T, bool, error) {
	key := c.buildKey(id)

	data, err := c.client.Get(ctx, key)
	if err != nil {
		return nil, false, fmt.Errorf("cache get error: %w", err)
	}

	if data == "" {
		return nil, false, nil // Cache miss
	}

	var result T
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		// Invalid cache data, treat as miss
		return nil, false, nil
	}

	return &result, true, nil
}

// Set stores data in cache with TTL
func (c *Cacheable[T]) Set(ctx context.Context, id string, data *T) error {
	key := c.buildKey(id)

	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return c.client.Set(ctx, key, string(bytes), c.ttl)
}

// Delete removes data from cache
func (c *Cacheable[T]) Delete(ctx context.Context, id string) error {
	key := c.buildKey(id)
	return c.client.Delete(ctx, key)
}

// Invalidate removes the cached item
func (c *Cacheable[T]) Invalidate(ctx context.Context, id string) error {
	return c.Delete(ctx, id)
}

// GetOrFetch implements cache-aside: try cache first, on miss call fetcher
func (c *Cacheable[T]) GetOrFetch(ctx context.Context, id string, fetcher func(ctx context.Context) (*T, error)) (*T, error) {
	// Try cache first
	cached, hit, err := c.Get(ctx, id)
	if err == nil && hit && cached != nil {
		return cached, nil
	}

	// Cache miss, fetch from source
	data, err := fetcher(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetcher error: %w", err)
	}

	if data == nil {
		return nil, nil
	}

	// Store in cache (best effort, don't fail if cache write fails)
	_ = c.Set(ctx, id, data)

	return data, nil
}

// Refresh updates cache if it exists
func (c *Cacheable[T]) Refresh(ctx context.Context, id string, data *T) error {
	key := c.buildKey(id)

	// Check if key exists first
	exists, err := c.client.Exists(ctx, key)
	if err != nil {
		return fmt.Errorf("failed to check cache existence: %w", err)
	}

	if exists == 0 {
		return nil // Key doesn't exist, nothing to refresh
	}

	return c.Set(ctx, id, data)
}

// SetWithCustomTTL stores data with a custom TTL
func (c *Cacheable[T]) SetWithCustomTTL(ctx context.Context, id string, data *T, ttl time.Duration) error {
	key := c.buildKey(id)

	bytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	return c.client.Set(ctx, key, string(bytes), ttl)
}

// GetTTL returns remaining TTL for a key
func (c *Cacheable[T]) GetTTL(ctx context.Context, id string) (time.Duration, error) {
	key := c.buildKey(id)
	return c.client.TTL(ctx, key)
}

// CacheStats holds cache operation statistics
type CacheStats struct {
	Hits    int64
	Misses  int64
	Sets    int64
	Deletes int64
}

// CacheWithStats wraps Cacheable with statistics tracking
type CacheWithStats[T any] struct {
	*Cacheable[T]
	stats CacheStats
}

// NewCacheWithStats creates a cache wrapper with statistics
func NewCacheWithStats[T any](client *RedisClient, prefix string, ttl time.Duration) *CacheWithStats[T] {
	return &CacheWithStats[T]{
		Cacheable: NewCacheable[T](client, prefix, ttl),
		stats:     CacheStats{},
	}
}

// GetWithStats retrieves data and updates statistics
func (c *CacheWithStats[T]) GetWithStats(ctx context.Context, id string) (*T, bool, error) {
	data, hit, err := c.Get(ctx, id)
	if err == nil {
		if hit {
			c.stats.Hits++
		} else {
			c.stats.Misses++
		}
	}
	return data, hit, err
}

// SetWithStats stores data and updates statistics
func (c *CacheWithStats[T]) SetWithStats(ctx context.Context, id string, data *T) error {
	c.stats.Sets++
	return c.Set(ctx, id, data)
}

// DeleteWithStats deletes data and updates statistics
func (c *CacheWithStats[T]) DeleteWithStats(ctx context.Context, id string) error {
	c.stats.Deletes++
	return c.Delete(ctx, id)
}

// GetStats returns current cache statistics
func (c *CacheWithStats[T]) GetStats() CacheStats {
	return c.stats
}

// GetHitRate calculates cache hit rate
func (c *CacheWithStats[T]) GetHitRate() float64 {
	total := c.stats.Hits + c.stats.Misses
	if total == 0 {
		return 0
	}
	return float64(c.stats.Hits) / float64(total) * 100
}
