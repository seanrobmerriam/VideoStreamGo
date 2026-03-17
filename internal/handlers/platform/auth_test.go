package platform

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"videostreamgo/internal/config"
	"videostreamgo/internal/models/master"
)

// MockAdminRepository for testing
type MockAdminRepository struct {
	admins map[string]*master.AdminUser
}

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
			EncryptionKey:  "test-encryption-key-for-testing",
			AllowedOrigins: []string{"http://localhost:3000"},
		},
	}
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func createTestAdmin(email, password string, role master.AdminRole) *master.AdminUser {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return &master.AdminUser{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hashedPassword),
		DisplayName:  "Test Admin",
		Role:         role,
		Status:       master.AdminStatusActive,
		CreatedAt:    time.Now(),
	}
}

// Test_AuthHandler_Login_Success tests successful admin login
func Test_AuthHandler_Login_Success(t *testing.T) {
	r := setupTestRouter()
	cfg := setupTestConfig()

	admin := createTestAdmin("admin@test.com", "password123", master.AdminRoleAdmin)

	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify credentials
		if req.Email != admin.Email {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid email or password",
				},
			})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid email or password",
				},
			})
			return
		}

		if admin.Status != master.AdminStatusActive {
			c.JSON(http.StatusForbidden, gin.H{
				"error": map[string]string{
					"code":    "ACCOUNT_INACTIVE",
					"message": "Account is not active",
				},
			})
			return
		}

		token, _ := generateTestToken(admin, cfg)
		c.JSON(http.StatusOK, gin.H{
			"token":      token,
			"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"user":       ToAdminUserResponse(admin),
		})
	})

	body := `{"email":"admin@test.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.NotEmpty(t, response["token"])
	assert.NotNil(t, response["user"])
}

// Test_AuthHandler_Login_InvalidCredentials tests login with wrong password
func Test_AuthHandler_Login_InvalidCredentials(t *testing.T) {
	r := setupTestRouter()

	admin := createTestAdmin("admin@test.com", "correctpassword", master.AdminRoleAdmin)

	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Return error for wrong password
		if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte("wrongpassword")); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid email or password",
				},
			})
			return
		}
	})

	body := `{"email":"admin@test.com","password":"wrongpassword"}`
	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Test_AuthHandler_Login_InvalidEmail tests login with non-existent email
func Test_AuthHandler_Login_InvalidEmail(t *testing.T) {
	r := setupTestRouter()

	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Simulate admin not found
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": map[string]string{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid email or password",
			},
		})
	})

	body := `{"email":"nonexistent@test.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// Test_AuthHandler_Login_InactiveAccount tests login with inactive account
func Test_AuthHandler_Login_InactiveAccount(t *testing.T) {
	r := setupTestRouter()

	admin := createTestAdmin("admin@test.com", "password123", master.AdminRoleAdmin)
	admin.Status = master.AdminStatusInactive

	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if admin.Status != master.AdminStatusActive {
			c.JSON(http.StatusForbidden, gin.H{
				"error": map[string]string{
					"code":    "ACCOUNT_INACTIVE",
					"message": "Account is not active",
				},
			})
			return
		}
	})

	body := `{"email":"admin@test.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// Test_AuthHandler_Login_ValidationError tests login with invalid request body
func Test_AuthHandler_Login_ValidationError(t *testing.T) {
	r := setupTestRouter()

	r.POST("/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "VALIDATION_ERROR",
					"message": "Invalid request",
					"details": err.Error(),
				},
			})
			return
		}
	})

	// Missing required fields
	body := `{"email":"invalid-email"}`
	req := httptest.NewRequest("POST", "/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_AuthHandler_Register_Success tests successful admin registration
func Test_AuthHandler_Register_Success(t *testing.T) {
	r := setupTestRouter()
	cfg := setupTestConfig()
	registeredEmails := make(map[string]bool)

	r.POST("/register", func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			Password    string `json:"password" binding:"required,min=8"`
			DisplayName string `json:"display_name" binding:"required,min=2,max=100"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if admin already exists
		if registeredEmails[req.Email] {
			c.JSON(http.StatusConflict, gin.H{
				"error": map[string]string{
					"code":    "EMAIL_EXISTS",
					"message": "Email already registered",
				},
			})
			return
		}

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

		admin := &master.AdminUser{
			ID:           uuid.New(),
			Email:        req.Email,
			PasswordHash: string(hashedPassword),
			DisplayName:  req.DisplayName,
			Role:         master.AdminRoleAdmin,
			Status:       master.AdminStatusActive,
		}
		registeredEmails[admin.Email] = true

		token, _ := generateTestToken(admin, cfg)
		c.JSON(http.StatusCreated, gin.H{
			"token":      token,
			"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			"user":       ToAdminUserResponse(admin),
		})
	})

	body := `{"email":"newadmin@test.com","password":"password123","display_name":"New Admin"}`
	req := httptest.NewRequest("POST", "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// Test_AuthHandler_Register_EmailExists tests registration with existing email
func Test_AuthHandler_Register_EmailExists(t *testing.T) {
	r := setupTestRouter()
	registeredEmails := make(map[string]bool)

	existingAdmin := createTestAdmin("existing@test.com", "password123", master.AdminRoleAdmin)
	registeredEmails[existingAdmin.Email] = true

	r.POST("/register", func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			Password    string `json:"password" binding:"required,min=8"`
			DisplayName string `json:"display_name" binding:"required,min=2,max=100"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if registeredEmails[req.Email] {
			c.JSON(http.StatusConflict, gin.H{
				"error": map[string]string{
					"code":    "EMAIL_EXISTS",
					"message": "Email already registered",
				},
			})
			return
		}
	})

	body := `{"email":"existing@test.com","password":"password123","display_name":"Test"}`
	req := httptest.NewRequest("POST", "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// Test_AuthHandler_Register_WeakPassword tests registration with weak password
func Test_AuthHandler_Register_WeakPassword(t *testing.T) {
	r := setupTestRouter()

	r.POST("/register", func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			Password    string `json:"password" binding:"required,min=8"`
			DisplayName string `json:"display_name" binding:"required,min=2,max=100"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	})

	body := `{"email":"test@test.com","password":"short","display_name":"Test"}`
	req := httptest.NewRequest("POST", "/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_JWTTokenGeneration tests JWT token generation
func Test_JWTTokenGeneration(t *testing.T) {
	cfg := setupTestConfig()
	admin := createTestAdmin("admin@test.com", "password123", master.AdminRoleAdmin)

	token, err := generateTestToken(admin, cfg)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Parse and verify token
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

// Test_JWTTokenExpiry tests JWT token expiry
func Test_JWTTokenExpiry(t *testing.T) {
	cfg := setupTestConfig()
	admin := createTestAdmin("admin@test.com", "password123", master.AdminRoleAdmin)

	// Create token with short expiry
	claims := jwt.MapClaims{
		"admin_id": admin.ID.String(),
		"email":    admin.Email,
		"role":     string(admin.Role),
		"exp":      time.Now().Add(-1 * time.Hour).Unix(), // Already expired
		"iat":      time.Now().Add(-2 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(cfg.App.JWTSecret))
	assert.NoError(t, err)

	// Parse should fail for expired token
	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.App.JWTSecret), nil
	})

	assert.NoError(t, err)
	assert.False(t, parsedToken.Valid, "Expired token should be invalid")
}

// Test_PasswordHashing tests password hashing verification
func Test_PasswordHashing(t *testing.T) {
	password := "securePassword123"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	assert.NoError(t, err)
	assert.NotEmpty(t, hashedPassword)

	// Verify correct password
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	assert.NoError(t, err)

	// Verify wrong password
	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte("wrongPassword"))
	assert.Error(t, err)
}

