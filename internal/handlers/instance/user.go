package instance

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"videostreamgo/internal/config"
	"videostreamgo/internal/middleware"
	instancemodels "videostreamgo/internal/models/instance"
	repo "videostreamgo/internal/repository/instance"
	"videostreamgo/internal/types"
)

// UserHandler handles user endpoints for instance API
type UserHandler struct {
	userRepo *repo.UserRepository
	cfg      *config.Config
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userRepo *repo.UserRepository, cfg *config.Config) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

// Register handles user registration
func (h *UserHandler) Register(c *gin.Context) {
	var req struct {
		Username    string `json:"username" binding:"required,min=3,max=50"`
		Email       string `json:"email" binding:"required,email"`
		Password    string `json:"password" binding:"required,min=8"`
		DisplayName string `json:"display_name" binding:"required,min=3,max=100"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	// Validate username format
	if !isValidUsername(req.Username) {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_USERNAME", "Username must contain only letters, numbers, and underscores", ""))
		return
	}

	// Check if username exists
	_, err := h.userRepo.GetByUsername(c.Request.Context(), req.Username)
	if err == nil {
		c.JSON(http.StatusConflict, types.ErrorResponse("USERNAME_EXISTS", "Username already taken", ""))
		return
	}

	// Check if email exists
	_, err = h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err == nil {
		c.JSON(http.StatusConflict, types.ErrorResponse("EMAIL_EXISTS", "Email already registered", ""))
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("HASH_ERROR", "Failed to process password", err.Error()))
		return
	}

	// Get instance ID from context
	instanceID := middleware.GetTenantID(c)

	user := &instancemodels.User{
		InstanceID:   instanceID,
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		DisplayName:  req.DisplayName,
		Role:         instancemodels.UserRoleUser,
		Status:       instancemodels.UserStatusActive,
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to create user", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.SuccessResponse(map[string]interface{}{
		"id":           user.ID,
		"username":     user.Username,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"created_at":   user.CreatedAt.Format(time.RFC3339),
	}, "User registered successfully"))
}

// Login handles user login
func (h *UserHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	user, err := h.userRepo.GetByEmail(c.Request.Context(), req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("INVALID_CREDENTIALS", "Invalid email or password", ""))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("INVALID_CREDENTIALS", "Invalid email or password", ""))
		return
	}

	if user.Status != instancemodels.UserStatusActive {
		c.JSON(http.StatusForbidden, types.ErrorResponse("ACCOUNT_INACTIVE", "Account is not active", ""))
		return
	}

	// Update last login
	h.userRepo.UpdateLastLogin(c.Request.Context(), user.ID)

	// Generate token (simplified - would use JWT in production)
	token := generateUserToken(user, h.cfg)

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"token":      token,
		"expires_at": time.Now().Add(7 * 24 * time.Hour).Format(time.RFC3339),
		"user":       ToUserResponse(user),
	}, "Login successful"))
}

// GetCurrentUser returns the current authenticated user
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get(string(types.ContextKeyUserID))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("NOT_AUTHENTICATED", "User not authenticated", ""))
		return
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TYPE_ERROR", "Invalid user ID type", ""))
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "User not found", ""))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(ToUserResponse(user), ""))
}

// UpdateCurrentUser updates the current user's profile
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	userID, exists := c.Get(string(types.ContextKeyUserID))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("NOT_AUTHENTICATED", "User not authenticated", ""))
		return
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TYPE_ERROR", "Invalid user ID type", ""))
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "User not found", ""))
		return
	}

	var req struct {
		DisplayName *string `json:"display_name"`
		AvatarURL   *string `json:"avatar_url"`
		Bio         *string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
	}
	if req.Bio != nil {
		user.Bio = *req.Bio
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to update user", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(ToUserResponse(user), "Profile updated successfully"))
}

// GetUser returns a user by ID
func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid user ID", ""))
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "User not found", ""))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(ToPublicUserResponse(user), ""))
}

// GetUserByUsername returns a user by username
func (h *UserHandler) GetUserByUsername(c *gin.Context) {
	username := c.Param("username")

	user, err := h.userRepo.GetByUsername(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "User not found", ""))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(ToPublicUserResponse(user), ""))
}

// ListUsers lists all users (admin only)
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Check if user is admin
	if !isUserAdmin(c) {
		c.JSON(http.StatusForbidden, types.ErrorResponse("FORBIDDEN", "Admin access required", ""))
		return
	}

	page := getIntParam(c, "page", 1)
	perPage := getIntParam(c, "per_page", 20)
	status := c.Query("status")

	users, total, err := h.userRepo.List(c.Request.Context(), (page-1)*perPage, perPage, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to list users", err.Error()))
		return
	}

	result := make([]map[string]interface{}, len(users))
	for i, user := range users {
		result[i] = ToUserResponse(&user)
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"users":    result,
		"total":    total,
		"page":     page,
		"per_page": perPage,
	}, ""))
}

// UpdateUser updates a user (admin only)
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// Check if user is admin
	if !isUserAdmin(c) {
		c.JSON(http.StatusForbidden, types.ErrorResponse("FORBIDDEN", "Admin access required", ""))
		return
	}

	id := c.Param("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid user ID", ""))
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "User not found", ""))
		return
	}

	var req struct {
		DisplayName *string                    `json:"display_name"`
		Role        *instancemodels.UserRole   `json:"role"`
		Status      *instancemodels.UserStatus `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.Role != nil {
		user.Role = *req.Role
	}
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to update user", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(ToUserResponse(user), "User updated successfully"))
}

// BanUser bans a user (admin only)
func (h *UserHandler) BanUser(c *gin.Context) {
	// Check if user is admin
	if !isUserAdmin(c) {
		c.JSON(http.StatusForbidden, types.ErrorResponse("FORBIDDEN", "Admin access required", ""))
		return
	}

	id := c.Param("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid user ID", ""))
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "User not found", ""))
		return
	}

	user.Status = instancemodels.UserStatusBanned

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to ban user", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":     user.ID,
		"status": user.Status,
	}, "User banned successfully"))
}

