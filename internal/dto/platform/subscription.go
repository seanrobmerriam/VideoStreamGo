package platform

import (
	"github.com/google/uuid"

	"videostreamgo/internal/models/master"
)

// CreateSubscriptionRequest represents a request to create a subscription
type CreateSubscriptionRequest struct {
	CustomerID   string `json:"customer_id" binding:"required,uuid"`
	PlanID       string `json:"plan_id" binding:"required,uuid"`
	BillingCycle string `json:"billing_cycle" binding:"required,oneof=monthly yearly"`
}

// UpdateSubscriptionRequest represents a request to update a subscription
type UpdateSubscriptionRequest struct {
	PlanID       *string `json:"plan_id" binding:"omitempty,uuid"`
	BillingCycle *string `json:"billing_cycle" binding:"omitempty,oneof=monthly yearly"`
	Status       *string `json:"status" binding:"omitempty,oneof=active cancelled past_due paused trialing"`
}

// SubscriptionResponse represents a subscription in API responses
type SubscriptionResponse struct {
	ID                 uuid.UUID                 `json:"id"`
	CustomerID         uuid.UUID                 `json:"customer_id"`
	CustomerName       string                    `json:"customer_name"`
	PlanID             uuid.UUID                 `json:"plan_id"`
	PlanName           string                    `json:"plan_name"`
	Status             master.SubscriptionStatus `json:"status"`
	BillingCycle       master.BillingCycle       `json:"billing_cycle"`
	MonthlyPrice       float64                   `json:"monthly_price"`
	YearlyPrice        float64                   `json:"yearly_price"`
	MaxStorageGB       int                       `json:"max_storage_gb"`
	MaxBandwidthGB     int                       `json:"max_bandwidth_gb"`
	MaxVideos          int                       `json:"max_videos"`
	MaxUsers           int                       `json:"max_users"`
	CurrentPeriodStart *string                   `json:"current_period_start,omitempty"`
	CurrentPeriodEnd   *string                   `json:"current_period_end,omitempty"`
	CreatedAt          string                    `json:"created_at"`
}

// ToSubscriptionResponse converts a Subscription to SubscriptionResponse
func ToSubscriptionResponse(subscription *master.Subscription, customerName, planName string, plan *master.SubscriptionPlan) SubscriptionResponse {
	resp := SubscriptionResponse{
		ID:           subscription.ID,
		CustomerID:   subscription.CustomerID,
		CustomerName: customerName,
		PlanID:       subscription.PlanID,
		PlanName:     planName,
		Status:       subscription.Status,
		BillingCycle: subscription.BillingCycle,
		CreatedAt:    subscription.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if plan != nil {
		resp.MonthlyPrice = plan.MonthlyPrice
		resp.YearlyPrice = plan.YearlyPrice
		resp.MaxStorageGB = plan.MaxStorageGB
		resp.MaxBandwidthGB = plan.MaxBandwidthGB
		resp.MaxVideos = plan.MaxVideos
		resp.MaxUsers = plan.MaxUsers
	}
	if subscription.CurrentPeriodStart != nil {
		start := subscription.CurrentPeriodStart.Format("2006-01-02T15:04:05Z07:00")
		resp.CurrentPeriodStart = &start
	}
	if subscription.CurrentPeriodEnd != nil {
		end := subscription.CurrentPeriodEnd.Format("2006-01-02T15:04:05Z07:00")
		resp.CurrentPeriodEnd = &end
	}
	return resp
}

// SubscriptionListResponse represents a list of subscriptions with pagination
type SubscriptionListResponse struct {
	Subscriptions []SubscriptionResponse `json:"subscriptions"`
	Total         int64                  `json:"total"`
	Page          int                    `json:"page"`
	PerPage       int                    `json:"per_page"`
}

// CreatePlanRequest represents a request to create a subscription plan
type CreatePlanRequest struct {
	Name           string  `json:"name" binding:"required,min=2,max=100"`
	Description    string  `json:"description"`
	MonthlyPrice   float64 `json:"monthly_price" binding:"required,gte=0"`
	YearlyPrice    float64 `json:"yearly_price" binding:"required,gte=0"`
	MaxStorageGB   int     `json:"max_storage_gb" binding:"required,gte=0"`
	MaxBandwidthGB int     `json:"max_bandwidth_gb" binding:"required,gte=0"`
	MaxVideos      int     `json:"max_videos" binding:"required,gte=0"`
	MaxUsers       int     `json:"max_users" binding:"required,gte=0"`
	Features       string  `json:"features"` // JSON string of features
}

// PlanResponse represents a subscription plan in API responses
type PlanResponse struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	MonthlyPrice   float64   `json:"monthly_price"`
	YearlyPrice    float64   `json:"yearly_price"`
	MaxStorageGB   int       `json:"max_storage_gb"`
	MaxBandwidthGB int       `json:"max_bandwidth_gb"`
	MaxVideos      int       `json:"max_videos"`
	MaxUsers       int       `json:"max_users"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      string    `json:"created_at"`
}

// ToPlanResponse converts a SubscriptionPlan to PlanResponse
func ToPlanResponse(plan *master.SubscriptionPlan) PlanResponse {
	return PlanResponse{
		ID:             plan.ID,
		Name:           plan.Name,
		Description:    plan.Description,
		MonthlyPrice:   plan.MonthlyPrice,
		YearlyPrice:    plan.YearlyPrice,
		MaxStorageGB:   plan.MaxStorageGB,
		MaxBandwidthGB: plan.MaxBandwidthGB,
		MaxVideos:      plan.MaxVideos,
		MaxUsers:       plan.MaxUsers,
		IsActive:       plan.IsActive,
		CreatedAt:      plan.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