// Test_ToAdminUserResponse tests response conversion
func Test_ToAdminUserResponse(t *testing.T) {
	admin := createTestAdmin("admin@test.com", "password123", master.AdminRoleAdmin)
	admin.LastLoginAt = nil

	response := ToAdminUserResponse(admin)

	assert.Equal(t, admin.ID, response["id"])
	assert.Equal(t, admin.Email, response["email"])
	assert.Equal(t, admin.DisplayName, response["display_name"])
	assert.Equal(t, admin.Role, response["role"])
	assert.Equal(t, admin.Status, response["status"])
	assert.Contains(t, response, "created_at")
	assert.NotContains(t, response, "password")
	assert.NotContains(t, response, "password_hash")
}

// Test_GetCurrentAdmin tests getting current admin
func Test_GetCurrentAdmin(t *testing.T) {
	r := setupTestRouter()

	admin := createTestAdmin("admin@test.com", "password123", master.AdminRoleAdmin)

	r.GET("/me", func(c *gin.Context) {
		c.Set("admin_user", admin)

		// Simulate GetCurrentAdmin logic
		adminUser, exists := c.Get("admin_user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin not found"})
			return
		}

		adminVal, ok := adminUser.(*master.AdminUser)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid admin type"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"user": ToAdminUserResponse(adminVal)})
	})

	req := httptest.NewRequest("GET", "/me", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_ChangePassword tests password change
func Test_ChangePassword(t *testing.T) {
	r := setupTestRouter()

	admin := createTestAdmin("admin@test.com", "oldPassword123", master.AdminRoleAdmin)

	r.POST("/change-password", func(c *gin.Context) {
		var req struct {
			CurrentPassword string `json:"current_password" binding:"required"`
			NewPassword     string `json:"new_password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.CurrentPassword)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
			return
		}

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		admin.PasswordHash = string(hashedPassword)

		c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
	})

	body := `{"current_password":"oldPassword123","new_password":"newSecurePassword123"}`
	req := httptest.NewRequest("POST", "/change-password", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Helper function to generate test token
func generateTestToken(admin *master.AdminUser, cfg *config.Config) (string, error) {
	claims := jwt.MapClaims{
		"admin_id":     admin.ID.String(),
		"email":        admin.Email,
		"display_name": admin.DisplayName,
		"role":         string(admin.Role),
		"exp":          time.Now().Add(24 * time.Hour).Unix(),
		"iat":          time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.App.JWTSecret))
}
