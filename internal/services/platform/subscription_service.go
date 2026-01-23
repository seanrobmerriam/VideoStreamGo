package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"videostreamgo/internal/dto/platform"
	"videostreamgo/internal/models/master"
	masterRepo "videostreamgo/internal/repository/master"
)

var (
	ErrPlanNotFound         = errors.New("plan not found")
	ErrPlanExists           = errors.New("plan already exists")
	ErrInvalidPlanLimits    = errors.New("invalid plan limits")
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

const unlimited = -1 // Constant for unlimited values

// SubscriptionService handles subscription plan management
type SubscriptionService struct {
	planRepo         *masterRepo.PlanRepository
	subscriptionRepo *masterRepo.SubscriptionRepository
	customerRepo     *masterRepo.CustomerRepository
	billingRepo      *masterRepo.BillingRecordRepository
}

// NewSubscriptionService creates a new SubscriptionService
func NewSubscriptionService(
	planRepo *masterRepo.PlanRepository,
	subscriptionRepo *masterRepo.SubscriptionRepository,
	customerRepo *masterRepo.CustomerRepository,
	billingRepo *masterRepo.BillingRecordRepository,
) *SubscriptionService {
	return &SubscriptionService{
		planRepo:         planRepo,
		subscriptionRepo: subscriptionRepo,
		customerRepo:     customerRepo,
		billingRepo:      billingRepo,
	}
}

// PlanTier constants for predefined plans
const (
	PlanTierStarter      = "starter"
	PlanTierProfessional = "professional"
	PlanTierEnterprise   = "enterprise"
)

// CreatePlan creates a new subscription plan
func (s *SubscriptionService) CreatePlan(ctx context.Context, req *platform.CreatePlanRequest) (*master.SubscriptionPlan, error) {
	// Validate plan limits
	if err := validatePlanLimits(req); err != nil {
		return nil, err
	}

	// Check if plan with same name exists
	plans, err := s.planRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing plans: %w", err)
	}

	for _, p := range plans {
		if p.Name == req.Name {
			return nil, ErrPlanExists
		}
	}

	// Parse features from JSON string
	var features master.JSONMap
	if req.Features != "" {
		if err := json.Unmarshal([]byte(req.Features), &features); err != nil {
			return nil, fmt.Errorf("invalid features JSON: %w", err)
		}
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
		Features:       features,
		IsActive:       true,
		SortOrder:      0,
	}

	if err := s.planRepo.Create(ctx, plan); err != nil {
		return nil, fmt.Errorf("failed to create plan: %w", err)
	}

	return plan, nil
}

// UpdatePlan updates an existing subscription plan
func (s *SubscriptionService) UpdatePlan(ctx context.Context, planID string, updates map[string]interface{}) (*master.SubscriptionPlan, error) {
	planUUID, err := uuid.Parse(planID)
	if err != nil {
		return nil, fmt.Errorf("invalid plan ID: %w", err)
	}

	plan, err := s.planRepo.GetByID(ctx, planUUID)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		plan.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		plan.Description = description
	}
	if monthlyPrice, ok := updates["monthly_price"].(float64); ok {
		plan.MonthlyPrice = monthlyPrice
	}
	if yearlyPrice, ok := updates["yearly_price"].(float64); ok {
		plan.YearlyPrice = yearlyPrice
	}
	if maxStorageGB, ok := updates["max_storage_gb"].(int); ok {
		plan.MaxStorageGB = maxStorageGB
	}
	if maxBandwidthGB, ok := updates["max_bandwidth_gb"].(int); ok {
		plan.MaxBandwidthGB = maxBandwidthGB
	}
	if maxVideos, ok := updates["max_videos"].(int); ok {
		plan.MaxVideos = maxVideos
	}
	if maxUsers, ok := updates["max_users"].(int); ok {
		plan.MaxUsers = maxUsers
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		plan.IsActive = isActive
	}
	if sortOrder, ok := updates["sort_order"].(int); ok {
		plan.SortOrder = sortOrder
	}

	if err := s.planRepo.Update(ctx, plan); err != nil {
		return nil, fmt.Errorf("failed to update plan: %w", err)
	}

	return plan, nil
}

