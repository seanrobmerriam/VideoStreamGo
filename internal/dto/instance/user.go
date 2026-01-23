package instance

import (
	"github.com/google/uuid"

	"videostreamgo/internal/models/instance"
)

// RegisterUserRequest represents a user registration request
type RegisterUserRequest struct {
	Username    string `json:"username" binding:"required,alphanum,min=3,max=50"`
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name" binding:"required,min=2,max=100"`
}

// LoginUserRequest represents a user login request
type LoginUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginUserResponse represents a successful user login response
type LoginUserResponse struct {
	Token     string       `json:"token"`
	ExpiresAt string       `json:"expires_at"`
	User      UserResponse `json:"user"`
}

// UserResponse represents a user in API responses
type UserResponse struct {
	ID              uuid.UUID           `json:"id"`
	Username        string              `json:"username"`
	Email           string              `json:"email"`
	DisplayName     string              `json:"display_name"`
	AvatarURL       string              `json:"avatar_url,omitempty"`
	Bio             string              `json:"bio,omitempty"`
	Role            instance.UserRole   `json:"role"`
	Status          instance.UserStatus `json:"status"`
	VideoCount      int                 `json:"video_count"`
	SubscriberCount int                 `json:"subscriber_count"`
	CreatedAt       string              `json:"created_at"`
}

// ToUserResponse converts a User to UserResponse
func ToUserResponse(user *instance.User) UserResponse {
	return UserResponse{
		ID:          user.ID,
		Username:    user.Username,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Bio:         user.Bio,
		Role:        user.Role,
		Status:      user.Status,
		CreatedAt:   user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// UpdateUserRequest represents a request to update user profile
type UpdateUserRequest struct {
	DisplayName *string `json:"display_name" binding:"omitempty,min=2,max=100"`
	AvatarURL   *string `json:"avatar_url" binding:"omitempty,url"`
	Bio         *string `json:"bio" binding:"omitempty,max=500"`
}

// ChangePasswordRequest represents a request to change password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// UserListResponse represents a list of users with pagination
type UserListResponse struct {
	Users   []UserResponse `json:"users"`
	Total   int64          `json:"total"`
	Page    int            `json:"page"`
	PerPage int            `json:"per_page"`
}
