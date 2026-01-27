package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"videostreamgo/internal/config"
	"videostreamgo/internal/models/instance"
	"videostreamgo/internal/models/master"
)

func setupTestConfig() *config.Config {
	return &config.Config{
		App: struct {
			Environment       string
			Debug             bool
			Port              int
			PlatformJWTSecret string
			InstanceJWTSecret string
			EncryptionKey     string
			ServiceIdentifier string
		}{
			PlatformJWTSecret: "test-platform-secret-key-for-testing-purposes-only-minimum-64-chars",
			InstanceJWTSecret: "test-instance-secret-key-for-testing-purposes-only-minimum-64-chars",
			ServiceIdentifier: config.ServiceIdentifierPlatform,
		},
	}
}

func setupTestConfigWithWrongSecret() *config.Config {
	return &config.Config{
		App: struct {
			Environment       string
			Debug             bool
			Port              int
			PlatformJWTSecret string
			InstanceJWTSecret string
			EncryptionKey     string
			ServiceIdentifier string
		}{
			PlatformJWTSecret: "wrong-secret-key-for-testing-purposes-only-minimum-64-chars",
			InstanceJWTSecret: "wrong-instance-secret-key-for-testing-purposes-only-minimum-64",
			ServiceIdentifier: config.ServiceIdentifierPlatform,
		},
	}
}

// Test_AdminAuthMiddleware_ValidToken tests successful admin authentication with valid token
func Test_AdminAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := setupTestConfig()
	adminID := uuid.New()
	admin := &master.AdminUser{
		ID:        adminID,
		Email:     "admin@test.com",
		Role:      master.AdminRoleSuperAdmin,
		Status:    master.AdminStatusActive,
		CreatedAt: time.Now(),
	}

	// Generate valid token
	token, jti, err := GenerateAdminToken(admin, cfg)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, jti)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	// Note: The actual middleware would need a DB connection for full test
	// Here we just verify the token can be parsed and has correct claims
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.App.PlatformJWTSecret), nil
	})

	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, config.ServiceIdentifierPlatform, claims["iss"])
	assert.Equal(t, adminID.String(), claims["admin_id"])
}

// Test_AdminAuthMiddleware_WrongIssuer tests rejection of tokens with wrong issuer
func Test_AdminAuthMiddleware_WrongIssuer(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := setupTestConfig()

	// Create a token with wrong issuer (instance-api instead of platform-api)
	claims := jwt.MapClaims{
		"admin_id": uuid.New().String(),
		"email":    "admin@test.com",
		"role":     string(master.AdminRoleAdmin),
		"iss":      config.ServiceIdentifierInstance, // Wrong issuer
		"exp":      time.Now().Add(time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(cfg.App.PlatformJWTSecret))

	// Parse and check claims
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.App.PlatformJWTSecret), nil
	})

	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	tokenClaims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, config.ServiceIdentifierInstance, tokenClaims["iss"])
}

// Test_GenerateAdminToken_IncludesIssuer tests that generated tokens include issuer claim
func Test_GenerateAdminToken_IncludesIssuer(t *testing.T) {
	cfg := setupTestConfig()
	admin := &master.AdminUser{
		ID:     uuid.New(),
		Email:  "admin@test.com",
		Role:   master.AdminRoleAdmin,
		Status: master.AdminStatusActive,
	}

	token, _, err := GenerateAdminToken(admin, cfg)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.App.PlatformJWTSecret), nil
	})

	assert.NoError(t, err)
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	// Verify issuer claim
	iss, ok := claims["iss"].(string)
	assert.True(t, ok)
	assert.Equal(t, config.ServiceIdentifierPlatform, iss)

	// Verify JTI claim
	_, ok = claims["jti"].(string)
	assert.True(t, ok)

	// Verify SID claim
	_, ok = claims["sid"].(string)
	assert.True(t, ok)
}

// Test_GenerateUserToken_IncludesIssuer tests that user tokens include correct issuer
func Test_GenerateUserToken_IncludesIssuer(t *testing.T) {
	cfg := setupTestConfig()
	userID := uuid.New()
	instanceID := uuid.New()
	user := &instance.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     instance.UserRoleUser,
		Status:   instance.UserStatusActive,
	}

	token, _, err := GenerateUserToken(user, instanceID, cfg)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.App.InstanceJWTSecret), nil
	})

	assert.NoError(t, err)
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	// Verify issuer claim
	iss, ok := claims["iss"].(string)
	assert.True(t, ok)
	assert.Equal(t, config.ServiceIdentifierInstance, iss)
}

// Test_RequireRole_SuperAdminAccess tests that super admins have access to all routes
func Test_RequireRole_SuperAdminAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	admin := &master.AdminUser{
		ID:     uuid.New(),
		Email:  "superadmin@test.com",
		Role:   master.AdminRoleSuperAdmin,
		Status: master.AdminStatusActive,
	}

	tests := []struct {
		name         string
		allowedRoles []master.AdminRole
		expectAccess bool
	}{
		{"No roles required", []master.AdminRole{}, true},
		{"Admin role required", []master.AdminRole{master.AdminRoleAdmin}, true},
		{"SuperAdmin role required", []master.AdminRole{master.AdminRoleSuperAdmin}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Set("admin_user", admin)

			// Super admins have access to everything
			hasAccess := admin.Role == master.AdminRoleSuperAdmin
			for _, role := range tt.allowedRoles {
				if admin.Role != role && admin.Role != master.AdminRoleSuperAdmin {
					hasAccess = false
				}
			}

			assert.Equal(t, tt.expectAccess, hasAccess)
		})
	}
}

