package auth

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"

	"videostreamgo/internal/cache"
	"videostreamgo/internal/config"
)

// Token revocation constants
const (
	RevokedTokensByUser = "revoked_tokens:user:%s"
	RevokedTokensByJTI  = "revoked_tokens:jti:%s"
	TokenRotationPrefix = "token_rotation:user:%s"
	RefreshTokenPrefix  = "refresh_token:%s"
)

// TokenManager handles token lifecycle including revocation and refresh
type TokenManager struct {
	redis *cache.RedisClient
	cfg   *config.Config
}

// NewTokenManager creates a new token manager
func NewTokenManager(redisClient *cache.RedisClient, cfg *config.Config) *TokenManager {
	return &TokenManager{
		redis: redisClient,
		cfg:   cfg,
	}
}

// RevokeToken revokes a token by its JTI
func (m *TokenManager) RevokeToken(ctx context.Context, jti string, reason string) error {
	if m.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf(RevokedTokensByJTI, jti)
	// Store revocation info with timestamp
	revocationData := fmt.Sprintf("%d:%s", time.Now().Unix(), reason)

	if err := m.redis.Set(ctx, cacheKey, revocationData, 24*time.Hour); err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	log.Printf("[TOKEN] Token revoked: jti=%s reason=%s", jti, reason)
	return nil
}

// IsTokenRevoked checks if a token has been revoked
func (m *TokenManager) IsTokenRevoked(ctx context.Context, jti string) (bool, string, error) {
	if m.redis == nil {
		return false, "", nil
	}

	cacheKey := fmt.Sprintf(RevokedTokensByJTI, jti)
	data, err := m.redis.Get(ctx, cacheKey)
	if err != nil {
		return false, "", err
	}

	if data == "" {
		return false, "", nil
	}

	return true, data, nil
}

// RevokeAllUserTokens revokes all tokens for a specific user
func (m *TokenManager) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, reason string) error {
	if m.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf(RevokedTokensByUser, userID.String())
	revocationData := fmt.Sprintf("%d:%s", time.Now().Unix(), reason)

	if err := m.redis.Set(ctx, cacheKey, revocationData, 24*time.Hour); err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	log.Printf("[TOKEN] All tokens revoked for user: user_id=%s reason=%s", userID.String(), reason)
	return nil
}

// AreUserTokensRevoked checks if any tokens for a user have been revoked
func (m *TokenManager) AreUserTokensRevoked(ctx context.Context, userID uuid.UUID) (bool, string, error) {
	if m.redis == nil {
		return false, "", nil
	}

	cacheKey := fmt.Sprintf(RevokedTokensByUser, userID.String())
	data, err := m.redis.Get(ctx, cacheKey)
	if err != nil {
		return false, "", err
	}

	if data == "" {
		return false, "", nil
	}

	return true, data, nil
}

// StoreRefreshToken stores a refresh token for a user
func (m *TokenManager) StoreRefreshToken(ctx context.Context, userID uuid.UUID, refreshToken string, expiry time.Duration) error {
	if m.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf(RefreshTokenPrefix, refreshToken)
	userData := fmt.Sprintf("%s:%d", userID.String(), time.Now().Unix())

	if err := m.redis.Set(ctx, cacheKey, userData, expiry); err != nil {
		return fmt.Errorf("failed to store refresh token: %w", err)
	}

	return nil
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func (m *TokenManager) ValidateRefreshToken(ctx context.Context, refreshToken string) (uuid.UUID, error) {
	if m.redis == nil {
		return uuid.Nil, fmt.Errorf("refresh token validation not available")
	}

	cacheKey := fmt.Sprintf(RefreshTokenPrefix, refreshToken)
	data, err := m.redis.Get(ctx, cacheKey)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid refresh token")
	}

	if data == "" {
		return uuid.Nil, fmt.Errorf("refresh token expired or revoked")
	}

	// Parse user ID from stored data (format: userID:timestamp)
	// First 36 chars should be UUID
	return uuid.Parse(data[:36])
}

// RevokeRefreshToken revokes a specific refresh token
func (m *TokenManager) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	if m.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf(RefreshTokenPrefix, refreshToken)
	return m.redis.Delete(ctx, cacheKey)
}

// RecordTokenRotation records when a user's tokens were rotated (password change, etc.)
func (m *TokenManager) RecordTokenRotation(ctx context.Context, userID uuid.UUID) error {
	if m.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf(TokenRotationPrefix, userID.String())
	if err := m.redis.Set(ctx, cacheKey, time.Now().Unix(), 24*time.Hour); err != nil {
		return fmt.Errorf("failed to record token rotation: %w", err)
	}

	return nil
}

// GetLastTokenRotation returns the timestamp of the last token rotation for a user
func (m *TokenManager) GetLastTokenRotation(ctx context.Context, userID uuid.UUID) (time.Time, error) {
	if m.redis == nil {
		return time.Time{}, nil
	}

	cacheKey := fmt.Sprintf(TokenRotationPrefix, userID.String())
	data, err := m.redis.Get(ctx, cacheKey)
	if err != nil || data == "" {
		return time.Time{}, nil // No rotation recorded
	}

	timestamp, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return time.Time{}, nil
	}

	return time.Unix(timestamp, 0), nil
}

// ValidateTokenNotRotated validates that a token was issued after the last rotation
func (m *TokenManager) ValidateTokenNotRotated(ctx context.Context, userID uuid.UUID, tokenIssuedAt time.Time) (bool, error) {
	lastRotation, err := m.GetLastTokenRotation(ctx, userID)
	if err != nil {
		return false, err
	}

	if lastRotation.IsZero() {
		return true, nil // No rotation recorded, token is valid
	}

	return tokenIssuedAt.After(lastRotation), nil
}

// CleanupExpiredRevocations removes expired revocation entries
func (m *TokenManager) CleanupExpiredRevocations(ctx context.Context) error {
	if m.redis == nil {
		return nil
	}

	// This would require scanning keys - in production, use a more efficient method
	log.Println("[TOKEN] Revocation cleanup not implemented - using for TTL automatic expiry")
	return nil
}

// TokenInfo represents information about a token
type TokenInfo struct {
	JTI       string     `json:"jti"`
	UserID    string     `json:"user_id"`
	IssuedAt  time.Time  `json:"issued_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	Revoked   bool       `json:"revoked"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}

// GetTokenInfo retrieves information about a token (for debugging/admin)
func (m *TokenManager) GetTokenInfo(ctx context.Context, jti string) (*TokenInfo, error) {
	info := &TokenInfo{JTI: jti}

	// Check if revoked
	revoked, revocationData, err := m.IsTokenRevoked(ctx, jti)
	if err != nil {
		return nil, err
	}

	if revoked {
		info.Revoked = true
		timestamp, _ := strconv.ParseInt(revocationData[:10], 10, 64)
		revokedAt := time.Unix(timestamp, 0)
		info.RevokedAt = &revokedAt
	}

	return info, nil
}
