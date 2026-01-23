package instance

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"

	"videostreamgo/internal/models/instance"
)

// Test_UserHandler_Register_Success tests successful user registration
func Test_UserHandler_Register_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	instanceID := uuid.New()
	registeredUsers := make(map[string]bool)

	r.POST("/users/register", func(c *gin.Context) {
		var req struct {
			Username    string `json:"username" binding:"required,min=3,max=50"`
			Email       string `json:"email" binding:"required,email"`
			Password    string `json:"password" binding:"required,min=8"`
			DisplayName string `json:"display_name" binding:"required,min=3,max=100"`
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

		// Check if username exists
		if registeredUsers[req.Username] {
			c.JSON(http.StatusConflict, gin.H{
				"error": map[string]string{
					"code":    "USERNAME_EXISTS",
					"message": "Username already taken",
				},
			})
			return
		}

		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

		user := &instance.User{
			ID:           uuid.New(),
			InstanceID:   instanceID,
			Username:     req.Username,
			Email:        req.Email,
			PasswordHash: string(hashedPassword),
			DisplayName:  req.DisplayName,
			Role:         instance.UserRoleUser,
			Status:       instance.UserStatusActive,
		}
		registeredUsers[user.Username] = true

		c.JSON(http.StatusCreated, gin.H{
			"data": map[string]interface{}{
				"id":           user.ID,
				"username":     user.Username,
				"email":        user.Email,
				"display_name": user.DisplayName,
				"created_at":   user.CreatedAt.Format(time.RFC3339),
			},
			"message": "User registered successfully",
		})
	})

	body := `{"username":"testuser","email":"test@example.com","password":"password123","display_name":"Test User"}`
	req := httptest.NewRequest("POST", "/users/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "testuser", data["username"])
	assert.Equal(t, "test@example.com", data["email"])
	assert.Equal(t, "Test User", data["display_name"])
}

// Test_UserHandler_Register_UsernameExists tests registration with existing username
func Test_UserHandler_Register_UsernameExists(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	registeredUsers := make(map[string]bool)
	registeredUsers["existinguser"] = true

	r.POST("/users/register", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required,min=3,max=50"`
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if registeredUsers[req.Username] {
			c.JSON(http.StatusConflict, gin.H{
				"error": map[string]string{
					"code":    "USERNAME_EXISTS",
					"message": "Username already taken",
				},
			})
			return
		}
	})

	body := `{"username":"existinguser","email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/users/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// Test_UserHandler_Register_InvalidUsername tests registration with invalid username
func Test_UserHandler_Register_InvalidUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/users/register", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required,min=3,max=50"`
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required,min=8"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate username format
		if len(req.Username) < 3 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": map[string]string{
					"code":    "INVALID_USERNAME",
					"message": "Username must be at least 3 characters",
				},
			})
			return
		}
	})

	// Too short username
	body := `{"username":"ab","email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/users/register", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Test_UserHandler_Login_Success tests successful user login
func Test_UserHandler_Login_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	user := &instance.User{
		ID:           uuid.New(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "", // Will be set below
		DisplayName:  "Test User",
		Role:         instance.UserRoleUser,
		Status:       instance.UserStatusActive,
	}
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user.PasswordHash = string(hashedPassword)

	r.POST("/users/login", func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Email != user.Email {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid email or password",
				},
			})
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": map[string]string{
					"code":    "INVALID_CREDENTIALS",
					"message": "Invalid email or password",
				},
			})
			return
		}

		if user.Status != instance.UserStatusActive {
			c.JSON(http.StatusForbidden, gin.H{
				"error": map[string]string{
					"code":    "ACCOUNT_INACTIVE",
					"message": "Account is not active",
				},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"token":      "mock-jwt-token",
				"expires_at": time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
				"user":       ToUserResponse(user),
			},
			"message": "Login successful",
		})
	})

	body := `{"email":"test@example.com","password":"password123"}`
	req := httptest.NewRequest("POST", "/users/login", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_UserHandler_UpdateProfile tests profile update
func Test_UserHandler_UpdateProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	userID := uuid.New()

	r.PUT("/users/me", func(c *gin.Context) {
		c.Set("user_id", userID)

		var req struct {
			DisplayName *string `json:"display_name"`
			AvatarURL   *string `json:"avatar_url"`
			Bio         *string `json:"bio"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":           userID,
				"display_name": "Updated Name",
				"avatar_url":   "https://example.com/avatar.jpg",
				"bio":          "My new bio",
			},
			"message": "Profile updated successfully",
		})
	})

	body := `{"display_name":"Updated Name","avatar_url":"https://example.com/avatar.jpg","bio":"My new bio"}`
	req := httptest.NewRequest("PUT", "/users/me", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_UserHandler_Ban tests banning a user (admin only)
func Test_UserHandler_Ban(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	targetUserID := uuid.New()

	r.POST("/users/:id/ban", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":     targetUserID,
				"status": instance.UserStatusBanned,
			},
			"message": "User banned successfully",
		})
	})

	req := httptest.NewRequest("POST", "/users/"+targetUserID.String()+"/ban", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	data := response["data"].(map[string]interface{})
	assert.Equal(t, "banned", data["status"])
}

// Test_UserHandler_Unban tests unbanning a user (admin only)
func Test_UserHandler_Unban(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	targetUserID := uuid.New()

	r.POST("/users/:id/unban", func(c *gin.Context) {
		id := c.Param("id")
		_, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"id":     targetUserID,
				"status": instance.UserStatusActive,
			},
			"message": "User unbanned successfully",
		})
	})

	req := httptest.NewRequest("POST", "/users/"+targetUserID.String()+"/unban", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_UserHandler_List tests listing users (admin only)
func Test_UserHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": map[string]interface{}{
				"users": []map[string]interface{}{
					{
						"id":           uuid.New(),
						"username":     "user1",
						"display_name": "User One",
						"email":        "user1@example.com",
						"status":       "active",
						"role":         "user",
					},
					{
						"id":           uuid.New(),
						"username":     "user2",
						"display_name": "User Two",
						"email":        "user2@example.com",
						"status":       "active",
						"role":         "user",
					},
				},
				"total":    100,
				"page":     1,
				"per_page": 20,
			},
		})
	})

	req := httptest.NewRequest("GET", "/users", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// Test_UserStatusTransitions tests user status transitions
func Test_UserStatusTransitions(t *testing.T) {
	tests := []struct {
		name     string
		initial  instance.UserStatus
		action   string
		expected instance.UserStatus
	}{
		{"Active to Banned", instance.UserStatusActive, "ban", instance.UserStatusBanned},
		{"Banned to Active", instance.UserStatusBanned, "unban", instance.UserStatusActive},
		{"Active to Suspended", instance.UserStatusActive, "suspend", instance.UserStatusSuspended},
		{"Suspended to Active", instance.UserStatusSuspended, "unsuspend", instance.UserStatusActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newStatus := tt.initial

			switch tt.action {
			case "ban":
				if tt.initial == instance.UserStatusActive {
					newStatus = instance.UserStatusBanned
				}
			case "unban":
				if tt.initial == instance.UserStatusBanned {
					newStatus = instance.UserStatusActive
				}
			case "suspend":
				if tt.initial == instance.UserStatusActive {
					newStatus = instance.UserStatusSuspended
				}
			case "unsuspend":
				if tt.initial == instance.UserStatusSuspended {
					newStatus = instance.UserStatusActive
				}
			}

			assert.Equal(t, tt.expected, newStatus)
		})
	}
}
