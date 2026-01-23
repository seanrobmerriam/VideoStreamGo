package platform

import (
	"github.com/google/uuid"

	"videostreamgo/internal/models/master"
)

// CreateCustomerRequest represents a request to create a new customer
type CreateCustomerRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=8"`
	CompanyName string `json:"company_name" binding:"required,min=2,max=255"`
	ContactName string `json:"contact_name" binding:"required,min=2,max=255"`
	Phone       string `json:"phone"`
	PlanID      string `json:"plan_id" binding:"required,uuid"`
}

// UpdateCustomerRequest represents a request to update a customer
type UpdateCustomerRequest struct {
	Email       *string `json:"email" binding:"omitempty,email"`
	CompanyName *string `json:"company_name" binding:"omitempty,min=2,max=255"`
	ContactName *string `json:"contact_name" binding:"omitempty,min=2,max=255"`
	Phone       *string `json:"phone"`
	Status      *string `json:"status" binding:"omitempty,oneof=active suspended cancelled pending"`
}

// CustomerResponse represents a customer in API responses
type CustomerResponse struct {
	ID            uuid.UUID             `json:"id"`
	Email         string                `json:"email"`
	CompanyName   string                `json:"company_name"`
	ContactName   string                `json:"contact_name"`
	Phone         string                `json:"phone"`
	Status        master.CustomerStatus `json:"status"`
	InstanceCount int                   `json:"instance_count"`
	CreatedAt     string                `json:"created_at"`
	UpdatedAt     string                `json:"updated_at"`
}

// ToResponse converts a Customer to CustomerResponse
func ToCustomerResponse(customer *master.Customer, instanceCount int) CustomerResponse {
	return CustomerResponse{
		ID:            customer.ID,
		Email:         customer.Email,
		CompanyName:   customer.CompanyName,
		ContactName:   customer.ContactName,
		Phone:         customer.Phone,
		Status:        customer.Status,
		InstanceCount: instanceCount,
		CreatedAt:     customer.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     customer.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// CustomerListResponse represents a list of customers with pagination
type CustomerListResponse struct {
	Customers []CustomerResponse `json:"customers"`
	Total     int64              `json:"total"`
	Page      int                `json:"page"`
	PerPage   int                `json:"per_page"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	Token     string           `json:"token"`
	ExpiresAt string           `json:"expires_at"`
	User      CustomerResponse `json:"user"`
}
