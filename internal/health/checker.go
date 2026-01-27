package health

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"videostreamgo/internal/cache"
	"videostreamgo/internal/config"
	"videostreamgo/internal/storage"
)

// Timeouts for health checks
const (
	TimeoutDB    = 2 * time.Second
	TimeoutRedis = 1 * time.Second
	TimeoutMinIO = 3 * time.Second
)

// HealthStatus represents the health status of a dependency
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnknown   HealthStatus = "unknown"
)

// DependencyCheck represents a health check result for a dependency
type DependencyCheck struct {
	Name         string        `json:"name"`
	Status       HealthStatus  `json:"status"`
	ResponseTime time.Duration `json:"response_time_ms"`
	Error        string        `json:"error,omitempty"`
	Details      interface{}   `json:"details,omitempty"`
}

// HealthCheckResult represents the complete health check result
type HealthCheckResult struct {
	Status        string            `json:"status"`
	Service       string            `json:"service"`
	Version       string            `json:"version"`
	Uptime        string            `json:"uptime"`
	UptimeSeconds float64           `json:"uptime_seconds"`
	StartTime     time.Time         `json:"start_time"`
	MemoryUsage   *MemoryStats      `json:"memory_usage"`
	Dependencies  []DependencyCheck `json:"dependencies"`
	Timestamp     time.Time         `json:"timestamp"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Alloc         uint64  `json:"alloc_bytes"`
	AllocMB       float64 `json:"alloc_mb"`
	TotalAlloc    uint64  `json:"total_alloc_bytes"`
	TotalAllocMB  float64 `json:"total_alloc_mb"`
	Sys           uint64  `json:"sys_bytes"`
	SysMB         float64 `json:"sys_mb"`
	NumGC         uint32  `json:"num_gc"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`
}

// Checker performs health checks on all dependencies
type Checker struct {
	masterDB    *gorm.DB
	redisClient *redis.Client
	minioClient *storage.MinioClient
	cfg         *config.Config
	startTime   time.Time
	mu          sync.RWMutex
	version     string
}

// NewChecker creates a new health checker
func NewChecker(masterDB *gorm.DB, redisClient *cache.RedisClient, minioClient *storage.MinioClient, cfg *config.Config, version string) *Checker {
	var redis *redis.Client
	if redisClient != nil {
		redis = redisClient.GetClient()
	}

	return &Checker{
		masterDB:    masterDB,
		redisClient: redis,
		minioClient: minioClient,
		cfg:         cfg,
		startTime:   time.Now(),
		version:     version,
	}
}

// Check performs all health checks
func (c *Checker) Check(ctx context.Context) *HealthCheckResult {
	c.mu.RLock()
	startTime := c.startTime
	c.mu.RUnlock()

	var wg sync.WaitGroup
	var mu sync.Mutex

	dependencies := make([]DependencyCheck, 0, 4)

	// Check database
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := c.checkDatabase(ctx)
		mu.Lock()
		dependencies = append(dependencies, result)
		mu.Unlock()
	}()

	// Check Redis
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := c.checkRedis(ctx)
		mu.Lock()
		dependencies = append(dependencies, result)
		mu.Unlock()
	}()

	// Check MinIO
	wg.Add(1)
	go func() {
		defer wg.Done()
		result := c.checkMinIO(ctx)
		mu.Lock()
		dependencies = append(dependencies, result)
		mu.Unlock()
	}()

	wg.Wait()

	// Determine overall status
	overallStatus := StatusHealthy
	hasUnhealthy := false
	hasDegraded := false

	for _, dep := range dependencies {
		switch dep.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		overallStatus = StatusUnhealthy
	} else if hasDegraded {
		overallStatus = StatusDegraded
	}

	// Calculate uptime
	uptime := time.Since(startTime)
	uptimeSeconds := uptime.Seconds()

	// Get memory stats
	memStats := c.getMemoryStats()

	return &HealthCheckResult{
		Status:        string(overallStatus),
		Service:       c.cfg.App.ServiceIdentifier,
		Version:       c.version,
		Uptime:        formatDuration(uptime),
		UptimeSeconds: uptimeSeconds,
		StartTime:     startTime,
		MemoryUsage:   memStats,
		Dependencies:  dependencies,
		Timestamp:     time.Now().UTC(),
	}
}

// checkDatabase performs a database health check
func (c *Checker) checkDatabase(ctx context.Context) DependencyCheck {
	start := time.Now()

	var result DependencyCheck
	result.Name = "database"
	result.ResponseTime = time.Since(start)

	if c.masterDB == nil {
		result.Status = StatusDegraded
		result.Error = "database client not configured"
		result.ResponseTime = time.Since(start)
		return result
	}

	ctx, cancel := context.WithTimeout(ctx, TimeoutDB)
	defer cancel()

	// Execute a simple query
	sql := "SELECT 1"
	if err := c.masterDB.WithContext(ctx).Raw(sql).Scan(&result).Error; err != nil {
		result.Status = StatusUnhealthy
		result.Error = err.Error()
	} else {
		result.Status = StatusHealthy
	}

	result.ResponseTime = time.Since(start)

	// Add connection pool stats
	sqlDB, err := c.masterDB.DB()
	if err == nil && sqlDB != nil {
		stats := sqlDB.Stats()
		result.Details = map[string]interface{}{
			"open_connections": stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
			"wait_count":       stats.WaitCount,
			"wait_duration_ms": stats.WaitDuration.Milliseconds(),
		}
	}

	return result
}

// checkRedis performs a Redis health check
func (c *Checker) checkRedis(ctx context.Context) DependencyCheck {
	start := time.Now()

	var result DependencyCheck
	result.Name = "redis"
	result.ResponseTime = time.Since(start)

	if c.redisClient == nil {
		result.Status = StatusDegraded
		result.Error = "redis client not configured"
		result.ResponseTime = time.Since(start)
		return result
	}

	ctx, cancel := context.WithTimeout(ctx, TimeoutRedis)
	defer cancel()

	// Execute PING command
	if err := c.redisClient.Ping(ctx).Err(); err != nil {
		result.Status = StatusUnhealthy
		result.Error = err.Error()
	} else {
		result.Status = StatusHealthy
	}

	result.ResponseTime = time.Since(start)

	// Add Redis info
	result.Details = map[string]interface{}{
		"connected": true,
		"addr":      c.redisClient.Options().Addr,
	}

	return result
}

// checkMinIO performs a MinIO health check
func (c *Checker) checkMinIO(ctx context.Context) DependencyCheck {
	start := time.Now()

	var result DependencyCheck
	result.Name = "minio"
	result.ResponseTime = time.Since(start)

	if c.minioClient == nil {
		result.Status = StatusDegraded
		result.Error = "minio client not configured"
		result.ResponseTime = time.Since(start)
		return result
	}

	ctx, cancel := context.WithTimeout(ctx, TimeoutMinIO)
	defer cancel()

	// Use the existing Health method
	healthResult := c.minioClient.Health(ctx)
	if healthResult["status"] == "healthy" {
		result.Status = StatusHealthy
	} else {
		result.Status = StatusUnhealthy
		if err, ok := healthResult["error"].(string); ok {
			result.Error = err
		}
	}

	result.Details = healthResult

	result.ResponseTime = time.Since(start)

	return result
}

// getMemoryStats returns memory usage statistics
func (c *Checker) getMemoryStats() *MemoryStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &MemoryStats{
		Alloc:         memStats.Alloc,
		AllocMB:       float64(memStats.Alloc) / 1024 / 1024,
		TotalAlloc:    memStats.TotalAlloc,
		TotalAllocMB:  float64(memStats.TotalAlloc) / 1024 / 1024,
		Sys:           memStats.Sys,
		SysMB:         float64(memStats.Sys) / 1024 / 1024,
		NumGC:         memStats.NumGC,
		GCCPUFraction: memStats.GCCPUFraction,
	}
}

// formatDuration formats a duration as a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%dm %.2fs", int(d.Minutes()), d.Seconds()-float64(int(d.Minutes())*60))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh %dm %.2fs", int(d.Hours()), int(d.Minutes())-int(d.Hours())*60, d.Seconds()-float64(int(d.Minutes())*60))
	}
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
}

// SimpleCheck performs a quick health check (for load balancers)
func (c *Checker) SimpleCheck(ctx context.Context) error {
	// Check database
	if c.masterDB != nil {
		dbCtx, dbCancel := context.WithTimeout(ctx, TimeoutDB)
		defer dbCancel()
		if err := c.masterDB.WithContext(dbCtx).Raw("SELECT 1").Error; err != nil {
			return fmt.Errorf("database unhealthy: %w", err)
		}
	}

	// Check Redis
	if c.redisClient != nil {
		redisCtx, redisCancel := context.WithTimeout(ctx, TimeoutRedis)
		defer redisCancel()
		if err := c.redisClient.Ping(redisCtx).Err(); err != nil {
			return fmt.Errorf("redis unhealthy: %w", err)
		}
	}

	return nil
}

// GetVersion returns the service version
func (c *Checker) GetVersion() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.version
}

// SetVersion sets the service version
func (c *Checker) SetVersion(version string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.version = version
}

// SetStartTime sets the service start time
func (c *Checker) SetStartTime(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.startTime = t
}
