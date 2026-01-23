package master

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InstanceStatus represents the status of a customer instance
type InstanceStatus string

const (
	InstanceStatusPending      InstanceStatus = "pending"
	InstanceStatusProvisioning InstanceStatus = "provisioning"
	InstanceStatusActive       InstanceStatus = "active"
	InstanceStatusSuspended    InstanceStatus = "suspended"
	InstanceStatusTerminated   InstanceStatus = "terminated"
)

// Instance represents a customer video tube site instance
type Instance struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CustomerID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"customer_id"`
	Name          string         `gorm:"type:varchar(255);not null" json:"name"`
	Subdomain     string         `gorm:"type:varchar(63);uniqueIndex;not null" json:"subdomain"`
	CustomDomains StringArray    `gorm:"type:text[];default:'{}'" json:"custom_domains"`
	Status        InstanceStatus `gorm:"type:varchar(50);default:'pending';index" json:"status"`
	PlanID        *uuid.UUID     `gorm:"type:uuid" json:"plan_id"`
	DatabaseName  string         `gorm:"type:varchar(63);not null" json:"database_name"`
	StorageBucket string         `gorm:"type:varchar(63);not null" json:"storage_bucket"`
	Metadata      JSONMap        `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt     time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	ActivatedAt   *time.Time     `json:"activated_at"`

	// Relationships
	Customer       Customer         `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Subscription   *Subscription    `gorm:"foreignKey:PlanID;references:PlanID" json:"subscription,omitempty"`
	InstanceConfig []InstanceConfig `gorm:"foreignKey:InstanceID" json:"instance_config,omitempty"`
	UsageMetrics   []UsageMetrics   `gorm:"foreignKey:InstanceID" json:"usage_metrics,omitempty"`
}

// TableName sets the table name for Instance
func (Instance) TableName() string {
	return "instances"
}

// BeforeCreate generates a UUID before creating a new Instance
func (i *Instance) BeforeCreate(tx *gorm.DB) error {
	if i.ID == uuid.Nil {
		i.ID = uuid.New()
	}
	return nil
}

// StringArray is a custom type for PostgreSQL text array
type StringArray []string