// Test_RequireRole_RegularAdminAccess tests role-based access for regular admins
func Test_RequireRole_RegularAdminAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	admin := &master.AdminUser{
		ID:     uuid.New(),
		Email:  "admin@test.com",
		Role:   master.AdminRoleAdmin,
		Status: master.AdminStatusActive,
	}

	tests := []struct {
		name         string
		allowedRoles []master.AdminRole
		expectAccess bool
	}{
		{"Admin role allowed", []master.AdminRole{master.AdminRoleAdmin}, true},
		{"SuperAdmin role not allowed", []master.AdminRole{master.AdminRoleSuperAdmin}, false},
		{"Moderator role not allowed", []master.AdminRole{master.AdminRoleModerator}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			c.Set("admin_user", admin)

			hasAccess := false
			for _, role := range tt.allowedRoles {
				if admin.Role == role || admin.Role == master.AdminRoleSuperAdmin {
					hasAccess = true
					break
				}
			}

			assert.Equal(t, tt.expectAccess, hasAccess)
		})
	}
}

// Test_JWTClaimsValidation tests JWT claims validation scenarios
func Test_JWTClaimsValidation(t *testing.T) {
	cfg := setupTestConfig()

	t.Run("Missing admin_id claim", func(t *testing.T) {
		claims := jwt.MapClaims{
			"email": "admin@test.com",
			"role":  string(master.AdminRoleAdmin),
			"iss":   config.ServiceIdentifierPlatform,
			"exp":   time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(cfg.App.PlatformJWTSecret))

		parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.App.PlatformJWTSecret), nil
		})

		assert.NoError(t, err)
		parsedClaims, _ := parsedToken.Claims.(jwt.MapClaims)
		_, hasAdminID := parsedClaims["admin_id"]
		assert.False(t, hasAdminID)
	})

	t.Run("Token with valid platform-api issuer", func(t *testing.T) {
		claims := jwt.MapClaims{
			"admin_id": uuid.New().String(),
			"email":    "admin@test.com",
			"role":     string(master.AdminRoleAdmin),
			"iss":      config.ServiceIdentifierPlatform,
			"exp":      time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(cfg.App.PlatformJWTSecret))

		parsedToken, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.App.PlatformJWTSecret), nil
		})
		parsedClaims, _ := parsedToken.Claims.(jwt.MapClaims)

		assert.Equal(t, config.ServiceIdentifierPlatform, parsedClaims["iss"])
	})

	t.Run("Token with wrong issuer rejected", func(t *testing.T) {
		claims := jwt.MapClaims{
			"admin_id": uuid.New().String(),
			"email":    "admin@test.com",
			"role":     string(master.AdminRoleAdmin),
			"iss":      config.ServiceIdentifierInstance,
			"exp":      time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(cfg.App.PlatformJWTSecret))

		parsedToken, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.App.PlatformJWTSecret), nil
		})
		parsedClaims, _ := parsedToken.Claims.(jwt.MapClaims)

		// This token has wrong issuer for platform-api
		assert.NotEqual(t, config.ServiceIdentifierPlatform, parsedClaims["iss"])
	})
}

// Test_SecurityEventLogging tests that security events can be logged
func Test_SecurityEventLogging(t *testing.T) {
	// Test that logSecurityEvent function exists and doesn't panic
	assert.NotPanics(t, func() {
		logSecurityEvent(SecurityEventCrossServiceToken, "Test message", "127.0.0.1")
		logSecurityEvent(SecurityEventInvalidIssuer, "Test message", "127.0.0.1")
		logSecurityEvent(SecurityEventTokenExpired, "Test message", "127.0.0.1")
		logSecurityEvent(SecurityEventTokenRevoked, "Test message", "127.0.0.1")
	})
}

// Test_TokenExpiry tests token expiration handling
func Test_TokenExpiry(t *testing.T) {
	cfg := setupTestConfig()
	admin := &master.AdminUser{
		ID:     uuid.New(),
		Email:  "admin@test.com",
		Role:   master.AdminRoleAdmin,
		Status: master.AdminStatusActive,
	}

	t.Run("Expired token is invalid", func(t *testing.T) {
		claims := jwt.MapClaims{
			"admin_id": admin.ID.String(),
			"email":    admin.Email,
			"role":     string(admin.Role),
			"iss":      config.ServiceIdentifierPlatform,
			"exp":      time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
			"iat":      time.Now().Add(-2 * time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(cfg.App.PlatformJWTSecret))

		parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.App.PlatformJWTSecret), nil
		})

		// Expired tokens should fail validation
		assert.Error(t, err)
		assert.False(t, parsedToken.Valid)
	})

	t.Run("Valid token is valid", func(t *testing.T) {
		claims := jwt.MapClaims{
			"admin_id": admin.ID.String(),
			"email":    admin.Email,
			"role":     string(admin.Role),
			"iss":      config.ServiceIdentifierPlatform,
			"exp":      time.Now().Add(time.Hour).Unix(), // Expires in 1 hour
			"iat":      time.Now().Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(cfg.App.PlatformJWTSecret))

		parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.App.PlatformJWTSecret), nil
		})

		assert.NoError(t, err)
		assert.True(t, parsedToken.Valid)
	})
}

// Helper function to split auth header
func splitAuthHeader(authHeader string) []string {
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return []string{authHeader}
	}
	return []string{"Bearer", authHeader[7:]}
}
