package master

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InstanceConfig stores branding and settings for each instance
type InstanceConfig struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	InstanceID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_instance_config_key" json:"instance_id"`
	ConfigKey   string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_instance_config_key" json:"config_key"`
	ConfigValue string    `gorm:"type:text;not null" json:"config_value"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Instance Instance `gorm:"foreignKey:InstanceID" json:"instance,omitempty"`
}

// TableName sets the table name for InstanceConfig
func (InstanceConfig) TableName() string {
	return "instance_config"
}

// BeforeCreate generates a UUID before creating a new InstanceConfig
func (ic *InstanceConfig) BeforeCreate(tx *gorm.DB) error {
	if ic.ID == uuid.Nil {
		ic.ID = uuid.New()
	}
	return nil
}

// Common instance configuration keys
const (
	ConfigSiteName        = "site_name"
	ConfigLogoURL         = "logo_url"
	ConfigFaviconURL      = "favicon_url"
	ConfigPrimaryColor    = "primary_color"
	ConfigSecondaryColor  = "secondary_color"
	ConfigAccentColor     = "accent_color"
	ConfigBackgroundColor = "background_color"
	ConfigTextColor       = "text_color"
	ConfigHeaderHTML      = "header_html"
	ConfigFooterHTML      = "footer_html"
	ConfigCustomCSS       = "custom_css"
	ConfigSocialLinks     = "social_links"
	ConfigFooterLinks     = "footer_links"
)
