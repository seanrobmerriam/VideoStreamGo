package platform

import (
	"github.com/google/uuid"

	"videostreamgo/internal/models/master"
)

// AdminLoginRequest represents an admin login request
type AdminLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AdminLoginResponse represents a successful admin login response
type AdminLoginResponse struct {
	Token     string            `json:"token"`
	ExpiresAt string            `json:"expires_at"`
	User      AdminUserResponse `json:"user"`
}

// AdminUserResponse represents an admin user in API responses
type AdminUserResponse struct {
	ID          uuid.UUID          `json:"id"`
	Email       string             `json:"email"`
	DisplayName string             `json:"display_name"`
	Role        master.AdminRole   `json:"role"`
	Status      master.AdminStatus `json:"status"`
	LastLoginAt *string            `json:"last_login_at,omitempty"`
	CreatedAt   string             `json:"created_at"`
}

// ToAdminUserResponse converts an AdminUser to AdminUserResponse
func ToAdminUserResponse(admin *master.AdminUser) AdminUserResponse {
	resp := AdminUserResponse{
		ID:          admin.ID,
		Email:       admin.Email,
		DisplayName: admin.DisplayName,
		Role:        admin.Role,
		Status:      admin.Status,
		CreatedAt:   admin.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if admin.LastLoginAt != nil {
		lastLogin := admin.LastLoginAt.Format("2006-01-02T15:04:05Z07:00")
		resp.LastLoginAt = &lastLogin
	}
	return resp
}

// CreateAdminRequest represents a request to create an admin user
type CreateAdminRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name" binding:"required,min=2,max=100"`
	Role        string `json:"role" binding:"required,oneof=super_admin admin moderator support"`
}

// UpdateAdminRequest represents a request to update an admin user
type UpdateAdminRequest struct {
	Email       *string `json:"email" binding:"omitempty,email"`
	DisplayName *string `json:"display_name" binding:"omitempty,min=2,max=100"`
	Role        *string `json:"role" binding:"omitempty,oneof=super_admin admin moderator support"`
	Status      *string `json:"status" binding:"omitempty,oneof=active inactive suspended"`
}

// AdminListResponse represents a list of admin users with pagination
type AdminListResponse struct {
	Admins  []AdminUserResponse `json:"admins"`
	Total   int64               `json:"total"`
	Page    int                 `json:"page"`
	PerPage int                 `json:"per_page"`
}

// ChangePasswordRequest represents a request to change password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}
