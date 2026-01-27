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

// setupMockRedis creates a miniredis instance for testing
func setupMockRedis(t *testing.T) (*miniredis.Miniredis, *RedisClient) {
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

func TestRedisClient_SetAndGet(t *testing.T) {
	mr, client := setupMockRedis(t)
	defer mr.Close()

	ctx := context.Background()

	// Test Set
	err := client.Set(ctx, "test_key", "test_value", time.Hour)
	assert.NoError(t, err)

	// Test Get
	val, err := client.Get(ctx, "test_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_value", val)

	// Test Get non-existent key
	val, err = client.Get(ctx, "non_existent")
	assert.NoError(t, err)
	assert.Empty(t, val)
}

func TestRedisClient_Delete(t *testing.T) {
	mr, client := setupMockRedis(t)
	defer mr.Close()

	ctx := context.Background()

	// Set a value
	err := client.Set(ctx, "delete_key", "delete_value", time.Hour)
	require.NoError(t, err)

	// Verify it exists
	val, err := client.Get(ctx, "delete_key")
	assert.NoError(t, err)
	assert.Equal(t, "delete_value", val)

	// Delete
	err = client.Delete(ctx, "delete_key")
	assert.NoError(t, err)

	// Verify it's gone
	val, err = client.Get(ctx, "delete_key")
	assert.NoError(t, err)
	assert.Empty(t, val)
}

func TestRedisClient_Exists(t *testing.T) {
	mr, client := setupMockRedis(t)
	defer mr.Close()

	ctx := context.Background()

	// Set a value
	err := client.Set(ctx, "exists_key", "exists_value", time.Hour)
	require.NoError(t, err)

	// Check exists
	count, err := client.Exists(ctx, "exists_key")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)

	// Check non-existent
	count, err = client.Exists(ctx, "non_existent")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

func TestRedisClient_Incr(t *testing.T) {
	mr, client := setupMockRedis(t)
	defer mr.Close()

	ctx := context.Background()

	// Test increment
	result, err := client.Incr(ctx, "counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), result)

	result, err = client.Incr(ctx, "counter")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), result)

	// Test increment by amount
	result, err = client.IncrBy(ctx, "counter", 5)
	assert.NoError(t, err)
	assert.Equal(t, int64(7), result)
}

func TestRedisClient_TTL(t *testing.T) {
	mr, client := setupMockRedis(t)
	defer mr.Close()

	ctx := context.Background()

	// Set with TTL
	err := client.Set(ctx, "ttl_key", "ttl_value", 5*time.Minute)
	require.NoError(t, err)

	// Check TTL
	ttl, err := client.TTL(ctx, "ttl_key")
	assert.NoError(t, err)
	assert.True(t, ttl > 0)
	assert.True(t, ttl <= 5*time.Minute)
}

func TestRedisClient_Health(t *testing.T) {
	mr, client := setupMockRedis(t)
	defer mr.Close()

	ctx := context.Background()

	health := client.Health(ctx)
	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, true, health["connected"])
	assert.Nil(t, health["error"])
}

func TestRedisClient_HealthUnhealthy(t *testing.T) {
	// Create client that will fail to connect
	cfg := &config.Config{
		Redis: struct {
			Host     string
			Port     int
			Password string
			Database int
			PoolSize int
		}{
			Host:     "localhost",
			Port:     9999, // Invalid port
			Password: "",
			Database: 0,
			PoolSize: 10,
		},
	}

	client := &RedisClient{
		client:    redis.NewClient(&redis.Options{Addr: "localhost:9999"}),
		cfg:       cfg,
		connected: false,
	}

	ctx := context.Background()
	health := client.Health(ctx)
	assert.Equal(t, "unhealthy", health["status"])
}
