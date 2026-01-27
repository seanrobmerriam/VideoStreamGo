package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Minimum JWT secret length requirement (64 characters)
const MinJWTSecretLength = 64

// Service identifiers for JWT issuer claims
const (
	ServiceIdentifierPlatform = "platform-api"
	ServiceIdentifierInstance = "instance-api"
)

// Config holds all configuration for the application
type Config struct {
	// Master Database Configuration
	MasterDB struct {
		Host            string
		Port            int
		Username        string
		Password        string
		Database        string
		SSLMode         string
		MaxOpenConns    int
		MaxIdleConns    int
		ConnMaxLifetime time.Duration
	}

	// Instance Database Configuration
	InstanceDB struct {
		Host            string
		Port            int
		Username        string
		Password        string
		DatabasePrefix  string
		SSLMode         string
		MaxOpenConns    int
		MaxIdleConns    int
		ConnMaxLifetime time.Duration
	}

	// Application Configuration
	App struct {
		Environment       string
		Debug             bool
		Port              int
		PlatformJWTSecret string
		InstanceJWTSecret string
		EncryptionKey     string
		ServiceIdentifier string
	}

	// Redis Configuration
	Redis struct {
		Host     string
		Port     int
		Password string
		Database int
		PoolSize int
	}

	// S3 Configuration
	S3 struct {
		Endpoint     string
		AccessKey    string
		SecretKey    string
		Bucket       string
		BucketPrefix string
		Region       string
		UseSSL       bool
		MaxRetries   int
		ChunkSizeMB  int
	}

	// Logging Configuration
	Logging struct {
		Level  string
		Format string
	}

	// Stripe Configuration
	Stripe struct {
		SecretKey      string
		PublishableKey string
		WebhookSecret  string
	}
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{}

	// Master Database Configuration
	cfg.MasterDB.Host = getEnv("MASTER_DB_HOST", "localhost")
	cfg.MasterDB.Port = getEnvInt("MASTER_DB_PORT", 5432)
	cfg.MasterDB.Username = getEnv("MASTER_DB_USER", "videostreamgo")
	cfg.MasterDB.Password = getEnv("MASTER_DB_PASSWORD", "securepassword")
	cfg.MasterDB.Database = getEnv("MASTER_DB_NAME", "videostreamgo_master")
	cfg.MasterDB.SSLMode = getEnv("MASTER_DB_SSLMODE", "require")
	cfg.MasterDB.MaxOpenConns = getEnvInt("MASTER_DB_MAX_OPEN_CONNS", 25)
	cfg.MasterDB.MaxIdleConns = getEnvInt("MASTER_DB_MAX_IDLE_CONNS", 5)
	cfg.MasterDB.ConnMaxLifetime = time.Duration(getEnvInt("MASTER_DB_CONN_MAX_LIFETIME", 300)) * time.Second

	// Instance Database Configuration
	cfg.InstanceDB.Host = getEnv("INSTANCE_DB_HOST", "localhost")
	cfg.InstanceDB.Port = getEnvInt("INSTANCE_DB_PORT", 5432)
	cfg.InstanceDB.Username = getEnv("INSTANCE_DB_USER", "videostreamgo")
	cfg.InstanceDB.Password = getEnv("INSTANCE_DB_PASSWORD", "securepassword")
	cfg.InstanceDB.DatabasePrefix = getEnv("INSTANCE_DB_PREFIX", "instance_")
	cfg.InstanceDB.SSLMode = getEnv("INSTANCE_DB_SSLMODE", "require")
	cfg.InstanceDB.MaxOpenConns = getEnvInt("INSTANCE_DB_MAX_OPEN_CONNS", 25)
	cfg.InstanceDB.MaxIdleConns = getEnvInt("INSTANCE_DB_MAX_IDLE_CONNS", 5)
	cfg.InstanceDB.ConnMaxLifetime = time.Duration(getEnvInt("INSTANCE_DB_CONN_MAX_LIFETIME", 300)) * time.Second

	// Application Configuration
	cfg.App.Environment = getEnv("APP_ENV", "development")
	cfg.App.Debug = getEnvBool("APP_DEBUG", true)
	cfg.App.Port = getEnvInt("APP_PORT", 8080)
	cfg.App.ServiceIdentifier = getEnv("SERVICE_IDENTIFIER", ServiceIdentifierPlatform)

	// Load and validate JWT secrets
	platformSecret := getEnv("PLATFORM_JWT_SECRET", "")
	instanceSecret := getEnv("INSTANCE_JWT_SECRET", "")

	// Generate secure random secrets if not provided
	if platformSecret == "" {
		platformSecret = generateSecureSecret()
		log.Printf("WARNING: PLATFORM_JWT_SECRET not set, generated secure random secret")
	}
	if instanceSecret == "" {
		instanceSecret = generateSecureSecret()
		log.Printf("WARNING: INSTANCE_JWT_SECRET not set, generated secure random secret")
	}

	// Validate secret lengths
	if err := validateJWTSecret(platformSecret); err != nil {
		return nil, fmt.Errorf("invalid PLATFORM_JWT_SECRET: %w", err)
	}
	if err := validateJWTSecret(instanceSecret); err != nil {
		return nil, fmt.Errorf("invalid INSTANCE_JWT_SECRET: %w", err)
	}

	cfg.App.PlatformJWTSecret = platformSecret
	cfg.App.InstanceJWTSecret = instanceSecret
	cfg.App.EncryptionKey = getEnv("ENCRYPTION_KEY", "your-32-byte-encryption-key")

	// Redis Configuration
	cfg.Redis.Host = getEnv("REDIS_HOST", "localhost")
	cfg.Redis.Port = getEnvInt("REDIS_PORT", 6379)
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.Database = getEnvInt("REDIS_DATABASE", 0)
	cfg.Redis.PoolSize = getEnvInt("REDIS_POOL_SIZE", 10)

	// S3 Configuration
	cfg.S3.Endpoint = getEnv("S3_ENDPOINT", "localhost:9000")
	cfg.S3.AccessKey = getEnv("S3_ACCESS_KEY", "minioadmin")
	cfg.S3.SecretKey = getEnv("S3_SECRET_KEY", "minioadmin")
	cfg.S3.Bucket = getEnv("S3_BUCKET", "videostreamgo")
	cfg.S3.BucketPrefix = getEnv("S3_BUCKET_PREFIX", "")
	cfg.S3.Region = getEnv("S3_REGION", "us-east-1")
	cfg.S3.UseSSL = getEnvBool("S3_USE_SSL", false)
	cfg.S3.MaxRetries = getEnvInt("S3_MAX_RETRIES", 3)
	cfg.S3.ChunkSizeMB = getEnvInt("S3_CHUNK_SIZE_MB", 10)

	// Logging Configuration
	cfg.Logging.Level = getEnv("LOG_LEVEL", "debug")
	cfg.Logging.Format = getEnv("LOG_FORMAT", "json")

	// Stripe Configuration
	cfg.Stripe.SecretKey = getEnv("STRIPE_SECRET_KEY", "")
	cfg.Stripe.PublishableKey = getEnv("STRIPE_PUBLISHABLE_KEY", "")
	cfg.Stripe.WebhookSecret = getEnv("STRIPE_WEBHOOK_SECRET", "")

	return cfg, nil
}

