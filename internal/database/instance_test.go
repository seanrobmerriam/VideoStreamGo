package database_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/config"
	"videostreamgo/internal/database"
)

// Test_TenantDBManagerConnectionPoolIsolation tests that connection pools are isolated per tenant
func Test_TenantDBManagerConnectionPoolIsolation(t *testing.T) {
	cfg, err := config.Load()
	assert.NoError(t, err)

	manager := database.NewTenantDBManager(cfg)

	// Create separate connection pools for different instances
	instance1ID := "11111111-1111-1111-1111-111111111111"
	instance2ID := "22222222-2222-2222-2222-222222222222"

	// Get connection pools for both instances
	db1, err1 := manager.GetDB(instance1ID)
	db2, err2 := manager.GetDB(instance2ID)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NotNil(t, db1)
	assert.NotNil(t, db2)

	// Verify pools are different
	poolInfo1, err := manager.GetPoolInfo(instance1ID)
	poolInfo2, err := manager.GetPoolInfo(instance2ID)

	assert.NoError(t, err)
	assert.NotNil(t, poolInfo1)
	assert.NotNil(t, poolInfo2)
	assert.NotEqual(t, poolInfo1.DatabaseName, poolInfo2.DatabaseName)

	// Clean up
	manager.Close(instance1ID)
	manager.Close(instance2ID)
}

// Test_TenantDBManagerGetPoolStats tests pool statistics retrieval
func Test_TenantDBManagerGetPoolStats(t *testing.T) {
	cfg, err := config.Load()
	assert.NoError(t, err)

	manager := database.NewTenantDBManager(cfg)

	// Add a connection pool
	instanceID := "33333333-3333-3333-3333-333333333333"
	_, err = manager.GetDB(instanceID)
	assert.NoError(t, err)

	// Get pool stats
	stats := manager.GetPoolStats()
	assert.Contains(t, stats, instanceID)

	poolStats := stats[instanceID]
	assert.Equal(t, instanceID, poolStats.InstanceID)

	// Clean up
	manager.Close(instanceID)
}

// Test_TenantDBManagerHealthCheck tests health check functionality
func Test_TenantDBManagerHealthCheck(t *testing.T) {
	cfg, err := config.Load()
	assert.NoError(t, err)

	manager := database.NewTenantDBManager(cfg)

	// Add a connection pool
	instanceID := "44444444-4444-4444-4444-444444444444"
	_, err = manager.GetDB(instanceID)
	assert.NoError(t, err)

	// Perform health check
	health := manager.HealthCheck()
	assert.Contains(t, health, instanceID)

	healthStatus := health[instanceID].(map[string]interface{})
	assert.Equal(t, "healthy", healthStatus["status"])

	// Clean up
	manager.Close(instanceID)
}

// Test_TenantDBManagerCloseAll tests closing all pools
func Test_TenantDBManagerCloseAll(t *testing.T) {
	cfg, err := config.Load()
	assert.NoError(t, err)

	manager := database.NewTenantDBManager(cfg)

	// Add multiple connection pools
	instance1ID := "55555555-5555-5555-5555-555555555555"
	instance2ID := "66666666-6666-6666-6666-666666666666"

	_, err = manager.GetDB(instance1ID)
	assert.NoError(t, err)
	_, err = manager.GetDB(instance2ID)
	assert.NoError(t, err)

	// Close all pools
	err = manager.CloseAll()
	assert.NoError(t, err)

	// Verify all pools are closed
	stats := manager.GetPoolStats()
	assert.Empty(t, stats)
}

// Test_DefaultTenantDBConfig tests default configuration values
func Test_DefaultTenantDBConfig(t *testing.T) {
	cfg := database.DefaultTenantDBConfig()

	assert.Equal(t, 25, cfg.MaxOpenConns)
	assert.Equal(t, 5, cfg.MaxIdleConns)
	assert.Equal(t, 5*60, int(cfg.ConnMaxLifetime.Seconds()))
	assert.Equal(t, 1*60, int(cfg.ConnMaxIdleTime.Seconds()))
	assert.Equal(t, 30, int(cfg.QueryTimeout.Seconds()))
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 500, int(cfg.RetryDelay.Milliseconds()))
}

// Test_TenantDBManagerSetTenantConfig tests setting custom configuration
func Test_TenantDBManagerSetTenantConfig(t *testing.T) {
	cfg, err := config.Load()
	assert.NoError(t, err)

	manager := database.NewTenantDBManager(cfg)

	instanceID := "77777777-7777-7777-7777-777777777777"

	// Set custom configuration
	customConfig := database.TenantDBConfig{
		MaxOpenConns: 50,
		MaxIdleConns: 10,
		MaxRetries:   5,
		RetryDelay:   1000,
	}

	manager.SetTenantConfig(instanceID, customConfig)

	// Note: Custom config would be used when creating new pools
	// This test verifies the configuration can be set without error

	// Clean up
	manager.CloseAll()
}

// Test_TenantDBManagerPingAll tests pinging all pools
func Test_TenantDBManagerPingAll(t *testing.T) {
	cfg, err := config.Load()
	assert.NoError(t, err)

	manager := database.NewTenantDBManager(cfg)

	// Add a connection pool
	instanceID := "88888888-8888-8888-8888-888888888888"
	_, err = manager.GetDB(instanceID)
	assert.NoError(t, err)

	// Ping all pools
	errors := manager.PingAll()
	// The pool might not be healthy if DB is not available, but it shouldn't panic
	assert.NotNil(t, errors)

	// Clean up
	manager.Close(instanceID)
}

// Test_PoolInfoStructure tests pool info structure
func Test_PoolInfoStructure(t *testing.T) {
	info := database.PoolInfo{
		InstanceID:   "test-instance",
		DatabaseName: "test_db",
		MaxOpenConns: 25,
		MaxIdleConns: 5,
	}

	assert.Equal(t, "test-instance", info.InstanceID)
	assert.Equal(t, "test_db", info.DatabaseName)
	assert.Equal(t, 25, info.MaxOpenConns)
	assert.Equal(t, 5, info.MaxIdleConns)
}
