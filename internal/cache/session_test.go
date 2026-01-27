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

// setupTestRedis creates a miniredis instance for testing
func setupSessionTestRedis(t *testing.T) (*miniredis.Miniredis, *RedisClient) {
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

func TestSessionStore_Create(t *testing.T) {
	mr, client := setupSessionTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	store := NewSessionStore(client)

	session := &SessionData{
		UserID:     "user_123",
		Email:      "test@example.com",
		Username:   "testuser",
		Role:       "admin",
		IPAddress:  "192.168.1.1",
		UserAgent:  "Mozilla/5.0",
		InstanceID: "instance_456",
	}

	err := store.Create(ctx, session)
	assert.NoError(t, err)
	assert.NotEmpty(t, session.SessionID)
	assert.False(t, session.CreatedAt.IsZero())
	assert.False(t, session.ExpiresAt.IsZero())
}

func TestSessionStore_Get(t *testing.T) {
	mr, client := setupSessionTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	store := NewSessionStore(client)

	// Create session
	session := &SessionData{
		UserID:     "user_456",
		Email:      "gettest@example.com",
		Username:   "getuser",
		Role:       "user",
		InstanceID: "instance_789",
	}
	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Get session
	retrieved, err := store.Get(ctx, session.UserID, session.SessionID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, session.UserID, retrieved.UserID)
	assert.Equal(t, session.Email, retrieved.Email)
	assert.Equal(t, session.Username, retrieved.Username)
}

func TestSessionStore_GetNotFound(t *testing.T) {
	mr, client := setupSessionTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	store := NewSessionStore(client)

	// Get non-existent session
	retrieved, err := store.Get(ctx, "nonexistent_user", "nonexistent_session")
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestSessionStore_Refresh(t *testing.T) {
	mr, client := setupSessionTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	store := NewSessionStore(client)

	// Create session
	session := &SessionData{
		UserID:    "user_refresh",
		Email:     "refresh@example.com",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	err := store.Create(ctx, session)
	require.NoError(t, err)

	originalExpiry := session.ExpiresAt

	// Refresh session
	err = store.Refresh(ctx, session.UserID, session.SessionID, 30*time.Minute)
	assert.NoError(t, err)

	// Verify expiry was extended
	retrieved, err := store.Get(ctx, session.UserID, session.SessionID)
	assert.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.True(t, retrieved.ExpiresAt.After(originalExpiry))
}

func TestSessionStore_Invalidate(t *testing.T) {
	mr, client := setupSessionTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	store := NewSessionStore(client)

	// Create session
	session := &SessionData{
		UserID: "user_invalidate",
		Email:  "invalidate@example.com",
	}
	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Invalidate session
	err = store.Invalidate(ctx, session.UserID, session.SessionID)
	assert.NoError(t, err)

	// Verify it's gone
	retrieved, err := store.Get(ctx, session.UserID, session.SessionID)
	assert.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestSessionStore_InvalidateAll(t *testing.T) {
	mr, client := setupSessionTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	store := NewSessionStore(client)

	userID := "user_multi"

	// Create multiple sessions
	for i := 0; i < 3; i++ {
		session := &SessionData{
			UserID: userID,
			Email:  "multi@example.com",
		}
		err := store.Create(ctx, session)
		require.NoError(t, err)
	}

	// Get user sessions
	sessions, err := store.GetUserSessions(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, sessions, 3)

	// Invalidate all
	err = store.InvalidateAll(ctx, userID)
	assert.NoError(t, err)

	// Verify all gone
	sessions, err = store.GetUserSessions(ctx, userID)
	assert.NoError(t, err)
	assert.Len(t, sessions, 0)
}

func TestSessionStore_ValidateSession(t *testing.T) {
	mr, client := setupSessionTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	store := NewSessionStore(client)

	// Create session
	session := &SessionData{
		UserID: "user_validate",
		Email:  "validate@example.com",
	}
	err := store.Create(ctx, session)
	require.NoError(t, err)

	// Validate existing session
	valid, err := store.ValidateSession(ctx, session.UserID, session.SessionID)
	assert.NoError(t, err)
	assert.True(t, valid)

	// Validate non-existent session
	valid, err = store.ValidateSession(ctx, "nonexistent", "nonexistent")
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestSessionStore_CreateDuplicate(t *testing.T) {
	mr, client := setupSessionTestRedis(t)
	defer mr.Close()

	ctx := context.Background()
	store := NewSessionStore(client)

	session := &SessionData{
		UserID:    "user_dup",
		Email:     "dup@example.com",
		SessionID: "fixed_session_id", // Pre-set session ID
	}

	// Create first session
	err := store.Create(ctx, session)
	assert.NoError(t, err)

	// Try to create duplicate with same session ID
	duplicate := &SessionData{
		UserID:    "user_dup",
		Email:     "dup@example.com",
		SessionID: "fixed_session_id",
	}
	err = store.Create(ctx, duplicate)
	assert.Error(t, err)
}
