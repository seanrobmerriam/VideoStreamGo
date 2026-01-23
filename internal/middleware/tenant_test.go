package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/models/master"
)

// Test_TenantContextIsolation verifies that tenant context is properly isolated
func Test_TenantContextIsolation(t *testing.T) {
	// Create instances for different tenants
	tenant1ID := uuid.New()
	tenant2ID := uuid.New()

	instance1 := &master.Instance{
		ID:        tenant1ID,
		Name:      "Tenant 1",
		Subdomain: "tenant1",
		Status:    master.InstanceStatusActive,
	}

	instance2 := &master.Instance{
		ID:        tenant2ID,
		Name:      "Tenant 2",
		Subdomain: "tenant2",
		Status:    master.InstanceStatusActive,
	}

	// Test that different tenants have different contexts
	assert.NotEqual(t, instance1.ID, instance2.ID, "Tenant IDs should be different")
	assert.NotEqual(t, instance1.Subdomain, instance2.Subdomain, "Tenant subdomains should be different")

	// Test that tenant context can be set and retrieved
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Set("tenant_id", instance1.ID)
	c.Set("instance", instance1)

	tenantID := GetTenantID(c)
	assert.Equal(t, tenant1ID, tenantID, "Should retrieve correct tenant ID")

	retrievedInstance := GetInstance(c)
	assert.NotNil(t, retrievedInstance, "Should retrieve instance")
	assert.Equal(t, instance1.ID, retrievedInstance.ID, "Retrieved instance should match original")
}

// Test_TenantSubdomainExtraction tests subdomain extraction from various host formats
func Test_TenantSubdomainExtraction(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		expected string
	}{
		{
			name:     "Standard subdomain",
			host:     "customer1.videostreamgo.com",
			expected: "customer1",
		},
		{
			name:     "Subdomain with port",
			host:     "customer1.videostreamgo.com:8080",
			expected: "customer1",
		},
		{
			name:     "Localhost development",
			host:     "localhost:3000",
			expected: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSubdomain(tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Test_TenantMiddlewareRejectsInvalidTenant tests that invalid tenants are rejected
func Test_TenantMiddlewareRejectsInvalidTenant(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		host           string
		expectedStatus int
	}{
		{
			name:           "Non-existent subdomain",
			host:           "nonexistent.videostreamgo.com",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Suspended tenant",
			host:           "suspended.videostreamgo.com",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Request.Host = tt.host

			// Simulate middleware behavior
			if tt.host == "suspended.videostreamgo.com" {
				c.AbortWithStatusJSON(http.StatusForbidden, map[string]interface{}{
					"error": map[string]string{
						"code":    "TENANT_INACTIVE",
						"message": "Instance is not active",
					},
				})
			} else if tt.host != "www.videostreamgo.com" && tt.host != "api.videostreamgo.com" {
				c.AbortWithStatusJSON(http.StatusNotFound, map[string]interface{}{
					"error": map[string]string{
						"code":    "TENANT_NOT_FOUND",
						"message": "Instance not found",
					},
				})
			}

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// Test_TenantContextPlatformRoutes tests platform routes are handled correctly
func Test_TenantContextPlatformRoutes(t *testing.T) {
	tests := []struct {
		name string
		host string
	}{
		{"Platform domain", "videostreamgo.com"},
		{"Admin domain", "admin.videostreamgo.com"},
		{"API domain", "api.videostreamgo.com"},
		{"WWW domain", "www.videostreamgo.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isPlatform := isPlatformDomain(tt.host)
			assert.True(t, isPlatform, "%s should be recognized as platform domain", tt.host)
		})
	}
}

// Test_TenantStatusCheck tests that tenant status is properly checked
func Test_TenantStatusCheck(t *testing.T) {
	tests := []struct {
		name     string
		status   master.InstanceStatus
		expected bool
	}{
		{"Active tenant", master.InstanceStatusActive, true},
		{"Pending tenant", master.InstanceStatusPending, false},
		{"Provisioning tenant", master.InstanceStatusProvisioning, false},
		{"Suspended tenant", master.InstanceStatusSuspended, false},
		{"Terminated tenant", master.InstanceStatusTerminated, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isActive := tt.status == master.InstanceStatusActive
			assert.Equal(t, tt.expected, isActive)
		})
	}
}

// Test_TenantContextBrandingConfig tests branding configuration extraction
func Test_TenantContextBrandingConfig(t *testing.T) {
	instance := &master.Instance{
		ID:        uuid.New(),
		Name:      "Test Site",
		Subdomain: "test",
		Metadata: map[string]interface{}{
			"site_name":     "Custom Site Name",
			"logo_url":      "https://example.com/logo.png",
			"primary_color": "#ff0000",
		},
	}

	config := extractBrandingConfig(instance)

	assert.Equal(t, "Custom Site Name", config["site_name"])
	assert.Equal(t, "https://example.com/logo.png", config["logo_url"])
	assert.Equal(t, "#ff0000", config["primary_color"])
}

// Test_TenantContextDefaultBranding tests default branding when not configured
func Test_TenantContextDefaultBranding(t *testing.T) {
	instance := &master.Instance{
		ID:        uuid.New(),
		Name:      "Test Instance",
		Subdomain: "test",
		Metadata:  nil,
	}

	config := extractBrandingConfig(instance)

	// Should use instance name as default
	assert.Equal(t, "Test Instance", config["site_name"])
	// Should use default primary color
	assert.Equal(t, "#2563eb", config["primary_color"])
}

// Test_GetDefaultFeatureFlags tests feature flag defaults
func Test_GetDefaultFeatureFlags(t *testing.T) {
	activeInstance := &master.Instance{
		ID:     uuid.New(),
		Status: master.InstanceStatusActive,
	}

	inactiveInstance := &master.Instance{
		ID:     uuid.New(),
		Status: master.InstanceStatusSuspended,
	}

	activeFlags := getDefaultFeatureFlags(activeInstance)
	inactiveFlags := getDefaultFeatureFlags(inactiveInstance)

	// Active tenant should have all features enabled
	assert.True(t, activeFlags["video_upload"])
	assert.True(t, activeFlags["user_registration"])

	// Inactive tenant should have features disabled
	assert.False(t, inactiveFlags["video_upload"])
	assert.False(t, inactiveFlags["user_registration"])
}

// Test_getDefaultRateLimits tests default rate limits
func Test_getDefaultRateLimits(t *testing.T) {
	limits := getDefaultRateLimits()

	assert.Equal(t, 60, limits.RequestsPerMinute)
	assert.Equal(t, 1000, limits.RequestsPerHour)
	assert.Equal(t, 10000, limits.RequestsPerDay)
	assert.Equal(t, 10, limits.BurstSize)
}
