package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"videostreamgo/internal/config"
)

// TenantDBConfig holds database configuration for a specific tenant
type TenantDBConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	QueryTimeout    time.Duration
	MaxRetries      int
	RetryDelay      time.Duration
}

// DefaultTenantDBConfig returns default configuration for tenant databases
func DefaultTenantDBConfig() TenantDBConfig {
	return TenantDBConfig{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
		QueryTimeout:    30 * time.Second,
		MaxRetries:      3,
		RetryDelay:      500 * time.Millisecond,
	}
}

// PoolInfo holds information about a connection pool
type PoolInfo struct {
	InstanceID      string
	DatabaseName    string
	MaxOpenConns    int
	MaxIdleConns    int
	OpenConnections int
	InUse           int
	Idle            int
	WaitCount       int64
	WaitDuration    time.Duration
	LastUsed        time.Time
	CreatedAt       time.Time
}

// TenantDBManager manages database connections for per-tenant instance databases
type TenantDBManager struct {
	config     *config.Config
	pools      map[string]*gorm.DB
	poolInfo   map[string]*PoolInfo
	poolsMutex sync.RWMutex
	configs    map[string]TenantDBConfig
	configsMu  sync.RWMutex
	logger     *DBLogger
}

// NewTenantDBManager creates a new instance database manager
func NewTenantDBManager(cfg *config.Config) *TenantDBManager {
	return &TenantDBManager{
		config:   cfg,
		pools:    make(map[string]*gorm.DB),
		poolInfo: make(map[string]*PoolInfo),
		configs:  make(map[string]TenantDBConfig),
		logger:   NewDBLogger(),
	}
}

// GetDB returns a database connection for the specified instance
func (m *TenantDBManager) GetDB(instanceID string) (*gorm.DB, error) {
	return m.GetDBWithContext(context.Background(), instanceID)
}

// GetDBWithContext returns a database connection with context for the specified instance
func (m *TenantDBManager) GetDBWithContext(ctx context.Context, instanceID string) (*gorm.DB, error) {
	// Check if pool already exists
	m.poolsMutex.RLock()
	if pool, exists := m.pools[instanceID]; exists {
		m.poolsMutex.RUnlock()
		return pool, nil
	}
	m.poolsMutex.RUnlock()

	// Create new pool for instance
	m.poolsMutex.Lock()
	defer m.poolsMutex.Unlock()

	// Double-check after acquiring write lock
	if pool, exists := m.pools[instanceID]; exists {
		return pool, nil
	}

	// Get instance database configuration
	databaseName := m.getDatabaseName(instanceID)
	tenantConfig := m.getTenantConfig(instanceID)

	// Create new connection pool with retry logic
	var pool *gorm.DB
	var err error
	for i := 0; i <= tenantConfig.MaxRetries; i++ {
		pool, err = m.createPoolWithConfig(databaseName, tenantConfig)
		if err == nil {
			break
		}

		if i < tenantConfig.MaxRetries {
			m.logger.LogRetry(instanceID, i+1, tenantConfig.MaxRetries, err)
			time.Sleep(tenantConfig.RetryDelay * time.Duration(i+1))
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to instance database after %d retries: %w", tenantConfig.MaxRetries, err)
	}

	// Store pool info
	m.poolInfo[instanceID] = &PoolInfo{
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		MaxOpenConns: tenantConfig.MaxOpenConns,
		MaxIdleConns: tenantConfig.MaxIdleConns,
		CreatedAt:    time.Now(),
	}

	m.pools[instanceID] = pool
	m.logger.LogPoolCreated(instanceID, databaseName)

	return pool, nil
}

// createPoolWithConfig creates a new database connection pool with custom configuration
func (m *TenantDBManager) createPoolWithConfig(databaseName string, cfg TenantDBConfig) (*gorm.DB, error) {
	dsn := m.config.InstanceDSN(databaseName)

	gormConfig := &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "",
			SingularTable: false,
		},
		Logger: m.logger,
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database %s: %w", databaseName, err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	return db, nil
}

// getTenantConfig returns the configuration for a specific tenant
func (m *TenantDBManager) getTenantConfig(instanceID string) TenantDBConfig {
	m.configsMu.RLock()
	if cfg, exists := m.configs[instanceID]; exists {
		m.configsMu.RUnlock()
		return cfg
	}
	m.configsMu.RUnlock()

	// Return default config
	return DefaultTenantDBConfig()
}

