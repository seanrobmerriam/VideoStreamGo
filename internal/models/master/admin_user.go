package master

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AdminRole represents the role of an admin user
type AdminRole string

const (
	AdminRoleSuperAdmin AdminRole = "super_admin"
	AdminRoleAdmin      AdminRole = "admin"
	AdminRoleModerator  AdminRole = "moderator"
	AdminRoleSupport    AdminRole = "support"
)

// AdminStatus represents the status of an admin user
type AdminStatus string

const (
	AdminStatusActive    AdminStatus = "active"
	AdminStatusInactive  AdminStatus = "inactive"
	AdminStatusSuspended AdminStatus = "suspended"
)

// AdminUser represents a platform admin user
type AdminUser struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email        string          `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string          `gorm:"type:varchar(255);not null" json:"-"`
	DisplayName  string          `gorm:"type:varchar(100)" json:"display_name"`
	Role         AdminRole       `gorm:"type:varchar(50);default:'admin'" json:"role"`
	Status       AdminStatus     `gorm:"type:varchar(50);default:'active';index" json:"status"`
	Permissions  JSONStringArray `gorm:"type:jsonb;default:'[]'" json:"permissions"`
	LastLoginAt  *time.Time      `json:"last_login_at"`
	CreatedAt    time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time       `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName sets the table name for AdminUser
func (AdminUser) TableName() string {
	return "admin_users"
}

// BeforeCreate generates a UUID before creating a new AdminUser
func (au *AdminUser) BeforeCreate(tx *gorm.DB) error {
	if au.ID == uuid.Nil {
		au.ID = uuid.New()
	}
	return nil
}

// IsSuperAdmin checks if the admin user is a super admin
func (au *AdminUser) IsSuperAdmin() bool {
	return au.Role == AdminRoleSuperAdmin
}

// HasPermission checks if the admin user has a specific permission
func (au *AdminUser) HasPermission(permission string) bool {
	if au.IsSuperAdmin() {
		return true
	}
	for _, p := range au.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// PlatformSettings represents platform-wide settings
type PlatformSettings struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Key         string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"key"`
	Value       string    `gorm:"type:text;not null" json:"value"`
	Description string    `gorm:"type:text" json:"description"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName sets the table name for PlatformSettings
func (PlatformSettings) TableName() string {
	return "platform_settings"
}

// Common platform settings keys
const (
	PlatformSettingMaintenanceMode   = "maintenance_mode"
	PlatformSettingAllowNewSignups   = "allow_new_signups"
	PlatformSettingDefaultPlanID     = "default_plan_id"
	PlatformSettingSupportEmail      = "support_email"
	PlatformSettingTermsOfServiceURL = "tos_url"
	PlatformSettingPrivacyPolicyURL  = "privacy_policy_url"
)
