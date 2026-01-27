package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"videostreamgo/internal/config"
)

type testUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// setupTestRedis creates a miniredis instance for testing
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *RedisClient) {
	mr, err := miniredis.Run()
	require.NoError(t, err)

	cfg := &config.Config{
		Redis: struct {
			Host     string
			Port     int
			Password string
			Database int
			PoolSize int
		}{
			Host:     mr.Host(),
			Port:     6379,
			Password: "",
			Database: 0,
			PoolSize: 10,
		},
	}

	client := &RedisClient{
		client:    redis.NewClient(&redis.Options{Addr: mr.Addr()}),
		cfg:       cfg,
		connected: true,
	}

	return mr, client
}

func TestCacheable_GetSet(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	cacheable := NewCacheable[testUser](client, "user", TTLUser)

	// Test Set
	user := &testUser{ID: "123", Name: "John Doe", Email: "john@example.com"}
	err := cacheable.Set(ctx, "123", user)
	assert.NoError(t, err)

	// Test Get
	retrieved, hit, err := cacheable.Get(ctx, "123")
	assert.NoError(t, err)
	assert.True(t, hit)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "123", retrieved.ID)
	assert.Equal(t, "John Doe", retrieved.Name)
	assert.Equal(t, "john@example.com", retrieved.Email)
}

func TestCacheable_GetMiss(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	cacheable := NewCacheable[testUser](client, "user", TTLUser)

	// Test Get non-existent key
	retrieved, hit, err := cacheable.Get(ctx, "non_existent")
	assert.NoError(t, err)
	assert.False(t, hit)
	assert.Nil(t, retrieved)
}

func TestCacheable_Delete(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	cacheable := NewCacheable[testUser](client, "user", TTLUser)

	// Set a value
	user := &testUser{ID: "456", Name: "Jane Doe", Email: "jane@example.com"}
	err := cacheable.Set(ctx, "456", user)
	require.NoError(t, err)

	// Delete
	err = cacheable.Delete(ctx, "456")
	assert.NoError(t, err)

	// Verify it's gone
	retrieved, hit, err := cacheable.Get(ctx, "456")
	assert.NoError(t, err)
	assert.False(t, hit)
	assert.Nil(t, retrieved)
}

func TestCacheable_GetOrFetch(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	cacheable := NewCacheable[testUser](client, "user", TTLUser)

	fetchCount := 0
	fetcher := func(ctx context.Context) (*testUser, error) {
		fetchCount++
		return &testUser{ID: "789", Name: "Fetched User", Email: "fetched@example.com"}, nil
	}

	// First call - should fetch from source
	result, err := cacheable.GetOrFetch(ctx, "789", fetcher)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, fetchCount)
	assert.Equal(t, "Fetched User", result.Name)

	// Second call - should hit cache
	result, err = cacheable.GetOrFetch(ctx, "789", fetcher)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, fetchCount) // Still 1, no additional fetch
}

func TestCacheable_Refresh(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	cacheable := NewCacheable[testUser](client, "user", TTLUser)

	// Set initial value
	original := &testUser{ID: "refresh", Name: "Original", Email: "original@example.com"}
	err := cacheable.Set(ctx, "refresh", original)
	require.NoError(t, err)

	// Refresh with new value
	updated := &testUser{ID: "refresh", Name: "Updated", Email: "updated@example.com"}
	err = cacheable.Refresh(ctx, "refresh", updated)
	assert.NoError(t, err)

	// Verify updated value
	retrieved, hit, err := cacheable.Get(ctx, "refresh")
	assert.NoError(t, err)
	assert.True(t, hit)
	assert.Equal(t, "Updated", retrieved.Name)
}

func TestCacheable_VersionedKeys(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	cacheable := NewCacheable[testUser](client, "user", TTLUser)

	// Set a value
	user := &testUser{ID: "v1_test", Name: "Versioned", Email: "versioned@example.com"}
	err := cacheable.Set(ctx, "v1_test", user)
	require.NoError(t, err)

	// Verify key format (should be user:user:v1_test based on buildKey implementation)
	keys, err := client.Keys(ctx, "user:user:*")
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Equal(t, "user:user:v1_test", keys[0])
}

func TestCacheWithStats(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	cacheable := NewCacheWithStats[testUser](client, "stats_user", TTLUser)

	// Initial stats
	stats := cacheable.GetStats()
	assert.Equal(t, int64(0), stats.Hits)
	assert.Equal(t, int64(0), stats.Misses)
	assert.Equal(t, int64(0), stats.Sets)

	// Set
	user := &testUser{ID: "stats_1", Name: "Stats User", Email: "stats@example.com"}
	err := cacheable.SetWithStats(ctx, "stats_1", user)
	assert.NoError(t, err)

	stats = cacheable.GetStats()
	assert.Equal(t, int64(1), stats.Sets)

	// Miss
	_, hit, err := cacheable.GetWithStats(ctx, "non_existent")
	assert.NoError(t, err)
	assert.False(t, hit)

	stats = cacheable.GetStats()
	assert.Equal(t, int64(1), stats.Misses)

	// Hit
	retrieved, hit, err := cacheable.GetWithStats(ctx, "stats_1")
	assert.NoError(t, err)
	assert.True(t, hit)
	assert.NotNil(t, retrieved)

	stats = cacheable.GetStats()
	assert.Equal(t, int64(1), stats.Hits)

	// Hit rate
	hitRate := cacheable.GetHitRate()
	assert.Equal(t, 50.0, hitRate)
}

func TestCacheable_TTLPresets(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	ctx := context.Background()

	// Test user TTL (5 minutes)
	userCache := NewCacheable[testUser](client, "user", TTLUser)
	user := &testUser{ID: "ttl_test", Name: "TTL Test", Email: "ttl@test.com"}
	err := userCache.Set(ctx, "ttl_test", user)
	require.NoError(t, err)

	ttl, err := userCache.GetTTL(ctx, "ttl_test")
	assert.NoError(t, err)
	assert.True(t, ttl > 4*time.Minute)
	assert.True(t, ttl <= 5*time.Minute)

	// Test config TTL (15 minutes)
	configCache := NewCacheable[testUser](client, "config", TTLConfig)
	err = configCache.Set(ctx, "config_1", user)
	require.NoError(t, err)

	ttl, err = configCache.GetTTL(ctx, "config_1")
	assert.NoError(t, err)
	assert.True(t, ttl > 14*time.Minute)
	assert.True(t, ttl <= 15*time.Minute)

	// Test static TTL (1 hour)
	staticCache := NewCacheable[testUser](client, "static", TTLStatic)
	err = staticCache.Set(ctx, "static_1", user)
	require.NoError(t, err)

	ttl, err = staticCache.GetTTL(ctx, "static_1")
	assert.NoError(t, err)
	assert.True(t, ttl > 59*time.Minute)
	assert.True(t, ttl <= 60*time.Minute)
}
