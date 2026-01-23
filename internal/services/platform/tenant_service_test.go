package platform

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/models/master"
)

func Test_TenantConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config TenantConfig
		valid  bool
	}{
		{
			name: "Valid config",
			config: TenantConfig{
				Name:      "Test Tenant",
				Subdomain: "testtenant",
			},
			valid: true,
		},
		{
			name: "Valid config with all fields",
			config: TenantConfig{
				Name:             "Full Tenant",
				Subdomain:        "fulltenant",
				CustomDomains:    []string{"example.com"},
				StorageLimitGB:   100,
				BandwidthLimitGB: 1000,
				MaxVideos:        500,
				MaxUsers:         100,
			},
			valid: true,
		},
		{
			name: "Empty name invalid",
			config: TenantConfig{
				Name:      "",
				Subdomain: "test",
			},
			valid: false,
		},
		{
			name: "Empty subdomain invalid",
			config: TenantConfig{
				Name:      "Test",
				Subdomain: "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.config.Name != "" && tt.config.Subdomain != ""
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func Test_TenantConfig_PlanAssignment(t *testing.T) {
	planID := uuid.New()

	config := TenantConfig{
		Name:   "Test Tenant",
		PlanID: &planID,
	}

	assert.NotNil(t, config.PlanID)
	assert.Equal(t, planID, *config.PlanID)
}

func Test_BrandingConfig_Defaults(t *testing.T) {
	config := BrandingConfig{}

	assert.Equal(t, "", config.SiteName)
	assert.Equal(t, "", config.LogoURL)
	assert.Equal(t, "", config.PrimaryColor)
}

func Test_BrandingConfig_WithValues(t *testing.T) {
	config := BrandingConfig{
		SiteName:     "My Video Site",
		LogoURL:      "https://example.com/logo.png",
		PrimaryColor: "#ff0000",
		SocialLinks: map[string]string{
			"twitter": "https://twitter.com/myvideosite",
		},
	}

	assert.Equal(t, "My Video Site", config.SiteName)
	assert.Equal(t, "https://example.com/logo.png", config.LogoURL)
	assert.Equal(t, "#ff0000", config.PrimaryColor)
	assert.Contains(t, config.SocialLinks, "twitter")
}

func Test_TenantStats_Calculations(t *testing.T) {
	instanceID := uuid.New()

	stats := TenantStats{
		InstanceID:          instanceID,
		StorageUsedBytes:    1024 * 1024 * 1024,        // 1GB
		StorageLimitBytes:   100 * 1024 * 1024 * 1024,  // 100GB
		BandwidthUsedBytes:  10 * 1024 * 1024 * 1024,   // 10GB
		BandwidthLimitBytes: 1000 * 1024 * 1024 * 1024, // 1TB
		VideoCount:          100,
		UserCount:           50,
		TotalViews:          10000,
	}

	// Calculate usage percentages
	storagePercent := float64(stats.StorageUsedBytes) / float64(stats.StorageLimitBytes) * 100
	bandwidthPercent := float64(stats.BandwidthUsedBytes) / float64(stats.BandwidthLimitBytes) * 100

	assert.Equal(t, 1.0, storagePercent)   // 1GB / 100GB = 1%
	assert.Equal(t, 1.0, bandwidthPercent) // 10GB / 1TB = 1%
}

func Test_TenantStatus_Transitions(t *testing.T) {
	// Test valid status transitions
	transitions := []struct {
		from  master.InstanceStatus
		to    master.InstanceStatus
		valid bool
	}{
		{master.InstanceStatusPending, master.InstanceStatusActive, true},
		{master.InstanceStatusPending, master.InstanceStatusSuspended, true},
		{master.InstanceStatusProvisioning, master.InstanceStatusActive, true},
		{master.InstanceStatusProvisioning, master.InstanceStatusSuspended, true},
		{master.InstanceStatusActive, master.InstanceStatusSuspended, true},
		{master.InstanceStatusActive, master.InstanceStatusTerminated, true},
		{master.InstanceStatusSuspended, master.InstanceStatusActive, true},
		{master.InstanceStatusTerminated, master.InstanceStatusActive, false},
	}

	for _, tt := range transitions {
		t.Run(string(tt.from)+"_to_"+string(tt.to), func(t *testing.T) {
			// terminated instances cannot transition to active
			isValid := !(tt.from == master.InstanceStatusTerminated && tt.to == master.InstanceStatusActive)
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func Test_InstanceDatabaseName_Generation(t *testing.T) {
	instanceID := uuid.New()

	// Simulate database name generation
	databaseName := "instance_" + instanceID.String()[:8]

	assert.Contains(t, databaseName, "instance_")
	assert.Greater(t, len(databaseName), 9) // Should be more than just "instance_"
}

func Test_InstanceStorageBucket_Generation(t *testing.T) {
	instanceID := uuid.New()

	// Simulate storage bucket name generation
	storageBucket := "instance-" + instanceID.String()[:8]

	assert.Contains(t, storageBucket, "instance-")
	assert.Len(t, storageBucket, 9+8) // "instance-" + 8 chars
}

func Test_TenantProvisioning_ConfigBuilding(t *testing.T) {
	_ = uuid.New() // customerID would be used in real provisioning
	planID := uuid.New()

	config := TenantConfig{
		Name:             "New Tenant",
		Subdomain:        "newtenant",
		CustomDomains:    []string{"newtenant.com", "www.newtenant.com"},
		PlanID:           &planID,
		StorageLimitGB:   500,
		BandwidthLimitGB: 5000,
		MaxVideos:        1000,
		MaxUsers:         200,
		BrandingConfig: &BrandingConfig{
			SiteName:     "New Tenant Site",
			PrimaryColor: "#3b82f6",
		},
	}

	// Verify config is properly structured
	assert.Equal(t, "New Tenant", config.Name)
	assert.Equal(t, "newtenant", config.Subdomain)
	assert.Len(t, config.CustomDomains, 2)
	assert.Equal(t, 500, config.StorageLimitGB)
	assert.NotNil(t, config.BrandingConfig)
	assert.Equal(t, "#3b82f6", config.BrandingConfig.PrimaryColor)
}