// SetTenantConfig sets custom configuration for a specific tenant
func (m *TenantDBManager) SetTenantConfig(instanceID string, cfg TenantDBConfig) {
	m.configsMu.Lock()
	m.configs[instanceID] = cfg
	m.configsMu.Unlock()
}

// getDatabaseName returns the database name for an instance
func (m *TenantDBManager) getDatabaseName(instanceID string) string {
	// Database name format: instance_{instance_id[:8]}
	if len(instanceID) >= 8 {
		return m.config.InstanceDB.DatabasePrefix + instanceID[:8]
	}
	return m.config.InstanceDB.DatabasePrefix + instanceID
}

// WithTransaction executes a function within a database transaction
func (m *TenantDBManager) WithTransaction(instanceID string, fn func(tx *gorm.DB) error) error {
	db, err := m.GetDB(instanceID)
	if err != nil {
		return err
	}

	return db.Transaction(fn)
}

// WithTimeout returns a database connection with timeout context
func (m *TenantDBManager) WithTimeout(instanceID string, timeout time.Duration) (context.Context, context.CancelFunc, *gorm.DB, error) {
	db, err := m.GetDB(instanceID)
	if err != nil {
		return nil, nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return ctx, cancel, db.WithContext(ctx), nil
}

// Close closes and removes an instance's database connection pool
func (m *TenantDBManager) Close(instanceID string) error {
	m.poolsMutex.Lock()
	defer m.poolsMutex.Unlock()

	if pool, exists := m.pools[instanceID]; exists {
		sqlDB, err := pool.DB()
		if err != nil {
			delete(m.pools, instanceID)
			delete(m.poolInfo, instanceID)
			return err
		}
		sqlDB.Close()
		delete(m.pools, instanceID)
		delete(m.poolInfo, instanceID)
		m.logger.LogPoolClosed(instanceID)
	}

	return nil
}

// CloseAll closes all instance database connection pools
func (m *TenantDBManager) CloseAll() error {
	m.poolsMutex.Lock()
	defer m.poolsMutex.Unlock()

	for id, pool := range m.pools {
		sqlDB, err := pool.DB()
		if err != nil {
			m.logger.LogError(id, err)
			continue
		}
		sqlDB.Close()
		delete(m.pools, id)
		delete(m.poolInfo, id)
		m.logger.LogPoolClosed(id)
	}

	return nil
}

// GetPoolInfo returns information about a specific connection pool
func (m *TenantDBManager) GetPoolInfo(instanceID string) (*PoolInfo, error) {
	m.poolsMutex.RLock()
	defer m.poolsMutex.RUnlock()

	info, exists := m.poolInfo[instanceID]
	if !exists {
		return nil, fmt.Errorf("pool not found for instance: %s", instanceID)
	}

	// Update current stats
	if pool, exists := m.pools[instanceID]; exists {
		sqlDB, err := pool.DB()
		if err != nil {
			return info, nil
		}

		dbStats := sqlDB.Stats()
		info.OpenConnections = dbStats.OpenConnections
		info.InUse = dbStats.InUse
		info.Idle = dbStats.Idle
		info.WaitCount = dbStats.WaitCount
		info.WaitDuration = dbStats.WaitDuration
	}

	return info, nil
}

// GetPoolStats returns statistics for all connection pools
func (m *TenantDBManager) GetPoolStats() map[string]PoolInfo {
	m.poolsMutex.RLock()
	defer m.poolsMutex.RUnlock()

	stats := make(map[string]PoolInfo)
	for id, info := range m.poolInfo {
		// Update current stats
		if pool, exists := m.pools[id]; exists {
			sqlDB, err := pool.DB()
			if err != nil {
				continue
			}

			dbStats := sqlDB.Stats()
			info.OpenConnections = dbStats.OpenConnections
			info.InUse = dbStats.InUse
			info.Idle = dbStats.Idle
			info.WaitCount = dbStats.WaitCount
			info.WaitDuration = dbStats.WaitDuration
			info.LastUsed = time.Now()
		}
		stats[id] = *info
	}

	return stats
}

// Ping checks if an instance database connection is alive
func (m *TenantDBManager) Ping(instanceID string) error {
	m.poolsMutex.RLock()
	defer m.poolsMutex.RUnlock()

	pool, exists := m.pools[instanceID]
	if !exists {
		return fmt.Errorf("pool not found for instance: %s", instanceID)
	}

	sqlDB, err := pool.DB()
	if err != nil {
		return err
	}

	return sqlDB.Ping()
}

// PingAll checks if all instance database connections are alive
func (m *TenantDBManager) PingAll() map[string]error {
	m.poolsMutex.RLock()
	defer m.poolsMutex.RUnlock()

	errors := make(map[string]error)
	for id, pool := range m.pools {
		sqlDB, err := pool.DB()
		if err != nil {
			errors[id] = err
			continue
		}
		if err := sqlDB.Ping(); err != nil {
			errors[id] = err
		}
	}

	return errors
}

// HealthCheck performs a health check on all instance databases
func (m *TenantDBManager) HealthCheck() map[string]interface{} {
	m.poolsMutex.RLock()
	defer m.poolsMutex.RUnlock()

	health := make(map[string]interface{})
	for id, pool := range m.pools {
		sqlDB, err := pool.DB()
		if err != nil {
			health[id] = map[string]interface{}{
				"status":  "unhealthy",
				"error":   err.Error(),
				"healthy": false,
			}
			continue
		}

		stats := sqlDB.Stats()
		health[id] = map[string]interface{}{
			"status":           "healthy",
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
			"healthy":          true,
		}
	}

	return health
}

// HealthCheckWithInstance performs a health check on a specific instance database
func (m *TenantDBManager) HealthCheckWithInstance(instanceID string) map[string]interface{} {
	m.poolsMutex.RLock()
	pool, exists := m.pools[instanceID]
	m.poolsMutex.RUnlock()

	if !exists {
		return map[string]interface{}{
			"status":  "not_found",
			"healthy": false,
		}
	}

	sqlDB, err := pool.DB()
	if err != nil {
		return map[string]interface{}{
			"status":  "unhealthy",
			"error":   err.Error(),
			"healthy": false,
		}
	}

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"status":           "healthy",
		"open_connections": stats.OpenConnections,
		"in_use":           stats.InUse,
		"idle":             stats.Idle,
		"healthy":          true,
	}
}

