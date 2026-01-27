package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"videostreamgo/internal/cache"
	"videostreamgo/internal/config"
)

// Service token constants
const (
	ServiceTokenExpiry      = 5 * time.Minute
	ServiceTokenCacheExpiry = 4 * time.Minute // Slightly less than expiry to ensure refresh
)

// Service token scopes
const (
	ScopeCreateInstance  = "create_instance"
	ScopeManageInstances = "manage_instances"
	ScopeViewMetrics     = "view_metrics"
	ScopeAdminOperations = "admin_operations"
)

// ErrInvalidToken is returned when a service token is invalid
var ErrInvalidToken = errors.New("invalid service token")

// ErrInvalidScope is returned when the token doesn't have the required scope
var ErrInvalidScope = errors.New("invalid or insufficient scope")

// ErrTokenExpired is returned when a service token has expired
var ErrTokenExpired = errors.New("service token has expired")

// ServiceClaims represents JWT claims for service-to-service authentication
type ServiceClaims struct {
	jwt.RegisteredClaims
	Issuer   string `json:"iss"`
	Audience string `json:"aud"`
	Scope    string `json:"scope"`
	Service  string `json:"service"`
	JTI      string `json:"jti"`
}

// ServiceTokenManager handles service-to-service JWT authentication
type ServiceTokenManager struct {
	privateKey   *rsa.PrivateKey
	publicKey    *rsa.PublicKey
	publicKeyPEM string
	cfg          *config.Config
	redis        *cache.RedisClient
}

// NewServiceTokenManager creates a new service token manager
func NewServiceTokenManager(cfg *config.Config, redisClient *cache.RedisClient) (*ServiceTokenManager, error) {
	manager := &ServiceTokenManager{
		cfg:   cfg,
		redis: redisClient,
	}

	// Try to load existing keys from config/environment
	privateKeyPEM := cfg.App.EncryptionKey // Using encryption key as fallback for key storage
	if privateKeyPEM == "" || len(privateKeyPEM) < 2048 {
		// Generate new RSA key pair
		if err := manager.generateKeyPair(); err != nil {
			return nil, fmt.Errorf("failed to generate service token key pair: %w", err)
		}
	} else {
		// Load existing key (simplified - in production, use proper key storage)
		if err := manager.loadOrGenerateKeyPair(); err != nil {
			return nil, fmt.Errorf("failed to load service token key pair: %w", err)
		}
	}

	return manager, nil
}

// generateKeyPair generates a new RSA-2048 key pair for service tokens
func (m *ServiceTokenManager) generateKeyPair() error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	m.privateKey = privateKey
	m.publicKey = &privateKey.PublicKey

	// Encode public key to PEM format for sharing
	m.publicKeyPEM = m.encodePublicKeyToPEM()
	return nil
}

// loadOrGenerateKeyPair loads existing keys or generates new ones
func (m *ServiceTokenManager) loadOrGenerateKeyPair() error {
	// For now, generate new keys (in production, load from secure storage)
	return m.generateKeyPair()
}

// encodePublicKeyToPEM encodes the public key to PEM format
func (m *ServiceTokenManager) encodePublicKeyToPEM() string {
	pubBytes, err := x509.MarshalPKIXPublicKey(m.publicKey)
	if err != nil {
		return ""
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}

	return string(pem.EncodeToMemory(block))
}

// GetPublicKeyPEM returns the PEM-encoded public key
func (m *ServiceTokenManager) GetPublicKeyPEM() string {
	return m.publicKeyPEM
}

// GenerateServiceToken generates a new service token for inter-service calls
func (m *ServiceTokenManager) GenerateServiceToken(
	ctx context.Context,
	issuer string,
	audience string,
	scope string,
	service string,
) (string, string, error) {
	jti := uuid.New().String()

	claims := ServiceClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ServiceTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        jti,
		},
		Issuer:   issuer,
		Audience: audience,
		Scope:    scope,
		Service:  service,
		JTI:      jti,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(m.privateKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to sign service token: %w", err)
	}

	// Cache the token in Redis
	if m.redis != nil {
		cacheKey := fmt.Sprintf("service_token:%s", jti)
		if err := m.redis.Set(ctx, cacheKey, tokenString, ServiceTokenCacheExpiry); err != nil {
			log.Printf("Warning: failed to cache service token: %v", err)
		}
	}

	return tokenString, jti, nil
}

