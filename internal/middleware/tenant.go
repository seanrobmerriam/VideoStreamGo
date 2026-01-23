package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
	"videostreamgo/internal/types"
)

// tenantContextKey is the context key for tenant information
const tenantContextKey = "tenant_context"

// PlatformDomains contains the platform's own domains
var PlatformDomains = []string{
	"videostreamgo.com",
	"www.videostreamgo.com",
	"api.videostreamgo.com",
	"admin.videostreamgo.com",
	"localhost",
}

// TenantContext holds comprehensive tenant information
type TenantContext struct {
	InstanceID     uuid.UUID
	CustomerID     uuid.UUID
	Name           string
	Subdomain      string
	CustomDomains  []string
	DatabaseName   string
	StorageBucket  string
	Status         master.InstanceStatus
	PlanID         *uuid.UUID
	CreatedAt      time.Time
	ActivatedAt    *time.Time
	BrandingConfig map[string]string
	FeatureFlags   map[string]bool
	RateLimits     RateLimitConfig
}

// RateLimitConfig holds rate limiting configuration for a tenant
type RateLimitConfig struct {
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
	BurstSize         int
}

// NewTenantMiddleware creates a new tenant middleware
func NewTenantMiddleware(repo *masterRepo.InstanceRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantCtx, err := resolveTenant(c, repo)
		if err != nil {
			handleTenantError(c, err)
			return
		}

		// Set tenant context in Gin context
		c.Set(string(types.ContextKeyTenantID), tenantCtx.InstanceID)
		c.Set(string(types.ContextKeyInstance), &master.Instance{
			ID:            tenantCtx.InstanceID,
			CustomerID:    tenantCtx.CustomerID,
			Name:          tenantCtx.Name,
			Subdomain:     tenantCtx.Subdomain,
			CustomDomains: tenantCtx.CustomDomains,
			DatabaseName:  tenantCtx.DatabaseName,
			StorageBucket: tenantCtx.StorageBucket,
			Status:        tenantCtx.Status,
			PlanID:        tenantCtx.PlanID,
		})
		c.Set(string(types.ContextKeyInstanceID), tenantCtx.InstanceID)
		c.Set(tenantContextKey, tenantCtx)

		// Set tenant context in request context for propagation
		ctx := context.WithValue(c.Request.Context(), tenantContextKey, tenantCtx)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// resolveTenant resolves the tenant from the request
func resolveTenant(c *gin.Context, repo *masterRepo.InstanceRepository) (*TenantContext, error) {
	host := extractHost(c.Request.Host)
	if host == "" {
		return nil, &TenantError{
			Code:    "MISSING_HOST",
			Message: "Host header is missing",
			Status:  http.StatusBadRequest,
		}
	}

	// Check if it's a platform domain
	if isPlatformDomain(host) {
		return createPlatformTenantContext(), nil
	}

	// Extract subdomain
	subdomain := extractSubdomain(host)
	if subdomain == "" {
		return nil, &TenantError{
			Code:    "INVALID_SUBDOMAIN",
			Message: "Could not extract subdomain from host",
			Status:  http.StatusBadRequest,
		}
	}

	// Look up instance by subdomain
	return resolveSubdomain(c, subdomain, repo)
}

// extractHost extracts the host from the request, removing port if present
func extractHost(host string) string {
	if host == "" {
		return ""
	}

	// Handle IPv6 addresses
	if strings.HasPrefix(host, "[") {
		if idx := strings.LastIndex(host, "]"); idx != -1 {
			return host[:idx+1]
		}
		return host
	}

	// Remove port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		return host[:idx]
	}

	return host
}

// extractSubdomain extracts the subdomain from a host
func extractSubdomain(host string) string {
	parts := strings.Split(host, ".")

	// Handle localhost for development
	if len(parts) > 0 && parts[len(parts)-1] == "localhost" {
		if len(parts) >= 2 {
			return parts[0]
		}
		return ""
	}

	// Handle development with port (e.g., localhost:8080)
	if len(parts) > 0 && strings.Contains(parts[0], ":") {
		hostParts := strings.Split(parts[0], ":")
		return hostParts[0]
	}

	// Standard subdomain extraction (subdomain.domain.tld)
	if len(parts) >= 3 {
		return parts[0]
	}

	return ""
}

// isSubdomain checks if the host is a subdomain (not the bare domain)
func isSubdomain(host string) bool {
	parts := strings.Split(host, ".")

	// Localhost handling
	if len(parts) > 0 && parts[len(parts)-1] == "localhost" {
		return len(parts) >= 2 && parts[0] != ""
	}

	// For production, anything other than the bare domain is a subdomain
	return len(parts) > 2
}

// isPlatformDomain checks if the host is a platform domain
func isPlatformDomain(host string) bool {
	host = strings.ToLower(host)
	for _, domain := range PlatformDomains {
		if host == domain {
			return true
		}
	}
	return false
}

// createPlatformTenantContext creates a tenant context for platform routes
func createPlatformTenantContext() *TenantContext {
	return &TenantContext{
		InstanceID:    uuid.Nil,
		CustomerID:    uuid.Nil,
		Name:          "platform",
		Subdomain:     "platform",
		DatabaseName:  "",
		StorageBucket: "",
		Status:        master.InstanceStatusActive,
		FeatureFlags: map[string]bool{
			"platform_access": true,
		},
	}
}