// MasterDSN returns the connection string for the master database
func (c *Config) MasterDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.MasterDB.Host,
		c.MasterDB.Port,
		c.MasterDB.Username,
		c.MasterDB.Password,
		c.MasterDB.Database,
		c.MasterDB.SSLMode,
	)
}

// InstanceDSN returns the connection string for an instance database
func (c *Config) InstanceDSN(databaseName string) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.InstanceDB.Host,
		c.InstanceDB.Port,
		c.InstanceDB.Username,
		c.InstanceDB.Password,
		databaseName,
		c.InstanceDB.SSLMode,
	)
}

// GetEnv retrieves environment variable with default value
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// getEnvInt retrieves environment variable as int with default value
func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvBool retrieves environment variable as bool with default value
func getEnvBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// generateSecureSecret generates a cryptographically random secret
func generateSecureSecret() string {
	bytes := make([]byte, 64) // 64 bytes = 128 hex characters
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to time-based if crypto random fails
		return hex.EncodeToString([]byte(time.Now().String()))
	}
	return hex.EncodeToString(bytes)
}

// validateJWTSecret validates that a JWT secret meets minimum requirements
func validateJWTSecret(secret string) error {
	if len(secret) < MinJWTSecretLength {
		return fmt.Errorf("secret must be at least %d characters, got %d", MinJWTSecretLength, len(secret))
	}
	return nil
}

// GetJWTSecret returns the appropriate JWT secret based on service identifier
func (c *Config) GetJWTSecret() string {
	if c.App.ServiceIdentifier == ServiceIdentifierInstance {
		return c.App.InstanceJWTSecret
	}
	return c.App.PlatformJWTSecret
}

// GetJWTSecretForIssuer returns the JWT secret for a specific issuer
func (c *Config) GetJWTSecretForIssuer(issuer string) (string, error) {
	switch issuer {
	case ServiceIdentifierPlatform:
		return c.App.PlatformJWTSecret, nil
	case ServiceIdentifierInstance:
		return c.App.InstanceJWTSecret, nil
	default:
		return "", fmt.Errorf("unknown issuer: %s", issuer)
	}
}

// IsValidIssuer checks if the given issuer is valid
func (c *Config) IsValidIssuer(issuer string) bool {
	return issuer == ServiceIdentifierPlatform || issuer == ServiceIdentifierInstance
}