// GetDBByInstanceUUID returns a database connection for the specified instance UUID
func (m *TenantDBManager) GetDBByInstanceUUID(instanceUUID uuid.UUID) (*gorm.DB, error) {
	return m.GetDB(instanceUUID.String())
}

// TenantDBLogger is a custom logger for database operations
type DBLogger struct {
	logger *log.Logger
}

// NewDBLogger creates a new database logger
func NewDBLogger() *DBLogger {
	return &DBLogger{
		logger: log.New(log.Writer(), "[DB] ", log.LstdFlags|log.Lmicroseconds),
	}
}

// Info logs an info message
func (l *DBLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	l.logger.Printf("[INFO] "+msg, args...)
}

// Warn logs a warning message
func (l *DBLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	l.logger.Printf("[WARN] "+msg, args...)
}

// Error logs an error message
func (l *DBLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	l.logger.Printf("[ERROR] "+msg, args...)
}

// Trace traces a SQL query
func (l *DBLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	l.logger.Printf("[TRACE] Query: %s, Rows: %d, Duration: %v, Error: %v", sql, rows, elapsed, err)
}

// LogMode returns the logger with the specified level
func (l *DBLogger) LogMode(level logger.LogLevel) logger.Interface {
	return l
}

// Log logs a message
func (l *DBLogger) Log(format string, args ...interface{}) {
	l.logger.Printf(format, args...)
}

// LogPoolCreated logs pool creation
func (l *DBLogger) LogPoolCreated(instanceID, databaseName string) {
	l.logger.Printf("Pool created for instance %s (database: %s)", instanceID, databaseName)
}

// LogPoolClosed logs pool closure
func (l *DBLogger) LogPoolClosed(instanceID string) {
	l.logger.Printf("Pool closed for instance %s", instanceID)
}

// LogRetry logs retry attempts
func (l *DBLogger) LogRetry(instanceID string, attempt, maxRetries int, err error) {
	l.logger.Printf("Retry %d/%d for instance %s: %v", attempt, maxRetries, instanceID, err)
}

// LogError logs an error
func (l *DBLogger) LogError(instanceID string, err error) {
	l.logger.Printf("Error for instance %s: %v", instanceID, err)
}

// LogQuery logs a query with tenant context
func (l *DBLogger) LogQuery(instanceID, query string, duration time.Duration) {
	l.logger.Printf("Query for instance %s completed in %v: %s", instanceID, duration, query)
}
