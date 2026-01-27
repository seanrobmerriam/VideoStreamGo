package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"videostreamgo/internal/config"
)

// Default timeout for Redis operations
const DefaultTimeout = 5 * time.Second

// RedisClient wraps the go-redis client with connection pooling and retry logic
type RedisClient struct {
	client    *redis.Client
	cfg       *config.Config
	connected bool
}

// NewRedisClient creates a new Redis client with connection pooling
func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:            fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password:        cfg.Redis.Password,
		DB:              cfg.Redis.Database,
		PoolSize:        cfg.Redis.PoolSize,
		MinIdleConns:    10,
		MaxRetries:      3,
		MinRetryBackoff: 100 * time.Millisecond,
		MaxRetryBackoff: 500 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		PoolTimeout:     10 * time.Second,
	})

	return &RedisClient{
		client: client,
		cfg:    cfg,
	}, nil
}

// Connect establishes connection with exponential backoff retry
func (r *RedisClient) Connect(ctx context.Context) error {
	// Test connection with retry logic
	maxRetries := 5
	baseDelay := 100 * time.Millisecond

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := r.client.Ping(ctx).Err(); err != nil {
			lastErr = err
			delay := baseDelay * time.Duration(1<<attempt) // Exponential backoff
			log.Printf("Redis connection attempt %d failed: %v, retrying in %v", attempt+1, err, delay)
			select {
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during Redis connection: %w", ctx.Err())
			case <-time.After(delay):
				continue
			}
		}
		r.connected = true
		log.Printf("Redis connected successfully to %s:%d", r.cfg.Redis.Host, r.cfg.Redis.Port)
		return nil
	}

	return fmt.Errorf("failed to connect to Redis after %d attempts: %w", maxRetries, lastErr)
}

// Ping checks if Redis is responsive
func (r *RedisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Health returns health status of Redis connection
func (r *RedisClient) Health(ctx context.Context) map[string]interface{} {
	pingErr := r.Ping(ctx)
	status := "healthy"
	if pingErr != nil {
		status = "unhealthy"
	}

	return map[string]interface{}{
		"status":    status,
		"connected": r.connected,
		"error":     pingErr,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
}

// GetClient returns the underlying redis client for advanced operations
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// Close gracefully shuts down the Redis connection
func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// Set stores a key-value pair with optional TTL
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.Set(ctx, key, value, ttl).Err()
}

// Get retrieves a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// Delete removes a key
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if keys exist
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.Exists(ctx, keys...).Result()
}

// Incr increments a key
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.Incr(ctx, key).Result()
}

// IncrBy increments a key by amount
func (r *RedisClient) IncrBy(ctx context.Context, key string, amount int64) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.IncrBy(ctx, key, amount).Result()
}

// TTL returns the remaining time to live of a key
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.TTL(ctx, key).Result()
}

// ZAdd adds a member to a sorted set
func (r *RedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.ZAdd(ctx, key, members...).Err()
}

// ZRemRangeByScore removes members by score range
func (r *RedisClient) ZRemRangeByScore(ctx context.Context, key, min, max string) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.ZRemRangeByScore(ctx, key, min, max).Err()
}

// ZRange returns a range of members from a sorted set
func (r *RedisClient) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.ZRange(ctx, key, start, stop).Result()
}

// ZCard returns the cardinality of a sorted set
func (r *RedisClient) ZCard(ctx context.Context, key string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.ZCard(ctx, key).Result()
}

// HSet sets field in a hash
func (r *RedisClient) HSet(ctx context.Context, key string, field string, value interface{}) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.HSet(ctx, key, field, value).Err()
}

// HGet gets a field from a hash
func (r *RedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.HGet(ctx, key, field).Result()
}

// HGetAll gets all fields from a hash
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.HGetAll(ctx, key).Result()
}

// HDel deletes fields from a hash
func (r *RedisClient) HDel(ctx context.Context, key string, fields ...string) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.HDel(ctx, key, fields...).Err()
}

// SetNX sets a value if it doesn't exist (atomic)
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.SetNX(ctx, key, value, expiration).Result()
}

// Keys returns all keys matching a pattern
func (r *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()

	return r.client.Keys(ctx, pattern).Result()
}
