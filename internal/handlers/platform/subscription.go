package platform

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"videostreamgo/internal/config"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
	"videostreamgo/internal/types"
)

// SubscriptionHandler handles subscription endpoints
type SubscriptionHandler struct {
	subscriptionRepo *masterRepo.SubscriptionRepository
	planRepo         *masterRepo.PlanRepository
	customerRepo     *masterRepo.CustomerRepository
	cfg              *config.Config
}

// NewSubscriptionHandler creates a new SubscriptionHandler
func NewSubscriptionHandler(subscriptionRepo *masterRepo.SubscriptionRepository, planRepo *masterRepo.PlanRepository, customerRepo *masterRepo.CustomerRepository, cfg *config.Config) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		customerRepo:     customerRepo,
		cfg:              cfg,
	}
}

// List returns all subscriptions
func (h *SubscriptionHandler) List(c *gin.Context) {
	page := getIntParam(c, "page", 1)
	perPage := getIntParam(c, "per_page", 20)
	status := c.Query("status")

	subscriptions, total, err := h.subscriptionRepo.List(c.Request.Context(), (page-1)*perPage, perPage, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to list subscriptions", err.Error()))
		return
	}

	result := make([]map[string]interface{}, len(subscriptions))
	for i, sub := range subscriptions {
		customer, _ := h.customerRepo.GetByID(c.Request.Context(), sub.CustomerID)
		plan, _ := h.planRepo.GetByID(c.Request.Context(), sub.PlanID)

		customerName := ""
		if customer != nil {
			customerName = customer.CompanyName
		}
		planName := ""
		monthlyPrice := 0.0
		if plan != nil {
			planName = plan.Name
			monthlyPrice = plan.MonthlyPrice
		}

		result[i] = map[string]interface{}{
			"id":            sub.ID,
			"customer_id":   sub.CustomerID,
			"customer_name": customerName,
			"plan_id":       sub.PlanID,
			"plan_name":     planName,
			"status":        sub.Status,
			"billing_cycle": sub.BillingCycle,
			"monthly_price": monthlyPrice,
			"created_at":    sub.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"subscriptions": result,
		"total":         total,
		"page":          page,
		"per_page":      perPage,
	}, ""))
}

// Create creates a new subscription
func (h *SubscriptionHandler) Create(c *gin.Context) {
	var req struct {
		CustomerID   string `json:"customer_id" binding:"required,uuid"`
		PlanID       string `json:"plan_id" binding:"required,uuid"`
		BillingCycle string `json:"billing_cycle" binding:"required,oneof=monthly yearly"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	customerID, _ := uuid.Parse(req.CustomerID)
	planID, _ := uuid.Parse(req.PlanID)

	now := time.Now()
	periodEnd := now.AddDate(0, 1, 0)
	if req.BillingCycle == "yearly" {
		periodEnd = now.AddDate(1, 0, 0)
	}

	subscription := &master.Subscription{
		CustomerID:         customerID,
		PlanID:             planID,
		Status:             master.SubscriptionStatusActive,
		BillingCycle:       master.BillingCycle(req.BillingCycle),
		CurrentPeriodStart: &now,
		CurrentPeriodEnd:   &periodEnd,
	}

	if err := h.subscriptionRepo.Create(c.Request.Context(), subscription); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to create subscription", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.SuccessResponse(map[string]interface{}{
		"id":                   subscription.ID,
		"customer_id":          subscription.CustomerID,
		"plan_id":              subscription.PlanID,
		"status":               subscription.Status,
		"billing_cycle":        subscription.BillingCycle,
		"current_period_start": subscription.CurrentPeriodStart.Format("2006-01-02"),
		"current_period_end":   subscription.CurrentPeriodEnd.Format("2006-01-02"),
		"created_at":           subscription.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, "Subscription created successfully"))
}

// Get returns a subscription by ID
func (h *SubscriptionHandler) Get(c *gin.Context) {
	id := c.Param("id")
	subID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid subscription ID", ""))
		return
	}

	subscription, err := h.subscriptionRepo.GetByID(c.Request.Context(), subID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Subscription not found", ""))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":                   subscription.ID,
		"customer_id":          subscription.CustomerID,
		"plan_id":              subscription.PlanID,
		"status":               subscription.Status,
		"billing_cycle":        subscription.BillingCycle,
		"current_period_start": formatTimePtr(subscription.CurrentPeriodStart),
		"current_period_end":   formatTimePtr(subscription.CurrentPeriodEnd),
		"created_at":           subscription.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, ""))
}

// Update updates a subscription
func (h *SubscriptionHandler) Update(c *gin.Context) {
	id := c.Param("id")
	subID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid subscription ID", ""))
		return
	}

	subscription, err := h.subscriptionRepo.GetByID(c.Request.Context(), subID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Subscription not found", ""))
		return
	}

	var req struct {
		PlanID       *string `json:"plan_id"`
		BillingCycle *string `json:"billing_cycle"`
		Status       *string `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	if req.PlanID != nil {
		planID, _ := uuid.Parse(*req.PlanID)
		subscription.PlanID = planID
	}
	if req.BillingCycle != nil {
		subscription.BillingCycle = master.BillingCycle(*req.BillingCycle)
	}
	if req.Status != nil {
		subscription.Status = master.SubscriptionStatus(*req.Status)
	}

	if err := h.subscriptionRepo.Update(c.Request.Context(), subscription); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to update subscription", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":     subscription.ID,
		"status": subscription.Status,
	}, "Subscription updated successfully"))
}

