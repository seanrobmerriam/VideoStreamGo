package platform

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"videostreamgo/internal/config"
	"videostreamgo/internal/middleware"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
	"videostreamgo/internal/types"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	adminRepo *masterRepo.AdminRepository
	cfg       *config.Config
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(adminRepo *masterRepo.AdminRepository, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		adminRepo: adminRepo,
		cfg:       cfg,
	}
}

// Login handles admin login
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	admin, err := h.adminRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("INVALID_CREDENTIALS", "Invalid email or password", ""))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("INVALID_CREDENTIALS", "Invalid email or password", ""))
		return
	}

	if admin.Status != master.AdminStatusActive {
		c.JSON(http.StatusForbidden, types.ErrorResponse("ACCOUNT_INACTIVE", "Account is not active", ""))
		return
	}

	token, err := middleware.GenerateAdminToken(admin, h.cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TOKEN_ERROR", "Failed to generate token", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(gin.H{
		"token":      token,
		"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"user":       ToAdminUserResponse(admin),
	}, "Login successful"))
}

// Register handles admin registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=8"`
		DisplayName string `json:"display_name" binding:"required,min=2,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	// Check if admin already exists
	_, err := h.adminRepo.GetByEmail(c.Request.Context(), req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, types.ErrorResponse("EMAIL_EXISTS", "Email already registered", ""))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("HASH_ERROR", "Failed to process password", err.Error()))
		return
	}

	admin := &master.AdminUser{
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		DisplayName:  req.DisplayName,
		Role:         master.AdminRoleAdmin,
		Status:       master.AdminStatusActive,
	}

	if err := h.adminRepo.Create(c.Request.Context(), admin); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to create admin", err.Error()))
		return
	}

	token, err := middleware.GenerateAdminToken(admin, h.cfg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TOKEN_ERROR", "Failed to generate token", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.SuccessResponse(gin.H{
		"token":      token,
		"expires_at": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"user":       ToAdminUserResponse(admin),
	}, "Registration successful"))
}

// GetCurrentAdmin returns the current authenticated admin
func (h *AuthHandler) GetCurrentAdmin(c *gin.Context) {
	admin, exists := c.Get(string(types.ContextKeyAdminUser))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("NOT_AUTHENTICATED", "Admin not found", ""))
		return
	}

	adminUser, ok := admin.(*master.AdminUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TYPE_ERROR", "Invalid admin type", ""))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(ToAdminUserResponse(adminUser), ""))
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	admin, exists := c.Get(string(types.ContextKeyAdminUser))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("NOT_AUTHENTICATED", "Admin not found", ""))
		return
	}

	adminUser, ok := admin.(*master.AdminUser)
	if !ok {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TYPE_ERROR", "Invalid admin type", ""))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(adminUser.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("INVALID_PASSWORD", "Current password is incorrect", ""))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("HASH_ERROR", "Failed to process password", err.Error()))
		return
	}

	adminUser.PasswordHash = string(hashedPassword)
	if err := h.adminRepo.Update(c.Request.Context(), adminUser); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to update password", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(nil, "Password updated successfully"))
}

// ToAdminUserResponse converts AdminUser to response format
func ToAdminUserResponse(admin *master.AdminUser) map[string]interface{} {
	resp := map[string]interface{}{
		"id":           admin.ID,
		"email":        admin.Email,
		"display_name": admin.DisplayName,
		"role":         admin.Role,
		"status":       admin.Status,
		"created_at":   admin.CreatedAt.Format(time.RFC3339),
	}
	if admin.LastLoginAt != nil {
		resp["last_login_at"] = admin.LastLoginAt.Format(time.RFC3339)
	}
	return resp
}
