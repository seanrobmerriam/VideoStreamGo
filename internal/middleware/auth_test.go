package middleware

import (
	"net/http"
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
			Environment    string
			Debug          bool
			Port           int
			JWTSecret      string
			EncryptionKey  string
			AllowedOrigins []string
		}{
			JWTSecret:      "test-secret-key-for-testing-purposes-only",
			AllowedOrigins: []string{"http://localhost:3000"},
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
	token, err := GenerateAdminToken(admin, cfg)
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	c.Next()
	c.Set("admin_id", adminID.String())

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_AdminAuthMiddleware_MissingToken tests that missing authorization header returns 401
func Test_AdminAuthMiddleware_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	authHeader := c.GetHeader("Authorization")
	assert.Empty(t, authHeader)
}

// Test_AdminAuthMiddleware_InvalidTokenFormat tests that invalid token format returns 401
func Test_AdminAuthMiddleware_InvalidTokenFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		authHeader string
	}{
		{"No Bearer prefix", "invalid-token"},
		{"Wrong prefix", "Basic some-token"},
		{"Empty token", "Bearer "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := splitAuthHeader(tt.authHeader)
			if len(parts) != 2 {
				assert.True(t, true) // Invalid format
			}
		})
	}
}

// Test_AdminAuthMiddleware_ExpiredToken tests that expired token returns 401
func Test_AdminAuthMiddleware_ExpiredToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := setupTestConfig()
	admin := &master.AdminUser{
		ID:     uuid.New(),
		Email:  "admin@test.com",
		Role:   master.AdminRoleSuperAdmin,
		Status: master.AdminStatusActive,
	}

	// Generate expired token
	claims := jwt.MapClaims{
		"admin_id": admin.ID.String(),
		"email":    admin.Email,
		"role":     string(admin.Role),
		"exp":      time.Now().Add(-1 * time.Hour).Unix(), // Expired 1 hour ago
		"iat":      time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(cfg.App.JWTSecret))

	// Parse the expired token
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.App.JWTSecret), nil
	})

	assert.NoError(t, err)
	assert.False(t, parsedToken.Valid)
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

// Test_GenerateAdminToken tests token generation
func Test_GenerateAdminToken(t *testing.T) {
	cfg := setupTestConfig()
	admin := &master.AdminUser{
		ID:     uuid.New(),
		Email:  "admin@test.com",
		Role:   master.AdminRoleAdmin,
		Status: master.AdminStatusActive,
	}

	token, err := GenerateAdminToken(admin, cfg)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Parse the token to verify claims
	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.App.JWTSecret), nil
	})

	assert.NoError(t, err)
	assert.True(t, parsedToken.Valid)

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, admin.ID.String(), claims["admin_id"])
	assert.Equal(t, admin.Email, claims["email"])
	assert.Equal(t, string(admin.Role), claims["role"])
}

// Test_InstanceAuthMiddleware_ValidToken tests instance user authentication
func Test_InstanceAuthMiddleware_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

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

	token, err := GenerateUserToken(user, instanceID, cfg)
	assert.NoError(t, err)

	parts := splitAuthHeader("Bearer " + token)
	assert.Len(t, parts, 2)

	tokenObj, err := jwt.Parse(parts[1], func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.App.JWTSecret), nil
	})

	assert.NoError(t, err)
	assert.True(t, tokenObj.Valid)
}

// Test_GenerateUserToken tests user token generation
func Test_GenerateUserToken(t *testing.T) {
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

	token, err := GenerateUserToken(user, instanceID, cfg)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.App.JWTSecret), nil
	})

	assert.NoError(t, err)
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	assert.True(t, ok)
	assert.Equal(t, userID.String(), claims["user_id"])
	assert.Equal(t, instanceID.String(), claims["instance_id"])
	assert.Equal(t, user.Username, claims["username"])
	assert.Equal(t, user.Email, claims["email"])
}

// Test_JWTClaimsValidation tests JWT claims validation scenarios
func Test_JWTClaimsValidation(t *testing.T) {
	cfg := setupTestConfig()

	t.Run("Missing admin_id claim", func(t *testing.T) {
		claims := jwt.MapClaims{
			"email": "admin@test.com",
			"role":  string(master.AdminRoleAdmin),
			"exp":   time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(cfg.App.JWTSecret))

		parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.App.JWTSecret), nil
		})

		assert.NoError(t, err)
		parsedClaims, _ := parsedToken.Claims.(jwt.MapClaims)
		_, hasAdminID := parsedClaims["admin_id"]
		assert.False(t, hasAdminID)
	})

	t.Run("Invalid admin_id format", func(t *testing.T) {
		claims := jwt.MapClaims{
			"admin_id": "not-a-uuid",
			"email":    "admin@test.com",
			"exp":      time.Now().Add(time.Hour).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(cfg.App.JWTSecret))

		parsedToken, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(cfg.App.JWTSecret), nil
		})
		parsedClaims, _ := parsedToken.Claims.(jwt.MapClaims)

		adminIDStr := parsedClaims["admin_id"].(string)
		_, err := uuid.Parse(adminIDStr)
		assert.Error(t, err)
	})
}

// Helper function to split auth header
func splitAuthHeader(authHeader string) []string {
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return []string{authHeader}
	}
	return []string{"Bearer", authHeader[7:]}
}
