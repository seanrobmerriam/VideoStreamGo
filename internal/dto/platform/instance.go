package platform

import (
	"github.com/google/uuid"

	"videostreamgo/internal/models/master"
)

// CreateInstanceRequest represents a request to create a new instance
type CreateInstanceRequest struct {
	CustomerID    string   `json:"customer_id" binding:"required,uuid"`
	Name          string   `json:"name" binding:"required,min=3,max=255"`
	Subdomain     string   `json:"subdomain" binding:"required,subdomain"`
	CustomDomains []string `json:"custom_domains"`
	PlanID        string   `json:"plan_id" binding:"required,uuid"`
}

// UpdateInstanceRequest represents a request to update an instance
type UpdateInstanceRequest struct {
	Name          *string  `json:"name" binding:"omitempty,min=3,max=255"`
	Subdomain     *string  `json:"subdomain" binding:"omitempty,subdomain"`
	CustomDomains []string `json:"custom_domains"`
	Status        *string  `json:"status" binding:"omitempty,oneof=pending provisioning active suspended terminated"`
}

// InstanceResponse represents an instance in API responses
type InstanceResponse struct {
	ID            uuid.UUID             `json:"id"`
	CustomerID    uuid.UUID             `json:"customer_id"`
	CustomerName  string                `json:"customer_name"`
	Name          string                `json:"name"`
	Subdomain     string                `json:"subdomain"`
	CustomDomains []string              `json:"custom_domains"`
	Status        master.InstanceStatus `json:"status"`
	PlanID        uuid.UUID             `json:"plan_id"`
	PlanName      string                `json:"plan_name"`
	DatabaseName  string                `json:"database_name"`
	StorageBucket string                `json:"storage_bucket"`
	CreatedAt     string                `json:"created_at"`
	UpdatedAt     string                `json:"updated_at"`
	ActivatedAt   *string               `json:"activated_at,omitempty"`
}

// ToInstanceResponse converts an Instance to InstanceResponse
func ToInstanceResponse(instance *master.Instance, customerName, planName string) InstanceResponse {
	resp := InstanceResponse{
		ID:            instance.ID,
		CustomerID:    instance.CustomerID,
		CustomerName:  customerName,
		Name:          instance.Name,
		Subdomain:     instance.Subdomain,
		CustomDomains: instance.CustomDomains,
		Status:        instance.Status,
		PlanID:        *instance.PlanID,
		PlanName:      planName,
		DatabaseName:  instance.DatabaseName,
		StorageBucket: instance.StorageBucket,
		CreatedAt:     instance.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:     instance.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if instance.ActivatedAt != nil {
		activated := instance.ActivatedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.ActivatedAt = &activated
	}
	return resp
}

// InstanceListResponse represents a list of instances with pagination
type InstanceListResponse struct {
	Instances []InstanceResponse `json:"instances"`
	Total     int64              `json:"total"`
	Page      int                `json:"page"`
	PerPage   int                `json:"per_page"`
}

// ProvisionInstanceResponse represents the response after provisioning an instance
type ProvisionInstanceResponse struct {
	Instance      InstanceResponse `json:"instance"`
	Message       string           `json:"message"`
	DatabaseName  string           `json:"database_name"`
	StorageBucket string           `json:"storage_bucket"`
}