// UnbanUser unbans a user (admin only)
func (h *UserHandler) UnbanUser(c *gin.Context) {
	// Check if user is admin
	if !isUserAdmin(c) {
		c.JSON(http.StatusForbidden, types.ErrorResponse("FORBIDDEN", "Admin access required", ""))
		return
	}

	id := c.Param("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid user ID", ""))
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "User not found", ""))
		return
	}

	user.Status = instancemodels.UserStatusActive

	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to unban user", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":     user.ID,
		"status": user.Status,
	}, "User unbanned successfully"))
}

// DeleteUser soft deletes a user (admin only)
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// Check if user is admin
	if !isUserAdmin(c) {
		c.JSON(http.StatusForbidden, types.ErrorResponse("FORBIDDEN", "Admin access required", ""))
		return
	}

	id := c.Param("id")
	userID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid user ID", ""))
		return
	}

	if err := h.userRepo.Delete(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("DELETE_ERROR", "Failed to delete user", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(nil, "User deleted successfully"))
}

// Helper functions

func isValidUsername(username string) bool {
	if len(username) < 3 || len(username) > 50 {
		return false
	}
	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func generateUserToken(user *instancemodels.User, cfg *config.Config) string {
	// In production, this would generate a proper JWT token
	return "token-" + user.ID.String() + "-" + time.Now().Format(time.RFC3339)
}

// ToUserResponse converts a User to response format (full info for authenticated requests)
func ToUserResponse(user *instancemodels.User) map[string]interface{} {
	return map[string]interface{}{
		"id":             user.ID,
		"username":       user.Username,
		"email":          user.Email,
		"display_name":   user.DisplayName,
		"avatar_url":     user.AvatarURL,
		"bio":            user.Bio,
		"role":           user.Role,
		"status":         user.Status,
		"email_verified": user.EmailVerified,
		"created_at":     user.CreatedAt.Format(time.RFC3339),
	}
}

// ToPublicUserResponse converts a User to public response format (limited info for public requests)
func ToPublicUserResponse(user *instancemodels.User) map[string]interface{} {
	return map[string]interface{}{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"avatar_url":   user.AvatarURL,
		"bio":          user.Bio,
		"created_at":   user.CreatedAt.Format(time.RFC3339),
	}
}

func getIntParam(c *gin.Context, key string, defaultValue int) int {
	value := c.GetInt(key)
	if value == 0 {
		return defaultValue
	}
	return value
}

// isUserAdmin checks if the authenticated user has admin role
func isUserAdmin(c *gin.Context) bool {
	user, exists := c.Get(string(types.ContextKeyUser))
	if !exists {
		return false
	}

	userObj, ok := user.(*instancemodels.User)
	if !ok {
		return false
	}

	return userObj.Role == instancemodels.UserRoleAdmin
}

// ChangePassword handles password change for authenticated user
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get(string(types.ContextKeyUserID))
	if !exists {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("NOT_AUTHENTICATED", "User not authenticated", ""))
		return
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("TYPE_ERROR", "Invalid user ID type", ""))
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "User not found", ""))
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse("INVALID_PASSWORD", "Current password is incorrect", ""))
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("HASH_ERROR", "Failed to process password", err.Error()))
		return
	}

	user.PasswordHash = string(hashedPassword)
	if err := h.userRepo.Update(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to update password", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(nil, "Password updated successfully"))
}