// ValidateServiceToken validates a service token and returns its claims
func (m *ServiceTokenManager) ValidateServiceToken(ctx context.Context, tokenString string) (*ServiceClaims, error) {
	// First check if token is revoked
	token, err := jwt.ParseWithClaims(tokenString, &ServiceClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*ServiceClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Check if token is in revocation list
	if m.redis != nil {
		cacheKey := fmt.Sprintf("service_token:%s", claims.JTI)
		cachedToken, err := m.redis.Get(ctx, cacheKey)
		if err == nil && cachedToken == "" {
			// Token was revoked or expired
			return nil, ErrTokenExpired
		}
	}

	return claims, nil
}

// RequireScope checks if the claims contain the required scope
func (c *ServiceClaims) RequireScope(requiredScope string) error {
	scopes := strings.Split(c.Scope, " ")
	for _, s := range scopes {
		if s == requiredScope || s == ScopeAdminOperations {
			return nil
		}
	}
	return fmt.Errorf("%w: required '%s', got '%s'", ErrInvalidScope, requiredScope, c.Scope)
}

// RevokeServiceToken revokes a service token by its JTI
func (m *ServiceTokenManager) RevokeServiceToken(ctx context.Context, jti string) error {
	if m.redis == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("service_token:%s", jti)
	// Set with very short expiry to effectively revoke
	return m.redis.Delete(ctx, cacheKey)
}

// ServiceTokenMiddleware creates middleware for validating service tokens
func ServiceTokenMiddleware(manager *ServiceTokenManager, requiredScope string) func(ctx context.Context, tokenString string) error {
	return func(ctx context.Context, tokenString string) error {
		claims, err := manager.ValidateServiceToken(ctx, tokenString)
		if err != nil {
			return err
		}

		return claims.RequireScope(requiredScope)
	}
}

// GetServiceTokenCacheKey returns the cache key for a service token
func GetServiceTokenCacheKey(jti string) string {
	return fmt.Sprintf("service_token:%s", jti)
}

// GenerateRSAPrivateKey generates a new RSA private key for service tokens
func GenerateRSAPrivateKey() (*rsa.PrivateKey, error) {
	return rsa.GenerateKey(rand.Reader, 2048)
}

// RSAPrivateKeyToPEM converts an RSA private key to PEM format
func RSAPrivateKeyToPEM(privateKey *rsa.PrivateKey) string {
	bytes := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: bytes,
	}
	return string(pem.EncodeToMemory(block))
}

// PEMToRSAPrivateKey converts PEM format back to RSA private key
func PEMToRSAPrivateKey(pemString string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// RSAPublicKeyToPEM converts an RSA public key to PEM format
func RSAPublicKeyToPEM(publicKey *rsa.PublicKey) (string, error) {
	pubBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", err
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}

	return string(pem.EncodeToMemory(block)), nil
}

// PEMToRSAPublicKey converts PEM format back to RSA public key
func PEMToRSAPublicKey(pemString string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemString))
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaKey, nil
}

// ExportPublicKeyForService exports the public key in a format suitable for other services
func (m *ServiceTokenManager) ExportPublicKeyForService() map[string]interface{} {
	return map[string]interface{}{
		"kty": "RSA",
		"alg": "RS256",
		"use": "sig",
		"n":   base64.RawURLEncoding.EncodeToString(m.publicKey.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(m.publicKey.E)).Bytes()),
	}
}

// MarshalPublicKey marshals the public key to JSON for API responses
func (m *ServiceTokenManager) MarshalPublicKey() (string, error) {
	data := m.ExportPublicKeyForService()
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
