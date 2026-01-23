package types

import (
	"time"

	"github.com/google/uuid"
)

// PaginationParams holds common pagination parameters
type PaginationParams struct {
	Page    int `form:"page" binding:"min=1"`
	PerPage int `form:"per_page" binding:"min=1,max=100"`
}

// GetOffset calculates the offset for database queries
func (p *PaginationParams) GetOffset() int {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PerPage < 1 {
		p.PerPage = 20
	}
	return (p.Page - 1) * p.PerPage
}

// GetLimit returns the limit for database queries
func (p *PaginationParams) GetLimit() int {
	if p.PerPage < 1 {
		p.PerPage = 20
	}
	if p.PerPage > 100 {
		p.PerPage = 100
	}
	return p.PerPage
}

// PaginatedResponse is a generic paginated response
type PaginatedResponse[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
}

// NewPaginatedResponse creates a new paginated response
func NewPaginatedResponse[T any](data []T, total int64, page, perPage int) PaginatedResponse[T] {
	totalPages := int(total) / perPage
	if int(total)%perPage > 0 {
		totalPages++
	}
	return PaginatedResponse[T]{
		Data:       data,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}
}

// APIResponse is a standard API response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents an error response
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// SuccessResponse creates a success response
func SuccessResponse(data interface{}, message string) APIResponse {
	return APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
}

// ErrorResponse creates an error response
func ErrorResponse(code, message, details string) APIResponse {
	return APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// Context keys for middleware
type ContextKey string

const (
	ContextKeyAdminUser  ContextKey = "admin_user"
	ContextKeyAdminID    ContextKey = "admin_id"
	ContextKeyCustomer   ContextKey = "customer"
	ContextKeyInstance   ContextKey = "instance"
	ContextKeyUser       ContextKey = "user"
	ContextKeyUserID     ContextKey = "user_id"
	ContextKeyTenantID   ContextKey = "tenant_id"
	ContextKeyRequestID  ContextKey = "request_id"
	ContextKeyInstanceID ContextKey = "instance_id"
)

// TenantContext holds tenant information
type TenantContext struct {
	InstanceID   uuid.UUID
	Subdomain    string
	DatabaseName string
}

// AdminClaims represents JWT claims for admin users
type AdminClaims struct {
	AdminID uuid.UUID `json:"admin_id"`
	Email   string    `json:"email"`
	Role    string    `json:"role"`
}

// UserClaims represents JWT claims for instance users
type UserClaims struct {
	UserID     uuid.UUID `json:"user_id"`
	InstanceID uuid.UUID `json:"instance_id"`
	Username   string    `json:"username"`
	Email      string    `json:"email"`
	Role       string    `json:"role"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         uuid.UUID `json:"id"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Action     string    `json:"action"`
	AdminID    uuid.UUID `json:"admin_id,omitempty"`
	UserID     uuid.UUID `json:"user_id,omitempty"`
	Changes    string    `json:"changes,omitempty"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}