// Cancel cancels a subscription
func (h *SubscriptionHandler) Cancel(c *gin.Context) {
	id := c.Param("id")
	subID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid subscription ID", ""))
		return
	}

	subscription, err := h.subscriptionRepo.GetByID(c.Request.Context(), subID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Subscription not found", ""))
		return
	}

	subscription.Status = master.SubscriptionStatusCancelled
	if err := h.subscriptionRepo.Update(c.Request.Context(), subscription); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to cancel subscription", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(nil, "Subscription cancelled successfully"))
}

// ListPlans returns all subscription plans
func (h *SubscriptionHandler) ListPlans(c *gin.Context) {
	plans, err := h.planRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("LIST_ERROR", "Failed to list plans", err.Error()))
		return
	}

	result := make([]map[string]interface{}, len(plans))
	for i, plan := range plans {
		result[i] = map[string]interface{}{
			"id":               plan.ID,
			"name":             plan.Name,
			"description":      plan.Description,
			"monthly_price":    plan.MonthlyPrice,
			"yearly_price":     plan.YearlyPrice,
			"max_storage_gb":   plan.MaxStorageGB,
			"max_bandwidth_gb": plan.MaxBandwidthGB,
			"max_videos":       plan.MaxVideos,
			"max_users":        plan.MaxUsers,
			"is_active":        plan.IsActive,
			"created_at":       plan.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"plans": result,
	}, ""))
}

// CreatePlan creates a new subscription plan
func (h *SubscriptionHandler) CreatePlan(c *gin.Context) {
	var req struct {
		Name           string  `json:"name" binding:"required"`
		Description    string  `json:"description"`
		MonthlyPrice   float64 `json:"monthly_price" binding:"required,gte=0"`
		YearlyPrice    float64 `json:"yearly_price" binding:"required,gte=0"`
		MaxStorageGB   int     `json:"max_storage_gb" binding:"required,gte=0"`
		MaxBandwidthGB int     `json:"max_bandwidth_gb" binding:"required,gte=0"`
		MaxVideos      int     `json:"max_videos" binding:"required,gte=0"`
		MaxUsers       int     `json:"max_users" binding:"required,gte=0"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	plan := &master.SubscriptionPlan{
		Name:           req.Name,
		Description:    req.Description,
		MonthlyPrice:   req.MonthlyPrice,
		YearlyPrice:    req.YearlyPrice,
		MaxStorageGB:   req.MaxStorageGB,
		MaxBandwidthGB: req.MaxBandwidthGB,
		MaxVideos:      req.MaxVideos,
		MaxUsers:       req.MaxUsers,
		IsActive:       true,
	}

	if err := h.planRepo.Create(c.Request.Context(), plan); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("CREATE_ERROR", "Failed to create plan", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, types.SuccessResponse(map[string]interface{}{
		"id":             plan.ID,
		"name":           plan.Name,
		"monthly_price":  plan.MonthlyPrice,
		"max_storage_gb": plan.MaxStorageGB,
		"created_at":     plan.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, "Plan created successfully"))
}

// GetPlan returns a plan by ID
func (h *SubscriptionHandler) GetPlan(c *gin.Context) {
	id := c.Param("id")
	planID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid plan ID", ""))
		return
	}

	plan, err := h.planRepo.GetByID(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Plan not found", ""))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":               plan.ID,
		"name":             plan.Name,
		"description":      plan.Description,
		"monthly_price":    plan.MonthlyPrice,
		"yearly_price":     plan.YearlyPrice,
		"max_storage_gb":   plan.MaxStorageGB,
		"max_bandwidth_gb": plan.MaxBandwidthGB,
		"max_videos":       plan.MaxVideos,
		"max_users":        plan.MaxUsers,
		"is_active":        plan.IsActive,
		"created_at":       plan.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, ""))
}

// UpdatePlan updates a plan
func (h *SubscriptionHandler) UpdatePlan(c *gin.Context) {
	id := c.Param("id")
	planID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid plan ID", ""))
		return
	}

	plan, err := h.planRepo.GetByID(c.Request.Context(), planID)
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse("NOT_FOUND", "Plan not found", ""))
		return
	}

	var req struct {
		Name         *string  `json:"name"`
		Description  *string  `json:"description"`
		MonthlyPrice *float64 `json:"monthly_price"`
		YearlyPrice  *float64 `json:"yearly_price"`
		MaxStorageGB *int     `json:"max_storage_gb"`
		IsActive     *bool    `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("VALIDATION_ERROR", "Invalid request", err.Error()))
		return
	}

	if req.Name != nil {
		plan.Name = *req.Name
	}
	if req.Description != nil {
		plan.Description = *req.Description
	}
	if req.MonthlyPrice != nil {
		plan.MonthlyPrice = *req.MonthlyPrice
	}
	if req.YearlyPrice != nil {
		plan.YearlyPrice = *req.YearlyPrice
	}
	if req.MaxStorageGB != nil {
		plan.MaxStorageGB = *req.MaxStorageGB
	}
	if req.IsActive != nil {
		plan.IsActive = *req.IsActive
	}

	if err := h.planRepo.Update(c.Request.Context(), plan); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("UPDATE_ERROR", "Failed to update plan", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(map[string]interface{}{
		"id":   plan.ID,
		"name": plan.Name,
	}, "Plan updated successfully"))
}

// DeletePlan deletes a plan
func (h *SubscriptionHandler) DeletePlan(c *gin.Context) {
	id := c.Param("id")
	planID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse("INVALID_ID", "Invalid plan ID", ""))
		return
	}

	if err := h.planRepo.Delete(c.Request.Context(), planID); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse("DELETE_ERROR", "Failed to delete plan", err.Error()))
		return
	}

	c.JSON(http.StatusOK, types.SuccessResponse(nil, "Plan deleted successfully"))
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}
