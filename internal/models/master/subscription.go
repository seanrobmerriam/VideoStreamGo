package master

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled"
	SubscriptionStatusPastDue   SubscriptionStatus = "past_due"
	SubscriptionStatusPaused    SubscriptionStatus = "paused"
	SubscriptionStatusTrialing  SubscriptionStatus = "trialing"
)

// BillingCycle represents the billing frequency
type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

// SubscriptionPlan represents a subscription plan for customers
type SubscriptionPlan struct {
	ID                   uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name                 string    `gorm:"type:varchar(100);not null" json:"name"`
	Description          string    `gorm:"type:text" json:"description"`
	MonthlyPrice         float64   `gorm:"type:decimal(10,2);not null;default:0" json:"monthly_price"`
	YearlyPrice          float64   `gorm:"type:decimal(10,2);not null;default:0" json:"yearly_price"`
	StripeMonthlyPriceID string    `gorm:"type:varchar(255)" json:"stripe_monthly_price_id"`
	StripeYearlyPriceID  string    `gorm:"type:varchar(255)" json:"stripe_yearly_price_id"`
	MaxStorageGB         int       `gorm:"not null;default:100" json:"max_storage_gb"`
	MaxBandwidthGB       int       `gorm:"not null;default:1000" json:"max_bandwidth_gb"`
	MaxVideos            int       `gorm:"not null;default:10000" json:"max_videos"`
	MaxUsers             int       `gorm:"not null;default:10000" json:"max_users"`
	Features             JSONMap   `gorm:"type:jsonb;default:'[]'" json:"features"`
	IsActive             bool      `gorm:"default:true" json:"is_active"`
	SortOrder            int       `gorm:"default:0" json:"sort_order"`
	CreatedAt            time.Time `gorm:"autoCreateTime" json:"created_at"`
}

// TableName sets the table name for SubscriptionPlan
func (SubscriptionPlan) TableName() string {
	return "subscription_plans"
}

// BeforeCreate generates a UUID before creating a new SubscriptionPlan
func (sp *SubscriptionPlan) BeforeCreate(tx *gorm.DB) error {
	if sp.ID == uuid.Nil {
		sp.ID = uuid.New()
	}
	return nil
}

// Subscription represents a customer's subscription to the platform
type Subscription struct {
	ID                   uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CustomerID           uuid.UUID          `gorm:"type:uuid;not null;index" json:"customer_id"`
	PlanID               uuid.UUID          `gorm:"type:uuid;not null;index" json:"plan_id"`
	Status               SubscriptionStatus `gorm:"type:varchar(50);default:'active';index" json:"status"`
	BillingCycle         BillingCycle       `gorm:"type:varchar(20);default:'monthly'" json:"billing_cycle"`
	StripeSubscriptionID string             `gorm:"type:varchar(255)" json:"stripe_subscription_id"`
	StripeCustomerID     string             `gorm:"type:varchar(255)" json:"stripe_customer_id"`
	CurrentPeriodStart   *time.Time         `json:"current_period_start"`
	CurrentPeriodEnd     *time.Time         `json:"current_period_end"`
	CancelAtPeriodEnd    bool               `gorm:"default:false" json:"cancel_at_period_end"`
	CreatedAt            time.Time          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time          `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Customer Customer         `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Plan     SubscriptionPlan `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
}

// TableName sets the table name for Subscription
func (Subscription) TableName() string {
	return "subscriptions"
}

// BeforeCreate generates a UUID before creating a new Subscription
func (s *Subscription) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
