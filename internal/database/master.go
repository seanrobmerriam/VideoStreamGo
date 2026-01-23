package database

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"videostreamgo/internal/config"
)

// MasterDB holds the master database connection
type MasterDB struct {
	DB     *gorm.DB
	config *config.Config
}

// NewMasterDB creates a new connection to the master database
func NewMasterDB(cfg *config.Config) (*MasterDB, error) {
	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: false,
		},
		Logger: logger.Default.LogMode(logger.Info),
	}

	if cfg.App.Environment == "production" {
		gormConfig.Logger = logger.Default.LogMode(logger.Warn)
	}

	db, err := gorm.Open(postgres.Open(cfg.MasterDSN()), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to master database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MasterDB.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MasterDB.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.MasterDB.ConnMaxLifetime)

	masterDB := &MasterDB{
		DB:     db,
		config: cfg,
	}

	return masterDB, nil
}

// Close closes the database connection
func (m *MasterDB) Close() error {
	sqlDB, err := m.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping checks if the database connection is alive
func (m *MasterDB) Ping() error {
	sqlDB, err := m.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// GetDB returns the underlying gorm.DB instance
func (m *MasterDB) GetDB() *gorm.DB {
	return m.DB
}

// Migrate runs database migrations for the master database
func (m *MasterDB) Migrate() error {
	// Auto-migrate all master models
	// Note: In production, use explicit migrations for more control
	err := m.DB.AutoMigrate(
	// Master models will be imported here
	)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Transaction executes a function within a database transaction
func (m *MasterDB) Transaction(fn func(tx *gorm.DB) error) error {
	return m.DB.Transaction(fn)
}

// WithTimeout returns a context with a timeout for database operations
func (m *MasterDB) WithTimeout(timeout time.Duration) (interface{}, error) {
	// This can be used for long-running operations
	return nil, nil
}

// HealthCheck performs a health check on the database connection
func (m *MasterDB) HealthCheck() map[string]interface{} {
	sqlDB, err := m.DB.DB()
	if err != nil {
		return map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	}

	stats := sqlDB.Stats()

	return map[string]interface{}{
		"status":           "healthy",
		"open_connections": stats.OpenConnections,
		"in_use":           stats.InUse,
		"idle":             stats.Idle,
		"wait_count":       stats.WaitCount,
		"wait_duration":    stats.WaitDuration.String(),
	}
}