// DeletePlan soft deletes a subscription plan
func (s *SubscriptionService) DeletePlan(ctx context.Context, planID string) error {
	planUUID, err := uuid.Parse(planID)
	if err != nil {
		return fmt.Errorf("invalid plan ID: %w", err)
	}

	// Check if plan is in use
	subscriptions, err := s.subscriptionRepo.GetByPlanID(ctx, planUUID)
	if err != nil {
		return fmt.Errorf("failed to check subscriptions: %w", err)
	}

	if len(subscriptions) > 0 {
		// Instead of deleting, deactivate the plan
		plan, err := s.planRepo.GetByID(ctx, planUUID)
		if err != nil {
			return ErrPlanNotFound
		}
		plan.IsActive = false
		return s.planRepo.Update(ctx, plan)
	}

	return s.planRepo.Delete(ctx, planUUID)
}

// GetPlans retrieves all active subscription plans
func (s *SubscriptionService) GetPlans(ctx context.Context) ([]master.SubscriptionPlan, error) {
	return s.planRepo.GetAll(ctx)
}

// GetPlan retrieves a specific subscription plan
func (s *SubscriptionService) GetPlan(ctx context.Context, planID string) (*master.SubscriptionPlan, error) {
	planUUID, err := uuid.Parse(planID)
	if err != nil {
		return nil, fmt.Errorf("invalid plan ID: %w", err)
	}

	plan, err := s.planRepo.GetByID(ctx, planUUID)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	return plan, nil
}

// AssignPlan assigns a plan to a customer
func (s *SubscriptionService) AssignPlan(ctx context.Context, customerID string, planID string, billingCycle master.BillingCycle) (*master.Subscription, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil {
		return nil, fmt.Errorf("invalid customer ID: %w", err)
	}

	planUUID, err := uuid.Parse(planID)
	if err != nil {
		return nil, fmt.Errorf("invalid plan ID: %w", err)
	}

	// Verify customer exists
	customer, err := s.customerRepo.GetByID(ctx, customerUUID)
	if err != nil {
		return nil, errors.New("customer not found")
	}

	// Verify plan exists
	plan, err := s.planRepo.GetByID(ctx, planUUID)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	// Check if customer already has an active subscription
	existingSub, err := s.subscriptionRepo.GetActiveByCustomerID(ctx, customerUUID)
	if err == nil && existingSub != nil {
		// Update existing subscription
		return s.UpdateSubscription(ctx, existingSub.ID, planUUID, billingCycle)
	}

	// Create new subscription
	subscription := &master.Subscription{
		CustomerID:   customerUUID,
		PlanID:       planUUID,
		Status:       master.SubscriptionStatusActive,
		BillingCycle: billingCycle,
	}

	if err := s.subscriptionRepo.Create(ctx, subscription); err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Create billing record for initial subscription
	billingRecord := &master.BillingRecord{
		CustomerID:  customerUUID,
		Amount:      plan.MonthlyPrice,
		Currency:    "USD",
		Status:      master.BillingRecordStatusPending,
		Type:        "subscription",
		Description: fmt.Sprintf("%s plan - %s billing", plan.Name, billingCycle),
	}

	if err := s.billingRepo.Create(ctx, billingRecord); err != nil {
		// Log error but don't fail the subscription creation
		fmt.Printf("failed to create billing record: %v\n", err)
	}

	_ = customer // Use customer to avoid unused variable error

	return subscription, nil
}

// UpdateSubscription updates a subscription to a new plan
func (s *SubscriptionService) UpdateSubscription(ctx context.Context, subscriptionID uuid.UUID, newPlanID uuid.UUID, billingCycle master.BillingCycle) (*master.Subscription, error) {
	subscription, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, ErrSubscriptionNotFound
	}

	// Verify the new plan exists
	_, err = s.planRepo.GetByID(ctx, newPlanID)
	if err != nil {
		return nil, ErrPlanNotFound
	}

	subscription.PlanID = newPlanID
	subscription.BillingCycle = billingCycle

	if err := s.subscriptionRepo.Update(ctx, subscription); err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	return subscription, nil
}

