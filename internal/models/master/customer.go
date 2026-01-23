package master

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CustomerStatus represents the status of a customer account
type CustomerStatus string

const (
	CustomerStatusActive    CustomerStatus = "active"
	CustomerStatusSuspended CustomerStatus = "suspended"
	CustomerStatusCancelled CustomerStatus = "cancelled"
	CustomerStatusPending   CustomerStatus = "pending"
)

// Customer represents a paying customer in the master database
type Customer struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email            string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash     string         `gorm:"type:varchar(255);not null" json:"-"`
	CompanyName      string         `gorm:"type:varchar(255);not null" json:"company_name"`
	ContactName      string         `gorm:"type:varchar(255)" json:"contact_name"`
	Phone            string         `gorm:"type:varchar(50)" json:"phone"`
	Status           CustomerStatus `gorm:"type:varchar(50);default:'active';index" json:"status"`
	StripeCustomerID string         `gorm:"type:varchar(255)" json:"stripe_customer_id"`
	BillingEmail     string         `gorm:"type:varchar(255)" json:"billing_email"`
	TaxID            string         `gorm:"type:varchar(100)" json:"tax_id"`
	Metadata         JSONMap        `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt        time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Instances      []Instance      `gorm:"foreignKey:CustomerID" json:"instances,omitempty"`
	Subscriptions  []Subscription  `gorm:"foreignKey:CustomerID" json:"subscriptions,omitempty"`
	BillingRecords []BillingRecord `gorm:"foreignKey:CustomerID" json:"billing_records,omitempty"`
}

// TableName sets the table name for Customer
func (Customer) TableName() string {
	return "customers"
}

// BeforeCreate generates a UUID before creating a new Customer
func (c *Customer) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// JSONMap is a custom type for JSONB columns
type JSONMap map[string]interface{}
