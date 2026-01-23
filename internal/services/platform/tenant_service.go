package platform

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"videostreamgo/internal/config"
	"videostreamgo/internal/database"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
)

// TenantConfig holds configuration for a new tenant
type TenantConfig struct {
	Name             string                 `json:"name" binding:"required"`
	Subdomain        string                 `json:"subdomain" binding:"required"`
	CustomDomains    []string               `json:"custom_domains,omitempty"`
	PlanID           *uuid.UUID             `json:"plan_id,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	BrandingConfig   *BrandingConfig        `json:"branding_config,omitempty"`
	FeatureFlags     map[string]bool        `json:"feature_flags,omitempty"`
	StorageLimitGB   int                    `json:"storage_limit_gb"`
	BandwidthLimitGB int                    `json:"bandwidth_limit_gb"`
	MaxVideos        int                    `json:"max_videos"`
	MaxUsers         int                    `json:"max_users"`
}

// BrandingConfig holds branding configuration for a tenant
type BrandingConfig struct {
	SiteName        string            `json:"site_name"`
	LogoURL         string            `json:"logo_url"`
	FaviconURL      string            `json:"favicon_url"`
	PrimaryColor    string            `json:"primary_color"`
	SecondaryColor  string            `json:"secondary_color"`
	AccentColor     string            `json:"accent_color"`
	BackgroundColor string            `json:"background_color"`
	TextColor       string            `json:"text_color"`
	HeaderHTML      string            `json:"header_html"`
	FooterHTML      string            `json:"footer_html"`
	CustomCSS       string            `json:"custom_css"`
	SocialLinks     map[string]string `json:"social_links"`
}

// TenantStats holds usage statistics for a tenant
type TenantStats struct {
	InstanceID          uuid.UUID  `json:"instance_id"`
	StorageUsedBytes    int64      `json:"storage_used_bytes"`
	StorageLimitBytes   int64      `json:"storage_limit_bytes"`
	BandwidthUsedBytes  int64      `json:"bandwidth_used_bytes"`
	BandwidthLimitBytes int64      `json:"bandwidth_limit_bytes"`
	VideoCount          int64      `json:"video_count"`
	UserCount           int64      `json:"user_count"`
	TotalViews          int64      `json:"total_views"`
	ActiveUsers30Days   int64      `json:"active_users_30_days"`
	LastActivityAt      *time.Time `json:"last_activity_at,omitempty"`
}

// TenantService handles tenant provisioning and management
type TenantService interface {
	ProvisionTenant(customerID string, config TenantConfig) (*master.Instance, error)
	SuspendTenant(instanceID string) error
	ActivateTenant(instanceID string) error
	DeleteTenant(instanceID string) error
	UpdateTenantConfig(instanceID string, config TenantConfig) error
	GetTenantStats(instanceID string) (*TenantStats, error)
	RotateTenantCredentials(instanceID string) error
}

// tenantService implements TenantService
type tenantService struct {
	masterDB       *gorm.DB
	instanceRepo   *masterRepo.InstanceRepository
	dbManager      *database.TenantDBManager
	config         *config.Config
	storageService StorageService
}

// StorageService handles S3 operations
type StorageService interface {
	CreateBucket(bucketName string) error
	DeleteBucket(bucketName string) error
	BucketExists(bucketName string) (bool, error)
}

// NewTenantService creates a new tenant service
func NewTenantService(
	masterDB *gorm.DB,
	instanceRepo *masterRepo.InstanceRepository,
	dbManager *database.TenantDBManager,
	cfg *config.Config,
	storageService StorageService,
) TenantService {
	return &tenantService{
		masterDB:       masterDB,
		instanceRepo:   instanceRepo,
		dbManager:      dbManager,
		config:         cfg,
		storageService: storageService,
	}
}

// ProvisionTenant creates a new tenant instance with all required resources
func (s *tenantService) ProvisionTenant(customerID string, config TenantConfig) (*master.Instance, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("invalid customer ID: %w", err)
	}

	ctx := context.Background()

	// Generate instance ID
	instanceID := uuid.New()

	// Generate database name
	databaseName := fmt.Sprintf("instance_%s", instanceID.String()[:8])

	// Generate storage bucket name
	storageBucket := fmt.Sprintf("instance-%s", instanceID.String()[:8])

	// Prepare metadata
	metadata := make(map[string]interface{})
	if config.Metadata != nil {
		for k, v := range config.Metadata {
			metadata[k] = v
		}
	}
	if config.BrandingConfig != nil {
		metadata["site_name"] = config.BrandingConfig.SiteName
		metadata["logo_url"] = config.BrandingConfig.LogoURL
		metadata["primary_color"] = config.BrandingConfig.PrimaryColor
	}

	// Create instance record
	instance := &master.Instance{
		ID:            instanceID,
		CustomerID:    customerUUID,
		Name:          config.Name,
		Subdomain:     config.Subdomain,
		CustomDomains: config.CustomDomains,
		Status:        master.InstanceStatusProvisioning,
		PlanID:        config.PlanID,
		DatabaseName:  databaseName,
		StorageBucket: storageBucket,
		Metadata:      metadata,
	}

	// Save instance to master database
	if err := s.instanceRepo.Create(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to create instance record: %w", err)
	}

	// Provision database
	if err := s.provisionDatabase(ctx, instance); err != nil {
		// Rollback instance creation on failure
		s.instanceRepo.Delete(ctx, instanceID)
		return nil, fmt.Errorf("failed to provision database: %w", err)
	}

	// Provision storage
	if err := s.provisionStorage(ctx, storageBucket); err != nil {
		// Cleanup database
		s.dropDatabase(ctx, databaseName)
		s.instanceRepo.Delete(ctx, instanceID)
		return nil, fmt.Errorf("failed to provision storage: %w", err)
	}

	// Update instance status to pending (waiting for DNS/SSL)
	instance.Status = master.InstanceStatusPending
	if err := s.instanceRepo.Update(ctx, instance); err != nil {
		return nil, fmt.Errorf("failed to update instance status: %w", err)
	}

	return instance, nil
}

// provisionDatabase creates the database and runs migrations
func (s *tenantService) provisionDatabase(ctx context.Context, instance *master.Instance) error {
	// Create database
	dbName := instance.DatabaseName
	if err := s.createDatabase(ctx, dbName); err != nil {
		return err
	}

	// Note: In production, you would run migrations here
	// For now, we just verify the database connection
	_, err := s.dbManager.GetDBWithContext(ctx, instance.ID.String())
	return err
}

// createDatabase creates a new database
func (s *tenantService) createDatabase(ctx context.Context, dbName string) error {
	// This would typically execute raw SQL to create the database
	// For now, we'll assume the database is created externally
	return nil
}

// dropDatabase drops a database
func (s *tenantService) dropDatabase(ctx context.Context, dbName string) error {
	// This would typically execute raw SQL to drop the database
	return nil
}

// provisionStorage creates the S3 bucket
func (s *tenantService) provisionStorage(ctx context.Context, bucketName string) error {
	return s.storageService.CreateBucket(bucketName)
}

// SuspendTenant suspends a tenant instance
func (s *tenantService) SuspendTenant(instanceID string) error {
	ctx := context.Background()

	id, err := uuid.Parse(instanceID)
	if err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}

	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if instance.Status == master.InstanceStatusSuspended {
		return nil // Already suspended
	}

	// Update status
	instance.Status = master.InstanceStatusSuspended
	if err := s.instanceRepo.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	// Close database connection pool
	if err := s.dbManager.Close(instanceID); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to close database connection: %v\n", err)
	}

	return nil
}

// ActivateTenant activates a suspended tenant instance
func (s *tenantService) ActivateTenant(instanceID string) error {
	ctx := context.Background()

	id, err := uuid.Parse(instanceID)
	if err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}

	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	if instance.Status == master.InstanceStatusActive {
		return nil // Already active
	}

	// Update status
	instance.Status = master.InstanceStatusActive
	if err := s.instanceRepo.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	return nil
}

// DeleteTenant deletes a tenant instance and all associated resources
func (s *tenantService) DeleteTenant(instanceID string) error {
	ctx := context.Background()

	id, err := uuid.Parse(instanceID)
	if err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}

	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	// Check if instance is already terminated
	if instance.Status == master.InstanceStatusTerminated {
		return nil // Already terminated
	}

	// Update status to terminated first
	instance.Status = master.InstanceStatusTerminated
	if err := s.instanceRepo.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	// Close database connection pool
	if err := s.dbManager.Close(instanceID); err != nil {
		// Log but don't fail
		fmt.Printf("Warning: failed to close database connection: %v\n", err)
	}

	// Delete storage bucket (async, as it may take time)
	go func() {
		if err := s.storageService.DeleteBucket(instance.StorageBucket); err != nil {
			fmt.Printf("Warning: failed to delete storage bucket: %v\n", err)
		}
	}()

	// Note: In production, you would also:
	// 1. Delete the database (or mark for deletion after backup)
	// 2. Revoke SSL certificates
	// 3. Remove DNS records
	// 4. Clean up any other resources

	// Soft delete the instance record
	if err := s.instanceRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete instance record: %w", err)
	}

	return nil
}

// UpdateTenantConfig updates the configuration for a tenant
func (s *tenantService) UpdateTenantConfig(instanceID string, config TenantConfig) error {
	ctx := context.Background()

	id, err := uuid.Parse(instanceID)
	if err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}

	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	// Update fields
	instance.Name = config.Name
	instance.Subdomain = config.Subdomain
	instance.CustomDomains = config.CustomDomains
	instance.PlanID = config.PlanID

	// Update metadata
	if config.Metadata != nil {
		for k, v := range config.Metadata {
			instance.Metadata[k] = v
		}
	}
	if config.BrandingConfig != nil {
		if instance.Metadata == nil {
			instance.Metadata = make(map[string]interface{})
		}
		instance.Metadata["site_name"] = config.BrandingConfig.SiteName
		instance.Metadata["logo_url"] = config.BrandingConfig.LogoURL
		instance.Metadata["primary_color"] = config.BrandingConfig.PrimaryColor
	}

	if err := s.instanceRepo.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	return nil
}

// GetTenantStats retrieves usage statistics for a tenant
func (s *tenantService) GetTenantStats(instanceID string) (*TenantStats, error) {
	ctx := context.Background()

	id, err := uuid.Parse(instanceID)
	if err != nil {
		return nil, fmt.Errorf("invalid instance ID: %w", err)
	}

	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("instance not found: %w", err)
	}

	// Calculate storage limits
	storageLimitBytes := int64(100 * 1024 * 1024 * 1024)    // Default 100GB
	bandwidthLimitBytes := int64(1000 * 1024 * 1024 * 1024) // Default 1TB

	stats := &TenantStats{
		InstanceID:          instance.ID,
		StorageLimitBytes:   storageLimitBytes,
		BandwidthLimitBytes: bandwidthLimitBytes,
	}

	// Note: In production, you would query actual usage from:
	// 1. Storage service (S3)
	// 2. Database (video count, user count)
	// 3. Analytics (views, bandwidth)

	return stats, nil
}

// RotateTenantCredentials rotates credentials for a tenant
func (s *tenantService) RotateTenantCredentials(instanceID string) error {
	ctx := context.Background()

	id, err := uuid.Parse(instanceID)
	if err != nil {
		return fmt.Errorf("invalid instance ID: %w", err)
	}

	instance, err := s.instanceRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("instance not found: %w", err)
	}

	// Close existing connection pool to force reconnection with new credentials
	if err := s.dbManager.Close(instanceID); err != nil {
		return fmt.Errorf("failed to close connection pool: %w", err)
	}

	// Note: In production, you would:
	// 1. Generate new database credentials
	// 2. Update the instance record with new credentials
	// 3. Update any secrets management system

	// Update instance to reflect credential rotation
	instance.UpdatedAt = time.Now()
	if err := s.instanceRepo.Update(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	return nil
}
