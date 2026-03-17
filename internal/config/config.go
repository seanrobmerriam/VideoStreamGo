package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
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
		Environment    string
		Debug          bool
		Port           int
		JWTSecret      string
		EncryptionKey string
		AllowedOrigins []string
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
		Endpoint  string
		AccessKey string
		SecretKey string
		Bucket    string
		Region    string
		UseSSL    bool
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

	// CORS Allowed Origins - comma-separated list of permitted origins
	// For production, explicitly list all allowed origins
	allowedOriginsEnv := getEnv("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:4321")
	cfg.App.AllowedOrigins = parseAllowedOrigins(allowedOriginsEnv)

	// Security-sensitive configuration - these MUST be set in production
	// JWT_SECRET must be at least 32 characters for HS256
	jwtSecret, err := getEnvRequiredWithMinLength("JWT_SECRET", 32)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_SECRET: %w", err)
	}
	cfg.App.JWTSecret = jwtSecret

	// ENCRYPTION_KEY must be at least 32 characters for AES-256
	encryptionKey, err := getEnvRequiredWithMinLength("ENCRYPTION_KEY", 32)
	if err != nil {
		return nil, fmt.Errorf("invalid ENCRYPTION_KEY: %w", err)
	}
	cfg.App.EncryptionKey = encryptionKey

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
	cfg.S3.Region = getEnv("S3_REGION", "us-east-1")
	cfg.S3.UseSSL = getEnvBool("S3_USE_SSL", false)

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

// getEnvRequired retrieves a required environment variable, returning an error if not set
func getEnvRequired(key string) (string, error) {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value, nil
	}
	return "", fmt.Errorf("required environment variable %s is not set", key)
}

// getEnvRequiredWithMinLength retrieves a required environment variable with minimum length validation
func getEnvRequiredWithMinLength(key string, minLength int) (string, error) {
	value, err := getEnvRequired(key)
	if err != nil {
		return "", err
	}
	if len(value) < minLength {
		return "", fmt.Errorf("environment variable %s must be at least %d characters", key, minLength)
	}
	return value, nil
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

// parseAllowedOrigins parses a comma-separated string of allowed origins
func parseAllowedOrigins(origins string) []string {
	if origins == "" {
		return []string{}
	}
	var result []string
	parts := strings.Split(origins, ",")
	for _, p := range parts {
		origin := strings.TrimSpace(p)
		if origin != "" {
			result = append(result, origin)
		}
	}
	return result
}
