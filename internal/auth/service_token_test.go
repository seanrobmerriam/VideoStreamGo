package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"videostreamgo/internal/config"
)

func TestServiceTokenManager_GenerateAndValidate(t *testing.T) {
	cfg := &config.Config{
		App: struct {
			Environment       string
			Debug             bool
			Port              int
			PlatformJWTSecret string
			InstanceJWTSecret string
			EncryptionKey     string
			ServiceIdentifier string
		}{
			ServiceIdentifier: config.ServiceIdentifierPlatform,
		},
	}

	manager, err := NewServiceTokenManager(cfg, nil)
	require.NoError(t, err)
	assert.NotNil(t, manager)

	// Generate a service token
	ctx := context.Background()
	token, jti, err := manager.GenerateServiceToken(
		ctx,
		config.ServiceIdentifierPlatform,
		config.ServiceIdentifierInstance,
		ScopeCreateInstance,
		"platform-api",
	)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, jti)

	// Validate the token
	claims, err := manager.ValidateServiceToken(ctx, token)
	require.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, config.ServiceIdentifierPlatform, claims.Issuer)
	assert.Equal(t, config.ServiceIdentifierInstance, claims.Audience)
	assert.Equal(t, ScopeCreateInstance, claims.Scope)
}

func TestServiceTokenManager_InvalidToken(t *testing.T) {
	cfg := &config.Config{
		App: struct {
			Environment       string
			Debug             bool
			Port              int
			PlatformJWTSecret string
			InstanceJWTSecret string
			EncryptionKey     string
			ServiceIdentifier string
		}{
			ServiceIdentifier: config.ServiceIdentifierPlatform,
		},
	}

	manager, err := NewServiceTokenManager(cfg, nil)
	require.NoError(t, err)

	ctx := context.Background()

	// Test with invalid token
	_, err = manager.ValidateServiceToken(ctx, "invalid-token")
	assert.Error(t, err)
}

func TestServiceClaims_RequireScope(t *testing.T) {
	claims := &ServiceClaims{
		Scope: "read write create_instance",
	}

	// Valid scope
	err := claims.RequireScope(ScopeCreateInstance)
	assert.NoError(t, err)

	// Scope in claims
	err = claims.RequireScope("read")
	assert.NoError(t, err)

	// Non-granted scope
	err = claims.RequireScope("some_other_scope")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid or insufficient scope")
}

func TestServiceClaims_RequireScope_WithAdminOperations(t *testing.T) {
	// When admin_operations is present, all scopes should be granted
	claims := &ServiceClaims{
		Scope: "read write create_instance admin_operations",
	}

	// Any scope should be granted with admin_operations
	err := claims.RequireScope(ScopeManageInstances)
	assert.NoError(t, err)

	err = claims.RequireScope(ScopeViewMetrics)
	assert.NoError(t, err)

	err = claims.RequireScope(ScopeAdminOperations)
	assert.NoError(t, err)

	err = claims.RequireScope("some_random_scope")
	assert.NoError(t, err)
}

func TestServiceTokenManager_PublicKeyExport(t *testing.T) {
	cfg := &config.Config{
		App: struct {
			Environment       string
			Debug             bool
			Port              int
			PlatformJWTSecret string
			InstanceJWTSecret string
			EncryptionKey     string
			ServiceIdentifier string
		}{
			ServiceIdentifier: config.ServiceIdentifierPlatform,
		},
	}

	manager, err := NewServiceTokenManager(cfg, nil)
	require.NoError(t, err)

	// Test public key export
	publicKeyPEM := manager.GetPublicKeyPEM()
	assert.NotEmpty(t, publicKeyPEM)

	// Test JWK export
	jwkData := manager.ExportPublicKeyForService()
	assert.NotEmpty(t, jwkData["kty"])
	assert.Equal(t, "RSA", jwkData["kty"])
	assert.NotEmpty(t, jwkData["n"])
	assert.NotEmpty(t, jwkData["e"])
}

func TestGenerateRSAPrivateKey(t *testing.T) {
	privateKey, err := GenerateRSAPrivateKey()
	require.NoError(t, err)
	assert.NotNil(t, privateKey)

	// Test PEM conversion
	pem := RSAPrivateKeyToPEM(privateKey)
	assert.NotEmpty(t, pem)

	// Test PEM parsing
	parsedKey, err := PEMToRSAPrivateKey(pem)
	require.NoError(t, err)
	assert.Equal(t, privateKey.D.String(), parsedKey.D.String())
}

func TestRSAPublicKeyConversion(t *testing.T) {
	privateKey, err := GenerateRSAPrivateKey()
	require.NoError(t, err)

	// Test public key PEM conversion
	publicKeyPEM, err := RSAPublicKeyToPEM(&privateKey.PublicKey)
	require.NoError(t, err)
	assert.NotEmpty(t, publicKeyPEM)

	// Test PEM parsing
	parsedKey, err := PEMToRSAPublicKey(publicKeyPEM)
	require.NoError(t, err)
	assert.Equal(t, privateKey.PublicKey.N.String(), parsedKey.N.String())
}

func TestServiceTokenExpiry(t *testing.T) {
	// Verify the expiry constant
	assert.Equal(t, 5*time.Minute, ServiceTokenExpiry)
	assert.Equal(t, 4*time.Minute, ServiceTokenCacheExpiry)
}
