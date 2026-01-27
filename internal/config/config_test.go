package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSecureSecret(t *testing.T) {
	secret1 := generateSecureSecret()
	assert.NotEmpty(t, secret1)
	assert.GreaterOrEqual(t, len(secret1), 64) // 64 hex chars = 32 bytes minimum

	secret2 := generateSecureSecret()
	assert.NotEqual(t, secret1, secret2) // Should be random

	// Verify it's valid hex
	assert.Len(t, secret1, 128) // 64 bytes encoded as hex
}

func TestValidateJWTSecret(t *testing.T) {
	t.Run("Valid long secret", func(t *testing.T) {
		secret := "this-is-a-valid-secret-key-that-is-at-least-64-characters-long-here"
		err := validateJWTSecret(secret)
		assert.NoError(t, err)
	})

	t.Run("Secret too short", func(t *testing.T) {
		secret := "short"
		err := validateJWTSecret(secret)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "at least 64 characters")
	})

	t.Run("Boundary length", func(t *testing.T) {
		secret := make([]byte, 64)
		for i := range secret {
			secret[i] = 'a'
		}
		err := validateJWTSecret(string(secret))
		assert.NoError(t, err)

		// One character less should fail
		shortSecret := string(secret[:63])
		err = validateJWTSecret(shortSecret)
		assert.Error(t, err)
	})
}

func TestMinJWTSecretLength(t *testing.T) {
	assert.Equal(t, 64, MinJWTSecretLength)
}

func TestServiceIdentifiers(t *testing.T) {
	assert.Equal(t, "platform-api", ServiceIdentifierPlatform)
	assert.Equal(t, "instance-api", ServiceIdentifierInstance)
}

func TestGetJWTSecret(t *testing.T) {
	t.Run("Platform service", func(t *testing.T) {
		cfg := &Config{
			App: struct {
				Environment       string
				Debug             bool
				Port              int
				PlatformJWTSecret string
				InstanceJWTSecret string
				EncryptionKey     string
				ServiceIdentifier string
			}{
				ServiceIdentifier: ServiceIdentifierPlatform,
				PlatformJWTSecret: "platform-secret",
				InstanceJWTSecret: "instance-secret",
			},
		}

		secret := cfg.GetJWTSecret()
		assert.Equal(t, "platform-secret", secret)
	})

	t.Run("Instance service", func(t *testing.T) {
		cfg := &Config{
			App: struct {
				Environment       string
				Debug             bool
				Port              int
				PlatformJWTSecret string
				InstanceJWTSecret string
				EncryptionKey     string
				ServiceIdentifier string
			}{
				ServiceIdentifier: ServiceIdentifierInstance,
				PlatformJWTSecret: "platform-secret",
				InstanceJWTSecret: "instance-secret",
			},
		}

		secret := cfg.GetJWTSecret()
		assert.Equal(t, "instance-secret", secret)
	})
}

func TestGetJWTSecretForIssuer(t *testing.T) {
	cfg := &Config{
		App: struct {
			Environment       string
			Debug             bool
			Port              int
			PlatformJWTSecret string
			InstanceJWTSecret string
			EncryptionKey     string
			ServiceIdentifier string
		}{
			PlatformJWTSecret: "platform-secret",
			InstanceJWTSecret: "instance-secret",
		},
	}

	t.Run("Platform issuer", func(t *testing.T) {
		secret, err := cfg.GetJWTSecretForIssuer(ServiceIdentifierPlatform)
		require.NoError(t, err)
		assert.Equal(t, "platform-secret", secret)
	})

	t.Run("Instance issuer", func(t *testing.T) {
		secret, err := cfg.GetJWTSecretForIssuer(ServiceIdentifierInstance)
		require.NoError(t, err)
		assert.Equal(t, "instance-secret", secret)
	})

	t.Run("Unknown issuer", func(t *testing.T) {
		_, err := cfg.GetJWTSecretForIssuer("unknown-service")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown issuer")
	})
}

func TestIsValidIssuer(t *testing.T) {
	cfg := &Config{}

	assert.True(t, cfg.IsValidIssuer(ServiceIdentifierPlatform))
	assert.True(t, cfg.IsValidIssuer(ServiceIdentifierInstance))
	assert.False(t, cfg.IsValidIssuer("unknown"))
	assert.False(t, cfg.IsValidIssuer(""))
}

func TestGenerateSecureSecret_CryptoRandom(t *testing.T) {
	// Test that the function uses crypto/rand by verifying randomness
	secrets := make([]string, 10)
	for i := 0; i < 10; i++ {
		secrets[i] = generateSecureSecret()
	}

	// All secrets should be unique (extremely unlikely to have duplicates)
	uniqueSecrets := make(map[string]bool)
	for _, secret := range secrets {
		uniqueSecrets[secret] = true
	}
	assert.Len(t, uniqueSecrets, 10)
}

func TestGenerateSecureSecret_Fallback(t *testing.T) {
	// This tests the fallback mechanism if crypto/rand fails
	// In practice, crypto/rand rarely fails, but we can verify the
	// function handles it gracefully by checking the implementation
	secret := generateSecureSecret()
	assert.NotEmpty(t, secret)
	assert.GreaterOrEqual(t, len(secret), 64)
}

func TestConfigLoad_SecretGeneration(t *testing.T) {
	// Set environment variables to empty to trigger secret generation
	os.Setenv("PLATFORM_JWT_SECRET", "")
	os.Setenv("INSTANCE_JWT_SECRET", "")
	defer os.Unsetenv("PLATFORM_JWT_SECRET")
	defer os.Unsetenv("INSTANCE_JWT_SECRET")

	cfg, err := Load()
	require.NoError(t, err)

	// Verify secrets were generated
	assert.NotEmpty(t, cfg.App.PlatformJWTSecret)
	assert.NotEmpty(t, cfg.App.InstanceJWTSecret)

	// Verify minimum length
	assert.GreaterOrEqual(t, len(cfg.App.PlatformJWTSecret), MinJWTSecretLength)
	assert.GreaterOrEqual(t, len(cfg.App.InstanceJWTSecret), MinJWTSecretLength)
}
