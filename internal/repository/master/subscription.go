package master

import (
	"context"

	"videostreamgo/internal/models/master"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionRepository handles database operations for subscriptions
type SubscriptionRepository struct {
	db *gorm.DB
}

// NewSubscriptionRepository creates a new SubscriptionRepository
func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// Create creates a new subscription
func (r *SubscriptionRepository) Create(ctx context.Context, subscription *master.Subscription) error {
	return r.db.WithContext(ctx).Create(subscription).Error
}

// GetByID retrieves a subscription by ID
func (r *SubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*master.Subscription, error) {
	var subscription master.Subscription
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetByCustomerID retrieves subscriptions by customer ID
func (r *SubscriptionRepository) GetByCustomerID(ctx context.Context, customerID uuid.UUID) ([]master.Subscription, error) {
	var subscriptions []master.Subscription
	err := r.db.WithContext(ctx).Where("customer_id = ?", customerID).Find(&subscriptions).Error
	return subscriptions, err
}

// GetActiveByCustomerID retrieves active subscription for a customer
func (r *SubscriptionRepository) GetActiveByCustomerID(ctx context.Context, customerID uuid.UUID) (*master.Subscription, error) {
	var subscription master.Subscription
	err := r.db.WithContext(ctx).Where("customer_id = ? AND status = ?", customerID, "active").First(&subscription).Error
	if err != nil {
		return nil, err
	}
	return &subscription, nil
}

// GetByPlanID retrieves subscriptions by plan ID
func (r *SubscriptionRepository) GetByPlanID(ctx context.Context, planID uuid.UUID) ([]master.Subscription, error) {
	var subscriptions []master.Subscription
	err := r.db.WithContext(ctx).Where("plan_id = ?", planID).Find(&subscriptions).Error
	return subscriptions, err
}

// GetActiveSubscriptions retrieves all active subscriptions
func (r *SubscriptionRepository) GetActiveSubscriptions(ctx context.Context) ([]master.Subscription, error) {
	var subscriptions []master.Subscription
	err := r.db.WithContext(ctx).Where("status = ?", master.SubscriptionStatusActive).Find(&subscriptions).Error
	return subscriptions, err
}

// Update updates a subscription
func (r *SubscriptionRepository) Update(ctx context.Context, subscription *master.Subscription) error {
	return r.db.WithContext(ctx).Save(subscription).Error
}

// Delete soft deletes a subscription
func (r *SubscriptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&master.Subscription{}, "id = ?", id).Error
}

// List retrieves subscriptions with pagination
func (r *SubscriptionRepository) List(ctx context.Context, offset, limit int, status string) ([]master.Subscription, int64, error) {
	var subscriptions []master.Subscription
	var total int64

	query := r.db.WithContext(ctx).Model(&master.Subscription{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&subscriptions).Error
	if err != nil {
		return nil, 0, err
	}

	return subscriptions, total, nil
}

// PlanRepository handles database operations for subscription plans
type PlanRepository struct {
	db *gorm.DB
}

// NewPlanRepository creates a new PlanRepository
func NewPlanRepository(db *gorm.DB) *PlanRepository {
	return &PlanRepository{db: db}
}

// Create creates a new plan
func (r *PlanRepository) Create(ctx context.Context, plan *master.SubscriptionPlan) error {
	return r.db.WithContext(ctx).Create(plan).Error
}

// GetByID retrieves a plan by ID
func (r *PlanRepository) GetByID(ctx context.Context, id uuid.UUID) (*master.SubscriptionPlan, error) {
	var plan master.SubscriptionPlan
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&plan).Error
	if err != nil {
		return nil, err
	}
	return &plan, nil
}

// GetAll retrieves all active plans
func (r *PlanRepository) GetAll(ctx context.Context) ([]master.SubscriptionPlan, error) {
	var plans []master.SubscriptionPlan
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Order("monthly_price ASC").Find(&plans).Error
	return plans, err
}

// Update updates a plan
func (r *PlanRepository) Update(ctx context.Context, plan *master.SubscriptionPlan) error {
	return r.db.WithContext(ctx).Save(plan).Error
}

// Delete soft deletes a plan
func (r *PlanRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&master.SubscriptionPlan{}, "id = ?", id).Error
}