// resolveSubdomain resolves a tenant by subdomain
func resolveSubdomain(c *gin.Context, subdomain string, repo *masterRepo.InstanceRepository) (*TenantContext, error) {
	instance, err := repo.GetBySubdomain(c.Request.Context(), subdomain)
	if err != nil {
		return nil, &TenantError{
			Code:    "TENANT_NOT_FOUND",
			Message: "Instance not found for subdomain: " + subdomain,
			Status:  http.StatusNotFound,
			Err:     err,
		}
	}

	if instance.Status != master.InstanceStatusActive {
		return nil, &TenantError{
			Code:    "TENANT_INACTIVE",
			Message: "Instance is not active",
			Status:  http.StatusForbidden,
		}
	}

	return buildTenantContext(instance), nil
}

// buildTenantContext builds a TenantContext from an Instance model
func buildTenantContext(instance *master.Instance) *TenantContext {
	return &TenantContext{
		InstanceID:     instance.ID,
		CustomerID:     instance.CustomerID,
		Name:           instance.Name,
		Subdomain:      instance.Subdomain,
		CustomDomains:  instance.CustomDomains,
		DatabaseName:   instance.DatabaseName,
		StorageBucket:  instance.StorageBucket,
		Status:         instance.Status,
		PlanID:         instance.PlanID,
		CreatedAt:      instance.CreatedAt,
		ActivatedAt:    instance.ActivatedAt,
		BrandingConfig: extractBrandingConfig(instance),
		FeatureFlags:   getDefaultFeatureFlags(instance),
		RateLimits:     getDefaultRateLimits(),
	}
}

// extractBrandingConfig extracts branding configuration from instance metadata
func extractBrandingConfig(instance *master.Instance) map[string]string {
	config := make(map[string]string)

	if instance.Metadata != nil {
		if siteName, ok := instance.Metadata["site_name"].(string); ok {
			config["site_name"] = siteName
		}
		if logoURL, ok := instance.Metadata["logo_url"].(string); ok {
			config["logo_url"] = logoURL
		}
		if primaryColor, ok := instance.Metadata["primary_color"].(string); ok {
			config["primary_color"] = primaryColor
		}
	}

	// Set defaults
	if config["site_name"] == "" {
		config["site_name"] = instance.Name
	}
	if config["primary_color"] == "" {
		config["primary_color"] = "#2563eb"
	}

	return config
}

// getDefaultFeatureFlags returns default feature flags based on instance status
func getDefaultFeatureFlags(instance *master.Instance) map[string]bool {
	flags := map[string]bool{
		"video_upload":      true,
		"user_registration": true,
		"comments":          true,
		"likes":             true,
		"playlists":         true,
	}

	// Disable features for non-active instances
	if instance.Status != master.InstanceStatusActive {
		flags["video_upload"] = false
		flags["user_registration"] = false
	}

	return flags
}

// getDefaultRateLimits returns default rate limits
func getDefaultRateLimits() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 60,
		RequestsPerHour:   1000,
		RequestsPerDay:    10000,
		BurstSize:         10,
	}
}

// TenantError represents an error during tenant resolution
type TenantError struct {
	Code    string
	Message string
	Status  int
	Err     error
}

func (e *TenantError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// handleTenantError handles tenant resolution errors
func handleTenantError(c *gin.Context, err error) {
	if tenantErr, ok := err.(*TenantError); ok {
		c.AbortWithStatusJSON(tenantErr.Status, types.ErrorResponse(
			tenantErr.Code,
			tenantErr.Message,
			"",
		))
		return
	}

	c.AbortWithStatusJSON(http.StatusInternalServerError, types.ErrorResponse(
		"TENANT_RESOLUTION_ERROR",
		"Failed to resolve tenant",
		err.Error(),
	))
}

// GetTenantContext retrieves the tenant context from Gin context
func GetTenantContext(c *gin.Context) *TenantContext {
	if tenantCtx, exists := c.Get(tenantContextKey); exists {
		if ctx, ok := tenantCtx.(*TenantContext); ok {
			return ctx
		}
	}
	return nil
}

// GetTenantFromContext retrieves the tenant context from the request context
func GetTenantFromContext(ctx context.Context) *TenantContext {
	if tenantCtx, ok := ctx.Value(tenantContextKey).(*TenantContext); ok {
		return tenantCtx
	}
	return nil
}

// GetTenantID retrieves the tenant ID from context
func GetTenantID(c *gin.Context) uuid.UUID {
	if tenantID, exists := c.Get(string(types.ContextKeyTenantID)); exists {
		if id, ok := tenantID.(uuid.UUID); ok {
			return id
		}
	}
	return uuid.Nil
}

// GetInstance retrieves the instance from context
func GetInstance(c *gin.Context) *master.Instance {
	if instance, exists := c.Get(string(types.ContextKeyInstance)); exists {
		if inst, ok := instance.(*master.Instance); ok {
			return inst
		}
	}
	return nil
}

// RequireTenant is a middleware that requires a valid tenant context
func RequireTenant() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantCtx := GetTenantContext(c)
		if tenantCtx == nil || tenantCtx.InstanceID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, types.ErrorResponse(
				"TENANT_REQUIRED",
				"Tenant context is required for this route",
				"",
			))
			return
		}

		if tenantCtx.Status != master.InstanceStatusActive {
			c.AbortWithStatusJSON(http.StatusForbidden, types.ErrorResponse(
				"TENANT_INACTIVE",
				"Instance is not active",
				"",
			))
			return
		}

		c.Next()
	}
}

// RequirePlatform is a middleware that requires platform (non-tenant) context
func RequirePlatform() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantCtx := GetTenantContext(c)
		if tenantCtx != nil && tenantCtx.InstanceID != uuid.Nil {
			c.AbortWithStatusJSON(http.StatusForbidden, types.ErrorResponse(
				"PLATFORM_ONLY",
				"This route is only available on the platform domain",
				"",
			))
			return
		}
		c.Next()
	}
}
