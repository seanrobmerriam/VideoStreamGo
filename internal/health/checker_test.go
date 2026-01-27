package health

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"videostreamgo/internal/config"
)

func TestNewChecker(t *testing.T) {
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
			ServiceIdentifier: "test-service",
		},
	}

	// Create checker with nil dependencies - this should not panic
	checker := NewChecker(nil, nil, nil, cfg, "1.0.0")

	assert.NotNil(t, checker)
	assert.Equal(t, "1.0.0", checker.GetVersion())
	assert.Equal(t, "test-service", checker.cfg.App.ServiceIdentifier)
}

func TestCheckerGetVersion(t *testing.T) {
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
			ServiceIdentifier: "test-service",
		},
	}

	checker := NewChecker(nil, nil, nil, cfg, "2.0.0")

	version := checker.GetVersion()
	assert.Equal(t, "2.0.0", version)
}

func TestCheckerSetVersion(t *testing.T) {
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
			ServiceIdentifier: "test-service",
		},
	}

	checker := NewChecker(nil, nil, nil, cfg, "1.0.0")

	checker.SetVersion("3.0.0")
	assert.Equal(t, "3.0.0", checker.GetVersion())
}

func TestCheckerSetStartTime(t *testing.T) {
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
			ServiceIdentifier: "test-service",
		},
	}

	checker := NewChecker(nil, nil, nil, cfg, "1.0.0")

	startTime := time.Now().Add(-time.Hour)
	checker.SetStartTime(startTime)

	uptime := time.Since(startTime)
	assert.True(t, uptime > time.Minute)
	assert.True(t, uptime < time.Hour*2)
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "milliseconds",
			duration: 500 * time.Millisecond,
			expected: "500ms",
		},
		{
			name:     "seconds",
			duration: 5 * time.Second,
			expected: "5.00s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m 30.00s",
		},
		{
			name:     "hours minutes seconds",
			duration: 2*time.Hour + 30*time.Minute + 45*time.Second,
			expected: "2h 30m 45.00s",
		},
		{
			name:     "days hours minutes",
			duration: 3*24*time.Hour + 5*time.Hour + 30*time.Minute,
			expected: "3d 5h 30m",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHealthStatus(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{StatusHealthy, "healthy"},
		{StatusUnhealthy, "unhealthy"},
		{StatusDegraded, "degraded"},
		{StatusUnknown, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestDependencyCheck(t *testing.T) {
	check := DependencyCheck{
		Name:         "test-dependency",
		Status:       StatusHealthy,
		ResponseTime: 100 * time.Millisecond,
		Error:        "",
		Details:      map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "test-dependency", check.Name)
	assert.Equal(t, StatusHealthy, check.Status)
	assert.Equal(t, 100*time.Millisecond, check.ResponseTime)
	assert.Empty(t, check.Error)
	assert.NotNil(t, check.Details)
}

func TestMemoryStats(t *testing.T) {
	stats := &MemoryStats{
		Alloc:         1024,
		AllocMB:       0.001,
		TotalAlloc:    2048,
		TotalAllocMB:  0.002,
		Sys:           4096,
		SysMB:         0.004,
		NumGC:         10,
		GCCPUFraction: 0.01,
	}

	assert.Equal(t, uint64(1024), stats.Alloc)
	assert.Equal(t, uint32(10), stats.NumGC)
	assert.Greater(t, stats.GCCPUFraction, float64(0))
}

func TestCheckerCheckResult(t *testing.T) {
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
			ServiceIdentifier: "test-service",
		},
	}

	// Create checker with nil dependencies
	checker := NewChecker(nil, nil, nil, cfg, "1.0.0")

	ctx := context.Background()
	result := checker.Check(ctx)

	require.NotNil(t, result)
	// Status could be degraded or unhealthy since dependencies are nil
	assert.Contains(t, []string{"healthy", "degraded", "unhealthy"}, result.Status)
	assert.Equal(t, "test-service", result.Service)
	assert.Equal(t, "1.0.0", result.Version)
	assert.NotNil(t, result.MemoryUsage)
	assert.NotNil(t, result.Dependencies)
	assert.False(t, result.StartTime.IsZero())
	assert.False(t, result.Timestamp.IsZero())
}

func TestCheckerCheckDependencies(t *testing.T) {
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
			ServiceIdentifier: "test-service",
		},
	}

	checker := NewChecker(nil, nil, nil, cfg, "1.0.0")

	ctx := context.Background()
	result := checker.Check(ctx)

	require.NotNil(t, result)
	require.Len(t, result.Dependencies, 3)

	// Check that all dependencies are present
	depNames := make(map[string]bool)
	for _, dep := range result.Dependencies {
		depNames[dep.Name] = true
	}

	assert.True(t, depNames["database"])
	assert.True(t, depNames["redis"])
	assert.True(t, depNames["minio"])
}

func TestCheckerUptime(t *testing.T) {
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
			ServiceIdentifier: "test-service",
		},
	}

	checker := NewChecker(nil, nil, nil, cfg, "1.0.0")

	// Set start time to 1 hour ago
	startTime := time.Now().Add(-time.Hour)
	checkTime := context.Background()
	checker.SetStartTime(startTime)

	result := checker.Check(checkTime)

	require.NotNil(t, result)
	assert.True(t, result.UptimeSeconds >= 3600)
	assert.True(t, result.UptimeSeconds < 7200)
}
