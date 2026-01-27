package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SessionData represents session data stored in Redis
type SessionData struct {
	UserID     string            `json:"user_id"`
	SessionID  string            `json:"session_id"`
	InstanceID string            `json:"instance_id,omitempty"`
	Email      string            `json:"email,omitempty"`
	Username   string            `json:"username,omitempty"`
	Role       string            `json:"role,omitempty"`
	IPAddress  string            `json:"ip_address,omitempty"`
	UserAgent  string            `json:"user_agent,omitempty"`
	CreatedAt  time.Time         `json:"created_at"`
	LastAccess time.Time         `json:"last_access"`
	ExpiresAt  time.Time         `json:"expires_at"`
	ExtraData  map[string]string `json:"extra_data,omitempty"`
}

// SessionStore handles Redis-backed session management
type SessionStore struct {
	client *RedisClient
}

// NewSessionStore creates a new session store
func NewSessionStore(client *RedisClient) *SessionStore {
	return &SessionStore{
		client: client,
	}
}

// sessionKey generates a Redis key for sessions
func (s *SessionStore) sessionKey(userID, sessionID string) string {
	return fmt.Sprintf("session:%s:%s", userID, sessionID)
}

// userSessionsKey generates a Redis key for user's session index
func (s *SessionStore) userSessionsKey(userID string) string {
	return fmt.Sprintf("user_sessions:%s", userID)
}

// Create creates a new session with atomic set-if-not-exists
func (s *SessionStore) Create(ctx context.Context, session *SessionData) error {
	if session.SessionID == "" {
		session.SessionID = uuid.New().String()
	}
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	session.LastAccess = session.CreatedAt

	// Calculate expiration time (default 24 hours from now)
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = session.CreatedAt.Add(24 * time.Hour)
	}

	// Serialize session data
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	// Calculate TTL
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("session expiration must be in the future")
	}

	// Atomic set-if-not-exists (SETNX)
	key := s.sessionKey(session.UserID, session.SessionID)
	created, err := s.client.SetNX(ctx, key, data, ttl)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	if !created {
		return fmt.Errorf("session already exists")
	}

	// Add to user's session index (for cleanup/enumeration)
	indexKey := s.userSessionsKey(session.UserID)
	indexTTL := 7 * 24 * time.Hour // Keep index longer than individual sessions
	if err := s.client.ZAdd(ctx, indexKey, redis.Z{
		Score:  float64(time.Now().Unix()),
		Member: session.SessionID,
	}); err != nil {
		// Log but don't fail - index is for convenience
		return nil
	}
	s.client.client.Expire(ctx, indexKey, indexTTL)

	return nil
}

// Get retrieves a session by user ID and session ID
func (s *SessionStore) Get(ctx context.Context, userID, sessionID string) (*SessionData, error) {
	key := s.sessionKey(userID, sessionID)
	data, err := s.client.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if data == "" {
		return nil, nil // Session not found
	}

	var session SessionData
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// Refresh extends the session TTL
func (s *SessionStore) Refresh(ctx context.Context, userID, sessionID string, extendBy time.Duration) error {
	key := s.sessionKey(userID, sessionID)

	// Get current session to verify it exists
	session, err := s.Get(ctx, userID, sessionID)
	if err != nil {
		return err
	}
	if session == nil {
		return fmt.Errorf("session not found")
	}

	// Update expiration time
	newExpiry := session.ExpiresAt.Add(extendBy)
	if newExpiry.Sub(session.CreatedAt) > 7*24*time.Hour {
		// Cap at 7 days
		newExpiry = session.CreatedAt.Add(7 * 24 * time.Hour)
	}

	// Update the session
	session.ExpiresAt = newExpiry
	session.LastAccess = time.Now()

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	ttl := time.Until(session.ExpiresAt)
	return s.client.Set(ctx, key, data, ttl)
}

// Invalidate removes a session (logout)
func (s *SessionStore) Invalidate(ctx context.Context, userID, sessionID string) error {
	key := s.sessionKey(userID, sessionID)

	// Delete the session
	if err := s.client.Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	// Remove from user's session index
	indexKey := s.userSessionsKey(userID)
	s.client.client.ZRem(ctx, indexKey, sessionID)

	return nil
}

// InvalidateAll removes all sessions for a user
func (s *SessionStore) InvalidateAll(ctx context.Context, userID string) error {
	// Get all session IDs for user
	indexKey := s.userSessionsKey(userID)
	sessionIDs, err := s.client.client.ZRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	// Delete all sessions
	for _, sessionID := range sessionIDs {
		key := s.sessionKey(userID, sessionID)
		if err := s.client.Delete(ctx, key); err != nil {
			continue // Continue deleting other sessions
		}
	}

	// Delete the index
	return s.client.Delete(ctx, indexKey)
}

// GetUserSessions retrieves all active sessions for a user
func (s *SessionStore) GetUserSessions(ctx context.Context, userID string) ([]*SessionData, error) {
	indexKey := s.userSessionsKey(userID)
	sessionIDs, err := s.client.client.ZRange(ctx, indexKey, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	var sessions []*SessionData
	for _, sessionID := range sessionIDs {
		session, err := s.Get(ctx, userID, sessionID)
		if err != nil || session == nil {
			continue // Skip invalid sessions
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// ValidateSession validates a session is still active
func (s *SessionStore) ValidateSession(ctx context.Context, userID, sessionID string) (bool, error) {
	key := s.sessionKey(userID, sessionID)
	exists, err := s.client.Exists(ctx, key)
	if err != nil {
		return false, fmt.Errorf("failed to validate session: %w", err)
	}
	return exists > 0, nil
}
