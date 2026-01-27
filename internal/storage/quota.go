package storage

import (
	"context"
	"fmt"
	"time"

	"videostreamgo/internal/cache"
	"videostreamgo/internal/config"
)

// StorageQuotaManager manages storage quotas per tenant
type StorageQuotaManager struct {
	redisClient  *cache.RedisClient
	minioClient  *MinioClient
	cfg          *config.Config
	defaultQuota int64
}

// StorageQuotaKey returns the Redis key for tenant storage usage
func StorageQuotaKey(tenantID string) string {
	return fmt.Sprintf("storage:%s:bytes", tenantID)
}

// StorageQuotaLimitKey returns the Redis key for tenant storage quota limit
func StorageQuotaLimitKey(tenantID string) string {
	return fmt.Sprintf("storage:%s:limit", tenantID)
}

// NewStorageQuotaManager creates a new storage quota manager
func NewStorageQuotaManager(redisClient *cache.RedisClient, minioClient *MinioClient, cfg *config.Config) *StorageQuotaManager {
	return &StorageQuotaManager{
		redisClient:  redisClient,
		minioClient:  minioClient,
		cfg:          cfg,
		defaultQuota: DefaultStorageQuota,
	}
}

// GetUsage returns the current storage usage for a tenant
func (m *StorageQuotaManager) GetUsage(ctx context.Context, tenantID string) (int64, error) {
	usageStr, err := m.redisClient.Get(ctx, StorageQuotaKey(tenantID))
	if err != nil {
		return 0, fmt.Errorf("failed to get storage usage: %w", err)
	}

	if usageStr == "" {
		return 0, nil
	}

	var usage int64
	_, err = fmt.Sscanf(usageStr, "%d", &usage)
	if err != nil {
		return 0, fmt.Errorf("failed to parse storage usage: %w", err)
	}

	return usage, nil
}

// GetQuota returns the storage quota limit for a tenant
func (m *StorageQuotaManager) GetQuota(ctx context.Context, tenantID string) (int64, error) {
	quotaStr, err := m.redisClient.Get(ctx, StorageQuotaLimitKey(tenantID))
	if err != nil {
		return 0, fmt.Errorf("failed to get storage quota: %w", err)
	}

	if quotaStr == "" {
		return m.defaultQuota, nil
	}

	var quota int64
	_, err = fmt.Sscanf(quotaStr, "%d", &quota)
	if err != nil {
		return m.defaultQuota, fmt.Errorf("failed to parse storage quota: %w", err)
	}

	return quota, nil
}

// SetQuota sets the storage quota limit for a tenant
func (m *StorageQuotaManager) SetQuota(ctx context.Context, tenantID string, quota int64) error {
	return m.redisClient.Set(ctx, StorageQuotaLimitKey(tenantID), quota, 0)
}

// CheckQuota checks if adding the specified bytes would exceed the tenant's quota
func (m *StorageQuotaManager) CheckQuota(ctx context.Context, tenantID string, bytesToAdd int64) (bool, int64, int64, error) {
	currentUsage, err := m.GetUsage(ctx, tenantID)
	if err != nil {
		return false, 0, 0, err
	}

	quota, err := m.GetQuota(ctx, tenantID)
	if err != nil {
		return false, 0, 0, err
	}

	newUsage := currentUsage + bytesToAdd
	hasQuota := newUsage <= quota

	return hasQuota, currentUsage, quota, nil
}

// CanUpload checks if an upload of the specified size is allowed
func (m *StorageQuotaManager) CanUpload(ctx context.Context, tenantID string, fileSize int64) (bool, error) {
	hasQuota, _, quota, err := m.CheckQuota(ctx, tenantID, fileSize)
	if err != nil {
		return false, err
	}

	if !hasQuota {
		return false, fmt.Errorf("storage quota exceeded: %d bytes limit", quota)
	}

	return true, nil
}

// IncrementUsage increments the storage usage for a tenant
func (m *StorageQuotaManager) IncrementUsage(ctx context.Context, tenantID string, bytes int64) error {
	_, err := m.redisClient.IncrBy(ctx, StorageQuotaKey(tenantID), bytes)
	return err
}

// DecrementUsage decrements the storage usage for a tenant
func (m *StorageQuotaManager) DecrementUsage(ctx context.Context, tenantID string, bytes int64) error {
	// Get current usage
	currentUsage, err := m.GetUsage(ctx, tenantID)
	if err != nil {
		return err
	}

	// Ensure we don't go below 0
	newUsage := currentUsage - bytes
	if newUsage < 0 {
		newUsage = 0
	}

	// Use Set instead of DecrBy to ensure we don't go negative
	return m.redisClient.Set(ctx, StorageQuotaKey(tenantID), newUsage, 0)
}

// UpdateUsageAfterDelete updates the storage usage after a soft delete
func (m *StorageQuotaManager) UpdateUsageAfterDelete(ctx context.Context, tenantID string, objectSize int64) error {
	// When soft-deleting, we move to deleted bucket but still count against quota
	// The usage is only decremented when permanently deleted
	return nil
}

// UpdateUsageAfterPermanentDelete updates the storage usage after permanent deletion
func (m *StorageQuotaManager) UpdateUsageAfterPermanentDelete(ctx context.Context, tenantID string, objectSize int64) error {
	return m.DecrementUsage(ctx, tenantID, objectSize)
}

// SetUsage sets the storage usage for a tenant (for initial sync or correction)
func (m *StorageQuotaManager) SetUsage(ctx context.Context, tenantID string, bytes int64) error {
	return m.redisClient.Set(ctx, StorageQuotaKey(tenantID), bytes, 0)
}

// UsageInfo returns complete usage information for a tenant
type UsageInfo struct {
	Usage     int64   `json:"usage_bytes"`
	Quota     int64   `json:"quota_bytes"`
	Available int64   `json:"available_bytes"`
	Percent   float64 `json:"usage_percent"`
}

// GetUsageInfo returns complete storage usage information for a tenant
func (m *StorageQuotaManager) GetUsageInfo(ctx context.Context, tenantID string) (*UsageInfo, error) {
	usage, err := m.GetUsage(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	quota, err := m.GetQuota(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	available := quota - usage
	if available < 0 {
		available = 0
	}

	percent := float64(0)
	if quota > 0 {
		percent = float64(usage) / float64(quota) * 100
	}

	return &UsageInfo{
		Usage:     usage,
		Quota:     quota,
		Available: available,
		Percent:   percent,
	}, nil
}

// RecalculateUsage recalculates the storage usage by scanning all objects
func (m *StorageQuotaManager) RecalculateUsage(ctx context.Context, tenantID string) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	var totalSize int64

	// List all objects in the tenant's prefix in uploads bucket
	for object := range m.minioClient.ListObjects(ctx, BucketUploads, tenantID+"/") {
		if object.Err != nil {
			continue
		}
		totalSize += object.Size
	}

	// List in exports bucket
	for object := range m.minioClient.ListObjects(ctx, BucketExports, tenantID+"/") {
		if object.Err != nil {
			continue
		}
		totalSize += object.Size
	}

	return m.SetUsage(ctx, tenantID, totalSize)
}
