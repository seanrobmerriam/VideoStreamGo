package instance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserSubscriptionStatus represents the status of a user's premium subscription
type UserSubscriptionStatus string

const (
	UserSubscriptionStatusActive    UserSubscriptionStatus = "active"
	UserSubscriptionStatusCancelled UserSubscriptionStatus = "cancelled"
	UserSubscriptionStatusExpired   UserSubscriptionStatus = "expired"
	UserSubscriptionStatusPastDue   UserSubscriptionStatus = "past_due"
)

// UserSubscription represents an end user's premium subscription within an instance
type UserSubscription struct {
	ID                   uuid.UUID              `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID           uuid.UUID              `gorm:"type:uuid;not null;index" json:"instance_id"`
	UserID               uuid.UUID              `gorm:"type:uuid;not null;index" json:"user_id"`
	Tier                 string                 `gorm:"type:varchar(50);default:'free'" json:"tier"` // free, premium, vip
	Status               UserSubscriptionStatus `gorm:"type:varchar(50);default:'active';index" json:"status"`
	StripeSubscriptionID string                 `gorm:"type:varchar(255)" json:"stripe_subscription_id"`
	StripeCustomerID     string                 `gorm:"type:varchar(255)" json:"stripe_customer_id"`
	CurrentPeriodStart   *time.Time             `json:"current_period_start"`
	CurrentPeriodEnd     *time.Time             `json:"current_period_end"`
	Features             JSONMap                `gorm:"type:jsonb;default:'{}'" json:"features"`
	CreatedAt            time.Time              `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time              `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName sets the table name for UserSubscription
func (UserSubscription) TableName() string {
	return "user_subscriptions"
}

// BeforeCreate generates a UUID before creating a new UserSubscription
func (us *UserSubscription) BeforeCreate(tx *gorm.DB) error {
	if us.ID == uuid.Nil {
		us.ID = uuid.New()
	}
	return nil
}

// IsActive checks if the subscription is currently active
func (us *UserSubscription) IsActive() bool {
	if us.Status != UserSubscriptionStatusActive {
		return false
	}
	if us.CurrentPeriodEnd == nil {
		return true
	}
	return time.Now().Before(*us.CurrentPeriodEnd)
}