// InitializeDefaultPlans creates the default subscription plans
func (s *SubscriptionService) InitializeDefaultPlans(ctx context.Context) error {
	plans, err := s.planRepo.GetAll(ctx)
	if err != nil {
		return fmt.Errorf("failed to get plans: %w", err)
	}

	if len(plans) > 0 {
		// Plans already initialized
		return nil
	}

	defaultPlans := []master.SubscriptionPlan{
		{
			Name:           "Starter",
			Description:    "Perfect for small businesses and startups getting started with video streaming",
			MonthlyPrice:   99.00,
			YearlyPrice:    990.00,
			MaxStorageGB:   100,
			MaxBandwidthGB: 1000,
			MaxVideos:      1000,
			MaxUsers:       10,
			SortOrder:      1,
		},
		{
			Name:           "Professional",
			Description:    "For growing businesses that need more capacity and features",
			MonthlyPrice:   299.00,
			YearlyPrice:    2990.00,
			MaxStorageGB:   500,
			MaxBandwidthGB: 5000,
			MaxVideos:      10000,
			MaxUsers:       100,
			SortOrder:      2,
		},
		{
			Name:           "Enterprise",
			Description:    "For large organizations with high volume needs",
			MonthlyPrice:   999.00,
			YearlyPrice:    9990.00,
			MaxStorageGB:   2000,
			MaxBandwidthGB: 50000,
			MaxVideos:      unlimited,
			MaxUsers:       unlimited,
			SortOrder:      3,
		},
	}

	for _, plan := range defaultPlans {
		plan.IsActive = true
		plan.Features = master.JSONMap{
			"custom_branding":   plan.MaxUsers >= 10,
			"analytics":         true,
			"priority_support":  plan.MonthlyPrice >= 299,
			"api_access":        plan.MonthlyPrice >= 299,
			"white_label":       plan.MonthlyPrice >= 999,
			"dedicated_support": plan.MonthlyPrice >= 999,
		}
		if err := s.planRepo.Create(ctx, &plan); err != nil {
			return fmt.Errorf("failed to create plan %s: %w", plan.Name, err)
		}
	}

	return nil
}

// validatePlanLimits validates plan limits
func validatePlanLimits(req *platform.CreatePlanRequest) error {
	if req.MaxStorageGB < 1 {
		return ErrInvalidPlanLimits
	}
	if req.MaxBandwidthGB < 1 {
		return ErrInvalidPlanLimits
	}
	if req.MaxVideos < 1 {
		return ErrInvalidPlanLimits
	}
	if req.MaxUsers < 1 {
		return ErrInvalidPlanLimits
	}
	if req.MonthlyPrice < 0 {
		return ErrInvalidPlanLimits
	}
	if req.YearlyPrice < 0 {
		return ErrInvalidPlanLimits
	}
	return nil
}

// GetActiveSubscription retrieves the active subscription for a customer
func (s *SubscriptionService) GetActiveSubscription(ctx context.Context, customerID uuid.UUID) (*master.Subscription, error) {
	return s.subscriptionRepo.GetActiveByCustomerID(ctx, customerID)
}

// GetSubscriptionWithPlan retrieves a subscription with its associated plan
func (s *SubscriptionService) GetSubscriptionWithPlan(ctx context.Context, subscriptionID uuid.UUID) (*master.Subscription, error) {
	subscription, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}

	plan, err := s.planRepo.GetByID(ctx, subscription.PlanID)
	if err != nil {
		return nil, err
	}

	subscription.Plan = *plan
	return subscription, nil
}

// GetCustomerSubscriptions retrieves all subscriptions for a customer
func (s *SubscriptionService) GetCustomerSubscriptions(ctx context.Context, customerID uuid.UUID) ([]master.Subscription, error) {
	return s.subscriptionRepo.GetByCustomerID(ctx, customerID)
}

// CancelSubscription cancels a customer's subscription
func (s *SubscriptionService) CancelSubscription(ctx context.Context, subscriptionID uuid.UUID, cancelImmediately bool) error {
	subscription, err := s.subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return ErrSubscriptionNotFound
	}

	if cancelImmediately {
		subscription.Status = master.SubscriptionStatusCancelled
	} else {
		subscription.CancelAtPeriodEnd = true
	}

	return s.subscriptionRepo.Update(ctx, subscription)
}

// GetPlanByTier retrieves a plan by its tier name
func (s *SubscriptionService) GetPlanByTier(ctx context.Context, tier string) (*master.SubscriptionPlan, error) {
	plans, err := s.planRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	for _, plan := range plans {
		if plan.Name == tier {
			return &plan, nil
		}
	}

	return nil, ErrPlanNotFound
}
