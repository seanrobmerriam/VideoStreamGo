package master

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LicenseType represents the type of license
type LicenseType string

const (
	LicenseTypeTrial      LicenseType = "trial"
	LicenseTypeStandard   LicenseType = "standard"
	LicenseTypeEnterprise LicenseType = "enterprise"
)

// LicenseStatus represents the status of a license
type LicenseStatus string

const (
	LicenseStatusActive    LicenseStatus = "active"
	LicenseStatusExpired   LicenseStatus = "expired"
	LicenseStatusRevoked   LicenseStatus = "revoked"
	LicenseStatusSuspended LicenseStatus = "suspended"
)

// License manages license keys for customers
type License struct {
	ID           uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CustomerID   uuid.UUID     `gorm:"type:uuid;not null;index" json:"customer_id"`
	LicenseKey   string        `gorm:"type:varchar(255);uniqueIndex;not null" json:"license_key"`
	LicenseType  LicenseType   `gorm:"type:varchar(50);default:'standard'" json:"license_type"`
	Status       LicenseStatus `gorm:"type:varchar(50);default:'active';index" json:"status"`
	ExpiresAt    time.Time     `gorm:"not null" json:"expires_at"`
	MaxInstances int           `gorm:"default:1" json:"max_instances"`
	Features     JSONMap       `gorm:"type:jsonb;default:'{}'" json:"features"`
	Metadata     JSONMap       `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt    time.Time     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time     `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Customer Customer `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
}

// TableName sets the table name for License
func (License) TableName() string {
	return "licenses"
}

// BeforeCreate generates a UUID and license key before creating a new License
func (l *License) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	if l.LicenseKey == "" {
		l.LicenseKey = generateLicenseKey()
	}
	return nil
}

// IsValid checks if the license is currently valid
func (l *License) IsValid() bool {
	if l.Status != LicenseStatusActive {
		return false
	}
	return time.Now().Before(l.ExpiresAt)
}

// generateLicenseKey generates a new license key
func generateLicenseKey() string {
	// Simple license key generation - in production use crypto/rand
	return uuid.New().String() + "-" + uuid.New().String()
}
