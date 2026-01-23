package master

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BillingRecordStatus represents the status of a billing record
type BillingRecordStatus string

const (
	BillingRecordStatusPending  BillingRecordStatus = "pending"
	BillingRecordStatusPaid     BillingRecordStatus = "paid"
	BillingRecordStatusFailed   BillingRecordStatus = "failed"
	BillingRecordStatusRefunded BillingRecordStatus = "refunded"
)

// BillingRecord represents a billing history record
type BillingRecord struct {
	ID                    uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CustomerID            uuid.UUID           `gorm:"type:uuid;not null;index" json:"customer_id"`
	SubscriptionID        *uuid.UUID          `gorm:"type:uuid;index" json:"subscription_id"`
	Amount                float64             `gorm:"type:decimal(10,2);not null" json:"amount"`
	Currency              string              `gorm:"type:varchar(3);default:'USD'" json:"currency"`
	Status                BillingRecordStatus `gorm:"type:varchar(50);default:'pending';index" json:"status"`
	Type                  string              `gorm:"type:varchar(50);default:'subscription'" json:"type"`
	InvoiceID             string              `gorm:"type:varchar(255)" json:"invoice_id"`
	StripePaymentIntentID string              `gorm:"type:varchar(255)" json:"stripe_payment_intent_id"`
	StripeChargeID        string              `gorm:"type:varchar(255)" json:"stripe_charge_id"`
	PeriodStart           *time.Time          `json:"period_start"`
	PeriodEnd             *time.Time          `json:"period_end"`
	Description           string              `gorm:"type:text" json:"description"`
	Metadata              JSONMap             `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt             time.Time           `gorm:"autoCreateTime;index" json:"created_at"`
	PaidAt                *time.Time          `json:"paid_at"`

	// Relationships
	Customer     Customer     `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Subscription Subscription `gorm:"foreignKey:SubscriptionID" json:"subscription,omitempty"`
}

// TableName sets the table name for BillingRecord
func (BillingRecord) TableName() string {
	return "billing_records"
}

// BeforeCreate generates a UUID before creating a new BillingRecord
func (br *BillingRecord) BeforeCreate(tx *gorm.DB) error {
	if br.ID == uuid.Nil {
		br.ID = uuid.New()
	}
	return nil
}

// Invoice represents a customer-facing invoice
type Invoice struct {
	ID            uuid.UUID  `json:"id"`
	InvoiceNumber string     `json:"invoice_number"`
	CustomerID    uuid.UUID  `json:"customer_id"`
	CustomerName  string     `json:"customer_name"`
	CustomerEmail string     `json:"customer_email"`
	Amount        float64    `json:"amount"`
	Currency      string     `json:"currency"`
	Status        string     `json:"status"`
	PeriodStart   time.Time  `json:"period_start"`
	PeriodEnd     time.Time  `json:"period_end"`
	CreatedAt     time.Time  `json:"created_at"`
	PaidAt        *time.Time `json:"paid_at,omitempty"`
}
